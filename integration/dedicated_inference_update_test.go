package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("dedicated-inference/update", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/dedicated-inferences/00000000-0000-4000-8000-000000000000":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal("failed to read request body")
				}

				var payload map[string]interface{}
				err = json.Unmarshal(body, &payload)
				if err != nil {
					t.Fatalf("failed to parse request body: %s", err)
				}

				// Verify the spec is present in the request
				if _, ok := payload["spec"]; !ok {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"id":"bad_request","message":"spec is required"}`))
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(dedicatedInferenceUpdateResponse))
			case "/v2/dedicated-inferences/99999999-9999-4999-8999-999999999999":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPatch {
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

	when("valid ID and spec are provided", func() {
		it("updates the dedicated inference endpoint", func() {
			specFile := createDedicatedInferenceUpdateSpecFile(t)

			aliases := []string{"update", "u"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"dedicated-inference",
					alias,
					"00000000-0000-4000-8000-000000000000",
					"--spec", specFile,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output for alias %q: %s", alias, output))
				expect.Equal(strings.TrimSpace(dedicatedInferenceUpdateOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("dedicated inference ID is missing", func() {
		it("returns an error", func() {
			specFile := createDedicatedInferenceUpdateSpecFile(t)

			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"update",
				"--spec", specFile,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing")
		})
	})

	when("spec flag is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"update",
				"00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "spec")
		})
	})

	when("dedicated inference does not exist", func() {
		it("returns a not found error", func() {
			specFile := createDedicatedInferenceUpdateSpecFile(t)

			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"update",
				"99999999-9999-4999-8999-999999999999",
				"--spec", specFile,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "404")
		})
	})

	when("using the di alias", func() {
		it("updates the dedicated inference endpoint", func() {
			specFile := createDedicatedInferenceUpdateSpecFile(t)

			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"di",
				"update",
				"00000000-0000-4000-8000-000000000000",
				"--spec", specFile,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a format flag", func() {
		it("displays only those columns", func() {
			specFile := createDedicatedInferenceUpdateSpecFile(t)

			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"update",
				"00000000-0000-4000-8000-000000000000",
				"--spec", specFile,
				"--format", "ID,Name,Status",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceUpdateFormatOutput), strings.TrimSpace(string(output)))
		})
	})
})

func createDedicatedInferenceUpdateSpecFile(t *testing.T) string {
	t.Helper()
	specJSON := `{
		"version": 0,
		"name": "updated-dedicated-inference",
		"region": "nyc2",
		"vpc": {"uuid": "00000000-0000-4000-8000-000000000001"},
		"enable_public_endpoint": true,
		"model_deployments": [
			{
				"model_slug": "mistral/mistral-7b-instruct-v3",
				"model_provider": "hugging_face",
				"accelerators": [
					{"scale": 3, "type": "prefill", "accelerator_slug": "gpu-mi300x1-192gb"},
					{"scale": 6, "type": "decode", "accelerator_slug": "gpu-mi300x1-192gb"}
				]
			}
		]
	}`
	tmpFile := t.TempDir() + "/update-spec.json"
	err := os.WriteFile(tmpFile, []byte(specJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write spec file: %s", err)
	}
	return tmpFile
}

const (
	dedicatedInferenceUpdateOutput = `
ID                                      Name                           Region    Status      VPC UUID                                Public Endpoint                           Private Endpoint                           Created At                       Updated At
00000000-0000-4000-8000-000000000000    updated-dedicated-inference    nyc2      UPDATING    00000000-0000-4000-8000-000000000001    public.dedicated-inference.example.com    private.dedicated-inference.example.com    2023-01-01 00:00:00 +0000 UTC    2023-01-02 00:00:00 +0000 UTC
`
	dedicatedInferenceUpdateFormatOutput = `
ID                                      Name                           Status
00000000-0000-4000-8000-000000000000    updated-dedicated-inference    UPDATING
`

	dedicatedInferenceUpdateResponse = `
{
  "dedicated_inference": {
    "id": "00000000-0000-4000-8000-000000000000",
    "name": "updated-dedicated-inference",
    "region": "nyc2",
    "status": "UPDATING",
    "vpc_uuid": "00000000-0000-4000-8000-000000000001",
    "endpoints": {
      "public_endpoint_fqdn": "public.dedicated-inference.example.com",
      "private_endpoint_fqdn": "private.dedicated-inference.example.com"
    },
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-02T00:00:00Z"
  }
}
`
)
