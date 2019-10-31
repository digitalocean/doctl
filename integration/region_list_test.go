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

var _ = suite("compute/region/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/regions":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(regionListResponse))
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
		it("lists regions", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"region",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(regionListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a format", func() {
		it("displays only those columns", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"region",
				"list",
				"--format", "Slug",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(regionListFormatOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing no-header", func() {
		it("displays only values, no headers", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"region",
				"list",
				"--no-header",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(regionListNoHeaderOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	regionListResponse = `
{
  "regions": [
    {
      "name": "New York 1",
      "slug": "nyc1",
      "features": [
        "private_networking",
        "backups"
      ],
      "available": true,
      "sizes": [
        "s-1vcpu-1gb"
      ]
    },
    {
      "name": "San Francisco 1",
      "slug": "sfo1",
      "features": [
        "private_networking",
        "backups"
      ],
      "available": true,
      "sizes": [
        "s-1vcpu-1gb"
      ]
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}
`
	regionListOutput = `
Slug    Name               Available
nyc1    New York 1         true
sfo1    San Francisco 1    true
`
	regionListFormatOutput = `
Slug
nyc1
sfo1
`
	regionListNoHeaderOutput = `
nyc1    New York 1         true
sfo1    San Francisco 1    true
`
)
