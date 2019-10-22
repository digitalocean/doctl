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

var _ = suite("compute/load-balancer/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/load_balancers/find-lb-id":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != "GET" {
					w.WriteHeader(http.StatusTeapot)
					return
				}

				w.Write([]byte(lbGetResponse))
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
		it("gets the specified load balancer", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"load-balancer",
				"get",
				"find-lb-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(lbGetOutput), strings.TrimSpace(string(output)))
		})
	})
})

const lbGetOutput = `
ID            IP                 Name             Status    Created At              Algorithm      Region    Tag    Droplet IDs    SSL      Sticky Sessions                                Health Check                                                                                                            Forwarding Rules
find-lb-id    104.131.186.241    example-lb-01    new       2017-02-01T22:22:58Z    round_robin    nyc3             3164445        false    type:none,cookie_name:,cookie_ttl_seconds:0    protocol:,port:0,path:,check_interval_seconds:0,response_timeout_seconds:0,healthy_threshold:0,unhealthy_threshold:0
`
const lbGetResponse = `
{
  "load_balancer": {
    "id": "find-lb-id",
    "name": "example-lb-01",
    "ip": "104.131.186.241",
    "algorithm": "round_robin",
    "status": "new",
    "created_at": "2017-02-01T22:22:58Z",
    "forwarding_rules": [],
    "health_check": {},
    "sticky_sessions": {
      "type": "none"
    },
    "region": {
      "name": "New York 3",
      "slug": "nyc3",
      "sizes": [ "s-1vcpu-1gb" ],
      "features": [
        "private_networking",
        "backups",
        "ipv6",
        "metadata",
        "install_agent"
      ],
      "available": true
    },
    "tag": "",
    "droplet_ids": [ 3164445 ],
    "redirect_http_to_https": false,
    "enable_proxy_protocol": false
  }
}`
