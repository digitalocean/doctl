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

var _ = suite("retries/server-error", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		var (
			requestCount int
			errResp      = `{"id": "server_error", "message": "something broke"}`
		)
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/account":
				requestCount++

				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				if requestCount < 5 {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(errResp))
					return
				}

				w.Write([]byte(accountGetResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("retries five time by default and succeeds", func() {
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

	it("retries are logged with trace flag", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"account",
			"get",
			"--trace",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Contains(strings.TrimSpace(string(output)), "retrying in")
	})

	when("respects the http-retry-max flag and gives up", func() {
		it("only displays the correct fields", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"account",
				"get",
				"--http-retry-max", "2",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expectedErr := fmt.Sprintf("Error: GET %s/v2/account: 500 something broke; giving up after 3 attempt(s)", server.URL)
			expect.Equal(strings.TrimSpace(string(output)), expectedErr)
		})
	})

	when("retries are disabled when http-retry-max is set to 0", func() {
		it("only displays the correct fields", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"account",
				"get",
				"--http-retry-max", "0",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)

			// Does not contain "giving up after"
			expectedErr := fmt.Sprintf("Error: GET %s/v2/account: 500 something broke", server.URL)
			expect.Equal(strings.TrimSpace(string(output)), expectedErr)
		})
	})

	when("doesn't retry 400-level errors", func() {
		it("only displays the correct fields", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "bad-token",
				"-u", server.URL,
				"account",
				"get",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)

			expect.NotContains(strings.TrimSpace(string(output)), "giving up after")
			expect.Contains(strings.TrimSpace(string(output)), "401")
		})
	})
})
