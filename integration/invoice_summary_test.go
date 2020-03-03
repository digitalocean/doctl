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

var _ = suite("invoices/summary", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/customers/my/invoices/example-invoice-uuid/summary":
				w.Write([]byte(invoiceSummaryResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("gets the specified invoice UUID summary", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"invoice",
			"summary",
			"example-invoice-uuid",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		expect.Equal(strings.TrimSpace(invoiceSummaryOutput), strings.TrimSpace(string(output)))
	})

})

const invoiceSummaryOutput string = `
Invoice UUID            Billing Period    Amount    User Name        Company         Email                   Product Charges Amount    Overages Amount    Taxes Amount    Credits and Adjustments Amount
example-invoice-uuid    2020-01           27.13     Frodo Baggins    DigitalOcean    fbaggins@example.com    12.34                     3.45               4.56            6.78
`
const invoiceSummaryResponse string = `
{
	"invoice_uuid": "example-invoice-uuid",
	"billing_period": "2020-01",
	"amount": "27.13",
	"user_name": "Frodo Baggins",
	"user_company": "DigitalOcean",
	"user_email": "fbaggins@example.com",
	"product_charges": {
		"name": "Product usage charges",
		"amount": "12.34",
		"items": [
		{
			"amount": "10.00",
			"name": "Spaces Subscription",
			"count": "1"
		},
		{
			"amount": "2.34",
			"name": "Database Clusters",
			"count": "1"
		}
		]
	},
	"overages": {
		"name": "Overages",
		"amount": "3.45"
	},
	"taxes": {
		"name": "Taxes",
		"amount": "4.56"
	},
	"credits_and_adjustments": {
		"name": "Credits & adjustments",
		"amount": "6.78"
	}
}
`
