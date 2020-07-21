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

var _ = suite("invoices/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/customers/my/invoices/example-invoice-uuid":
				w.Write([]byte(invoiceGetResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("gets the specified invoice UUID", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"invoice",
			"get",
			"example-invoice-uuid",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		expect.Equal(strings.TrimSpace(invoiceGetOutput), strings.TrimSpace(string(output)))
	})

})

const invoiceGetOutput string = `
Resource ID    Resource UUID              Product           Description                 Group Description    Amount    Duration    Duration Unit    Start Time              End Time                Project Name         Category
1234           droplet-1234-uuid          Droplets          My Example Droplet                               12.34     672         Hours            2018-06-20T08:44:38Z    2018-06-21T08:44:38Z    My project           iaas
2345           load-balancer-2345-uuid    Load Balancers    My Example Load Balancer    group                23.45     744         Hours            2018-06-20T08:44:38Z    2018-06-21T08:44:38Z    My Second Project    paas

`
const invoiceGetResponse string = `
{
	"invoice_items": [
		{
			"product": "Droplets",
			"resource_id": "1234",
			"resource_uuid": "droplet-1234-uuid",
			"group_description": "",
			"description": "My Example Droplet",
			"amount": "12.34",
			"duration": "672",
			"duration_unit": "Hours",
			"start_time": "2018-06-20T08:44:38Z",
			"end_time": "2018-06-21T08:44:38Z",
			"project_name": "My project",
			"category": "iaas"
		},
		{
			"product": "Load Balancers",
			"resource_id": "2345",
			"resource_uuid": "load-balancer-2345-uuid",
			"group_description": "group",
			"description": "My Example Load Balancer",
			"amount": "23.45",
			"duration": "744",
			"duration_unit": "Hours",
			"start_time": "2018-06-20T08:44:38Z",
			"end_time": "2018-06-21T08:44:38Z",
			"project_name": "My Second Project",
			"category": "paas"
		}
	],
	"meta": {
		"total": 2
	}
}
`
