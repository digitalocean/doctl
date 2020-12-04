package integration

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var (
	validRegistryName     = "my-registry"
	invalidRegistryName   = "not-my-registry"
	testGCBlobsDeleted    = uint64(42)
	testGCFreedBytes      = uint64(666)
	testGCStatus          = "requested"
	validGCUUID           = "mew-mew-id"
	invalidGCUUID         = "invalid-gc-uuid"
	testTime              = time.Date(2020, 4, 1, 0, 0, 0, 0, time.UTC)
	testGarbageCollection = &godo.GarbageCollection{
		UUID:         validGCUUID,
		RegistryName: validRegistryName,
		Status:       testGCStatus,
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
		BlobsDeleted: testGCBlobsDeleted,
		FreedBytes:   testGCFreedBytes,
	}
	gcResponseJSONTmpl = `
{
  "garbage_collection": {
    "uuid": "{{.UUID}}",
    "registry_name": "{{.RegistryName}}",
    "status": "{{.Status}}",
    "created_at": "{{.CreatedAt.Format "2006-01-02T15:04:05Z07:00"}}",
    "updated_at": "{{.UpdatedAt.Format "2006-01-02T15:04:05Z07:00"}}",
    "blobs_deleted": {{.BlobsDeleted}},
    "freed_bytes": {{.FreedBytes}}
  }
}`
	gcListResponseJSONTmpl = `
{
  "garbage_collections": [
    {
      "uuid": "{{.UUID}}",
      "registry_name": "{{.RegistryName}}",
      "status": "{{.Status}}",
      "created_at": "{{.CreatedAt.Format "2006-01-02T15:04:05Z07:00"}}",
      "updated_at": "{{.UpdatedAt.Format "2006-01-02T15:04:05Z07:00"}}",
      "blobs_deleted": {{.BlobsDeleted}},
      "freed_bytes": {{.FreedBytes}}
    }
  ],
	"meta": {
	    "total": 1
	}
}`
	gcGetOutput = `
UUID          Registry Name    Status       Created At                       Updated At                       Blobs Deleted    Bytes Freed
mew-mew-id    my-registry      requested    2020-04-01 00:00:00 +0000 UTC    2020-04-01 00:00:00 +0000 UTC    42               666
`
)

var _ = suite("registry/garbage-collection", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		gcResponseJSON := reifyTemplateStr(t, gcResponseJSONTmpl, testGarbageCollection)
		gcListResponseJSON := reifyTemplateStr(t, gcListResponseJSONTmpl, testGarbageCollection)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")
			auth := req.Header.Get("Authorization")

			if auth != "Bearer some-magic-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			switch req.URL.Path {
			case "/v2/registry":
				switch req.Method {
				case http.MethodGet:
					w.Write([]byte(registryGetResponse))
				default:
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			case "/v2/registry/" + invalidRegistryName + "/garbage-collection":
				w.WriteHeader(http.StatusNotFound)
			case "/v2/registry/" + validRegistryName + "/garbage-collection":
				switch req.Method {
				case http.MethodPost:
					w.WriteHeader(http.StatusCreated)
					w.Write([]byte(gcResponseJSON))
				case http.MethodGet:
					w.Write([]byte(gcResponseJSON))
				default:
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			case "/v2/registry/" + invalidRegistryName + "/garbage-collections":
				w.WriteHeader(http.StatusNotFound)
			case "/v2/registry/" + validRegistryName + "/garbage-collections":
				switch req.Method {
				case http.MethodGet:
					w.Write([]byte(gcListResponseJSON))
				default:
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			case "/v2/registry/" + invalidRegistryName + "/garbage-collection/" + validGCUUID,
				"/v2/registry/" + validRegistryName + "/garbage-collection/" + invalidGCUUID:
				w.WriteHeader(http.StatusNotFound)
			case "/v2/registry/" + validRegistryName + "/garbage-collection/" + validGCUUID:
				switch req.Method {
				case http.MethodPut:
					reqBody, err := ioutil.ReadAll(req.Body)
					expect.NoError(err)

					expectJSON := &strings.Builder{}
					err = json.NewEncoder(expectJSON).Encode(&godo.UpdateGarbageCollectionRequest{
						Cancel: true,
					})
					expect.NoError(err)
					expect.JSONEq(expectJSON.String(), string(reqBody))

					w.Write([]byte(gcResponseJSON))
				default:
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("starts a garbage collection", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"garbage-collection",
			"start", "--force",
		)
		output, err := cmd.CombinedOutput()
		expect.Equal(strings.TrimSpace(gcGetOutput), strings.TrimSpace(string(output)))
		expect.NoError(err)
	})

	it("gets the active garbage collection", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"garbage-collection",
			"get-active",
		)
		output, err := cmd.CombinedOutput()
		expect.Equal(strings.TrimSpace(gcGetOutput), strings.TrimSpace(string(output)))
		expect.NoError(err)
	})

	it("lists all garbage collections", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"garbage-collection",
			"list",
		)
		output, err := cmd.CombinedOutput()
		expect.Equal(strings.TrimSpace(gcGetOutput), strings.TrimSpace(string(output)))
		expect.NoError(err)
	})

	it("fails to cancel garbage collection/when no gc uuid given", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"garbage-collection",
			"cancel",
		)
		output, err := cmd.CombinedOutput()
		expectMsg := "Error: (garbage-collection.cancel) command is missing required arguments"
		expect.Equal(strings.TrimSpace(expectMsg), strings.TrimSpace(string(output)))
		expect.Error(err)
	})

	it("fails to cancel garbage collection/when invalid gc uuid given", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"garbage-collection",
			"cancel",
			"invalid-gc-uuid",
		)
		output, err := cmd.CombinedOutput()
		expectMsg := "Error: PUT " + server.URL + "/v2/registry/" + validRegistryName + "/garbage-collection/" + invalidGCUUID + ": 404"
		expect.Equal(strings.TrimSpace(expectMsg), strings.TrimSpace(string(output)))
		expect.Error(err)
	})

	it("fails to cancel garbage collection/when invalid registry name given", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"garbage-collection",
			"cancel",
			invalidRegistryName,
			validGCUUID,
		)
		output, err := cmd.CombinedOutput()
		expectMsg := "Error: PUT " + server.URL + "/v2/registry/" + invalidRegistryName + "/garbage-collection/" + validGCUUID + ": 404"
		expect.Equal(strings.TrimSpace(expectMsg), strings.TrimSpace(string(output)))
		expect.Error(err)
	})

	it("cancels garbage collection/when valid gc uuid given", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"garbage-collection",
			"cancel",
			validGCUUID,
		)
		output, err := cmd.CombinedOutput()
		expect.Equal("", strings.TrimSpace(string(output)))
		expect.NoError(err)
	})

	it("cancels garbage collection/when valid registry name and gc uuid given", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"garbage-collection",
			"cancel",
			validRegistryName,
			validGCUUID,
		)
		output, err := cmd.CombinedOutput()
		expect.Equal("", strings.TrimSpace(string(output)))
		expect.NoError(err)
	})
})

func reifyTemplateStr(t *testing.T, tmplStr string, v interface{}) string {
	tmpl, err := template.New("meow").Parse(tmplStr)
	require.NoError(t, err)

	s := &strings.Builder{}
	err = tmpl.Execute(s, v)
	require.NoError(t, err)

	return s.String()
}
