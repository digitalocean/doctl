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

var _ = suite("compute/microdroplet/create", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := io.ReadAll(req.Body)
				expect.NoError(err)

				var got map[string]any
				expect.NoError(json.Unmarshal(reqBody, &got))
				expect.Equal("sammy-microdroplet", got["name"])
				expect.Equal("nyc1", got["region"])
				expect.Equal("md-1vcpu-512mb", got["size"])
				expect.Equal("do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000", got["image"])

				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(microDropletCreateResponse))
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
		it("creates a microdroplet", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute", "microdroplet", "create", "sammy-microdroplet",
				"--region", "nyc1",
				"--size", "md-1vcpu-512mb",
				"--image", "do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(microDropletCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("missing required flags", func() {
		it("returns an error", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", "https://www.example.com",
				"compute", "microdroplet", "create", "sammy-microdroplet",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})
	})
})

const (
	microDropletCreateOutput = `
ID                                      Name                  Region    State       Size              Networking    Image                                                         Endpoint    Created At
b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2    sammy-microdroplet    nyc1      creating    md-1vcpu-512mb    public        do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000                2026-07-16T10:00:00Z
`
	microDropletCreateResponse = `
{
  "micro_droplet": {
    "id": "b2a2f7a4-8d34-4c1c-9c66-3f2b7f8f38f2",
    "name": "sammy-microdroplet",
    "region": "nyc1",
    "state": "creating",
    "size": "md-1vcpu-512mb",
    "networking": "public",
    "image": "do:microdroplet-image:0f0f0f0f-0000-0000-0000-000000000000",
    "created_at": "2026-07-16T10:00:00Z"
  }
}
`
)
