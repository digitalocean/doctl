package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("network/byoip-prefix/resource", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/byoip_prefixes/78d564a7-bc3f-4489-be14-1fb714969213/ips":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				responseJSON := byoipPrefixResourcesResponse
				w.Write([]byte(responseJSON))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}
				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("the valid request", func() {
		it("lists the byoip prefix resources", func() {
			aliases := []string{"resource", "resources"}
			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"network",
					"byoip-prefix",
					alias,
					"78d564a7-bc3f-4489-be14-1fb714969213",
				)
				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(norm(byoipPrefixResourcesOutput), norm(string(output)))
			}
		})
	})
})

const byoipPrefixResourcesResponse = `
{
  "ips": [
    {
      "id": 123,
      "byoip": "10.1.1.1",
      "resource": "do:droplet:456",
      "region": "nyc3",
      "assigned_at": "2024-06-24T12:34:56Z"
    }
  ]
}
`

const byoipPrefixResourcesOutput = `
ID    IP          Region    Resource         Assigned At
123   10.1.1.1    nyc3      do:droplet:456   2024-06-24 12:34:56 +0000 UTC
`
