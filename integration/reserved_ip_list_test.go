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

var _ = suite("compute/reserved-ip/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/reserved_ips":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(reservedIPListResponse))
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
		it("lists all reserved-ips", func() {
			aliases := []string{"list", "ls"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"reserved-ip",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(reservedIPListOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	reservedIPListOutput = `
IP         Region    Droplet ID    Droplet Name
8.8.8.8    nyc3      8888          hello
1.1.1.1    nyc3      1111
`
	reservedIPListResponse = `
{
  "reserved_ips": [
    {
      "ip": "8.8.8.8",
      "droplet": {"id": 8888, "name": "hello"},
      "region": {
        "name": "New York 3",
        "slug": "nyc3",
        "sizes": [ "s-1vcpu-1gb" ],
        "features": [ "metadata" ],
        "available": true
      },
      "locked": false
    },
    {
      "ip": "1.1.1.1",
      "droplet": {"id": 1111},
      "region": {
        "name": "New York 3",
        "slug": "nyc3",
        "sizes": [ "s-1vcpu-1gb" ],
        "features": [ "metadata" ],
        "available": true
      },
      "locked": false
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}
`
)
