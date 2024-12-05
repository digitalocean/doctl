package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/reserved-ipv6/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/reserved_ipv6":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := io.ReadAll(req.Body)
				expect.NoError(err)

				matchedRequest := reservedIPv6CreateRequest
				responseJSON := reservedIPv6CreateResponse

				expect.JSONEq(matchedRequest, string(reqBody))

				w.Write([]byte(responseJSON))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

	})

	when("the region flag is provided", func() {
		it("creates the reserved-ipv6", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"reserved-ipv6",
					alias,
					"--region", "nyc3",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(reservedIPv6CreateOutput), strings.TrimSpace(string(output)))
			}
		})
	})

})

const (
	reservedIPv6CreateOutput = `
IP                           Region    Droplet ID    Droplet Name
fd53:616d:6d60::1071:5001    nyc3                    
`
	reservedIPv6CreateResponse = `
{
  "reserved_ipv6": {
    "ip": "fd53:616d:6d60::1071:5001",
    "region_slug": "nyc3",
	"reserved_at": "2021-10-01T00:00:00Z",
  },
  "links": {}
}
`

	reservedIPv6CreateRequest = `
{"region_slug":"nyc3"}
`
)
