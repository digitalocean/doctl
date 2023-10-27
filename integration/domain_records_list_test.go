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

var _ = suite("compute/domain/records/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/domains/example.com/records":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.Write([]byte(domainRecordsListResponse))

			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is list", func() {
		it("lists domain records", func() {
			aliases := []string{"list", "ls"}

			for _, alias := range aliases {
				cmd := exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"domain",
					"records",
					alias,
					"example.com",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(domainRecordsListOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("format flag is passed", func() {
		it("flags and tag can be displayed", func() {
			aliases := []string{"list", "ls"}

			for _, alias := range aliases {
				cmd := exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"domain",
					"records",
					alias,
					"example.com",
					"--format", "ID,Type,Name,Flags,Tag",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(domainRecordsListFormattedOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	domainRecordsListOutput = `
ID       Type    Name    Data               Priority    Port    TTL     Weight
123      A       foo     127.0.0.1          0           0       3600    0
12345    CAA     @       letsencrypt.org    0           0       3600    0
`

	domainRecordsListFormattedOutput = `
ID       Type    Name    Flags    Tag
123      A       foo     0        
12345    CAA     @       1        issuewild
`

	domainRecordsListResponse = `
{
  "domain_records": [
	{
	  "id": 123,
	  "type": "A",
	  "name": "foo",
      "data": "127.0.0.1",
	  "ttl": 3600
    },
	{
		"id": 12345,
		"type": "CAA",
		"name": "@",
		"data": "letsencrypt.org",
		"flags": 1,
		"tag": "issuewild",
		"ttl": 3600
	}
  ]
}
`
)
