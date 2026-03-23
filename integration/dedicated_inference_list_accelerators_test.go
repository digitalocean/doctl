package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("dedicated-inference/list-accelerators", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/dedicated-inferences/00000000-0000-4000-8000-000000000000/accelerators":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				// Check for slug filter query param
				slugFilter := req.URL.Query().Get("slug")
				if slugFilter == "gpu-mi300x1-192gb" {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(dedicatedInferenceListAcceleratorsFilteredResponse))
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(dedicatedInferenceListAcceleratorsResponse))
			case "/v2/dedicated-inferences/99999999-9999-4999-8999-999999999999/accelerators":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"id":"not_found","message":"The resource you requested could not be found."}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("valid dedicated inference ID is provided", func() {
		it("lists the accelerators", func() {
			aliases := []string{"list-accelerators", "la"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"dedicated-inference",
					alias,
					"00000000-0000-4000-8000-000000000000",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output for alias %q: %s", alias, output))
				expect.Equal(strings.TrimSpace(dedicatedInferenceListAcceleratorsOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("dedicated inference ID is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"list-accelerators",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing")
		})
	})

	when("dedicated inference does not exist", func() {
		it("returns a not found error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"list-accelerators",
				"99999999-9999-4999-8999-999999999999",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "404")
		})
	})

	when("slug filter is provided", func() {
		it("lists only matching accelerators", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"list-accelerators",
				"00000000-0000-4000-8000-000000000000",
				"--slug", "gpu-mi300x1-192gb",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceListAcceleratorsFilteredOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a format flag", func() {
		it("displays only those columns", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"list-accelerators",
				"00000000-0000-4000-8000-000000000000",
				"--format", "ID,Slug,Status",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceListAcceleratorsFormatOutput), strings.TrimSpace(string(output)))
		})
	})

	when("using the di alias", func() {
		it("lists the accelerators", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"di",
				"list-accelerators",
				"00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceListAcceleratorsOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dedicatedInferenceListAcceleratorsOutput = `
ID         Name             Slug                 Status    Created At
acc-001    prefill-gpu-1    gpu-mi300x1-192gb    ACTIVE    2023-01-01 00:00:00 +0000 UTC
acc-002    decode-gpu-1     gpu-h100x1-80gb      ACTIVE    2023-01-01 00:00:00 +0000 UTC
`
	dedicatedInferenceListAcceleratorsFilteredOutput = `
ID         Name             Slug                 Status    Created At
acc-001    prefill-gpu-1    gpu-mi300x1-192gb    ACTIVE    2023-01-01 00:00:00 +0000 UTC
`
	dedicatedInferenceListAcceleratorsFormatOutput = `
ID         Slug                 Status
acc-001    gpu-mi300x1-192gb    ACTIVE
acc-002    gpu-h100x1-80gb      ACTIVE
`

	dedicatedInferenceListAcceleratorsResponse = `
{
  "accelerators": [
    {
      "id": "acc-001",
      "name": "prefill-gpu-1",
      "slug": "gpu-mi300x1-192gb",
      "status": "ACTIVE",
      "created_at": "2023-01-01T00:00:00Z"
    },
    {
      "id": "acc-002",
      "name": "decode-gpu-1",
      "slug": "gpu-h100x1-80gb",
      "status": "ACTIVE",
      "created_at": "2023-01-01T00:00:00Z"
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}
`
	dedicatedInferenceListAcceleratorsFilteredResponse = `
{
  "accelerators": [
    {
      "id": "acc-001",
      "name": "prefill-gpu-1",
      "slug": "gpu-mi300x1-192gb",
      "status": "ACTIVE",
      "created_at": "2023-01-01T00:00:00Z"
    }
  ],
  "links": {},
  "meta": {
    "total": 1
  }
}
`
)
