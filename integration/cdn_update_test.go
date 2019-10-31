package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/cdn/update", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect        *require.Assertions
		server        *httptest.Server
		requestBodies []string
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/cdn/endpoints/magic-cdn-id":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPut {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				requestBodies = append(requestBodies, string(reqBody))

				w.Write([]byte(cdnUpdateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("all flags are passed", func() {
		it("updates the specified cdn", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"cdn",
				"update",
				"magic-cdn-id",
				"--certificate-id", "some-cert-id",
				"--domain", "example.com",
				"--ttl", "60",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(cdnUpdateOutput), strings.TrimSpace(string(output)))

			expect.Len(requestBodies, 2)
			expect.JSONEq(`{"ttl":60}`, requestBodies[0])
			expect.JSONEq(`{"custom_domain": "example.com", "certificate_id": "some-cert-id"}`, requestBodies[1])
		})
	})
})

const (
	cdnUpdateOutput = `
ID              Origin         Endpoint       TTL     CustomDomain          CertificateID    CreatedAt
other-cdn-id    some-origin    some-static    3600    static.example.com    some-cert-id     2018-07-19 15:04:16 +0000 UTC
`
	cdnUpdateResponse = `
{
  "endpoint": {
    "id": "other-cdn-id",
    "origin": "some-origin",
    "endpoint": "some-static",
    "created_at": "2018-07-19T15:04:16Z",
    "certificate_id": "some-cert-id",
    "custom_domain": "static.example.com",
    "ttl": 3600
  }
}`
)
