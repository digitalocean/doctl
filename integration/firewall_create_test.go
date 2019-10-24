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

var _ = suite("compute/firewall/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/firewalls":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				w.Write([]byte(firewallCreateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("doing create", func() {
		it("does create", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"firewall",
				"create",
				"--name", "test-firewall",
				"--inbound-rules", `"protocol:tcp,ports:443"`,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(firewallCreateOutput), strings.TrimSpace(string(output)))
		})
	})
})

const firewallCreateOutput = `ID                                      Name             Status       Created At              Inbound Rules              Outbound Rules    Droplet IDs    Tags    Pending Changes
e4b9c960-d385-4950-84f3-d102162e6be5    test-firewall    succeeded    2019-10-24T20:30:26Z    protocol:tcp,ports:443,`

const firewallCreateResponse = `{
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
