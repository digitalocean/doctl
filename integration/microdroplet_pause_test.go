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

var _ = suite("compute/microdroplet/pause", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/microdroplets/b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2/pause":
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
				expect.Empty(strings.TrimSpace(string(reqBody)), "pause request body should be empty")

				w.Write([]byte(microDropletPauseResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("the id is provided", func() {
		it("pauses the microdroplet", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microdroplet", "pause",
				"b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(microDropletPauseOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	microDropletPauseOutput = `
ID                                      Name                  Region    State     Size                  Networking    Source                          Endpoint                           Ports    Created At
b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2    sammy-microdroplet    nyc1      paused    2vCPU/4096MiB/80GB    public        docker.io/library/nginx:1.27    sammy.microdroplets.example.com    8080     2026-07-16T10:00:00Z
`
	microDropletPauseResponse = `
{
  "micro_droplet": {
    "id": "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
    "name": "sammy-microdroplet",
    "region": "nyc1",
    "state": "paused",
    "size": {"cpu": 2, "memory": 4096, "disk": 80},
    "networking": "public",
    "source": {"oci_ref": "docker.io/library/nginx:1.27"},
    "urls": [{"hostname": "sammy.microdroplets.example.com", "port": 8080, "default": true, "status": "ACTIVE"}],
    "ports": [8080],
    "created_at": "2026-07-16T10:00:00Z"
  }
}
`
)
