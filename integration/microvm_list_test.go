package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/microvm/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect   *require.Assertions
		server   *httptest.Server
		gotQuery url.Values
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				gotQuery = req.URL.Query()
				w.Write([]byte(microVMListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("no flags are passed", func() {
		it("lists all microvms", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microvm", "list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(microVMListOutput), strings.TrimSpace(string(output)))
			expect.Empty(gotQuery.Get("region"))
			expect.Empty(gotQuery.Get("name"))
			expect.Empty(gotQuery.Get("tag_name"))
		})
	})

	when("a region is provided", func() {
		it("filters microvms by region", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microvm", "list",
				"--region", "nyc1",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(microVMListOutput), strings.TrimSpace(string(output)))
			expect.Equal("nyc1", gotQuery.Get("region"))
		})
	})

	when("name and tag are provided", func() {
		it("filters microvms by name and tag", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microvm", "list",
				"--name", "sammy-microvm",
				"--tag-name", "prod",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(microVMListOutput), strings.TrimSpace(string(output)))
			expect.Equal("sammy-microvm", gotQuery.Get("name"))
			expect.Equal("prod", gotQuery.Get("tag_name"))
		})
	})
})

const (
	microVMListOutput = `
ID                                      Name             Region    State      Size                  Networking    Source                          Endpoint                      Ports    Protocol    Tags    Failure Reason    Created At
b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2    sammy-microvm    nyc1      running    2vCPU/4096MiB/80GB    public        docker.io/library/nginx:1.27    sammy.microvms.example.com    8080     http        prod                      2026-07-16T10:00:00Z
`
	microVMListResponse = `
{
  "microvms": [
    {
      "id": "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
      "name": "sammy-microvm",
      "region": "nyc1",
      "state": "running",
      "size": {"cpu": 2, "memory": 4096, "disk": 80},
      "networking": "public",
      "source": {"oci_ref": "docker.io/library/nginx:1.27"},
      "urls": [{"hostname": "sammy.microvms.example.com", "port": 8080, "default": true, "status": "ACTIVE"}],
      "ports": [8080],
      "http_protocol": "http",
      "tags": ["prod"],
      "created_at": "2026-07-16T10:00:00Z"
    }
  ],
  "links": {},
  "meta": {"total": 1}
}
`
)
