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

var _ = suite("compute/firewall/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/firewalls/e4b9c960-d385-4950-84f3-d102162e6be5":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(firewallGetResponse))
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
		it("gets a firewall", func() {
			const id = "e4b9c960-d385-4950-84f3-d102162e6be5"
			aliases := []string{"get", "g"}

			for _, alias := range aliases {
				cmd := exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"firewall",
					alias,
					id,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output from command %s: %s", alias, output))
				expect.Equal(strings.TrimSpace(firewallGetOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const firewallGetResponse = `{
  "firewall": {
    "id":"e4b9c960-d385-4950-84f3-d102162e6be5",
    "name":"test-firewall",
    "status":"succeeded",
    "inbound_rules":[{
	  "protocol":"tcp",
	  "ports":"443",
	  "sources":{}
    }],
    "outbound_rules":[],
    "created_at":"2019-10-24T20:30:26Z",
    "droplet_ids":[],
    "tags":[],
	"pending_changes":[]
  }
}`

const firewallGetOutput = `
ID                                      Name             Status       Created At              Inbound Rules              Outbound Rules    Droplet IDs    Tags    Pending Changes
e4b9c960-d385-4950-84f3-d102162e6be5    test-firewall    succeeded    2019-10-24T20:30:26Z    protocol:tcp,ports:443,
`
