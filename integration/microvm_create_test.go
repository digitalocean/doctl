package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/microvm/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/microvms":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := io.ReadAll(req.Body)
				expect.NoError(err)

				var got map[string]any
				expect.NoError(json.Unmarshal(reqBody, &got))
				expect.Equal("sammy-microvm", got["name"])
				expect.Equal("nyc1", got["region"])
				expect.Equal(map[string]any{"cpu": float64(2), "memory": float64(4096)}, got["size"])
				expect.Equal(map[string]any{"oci_ref": "docker.io/library/nginx:1.27"}, got["source"])

				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(microVMCreateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("all required flags are passed", func() {
		it("creates a microvm", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microvm", "create", "sammy-microvm",
				"--region", "nyc1",
				"--cpu", "2",
				"--memory", "4096",
				"--oci-ref", "docker.io/library/nginx:1.27",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(microVMCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("missing required flags", func() {
		it("returns an error", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", "https://www.example.com",
				"compute", "microvm", "create", "sammy-microvm",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "exactly one of --oci-ref or --checkpoint-id is required")
		})
	})
})

const (
	microVMCreateOutput = `
ID                                      Name             Region    State       Size                  Networking    Source                          Endpoint    Ports    Protocol    Tags    Failure Reason    Created At
b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2    sammy-microvm    nyc1      creating    2vCPU/4096MiB/80GB    public        docker.io/library/nginx:1.27                8080     http        prod                      2026-07-16T10:00:00Z
`
	microVMCreateResponse = `
{
  "microvm": {
    "id": "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
    "name": "sammy-microvm",
    "region": "nyc1",
    "state": "creating",
    "size": {"cpu": 2, "memory": 4096, "disk": 80},
    "networking": "public",
    "source": {"oci_ref": "docker.io/library/nginx:1.27"},
    "urls": [{"hostname": "", "port": 8080, "default": true, "status": "PENDING"}],
    "ports": [8080],
    "http_protocol": "http",
    "tags": ["prod"],
    "created_at": "2026-07-16T10:00:00Z"
  }
}
`
)
