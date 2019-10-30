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

var _ = suite("compute/floating-ip/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/floating_ips/1.1.1.1":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(floatingIPGetResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

	})

	when("required flags are passed", func() {
		it("gets the specified load balancer", func() {
			aliases := []string{"get", "g"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"floating-ip",
					alias,
					"1.1.1.1",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(floatingIPGetOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	floatingIPGetOutput = `
IP         Region    Droplet ID    Droplet Name
1.1.1.1    nyc3
`
	floatingIPGetResponse = `
{
  "floating_ip": {
    "ip": "1.1.1.1",
    "droplet": null,
    "region": {
      "name": "New York 3",
      "slug": "nyc3",
      "sizes": [ "s-32vcpu-192gb" ],
      "features": [ "metadata" ],
      "available": true
    },
    "locked": false
  }
}
`
)
