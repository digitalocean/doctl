package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
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

			auth := req.Header.Get("Authorization")
			if auth != "Bearer some-magic-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Match the correct API patterns for registries
			switch {
			case strings.HasSuffix(req.URL.Path, "/garbage-collection") && req.Method == "POST":
				// Start garbage collection: POST /v2/registries/{registry}/garbage-collection
				registryName := strings.Split(req.URL.Path, "/")[3] // Extract registry name from path
				w.WriteHeader(http.StatusCreated)
				fmt.Fprintf(w, `{
					"garbage_collection": {
						"uuid": "gc-12345",
						"registry_name": "%s",
						"status": "requested",
						"created_at": "2023-01-01T00:00:00Z",
						"updated_at": "2023-01-01T00:00:00Z",
						"blobs_deleted": 0,
						"freed_bytes": 0
					}
				}`, registryName)
			case strings.HasSuffix(req.URL.Path, "/garbage-collection") && req.Method == "GET":
				// Get active garbage collection: GET /v2/registries/{registry}/garbage-collection
				registryName := strings.Split(req.URL.Path, "/")[3]
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
			case strings.HasSuffix(req.URL.Path, "/garbage-collections") && req.Method == "GET":
				// List garbage collections: GET /v2/registries/{registry}/garbage-collections
				registryName := strings.Split(req.URL.Path, "/")[3]
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
			default:
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, `{"error": "Not found", "path": "%s"}`, req.URL.Path)
			}
		}))
	})

	when("starting garbage collection", func() {
		it("starts garbage collection using registries command", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registries",
				"garbage-collection",
				"start",
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
				"garbage-collection",
				"list",
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
				"garbage-collection",
				"get-active",
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
