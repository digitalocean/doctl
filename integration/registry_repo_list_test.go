package integration

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("registry/repository/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/registry":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(registryGetResponse))
			case "/v2/registry/my-registry/repositories":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(repositoryListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("returns list of repositories in registry", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registry",
			"repository",
			"list",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(repositoryListOutput), strings.TrimSpace(string(output)))
	})
})

var (
	repositoryListOutput = `
Name       Latest Tag    Tag Count    Updated At
my-repo    my-tag        3            2020-04-01 00:00:00 +0000 UTC 
`
	repositoryListResponse = `{
		"repositories": [
			{
				"registry_name": "my-registry",
				"name": "my-repo",
				"tag_count": 3,
				"latest_tag": {
					"registry_name": "my-registry",
					"repository": "my-repo",
					"tag": "my-tag",
					"updated_at": "2020-04-01T00:00:00Z"
				}
			}
		],
		"meta": {
			"total": 1
		}
	}`
)
