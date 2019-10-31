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

var _ = suite("compute/certificate/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect   *require.Assertions
		cmd      *exec.Cmd
		baseArgs = []string{"find-cert-id"}
	)

	it.Before(func() {
		expect = require.New(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/certificates/find-cert-id":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(certGetResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"compute",
			"certificate",
		)
	})

	when("command is get", func() {
		it("gets the specified certificate", func() {
			args := append([]string{"get"}, baseArgs...)
			cmd.Args = append(cmd.Args, args...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(certGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is g", func() {
		it("gets the specified certificate", func() {
			args := append([]string{"g"}, baseArgs...)
			cmd.Args = append(cmd.Args, args...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(certGetOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	certGetOutput = `
ID                                      Name           DNS Names    SHA-1 Fingerprint                           Expiration Date         Created At              Type      State
892071a0-bb95-49bc-8021-3afd67a210bf    web-cert-01                 dfcc9f57d86bf58e321c2c6c31c7a971be244ac7    2017-02-22T00:23:00Z    2017-02-08T16:02:37Z    custom    verified
`
	certGetResponse = `
{
  "certificate": {
    "id": "892071a0-bb95-49bc-8021-3afd67a210bf",
    "name": "web-cert-01",
    "not_after": "2017-02-22T00:23:00Z",
    "sha1_fingerprint": "dfcc9f57d86bf58e321c2c6c31c7a971be244ac7",
    "created_at": "2017-02-08T16:02:37Z",
    "dns_names": [
      ""
    ],
    "state": "verified",
    "type": "custom"
  }
}
`
)
