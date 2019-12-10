package integration

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("balance/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/customers/my/balance":
				w.Write([]byte(`{
					"month_to_date_balance": "23.44",
					"account_balance": "12.23",
					"month_to_date_usage": "11.21",
					"generated_at": "2018-06-21T08:44:38Z"
				  }`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("returns the details of my balance", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"balance",
			"get",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(balanceOutput), strings.TrimSpace(string(output)))
	})
})

const balanceOutput string = `
Month-to-date Balance    Account Balance    Month-to-date Usage    Generated At
23.44                    12.23              11.21                  2018-06-21T08:44:38Z
`
