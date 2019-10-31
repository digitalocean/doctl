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

var _ = suite("compute/certificate/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
	)

	it.Before(func() {
		expect = require.New(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/certificates":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(certListResponse))
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

	when("command is list", func() {
		it("lists all certificates", func() {
			cmd.Args = append(cmd.Args, []string{"list"}...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(certListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is ls", func() {
		it("lists all certificates", func() {
			cmd.Args = append(cmd.Args, []string{"ls"}...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(certListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	certListOutput = `
ID                                      Name           DNS Names                      SHA-1 Fingerprint                           Expiration Date         Created At              Type            State
892071a0-bb95-49bc-8021-3afd67a210bf    web-cert-01                                   dfcc9f57d86bf58e321c2c6c31c7a971be244ac7    2017-02-22T00:23:00Z    2017-02-08T16:02:37Z    custom          verified
ba9b9c18-6c59-46c2-99df-70da170a42ba    web-cert-02    www.example.com,example.com    479c82b5c63cb6d3e6fac4624d58a33b267e166c    2018-06-07T17:44:12Z    2018-03-09T18:44:11Z    lets_encrypt    verified
`
	certListResponse = `
{
  "certificates": [
    {
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
    },
    {
      "id": "ba9b9c18-6c59-46c2-99df-70da170a42ba",
      "name": "web-cert-02",
      "not_after": "2018-06-07T17:44:12Z",
      "sha1_fingerprint": "479c82b5c63cb6d3e6fac4624d58a33b267e166c",
      "created_at": "2018-03-09T18:44:11Z",
      "dns_names": [
        "www.example.com",
        "example.com"
      ],
      "state": "verified",
      "type": "lets_encrypt"
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}
`
)
