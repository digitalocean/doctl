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

var _ = suite("prepayment/config", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/customers/my/prepayment_config":
				w.Write([]byte(`{
					"config": {
						"spend_limit": "100.00",
						"is_auto_prepay_enabled": true,
						"prepay_amount": "50.00",
						"prepay_threshold": "10.00",
						"created_at": "2026-06-22T19:31:51Z",
						"updated_at": "2026-06-25T18:40:07Z"
					},
					"status": {
						"balance": "25.00",
						"is_auto_prepay_enabled": true,
						"blocked": true,
						"eligible": true,
						"month_to_date_balance": "75.00"
					}
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

	it("returns the details of my prepayment configuration", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"prepayment",
			"config",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))

		expect.Equal(strings.TrimSpace(prepaymentConfigOutput), strings.TrimSpace(string(output)))
	})
})

var _ = suite("prepayment/status", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/customers/my/prepayment_status":
				w.Write([]byte(`{
					"status": {
						"balance": "25.00",
						"is_auto_prepay_enabled": true,
						"blocked": true,
						"eligible": true,
						"month_to_date_balance": "75.00"
					}
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

	it("returns the details of my prepayment status", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"prepayment",
			"status",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))

		expect.Equal(strings.TrimSpace(prepaymentStatusOutput), strings.TrimSpace(string(output)))
	})
})

const prepaymentConfigOutput string = `
Spend Limit    Auto Prepay Enabled    Prepay Amount    Prepay Threshold    Balance    Blocked    Eligible    Month-to-date Balance    Created At              Updated At
100.00         true                   50.00            10.00               25.00      true       true        75.00                    2026-06-22T19:31:51Z    2026-06-25T18:40:07Z
`

const prepaymentStatusOutput string = `
Balance    Auto Prepay Enabled    Blocked    Eligible    Month-to-date Balance
25.00      true                   true       true        75.00
`
