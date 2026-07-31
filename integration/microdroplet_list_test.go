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

var _ = suite("compute/microdroplet/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/microdroplets/instances":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				if region := req.URL.Query().Get("region"); region == "nyc1" {
					w.Write([]byte(microDropletListResponse))
					return
				}

				w.Write([]byte(microDropletListResponse))
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
		it("lists all microdroplets", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microdroplet", "list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(microDropletListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("a region is provided", func() {
		it("filters microdroplets by region", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microdroplet", "list",
				"--region", "nyc1",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(microDropletListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	microDropletListOutput = `
ID                                      Name                  Region    State      Size              Networking    Image                                                         Endpoint                                        Created At
b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2    sammy-microdroplet    nyc1      running    md-1vcpu-512mb    public        do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000    https://sammy.microdroplets.digitalocean.app    2026-07-16T10:00:00Z
`
	microDropletListResponse = `
{
  "micro_droplets": [
    {
      "id": "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
      "name": "sammy-microdroplet",
      "region": "nyc1",
      "state": "running",
      "size": "md-1vcpu-512mb",
      "networking": "public",
      "image": "do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000",
      "endpoint": "https://sammy.microdroplets.digitalocean.app",
      "created_at": "2026-07-16T10:00:00Z"
    }
  ],
  "links": {},
  "meta": {"total": 1}
}
`
)
