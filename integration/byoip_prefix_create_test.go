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

// Normalize both strings: trim and normalize whitespace
var norm = func(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	var out []string
	for _, line := range lines {
		// Turn multiple spaces/tabs into single spaces
		fields := strings.Fields(line)
		out = append(out, strings.Join(fields, " "))
	}
	return strings.Join(out, "\n")
}

var _ = suite("network/byoip-prefix/create", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := io.ReadAll(req.Body)
				expect.NoError(err)

				matchedRequest := byoipPrefixCreateRequest
				responseJSON := byoipPrefixCreateResponse

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

	when("the valid request", func() {
		it("creates the byoip prefix", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"network",
					"byoip-prefix",
					alias,
					"--region", "nyc3",
					"--prefix", "10.1.1.1/24",
					"--signature", "signature",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(norm(byoipPrefixCreateOutput), norm(string(output)))
			}
		})
	})

})

const (
	byoipPrefixCreateOutput = `
UUID									Region	Status
78d564a7-bc3f-4489-be14-1fb714969213	nyc3	active             
`
	byoipPrefixCreateResponse = `
{
"byoip_prefix": {
    "uuid": "78d564a7-bc3f-4489-be14-1fb714969213",
    "region": "nyc3",
    "status": "active"
	}
}
`

	byoipPrefixCreateRequest = `
{"region":"nyc3","prefix":"10.1.1.1/24","signature":"signature"}
`
)
