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

var _ = suite("invoices", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/customers/my/invoices":
				w.Write([]byte(invoiceListResponse))
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
			"list",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		expect.Equal(strings.TrimSpace(invoiceListOutput), strings.TrimSpace(string(output)))
	})

})

const invoiceListOutput string = `
Invoice UUID              Amount    Invoice Period
preview                   34.56     2020-02
example-invoice-uuid-1    12.34     2020-01
example-invoice-uuid-2    23.45     2019-12
`
const invoiceListResponse string = `
{
	"invoices": [
		{
		"invoice_uuid": "example-invoice-uuid-1",
		"amount": "12.34",
		"invoice_period": "2020-01"
		},
		{
		"invoice_uuid": "example-invoice-uuid-2",
		"amount": "23.45",
		"invoice_period": "2019-12"
		}
	],
	"invoice_preview": {
		"invoice_uuid": "example-invoice-uuid-preview",
		"amount": "34.56",
		"invoice_period": "2020-02",
		"updated_at": "2020-02-05T05:43:10Z"
	},
	"meta": {
		"total": 2
	}
}
`
