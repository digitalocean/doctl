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

var _ = suite("compute/floating-ip/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect   *require.Assertions
		cmd      *exec.Cmd
		baseArgs = []string{
			"--droplet-id", "1212",
		}
	)

	it.Before(func() {
		expect = require.New(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/floating_ips":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != "POST" {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				matchedRequest := floatingIPCreateRequest
				if !strings.Contains(string(reqBody), "droplet_id") {
					matchedRequest = floatingIPRegionCreateRequest
				}

				expect.JSONEq(matchedRequest, string(reqBody))

				w.Write([]byte(floatingIPCreateResponse))
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
			"floating-ip")
	})

	when("command is create", func() {
		it("creates the floating-ip", func() {
			args := append([]string{"create"}, baseArgs...)
			cmd.Args = append(cmd.Args, args...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(floatingIPCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is c", func() {
		it("creates the floating-ip", func() {
			args := append([]string{"c"}, []string{"--region", "newark"}...)
			cmd.Args = append(cmd.Args, args...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(floatingIPCreateOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	floatingIPCreateOutput = `
IP             Region    Droplet ID    Droplet Name
45.55.96.47    nyc3      1212          magic-name
`
	floatingIPCreateResponse = `
{
  "floating_ip": {
    "ip": "45.55.96.47",
    "droplet": {
      "id": 1212,
      "name": "magic-name"
    },
    "region": {
      "name": "New York 3",
      "slug": "nyc3",
      "sizes": [ "s-32vcpu-192gb" ],
      "features": [ "metadata" ],
      "available": true
    },
    "locked": false
  },
  "links": {}
}
`
	floatingIPCreateRequest = `
{"region":"","droplet_id":1212}
`
	floatingIPRegionCreateRequest = `
{"region":"newark"}
`
)
