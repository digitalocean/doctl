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

var _ = suite("compute/domain/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
	)

	it.Before(func() {
		expect = require.New(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/domains":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(domainListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"compute",
			"domain",
		)

	})

	when("command is list", func() {
		it("lists all domains", func() {
			cmd.Args = append(cmd.Args, []string{"list"}...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(domainListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is ls", func() {
		it("lists all domains", func() {
			cmd.Args = append(cmd.Args, []string{"ls"}...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(domainListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	domainListOutput = `
Domain         TTL
example.com    1800
exmaple.com    1800
`
	domainListResponse = `
{
  "domains": [
    {
      "name": "example.com",
      "ttl": 1800,
      "zone_file": "$ORIGIN example.com.\n$TTL 1800\nexample.com. IN SOA ns1.digitalocean.com. hostmaster.example.com. 1415982609 10800 3600 604800 1800"
    },
    {
      "name": "exmaple.com",
      "ttl": 1800,
      "zone_file": "$ORIGIN exmaple.com.\n$TTL 1800\nexmaple.com. IN SOA ns1.digitalocean.com. hostmaster.exmaple.com. 1415982609 10800 3600 604800 1800"
    }
  ],
  "links": {
  },
  "meta": {
    "total": 2
  }
}
`
)
