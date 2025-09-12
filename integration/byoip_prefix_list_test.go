package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("network/byoip-prefix/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/byoip_prefixes":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				responseJSON := byoipPrefixListResponse

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

	when("the valid request", func() {
		it("list the byoip prefix", func() {
			aliases := []string{"list", "ls"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"network",
					"byoip-prefix",
					alias,
				)

				output, err := cmd.CombinedOutput()

				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(norm(byoipPrefixListOutput), norm(string(output)))
			}
		})
	})

})

const (
	byoipPrefixListOutput = `
Prefix           Region    Status    UUID                                    Advertised    Failure Reason    Validations
10.1.1.1/24    	 nyc3      active    78d564a7-bc3f-4489-be14-1fb714969213   false                           []            
`
	byoipPrefixListResponse = `
{
"byoip_prefixes":[ {
    "uuid": "78d564a7-bc3f-4489-be14-1fb714969213",
    "region": "nyc3",
    "status": "active",
	"prefix": "10.1.1.1/24",
	"advertised": false,
	"validations": [],
	"failure_reason": ""
}]
}
`
)
