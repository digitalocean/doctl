package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/certificate/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/certificates":
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

				matchedRequest := certCustomCreateJSONReq
				if strings.Contains(string(reqBody), "lets_encrypt") {
					matchedRequest = certLECreateReqJSONReq
				}

				expect.JSONEq(string(reqBody), matchedRequest)

				w.Write([]byte(certCreateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("asking for a lets_encrypt cert", func() {
		it("creates the certificate", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"certificate",
				"create",
				"--name", "test-cert",
				"--dns-names", "example.com,exmaple.com",
				"--type", "lets_encrypt",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(certCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("asking for a custom cert", func() {
		it("creates the certificate", func() {
			testFiles := []string{"test-private.key", "test-cert.chain", "test-leaf.cert"}
			dir, err := ioutil.TempDir("", "doctl-tests-cert")
			expect.NoError(err)
			defer os.RemoveAll(dir)

			for _, f := range testFiles {
				err := ioutil.WriteFile(filepath.Join(dir, f), []byte(f), 0600)
				expect.NoError(err)
			}

			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"certificate",
				"create",
				"--name", "test-cert",
				"--type", "custom",
				"--private-key-path", filepath.Join(dir, testFiles[0]),
				"--certificate-chain-path", filepath.Join(dir, testFiles[1]),
				"--leaf-certificate-path", filepath.Join(dir, testFiles[2]),
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		})
	})
})

const (
	certCreateOutput = `
ID                                      Name           DNS Names    SHA-1 Fingerprint                           Expiration Date         Created At              Type      State
892071a0-bb95-49bc-8021-3afd67a210bf    web-cert-01                 dfcc9f57d86bf58e321c2c6c31c7a971be244ac7    2017-02-22T00:23:00Z    2017-02-08T16:02:37Z    custom    verified
`
	certCustomCreateJSONReq = `
{
  "name":"test-cert",
  "private_key":"test-private.key",
  "leaf_certificate":"test-leaf.cert",
  "certificate_chain":"test-cert.chain",
  "type":"custom"
}
`
	certLECreateReqJSONReq = `
{
  "name":"test-cert",
  "dns_names":["example.com","exmaple.com"],
  "type":"lets_encrypt"
}`
	certCreateResponse = `
{
  "certificate": {
    "id": "892071a0-bb95-49bc-8021-3afd67a210bf",
    "name": "web-cert-01",
    "not_after": "2017-02-22T00:23:00Z",
    "sha1_fingerprint": "dfcc9f57d86bf58e321c2c6c31c7a971be244ac7",
    "created_at": "2017-02-08T16:02:37Z",
    "dns_names": [],
    "state": "verified",
    "type": "custom"
  }
}
`
)
