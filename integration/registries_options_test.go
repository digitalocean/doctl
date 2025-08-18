package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("registries/options", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/registry/options", "/v2/registries/options":
				fmt.Fprintf(w, `{
					"options": {
						"subscription_tiers": [
							{
								"name": "Starter",
								"slug": "starter",
								"included_repositories": 1,
								"included_storage_bytes": 536870912,
								"allow_storage_overage": false,
								"included_bandwidth_bytes": 1073741824,
								"monthly_price_in_cents": 0,
								"eligible": true
							},
							{
								"name": "Basic",
								"slug": "basic",
								"included_repositories": 5,
								"included_storage_bytes": 5368709120,
								"allow_storage_overage": true,
								"included_bandwidth_bytes": 5368709120,
								"monthly_price_in_cents": 500,
								"eligible": true
							}
						],
						"available_regions": [
							"nyc1",
							"nyc3", 
							"ams3",
							"sfo3",
							"sgp1"
						]
					}
				}`)
			default:
				http.Error(w, "Not found", http.StatusNotFound)
			}
		}))
	})

	when("getting options", func() {
		it("gets subscription tiers using registries command", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registries",
				"options",
				"subscription-tiers",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "Starter")
			expect.Contains(string(output), "Basic")
			expect.Contains(string(output), "starter")
			expect.Contains(string(output), "basic")
		})

		it("gets available regions using registries command", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"registries",
				"options",
				"available-regions",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "nyc1")
			expect.Contains(string(output), "nyc3")
			expect.Contains(string(output), "ams3")
			expect.Contains(string(output), "sfo3")
			expect.Contains(string(output), "sgp1")
		})
	})

	it.After(func() {
		server.Close()
	})
})
