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

var _ = suite("tokens/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/tokens/123":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(tokensGetResponse))
			case "/v2/tokens":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(tokensGetLookupResponse))
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
		it("gets the specified token", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"tokens",
				"get",
				"123",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(tokensGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("the name is passed", func() {
		it("looks up the specified token", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"tokens",
				"get",
				"droplets-reader",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(tokensGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing the format flag", func() {
		it("changes the output", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"tokens",
				"get",
				"123",
				"--format", "Name,LastUsedAt",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(tokensGetFormattedOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	tokensGetOutput = `
ID     Name               Scopes                          Expires At                       Last Used At    Created At
123    droplets-reader    [account:read  droplet:read]    2023-01-11 19:21:53 +0000 UTC    2022-12-12      2022-12-12 19:21:53 +0000 UTC
`

	tokensGetFormattedOutput = `
Name               Last Used At
droplets-reader    2022-12-12
`
	tokensGetResponse = `
{
	"token": {
	"id":123,
	"name":"droplets-reader",
	"scopes":["account:read ","droplet:read"],
	"created_at":"2022-12-12T19:21:53Z",
	"last_used_at":"2022-12-12",
	"expiry_seconds":2592000
	}
}
`

	tokensGetLookupResponse = `
{
	"tokens": [{
	"id":123,
	"name":"droplets-reader",
	"scopes":["account:read ","droplet:read"],
	"created_at":"2022-12-12T19:21:53Z",
	"last_used_at":"2022-12-12",
	"expiry_seconds":2592000
	}]
}
`
)
