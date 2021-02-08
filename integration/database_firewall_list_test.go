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

var _ = suite("database/firewalls", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/1213/firewall":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(databasesListFirewallRuleResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is list", func() {
		it("list all of a database cluster's firewall rules", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"firewalls",
				"list",
				"1213",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databasesFirewallRuleOutput), strings.TrimSpace(string(output)))
		})
	})

})

const (
	databasesFirewallRuleOutput = `
UUID                                    ClusterUUID                             Type       Value
cdb689c2-56e6-48e6-869d-306c85af178d    d168d635-1c88-4616-b9b4-793b7c573927    ip_addr    107.13.36.145
`
	databasesListFirewallRuleResponse = `
	{
		"rules": [
		  {
			"uuid": "cdb689c2-56e6-48e6-869d-306c85af178d",
			"cluster_uuid": "d168d635-1c88-4616-b9b4-793b7c573927",
			"type": "ip_addr",
			"value": "107.13.36.145",
			"created_at": "2021-01-27T20:34:12Z"
		  }
		]
	  }`
)
