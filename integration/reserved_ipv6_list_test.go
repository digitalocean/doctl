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

var _ = suite("compute/reserved-ipv6/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/reserved_ipv6":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(reservedIPv6ListResponse))
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
		it("lists all reserved-ipv6s", func() {
			aliases := []string{"list", "ls"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"reserved-ipv6",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(reservedIPv6ListOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	reservedIPv6ListOutput = `
IP                           Region    Droplet ID    Droplet Name
fd53:616d:6d60::1071:5001    nyc3                    
fd53:616d:6d60::1071:5002    nyc3                    
`
	reservedIPv6ListResponse = `{
  "reserved_ipv6s": [
    {
      "ip": "fd53:616d:6d60::1071:5001",
      "region_slug": "nyc3",
	  "reserved_at": "2021-10-01T00:00:00Z"
    },
	{
      "ip": "fd53:616d:6d60::1071:5002",
      "region_slug": "nyc3",
	  "reserved_at": "2021-10-01T00:00:00Z"
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}`
)
