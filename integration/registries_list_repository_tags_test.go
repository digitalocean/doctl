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

var _ = suite("registries/list-repository-tags", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			// Use regex to match the URL pattern: /v2/registry/{registry_name}/repositories/{repository_name}/tags
			tagsRegex := regexp.MustCompile(`/v2/registry/([^/]+)/repositories/([^/]+)/tags`)
			match := tagsRegex.FindStringSubmatch(req.URL.Path)

			if len(match) > 2 {
				registryName := match[1]
				repositoryName := match[2]
				fmt.Fprintf(w, `{
					"tags": [
						{
							"registry_name": "%s",
							"repository": "%s",
							"tag": "latest",
							"manifest_digest": "sha256:abcd1234567890",
							"compressed_size_bytes": 1024,
							"size_bytes": 2048,
							"updated_at": "2023-01-01T00:00:00Z"
						},
						{
							"registry_name": "%s",
							"repository": "%s", 
							"tag": "v1.0.0",
							"manifest_digest": "sha256:efgh0987654321",
							"compressed_size_bytes": 2048,
							"size_bytes": 4096,
							"updated_at": "2023-01-02T00:00:00Z"
						}
					]
				}`, registryName, repositoryName, registryName, repositoryName)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		}))
	})

	when("listing repository tags", func() {
		it("lists tags for a specific repository using registries command", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registries",
				"list-repository-tags",
				"test-registry",
				"test-repo",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "latest")
			expect.Contains(string(output), "v1.0.0")
			expect.Contains(string(output), "sha256:abcd1234567890")
			expect.Contains(string(output), "sha256:efgh0987654321")
		})
	})

	it.After(func() {
		server.Close()
	})
})
