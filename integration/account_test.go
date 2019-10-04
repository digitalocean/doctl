package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/require"
)

func TestAccounts(t *testing.T) {
	spec.Run(t, "account/get", testAccountGet, spec.Report(report.Terminal{}))
	spec.Run(t, "account/ratelimit", testAccountRateLimit, spec.Report(report.Terminal{}))
}

func testAccountGet(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/account":
				w.Write([]byte(`{
					"account": {
					    "droplet_limit": 25,
					    "floating_ip_limit": 5,
					    "email": "sammy@digitalocean.com",
					    "uuid": "b6fr89dbf6d9156cace5f3c78dc9851d957381ef",
					    "email_verified": true,
					    "status": "active",
					    "status_message": ""
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

	it("returns the details of my account", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"account",
			"get",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(accountOutput), strings.TrimSpace(string(output)))
	})
}

func testAccountRateLimit(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/account":
				w.Header().Add("RateLimit-Limit", "200")
				w.Header().Add("RateLimit-Remaining", "199")
				w.Header().Add("RateLimit-Reset", "1565385881")

				w.Write([]byte(`{ "account":{}}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("returns the current rate limits for my account", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"account",
			"ratelimit",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		t := time.Unix(1565385881, 0)
		expectedOutput := strings.TrimSpace(fmt.Sprintf(ratelimitOutput, t))

		expect.Equal(expectedOutput, strings.TrimSpace(string(output)))
	})
}

const accountOutput string = `
Email                     Droplet Limit    Email Verified    UUID                                        Status
sammy@digitalocean.com    25               true              b6fr89dbf6d9156cace5f3c78dc9851d957381ef    active
`

const ratelimitOutput string = `
Limit    Remaining    Reset
200      199          %s
`
