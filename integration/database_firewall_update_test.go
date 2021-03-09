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

var _ = suite("database/firewalls", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)
	it.Before(func() {
		expect = require.New(t)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/d168d635-1c88-4616-b9b4-793b7c573927/firewall":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if req.Method == http.MethodPut {
					reqBody, err := ioutil.ReadAll(req.Body)
					expect.NoError(err)
					t.Log(string(reqBody))
					t.Log(databasesUpdateFirewallUpdateRequest)
					expect.JSONEq(databasesUpdateFirewallUpdateRequest, string(reqBody))
					w.Write([]byte(databasesUpdateFirewallRuleResponse))
				} else if req.Method == http.MethodGet {
					w.Write([]byte(databasesUpdateFirewallRuleResponse))
				} else {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}
				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is update", func() {
		it("update a database cluster's firewall rules", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"firewalls",
				"replace",
				"d168d635-1c88-4616-b9b4-793b7c573927",
				"--rule", "ip_addr:192.168.1.1",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databasesUpdateFirewallRuleOutput), strings.TrimSpace(string(output)))

			expected := strings.TrimSpace(databasesUpdateFirewallRuleOutput)
			actual := strings.TrimSpace(string(output))

			if expected != actual {
				t.Errorf("expected\n\n%s\n\nbut got\n\n%s\n\n", expected, actual)
			}
		})
	})

})

const (
	databasesUpdateFirewallUpdateRequest = `{"rules":[{"uuid":"","cluster_uuid":"","type":"ip_addr","value":"192.168.1.1","created_at":"0001-01-01T00:00:00Z"}]}`
	databasesUpdateFirewallRuleOutput    = `
UUID                                    ClusterUUID                             Type       Value
82ebbbd4-437c-4e11-bfd2-644ccb555de0    d168d635-1c88-4616-b9b4-793b7c573927    ip_addr    192.168.1.1`
	databasesUpdateFirewallRuleResponse = `{
		"rules":[
		   {
			  "uuid":"82ebbbd4-437c-4e11-bfd2-644ccb555de0",
			  "cluster_uuid":"d168d635-1c88-4616-b9b4-793b7c573927",
			  "type":"ip_addr",
			  "value":"192.168.1.1",
			  "created_at":"2021-01-29T19:59:35Z"
		   }
		]
	 }`
)
