package integration

import (
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

var _ = suite("compute/microvm/update", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect      *require.Assertions
		server      *httptest.Server
		expectedReq string
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/microvms/b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := io.ReadAll(req.Body)
				expect.NoError(err)
				expect.JSONEq(expectedReq, string(reqBody))

				w.Write([]byte(microVMUpdateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("name, tags, and idle timeout are provided", func() {
		it("updates the microvm", func() {
			expectedReq = `{"name":"renamed-microvm","tags":["prod","web"],"auto_pause":{"idle_timeout":"10m"}}`
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microvm", "update",
				"b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
				"--name", "renamed-microvm",
				"--tag", "prod",
				"--tag", "web",
				"--auto-pause-idle-timeout", "10m",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(microVMUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("clear-tags is provided", func() {
		it("sends an empty tag list", func() {
			expectedReq = `{"tags":[]}`
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microvm", "update",
				"b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
				"--clear-tags",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		})
	})

	when("no field is provided", func() {
		it("returns an error without calling the API", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microvm", "update",
				"b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "set at least one of")
		})
	})
})

const (
	microVMUpdateOutput = `
ID                                      Name               Region    State      Size                  Networking    Source                          Endpoint                      Ports    Protocol    Tags        Failure Reason    Created At
b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2    renamed-microvm    nyc1      running    2vCPU/4096MiB/80GB    public        docker.io/library/nginx:1.27    sammy.microvms.example.com    8080     http        prod,web                      2026-07-16T10:00:00Z
`
	microVMUpdateResponse = `
{
  "microvm": {
    "id": "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
    "name": "renamed-microvm",
    "region": "nyc1",
    "state": "running",
    "size": {"cpu": 2, "memory": 4096, "disk": 80},
    "networking": "public",
    "source": {"oci_ref": "docker.io/library/nginx:1.27"},
    "urls": [{"hostname": "sammy.microvms.example.com", "port": 8080, "default": true, "status": "ACTIVE"}],
    "ports": [8080],
    "http_protocol": "http",
    "tags": ["prod", "web"],
    "auto_pause": {"enabled": true, "idle_timeout": "10m"},
    "created_at": "2026-07-16T10:00:00Z"
  }
}
`
)
