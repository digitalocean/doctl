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

// prepayConfigPath is billing's public prepay route, which returns the config
// and the gate status together.
const prepayConfigPath = "/v2/customers/my/prepayment_config"

var _ = suite("harness-runtime/balance", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
		// respond is swapped per test to stand in for billing's reply.
		respond func(w http.ResponseWriter)
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case prepayConfigPath:
				respond(w)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("the team is funded", func() {
		it.Before(func() {
			respond = func(w http.ResponseWriter) {
				w.Write([]byte(`{
					"config": {
						"spend_limit": "100.00",
						"prepay_amount": "0.00",
						"prepay_threshold": "0.00"
					},
					"status": {
						"balance": "42.75",
						"eligible": true,
						"month_to_date_balance": "8.10"
					}
				}`))
			}
		})

		it("reports the balance and a clear gate", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"harness-runtime", "balance",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Equal(strings.TrimSpace(harnessBalanceOK), strings.TrimSpace(string(output)))
		})
	})

	when("the prepayment gate is engaged", func() {
		it.Before(func() {
			// proto3 JSON omits default values, so a disabled auto top-off is
			// absent rather than false. This mirrors the verified prod shape.
			respond = func(w http.ResponseWriter) {
				w.Write([]byte(`{
					"config": {
						"spend_limit": "0.00",
						"prepay_amount": "0.00",
						"prepay_threshold": "0.00"
					},
					"status": {
						"balance": "0.00",
						"blocked": true,
						"eligible": true,
						"month_to_date_balance": "0.00"
					}
				}`))
			}
		})

		it("says the balance is blocked rather than showing a bare zero", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"harness-runtime", "balance",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Equal(strings.TrimSpace(harnessBalanceBlocked), strings.TrimSpace(string(output)))
		})
	})

	when("auto top-off is enabled", func() {
		it.Before(func() {
			respond = func(w http.ResponseWriter) {
				w.Write([]byte(`{
					"config": {
						"spend_limit": "100.00",
						"is_auto_prepay_enabled": true,
						"prepay_amount": "25.00",
						"prepay_threshold": "5.00"
					},
					"status": {
						"balance": "3.00",
						"is_auto_prepay_enabled": true,
						"eligible": true,
						"month_to_date_balance": "22.00"
					}
				}`))
			}
		})

		it("shows what will be charged and when", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"harness-runtime", "balance",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Equal(strings.TrimSpace(harnessBalanceAutoTopOff), strings.TrimSpace(string(output)))
		})
	})

	when("the token cannot read billing", func() {
		it.Before(func() {
			respond = func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"id":"forbidden","message":"insufficient scope"}`))
			}
		})

		// The user can neither see nor fix the balance, so this is a state to
		// display, not a REST error to dump.
		it("reports the gate state as unknown and names the missing scope", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"harness-runtime", "balance",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Equal(strings.TrimSpace(harnessBalanceNoPermission), strings.TrimSpace(string(output)))
		})
	})
})

const harnessBalanceOK string = `
Balance    Month-to-date Balance    Auto Top-off    Status
$42.75     $8.10                    off             OK
`

const harnessBalanceBlocked string = `
Balance    Month-to-date Balance    Auto Top-off    Status
$0.00      $0.00                    off             Blocked — add funds
`

const harnessBalanceAutoTopOff string = `
Balance    Month-to-date Balance    Auto Top-off                      Status
$3.00      $22.00                   on (at $5.00, charging $25.00)    OK
`

const harnessBalanceNoPermission string = `
Balance    Month-to-date Balance    Auto Top-off    Status
unknown    unknown                  unknown         unknown (token lacks billing:read)
`
