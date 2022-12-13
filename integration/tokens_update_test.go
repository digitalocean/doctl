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

var _ = suite("tokens/update", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				request := godo.TokenUpdateRequest{}
				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)
				expect.Equal([]string{"account:read", "droplet:read"}, request.Scopes)

				w.Write([]byte(tokensUpdateResponse))

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
		it("updates the token", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"tokens",
				"update",
				"--scopes", "account:read",
				"--scopes", "droplet:read",
				"--updated-name", "droplets-reader",
				"123",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(tokensUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("the name is passed", func() {
		it("looks up and updates the token", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"tokens",
				"update",
				"--scopes", "account:read",
				"--scopes", "droplet:read",
				"--updated-name", "droplets-reader",
				"droplets-reader",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(tokensUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing the format flag", func() {
		it("changes the output", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"tokens",
				"update",
				"--format", "Name,Scopes",
				"--updated-name", "droplets-reader",
				"--scopes", "account:read,droplet:read",
				"123",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(tokensUpdateFormattedOutput), strings.TrimSpace(string(output)))
		})
	})

	when("missing the scopes flag", func() {
		it("it returns an error", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"tokens",
				"update",
				"123",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(tokensUpdateFlagOutputError), strings.TrimSpace(string(output)))
		})
	})
})

const (
	tokensUpdateOutput = `
ID     Name               Scopes                          Expires At                       Last Used At    Created At
123    droplets-reader    [account:read  droplet:read]    2023-01-11 19:21:53 +0000 UTC    2022-12-12      2022-12-12 19:21:53 +0000 UTC
`

	tokensUpdateFormattedOutput = `
Name               Scopes
droplets-reader    [account:read  droplet:read]
`

	tokensUpdateResponse = `
{
	"token":
		{
			"id":123,
			"name":"droplets-reader",
			"scopes":["account:read ","droplet:read"],
			"created_at":"2022-12-12T19:21:53Z",
			"last_used_at":"2022-12-12",
			"expiry_seconds":2592000
		}
}
`

	tokensUpdateFlagOutputError = `Error: must supply at least one of --scopes or --updated-name`
)
