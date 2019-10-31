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

var _ = suite("compute/size/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/sizes":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(sizeListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("all required flags are passed", func() {
		it("lists sizes", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"size",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(sizeListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a format", func() {
		it("displays only those columns", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"size",
				"list",
				"--format", "Slug,PriceMonthly",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(sizeListFormatOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing no-header", func() {
		it("displays only values, no headers", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"size",
				"list",
				"--no-header",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(sizeListNoHeaderOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	sizeListResponse = `
{
  "sizes": [
    {
      "slug": "512mb",
      "memory": 512,
      "vcpus": 1,
      "disk": 20,
      "transfer": 1,
      "price_monthly": 5,
      "price_hourly": 0.007439999841153622,
      "regions": [
        "nyc1"
      ],
      "available": true
    },
    {
      "slug": "s-1vcpu-1gb",
      "memory": 1024,
      "vcpus": 1,
      "disk": 25,
      "transfer": 1,
      "price_monthly": 5,
      "price_hourly": 0.007439999841153622,
      "regions": [
        "nyc1"
      ],
      "available": true
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}
`
	sizeListOutput = `
Slug           Memory    VCPUs    Disk    Price Monthly    Price Hourly
512mb          512       1        20      5.00             0.007440
s-1vcpu-1gb    1024      1        25      5.00             0.007440
`
	sizeListFormatOutput = `
Slug           Price Monthly
512mb          5.00
s-1vcpu-1gb    5.00

`
	sizeListNoHeaderOutput = `
512mb          512     1    20    5.00    0.007440
s-1vcpu-1gb    1024    1    25    5.00    0.007440
`
)
