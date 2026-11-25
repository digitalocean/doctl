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

// APICLI-5566: the add-funds card on a 402 tells the user to run
// `doctl harness-runtime balance`, so it must quote that table's figures under
// the table's labels. A card that blames a $0.00 balance while the table shows
// positive credit reads as a contradiction.
var _ = suite("harness-runtime/prepay-card", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
		// prepayStatus is swapped per test to stand in for billing's status.
		prepayStatus string
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/harness-runtime/triggers/by-session/sess_1":
				// No trigger is bound to the session.
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"id":"not_found","message":"no trigger is bound to this session"}`))
			case "/v2/harness-runtime/sessions/sess_1/resume":
				// The exact body harness-api writes when the gate blocks a team.
				w.WriteHeader(http.StatusPaymentRequired)
				w.Write([]byte(`{"error":{"code":402,"message":"team is blocked by the prepayment gate; add funds to resume"}}`))
			case prepayConfigPath:
				w.Write([]byte(`{"config":{"spend_limit":"0.00","prepay_amount":"0.00","prepay_threshold":"0.00"},"status":` + prepayStatus + `}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	resume := func() string {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"harness-runtime", "resume", "sess_1",
		)

		output, err := cmd.CombinedOutput()
		expect.Error(err)
		return strings.TrimSpace(string(output))
	}

	when("the team pays from credit", func() {
		it.Before(func() {
			prepayStatus = `{"balance":"0.00","eligible":true,"month_to_date_balance":"38.66","available_credit":"9.29"}`
		})

		it("quotes the balance and the available credit", func() {
			expect.Equal(strings.TrimSpace(prepayCardCredit), resume())
		})
	})

	when("the team pays with cash", func() {
		it.Before(func() {
			prepayStatus = `{"balance":"0.00","blocked":true,"eligible":true,"month_to_date_balance":"12.40","available_credit":"0.00"}`
		})

		it("quotes the balance and the available credit", func() {
			expect.Equal(strings.TrimSpace(prepayCardCash), resume())
		})
	})

	// zero-balance is the API's name for the gate pause, which counts cash and
	// credit together, so the help must not leave it reading as a zero balance.
	it("says in list help that zero-balance is the prepayment gate", func() {
		output, err := exec.Command(builtBinaryPath, "harness-runtime", "list", "--help").CombinedOutput()
		expect.NoError(err)
		expect.Contains(string(output), "zero-balance (paused by the prepayment gate)")
	})
})

const prepayCardCredit string = `
Error: Blocked by the prepayment gate
Your team's prepayment and credit don't cover its usage, so Harness Runtime can't start or resume sessions. Balance $0.00, Available Credit $9.29. Running sessions were paused and their work is saved.
status 402
→ Add funds at https://cloud.digitalocean.com/account/billing; doctl harness-runtime balance
`

const prepayCardCash string = `
Error: Blocked by the prepayment gate
Your team's prepayment and credit don't cover its usage, so Harness Runtime can't start or resume sessions. Balance $0.00, Available Credit $0.00. Running sessions were paused and their work is saved.
status 402
→ Add funds at https://cloud.digitalocean.com/account/billing; doctl harness-runtime balance
`
