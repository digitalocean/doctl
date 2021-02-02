package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("database/firewalls", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	mockResponses := []*godo.DatabaseFirewallRule{
		{
			UUID:        "cdb689c2-56e6-48e6-869d-306c85af178d",
			ClusterUUID: "d168d635-1c88-4616-b9b4-793b7c573927",
			Type:        "tag",
			Value:       "oldFirewall",
		},
	}

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
				switch req.Method {
				case http.MethodGet:
					data, err := json.Marshal(map[string]interface{}{
						"rules": mockResponses,
					})
					if err != nil {
						t.Fatalf("%+v", err)
					}

					w.Write(data)

				case http.MethodPut:
					v := map[string][]*godo.DatabaseFirewallRule{
						"rules": make([]*godo.DatabaseFirewallRule, 0),
					}
					if err := json.NewDecoder(req.Body).Decode(&v); err != nil {
						t.Fatalf("%+v", err)
					}

					// We're assuming the PUT request will only include the type
					// and value, so we generate the UUID to make it more like the
					// actual implementation.
					rules, ok := v["rules"]
					if !ok {
						t.Fatalf("expected rules to be present")
					}

					for _, rule := range rules {
						rule.UUID = "cdb689c2-56e6-48e6-869d-306c85af178d"
						rule.CreatedAt = time.Now()

						mockResponses = append(mockResponses, rule)
					}

					w.WriteHeader(http.StatusNoContent)
					return

				default:
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

	when("command is remove", func() {
		it("remove a database cluster's firewall rules", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"firewalls",
				"remove",
				"d168d635-1c88-4616-b9b4-793b7c573927",
				"--uuid", "cdb689c2-56e6-48e6-869d-306c85af178d",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databasesRemoveFirewallRuleOutput), strings.TrimSpace(string(output)))
			expected := strings.TrimSpace(databasesRemoveFirewallRuleOutput)
			actual := strings.TrimSpace(string(output))

			if expected != actual {
				t.Errorf("expected\n\n%s\n\nbut got\n\n%s\n\n", expected, actual)
			}
		})
	})

})

const (
	databasesRemoveFirewallRuleOutput = `
UUID                                    ClusterUUID                             Type    Value          Created At
cdb689c2-56e6-48e6-869d-306c85af178d    d168d635-1c88-4616-b9b4-793b7c573927    tag     oldFirewall    0001-01-01 00:00:00 +0000 UTC`
)
