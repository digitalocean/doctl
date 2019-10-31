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

var _ = suite("compute/floating-ip-action", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/floating_ips/77/actions/66":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(floatingIPActionResponse))
			case "/v2/floating_ips/1/actions":
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

				expect.JSONEq(`{"type":"unassign"}`, string(reqBody))

				w.Write([]byte(floatingIPActionResponse))
			case "/v2/floating_ips/1313/actions":
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

				expect.JSONEq(`{"droplet_id":1414,"type":"assign"}`, string(reqBody))

				w.Write([]byte(floatingIPActionResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is get", func() {
		it("gets the specified floating-ip action", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"floating-ip-action",
				"get",
				"77",
				"66",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(floatingIPActionOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is assign", func() {
		it("assigns the image", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"floating-ip-action",
				"assign",
				"1313",
				"1414",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(floatingIPActionOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is unassign", func() {
		it("unassigns the image", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"floating-ip-action",
				"unassign",
				"1",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(floatingIPActionOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	floatingIPActionOutput = `
ID          Status         Type         Started At                       Completed At    Resource ID    Resource Type    Region
68212728    in-progress    assign_ip    2015-10-15 17:45:44 +0000 UTC    <nil>           758603823      floating_ip      nyc3
	`
	floatingIPActionResponse = `
{
  "action": {
    "id": 68212728,
    "status": "in-progress",
    "type": "assign_ip",
    "started_at": "2015-10-15T17:45:44Z",
    "completed_at": null,
    "resource_id": 758603823,
    "resource_type": "floating_ip",
    "region": {
      "name": "New York 3",
      "slug": "nyc3",
      "sizes": [ "s-32vcpu-192gb" ],
      "features": [ "metadata" ],
      "available": true
    },
    "region_slug": "nyc3"
  }
}
`
)
