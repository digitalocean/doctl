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

var _ = suite("compute/cdn/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/cdn/endpoints/other-cdn-id":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(cdnGetResponse))
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
		it("gets the specified cdn", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"cdn",
				"get",
				"other-cdn-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(cdnGetOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	cdnGetOutput = `
ID              Origin                                       Endpoint                                         TTL     CustomDomain          CertificateID                           CreatedAt
other-cdn-id    static-images.nyc3.digitaloceanspaces.com    static-images.nyc3.cdn.digitaloceanspaces.com    3600    static.example.com    892071a0-bb95-49bc-8021-3afd67a210bf    2018-07-19 15:04:16 +0000 UTC
`
	cdnGetResponse = `
{
  "endpoint": {
    "id": "other-cdn-id",
    "origin": "static-images.nyc3.digitaloceanspaces.com",
    "endpoint": "static-images.nyc3.cdn.digitaloceanspaces.com",
    "created_at": "2018-07-19T15:04:16Z",
    "certificate_id": "892071a0-bb95-49bc-8021-3afd67a210bf",
    "custom_domain": "static.example.com",
    "ttl": 3600
  }
}`
)
