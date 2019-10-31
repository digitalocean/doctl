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

var _ = suite("database/user/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/some-database-id/users/some-user-id":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(databaseUserGetResponse))
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
		it("gets the database user", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"user",
				"get",
				"some-database-id",
				"some-user-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseUserGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("no header is passed", func() {
		it("does not display the header", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"user",
				"get",
				"some-database-id",
				"some-user-id",
				"--no-header",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseUserGetNoHeaderOutput), strings.TrimSpace(string(output)))
		})
	})

	when("format is passed", func() {
		it("presents the columns requested", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"user",
				"get",
				"some-database-id",
				"some-user-id",
				"--format", "Name",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseUserGetFormatOutput), strings.TrimSpace(string(output)))
		})
	})

	when("the output is json", func() {
		it("prints json", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"-o", "json",
				"database",
				"user",
				"get",
				"some-database-id",
				"some-user-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.JSONEq(databaseUserGetJSONOutput, string(output))
		})
	})
})

const (
	databaseUserGetOutput = `
Name            Role      Password
some-user-id    normal    jge5lfxtzhx42iff
`
	databaseUserGetNoHeaderOutput = `
some-user-id    normal    jge5lfxtzhx42iff
`
	databaseUserGetFormatOutput = `
Name
some-user-id
`
	databaseUserGetJSONOutput = `
[{ "name": "some-user-id", "role": "normal", "password": "jge5lfxtzhx42iff" }]`
	databaseUserGetResponse = `
{
  "user": {
    "name": "some-user-id",
    "role": "normal",
    "password": "jge5lfxtzhx42iff"
  }
}`
)
