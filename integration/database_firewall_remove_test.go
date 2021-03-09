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

	"github.com/digitalocean/godo"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

func (ms *mockServer) Remove(t *testing.T, w http.ResponseWriter, r *http.Request) {
	if !ms.auth(w, r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	v := map[string][]*godo.DatabaseFirewallRule{
		"rules": make([]*godo.DatabaseFirewallRule, 0),
	}
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
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
	}

	// The backend will always replace all firewall rules, so we do the same
	// in the mock implementation.
	// ms.rules will now only have the rule that was not removed.
	ms.rules = rules
	w.WriteHeader(http.StatusNoContent)
}

var _ = suite("database/firewalls", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	ms := &mockServer{
		// ms.rules had two rules to begin with.
		rules: []*godo.DatabaseFirewallRule{
			{
				UUID:        "cdb689c2-56e6-48e6-869d-306c85af178d",
				ClusterUUID: "d168d635-1c88-4616-b9b4-793b7c573927",
				Type:        "tag",
				Value:       "new-firewall-tag",
			},
			{
				UUID:        "cdb689c2-56e6-48e6-869d-306c85af178g",
				ClusterUUID: "d168d635-1c88-4616-b9b4-793b7c573927",
				Type:        "tag",
				Value:       "old-firewall-tag",
			},
		},
	}

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/d168d635-1c88-4616-b9b4-793b7c573927/firewall":
				switch req.Method {
				case http.MethodGet:
					ms.Get(t, w, req)

				case http.MethodPut:
					ms.Remove(t, w, req)

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
		it("remove a database cluster's firewall rule", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"firewalls",
				"remove",
				"d168d635-1c88-4616-b9b4-793b7c573927",
				"--uuid", "cdb689c2-56e6-48e6-869d-306c85af178g",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))

			expected := strings.TrimSpace(string(output))
			actual := strings.TrimSpace(databasesRemoveFirewallRuleOutput)

			expect.Equal(expected, actual)

			fmt.Println(expected)
			fmt.Println(actual)
		})
	})

})

const (
	databasesRemoveFirewallRuleOutput = `
UUID                                    ClusterUUID                             Type    Value
cdb689c2-56e6-48e6-869d-306c85af178d    d168d635-1c88-4616-b9b4-793b7c573927    tag     new-firewall-tag`
)
