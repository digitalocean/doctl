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

var _ = suite("registries/repository/delete-tag", func(t *testing.T, when spec.G, it spec.S) {
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
			case "/v2/registries/my-registry/repositories/my-repo/tags":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(`{
					"tags": [
						{
							"registry_name": "my-registry",
							"repository": "my-repo",
							"tag": "my-tag",
							"manifest_digest": "sha256:abcd",
							"compressed_size_bytes": 1,
							"size_bytes": 2,
							"updated_at": "2023-01-01T00:00:00Z"
						}
					]
				}`))
			case "/v2/registries/my-registry/repositories/my-repo/tags/my-tag":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			case "/v2/registries/my-registry/repositories/my-repo/tags/missing-tag":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				// API historically returns success for missing tags; CLI must
				// reject before this path is reached.
				w.WriteHeader(http.StatusNoContent)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("deletes repository tag", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registries",
			"repository",
			"delete-tag",
			"my-registry",
			"my-repo",
			"my-tag",
			"--force",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal("Successfully deleted 1 tag(s)", strings.TrimSpace(string(output)))
	})

	it("errors when the tag is missing", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"registries",
			"repository",
			"delete-tag",
			"my-registry",
			"my-repo",
			"missing-tag",
			"--force",
		)

		output, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), `tag "missing-tag" not found in repository my-registry/my-repo`)
	})
})
