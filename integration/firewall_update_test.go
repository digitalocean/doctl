package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/firewall/update", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodPut {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				body, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)
				expect.JSONEq(firewallUpdateRequestBody, string(body))

				w.Write([]byte(firewallUpdateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("an updated name and the old inbound rules are provided", func() {
		it("updates a firewall", func() {
			const id = "e4b9c960-d385-4950-84f3-d102162e6be5"
			aliases := []string{"update", "u"}

			for _, alias := range aliases {
				cmd := exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"firewall",
					alias,
					id,
					"--name", "updated-test-firewall",
					"--inbound-rules", `protocol:tcp,ports:443`,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output from command %s: %s", alias, output))
				expect.Equal(strings.TrimSpace(firewallUpdateOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	firewallUpdateOutput = `
ID                                      Name                     Status       Created At              Inbound Rules              Outbound Rules    Droplet IDs    Tags    Pending Changes
e4b9c960-d385-4950-84f3-d102162e6be5    updated-test-firewall    succeeded    2019-10-24T20:30:26Z    protocol:tcp,ports:443,`

	firewallUpdateRequestBody = `{
  "name":"updated-test-firewall",
  "inbound_rules":[{
	"protocol":"tcp",
	"ports":"443",
	"sources":{}
  }],
  "outbound_rules":null,
  "droplet_ids":[],
  "tags":[]
}`

	firewallUpdateResponse = `{
  "firewall": {
	"id":"e4b9c960-d385-4950-84f3-d102162e6be5",
	"name":"updated-test-firewall",
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
)
