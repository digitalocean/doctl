package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"regexp"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("registries/list-repositories", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			// Use regex to match the URL pattern: /v2/registry/{registry_name}/repositories
			registryNameRegex := regexp.MustCompile(`/v2/registry/([^/]+)/repositories`)
			match := registryNameRegex.FindStringSubmatch(req.URL.Path)

			if len(match) > 1 {
				registryName := match[1]
				fmt.Fprintf(w, `{
					"repositories": [
						{
							"registry_name": "%s",
							"name": "test-repo-1",
							"latest_tag": {
								"registry_name": "%s",
								"repository": "test-repo-1",
								"tag": "latest",
								"manifest_digest": "sha256:abcd1234",
								"compressed_size_bytes": 1024,
								"size_bytes": 2048,
								"updated_at": "2023-01-01T00:00:00Z"
							},
							"tag_count": 1
						},
						{
							"registry_name": "%s", 
							"name": "test-repo-2",
							"latest_tag": {
								"registry_name": "%s",
								"repository": "test-repo-2", 
								"tag": "v1.0.0",
								"manifest_digest": "sha256:efgh5678",
								"compressed_size_bytes": 2048,
								"size_bytes": 4096,
								"updated_at": "2023-01-02T00:00:00Z"
							},
							"tag_count": 2
						}
					]
				}`, registryName, registryName, registryName, registryName)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		}))
	})

	when("listing repositories", func() {
		it("lists repositories for a specific registry", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registry",
				"repository",
				"list",
				"--registry", "test-registry",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "test-repo-1")
			expect.Contains(string(output), "test-repo-2")
		})
	})

	when("using the new registries command", func() {
		it("lists repositories using registries command", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registries",
				"list-repositories",
				"test-registry",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "test-repo-1")
			expect.Contains(string(output), "test-repo-2")
		})
	})

	it.After(func() {
		server.Close()
	})
})
