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

var _ = suite("billingHistory", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/customers/my/billing_history":
				w.Write([]byte(billingHistoryListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("gets the customer's billing history", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"billing-history",
			"list",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		expect.Equal(strings.TrimSpace(billingHistoryListOutput), strings.TrimSpace(string(output)))
	})

})

const billingHistoryListOutput string = `
Date                    Type       Description             Amount    Invoice ID    Invoice UUID
2018-06-01T08:44:38Z    Invoice    Invoice for May 2018    12.34     123           example-uuid
2018-06-02T08:44:38Z    Payment    Payment (MC 2018)       -12.34
`
const billingHistoryListResponse string = `
{
	"billing_history": [
		{
			"description": "Invoice for May 2018",
			"amount": "12.34",
			"invoice_id": "123",
			"invoice_uuid": "example-uuid",
			"date": "2018-06-01T08:44:38Z",
			"type": "Invoice"
		},
		{
			"description": "Payment (MC 2018)",
			"amount": "-12.34",
			"date": "2018-06-02T08:44:38Z",
			"type": "Payment"
		}
	],
	"meta": {
		"total": 2
	}
}
`
