package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("tokens/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/tokens":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				request := godo.TokenCreateRequest{}
				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)
				expect.Equal([]string{"account:read", "droplet:read"}, request.Scopes)

				w.Write([]byte(tokensCreateResponse))
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
		it("creates the token", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"tokens",
				"create",
				"droplets-reader",
				"--scopes", "account:read",
				"--scopes", "droplet:read",
				"--expiry-seconds", "2592000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(tokensCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing the format flag", func() {
		it("changes the output", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"tokens",
				"create",
				"--format", "Name,AccessToken",
				"droplets-reader",
				"--scopes", "account:read,droplet:read",
				"--expiry-seconds", "2592000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(tokensCreateFormattedOutput), strings.TrimSpace(string(output)))
		})
	})

	when("missing the scopes flag", func() {
		it("it returns an error", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"tokens",
				"create",
				"droplets-reader",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(tokensCreateFlagOutputError), strings.TrimSpace(string(output)))
		})
	})
})

const (
	tokensCreateOutput = `
ID     Name               Scopes                          Expires At                       Access Token
123    droplets-reader    [account:read  droplet:read]    2023-01-11 19:21:53 +0000 UTC    dop_v1_shhhhhh
`

	tokensCreateFormattedOutput = `
Name               Access Token
droplets-reader    dop_v1_shhhhhh
`

	tokensCreateResponse = `
{
	"token":
		{
			"id":123,
			"name":"droplets-reader",
			"scopes":["account:read ","droplet:read"],
			"created_at":"2022-12-12T19:21:53Z",
			"last_used_at":"2022-12-12",
			"expiry_seconds":2592000,
			"access_token":"dop_v1_shhhhhh"
		}
}
`

	tokensCreateFlagOutputError = `Error: (token.create.scopes) command is missing required arguments`
)
