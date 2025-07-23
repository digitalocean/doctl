package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("registries/garbage-collection", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			// Use regex to match garbage collection URLs
			if strings.Contains(req.URL.Path, "/garbage-collection") {
				registryGCRegex := regexp.MustCompile(`/v2/registry/([^/]+)/garbage-collection`)
				match := registryGCRegex.FindStringSubmatch(req.URL.Path)

				if len(match) > 1 {
					registryName := match[1]

					switch req.Method {
					case "POST":
						// Start garbage collection
						fmt.Fprintf(w, `{
							"garbage_collection": {
								"uuid": "gc-12345",
								"registry_name": "%s",
								"status": "requested",
								"created_at": "2023-01-01T00:00:00Z"
							}
						}`, registryName)
					case "GET":
						// Get or list garbage collections
						if strings.Contains(req.URL.Path, "gc-12345") {
							// Get specific GC
							fmt.Fprintf(w, `{
								"garbage_collection": {
									"uuid": "gc-12345",
									"registry_name": "%s",
									"status": "completed",
									"created_at": "2023-01-01T00:00:00Z",
									"updated_at": "2023-01-01T01:00:00Z",
									"blobs_deleted": 10,
									"freed_bytes": 1024000
								}
							}`, registryName)
						} else {
							// List all GCs
							fmt.Fprintf(w, `{
								"garbage_collections": [
									{
										"uuid": "gc-12345",
										"registry_name": "%s",
										"status": "completed",
										"created_at": "2023-01-01T00:00:00Z",
										"updated_at": "2023-01-01T01:00:00Z",
										"blobs_deleted": 10,
										"freed_bytes": 1024000
									}
								]
							}`, registryName)
						}
					}
				}
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		}))
	})

	when("starting garbage collection", func() {
		it("starts garbage collection using registries command", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registries",
				"start-garbage-collection",
				"test-registry",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "gc-12345")
			expect.Contains(string(output), "requested")
		})
	})

	when("listing garbage collections", func() {
		it("lists garbage collections using registries command", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registries",
				"list-garbage-collections",
				"test-registry",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "gc-12345")
			expect.Contains(string(output), "completed")
		})
	})

	when("getting garbage collection", func() {
		it("gets active garbage collection using registries command", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registries",
				"get-garbage-collection",
				"test-registry",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "gc-12345")
			expect.Contains(string(output), "completed")
		})
	})

	it.After(func() {
		server.Close()
	})
})
