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

var _ = suite("compute/cdn/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/cdn/endpoints":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(cdnListResponse))
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
		it("lists the cdns", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"cdn",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(cdnListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	cdnListOutput = `
ID    Origin              Endpoint           TTL     CustomDomain          CertificateID    CreatedAt
1     static.other.com    static.blah.com    3600    static.example.com    some-cert-id     2018-07-19 15:04:16 +0000 UTC
2     static.other.com    static.blah.com    3600    static.example.com    some-cert-id     2018-07-19 15:04:16 +0000 UTC
`
	cdnListResponse = `
{
  "endpoints": [
    {
      "id": "1",
      "origin": "static.other.com",
      "endpoint": "static.blah.com",
      "created_at": "2018-07-19T15:04:16Z",
      "certificate_id": "some-cert-id",
      "custom_domain": "static.example.com",
      "ttl": 3600
    },
    {
      "id": "2",
      "origin": "static.other.com",
      "endpoint": "static.blah.com",
      "created_at": "2018-07-19T15:04:16Z",
      "certificate_id": "some-cert-id",
      "custom_domain": "static.example.com",
      "ttl": 3600
    }
  ],
  "meta": {
    "total": 2
  },
  "links": {
  }
}`
)
