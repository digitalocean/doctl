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

var _ = suite("compute/load-balancer/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
		cmd    *exec.Cmd
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/load_balancers":
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

				expect.JSONEq(glbCreateRequest, string(reqBody))

				w.Write([]byte(glbCreateResponse))
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
			"load-balancer",
		)
	})

	when("command is create with global config", func() {
		it("creates a global load balancer", func() {
			args := append([]string{"create"}, []string{
				"--name", "my-glb-name",
				"--type", "GLOBAL",
				"--domains", "name:test-domain-1 is_managed:true certificate_id:test-cert-id-1",
				"--domains", "name:test-domain-2 is_managed:false certificate_id:test-cert-id-2",
				"--glb-settings", "target_protocol:http,target_port:80",
				"--glb-cdn-settings", "is_enabled:true",
				"--target-lb-ids", "target-lb-id-1",
				"--target-lb-ids", "target-lb-id-2",
			}...)
			cmd.Args = append(cmd.Args, args...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(glbCreateOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	glbCreateRequest = `
{
  "name": "my-glb-name",
  "algorithm": "round_robin",
  "type": "GLOBAL",
  "health_check": {},
  "sticky_sessions": {},
  "disable_lets_encrypt_dns_records": false,
  "domains": [
    {
      "name": "test-domain-1",
      "is_managed": true,
      "certificate_id": "test-cert-id-1"
    },
    {
      "name": "test-domain-2",
      "is_managed": false,
      "certificate_id": "test-cert-id-2"
    }
  ],
  "glb_settings": {
    "target_protocol": "http",
    "target_port": 80,
    "cdn": {
      "is_enabled": true
    }
  },
  "target_load_balancer_ids": [
    "target-lb-id-1",
    "target-lb-id-2"
  ]
}`
	glbCreateResponse = `
{
  "load_balancer": {
    "id": "cf9f1aa1-e1f8-4f3a-ad71-124c45e204b8",
    "name": "my-glb-name",
    "ip": "",
    "size": "lb-small",
    "size_unit": 1,
    "type": "GLOBAL",
    "algorithm": "round_robin",
    "status": "new",
    "created_at": "2024-04-09T16:10:11Z",
    "forwarding_rules": [],
    "health_check": {
      "protocol": "http",
      "port": 80,
      "path": "/",
      "check_interval_seconds": 10,
      "response_timeout_seconds": 5,
      "healthy_threshold": 5,
      "unhealthy_threshold": 3
    },
    "sticky_sessions": {
      "type": "none"
    },
    "tag": "",
    "droplet_ids": [],
    "redirect_http_to_https": false,
    "enable_proxy_protocol": false,
    "enable_backend_keepalive": false,
    "project_id": "1e02c6d8-aa24-477e-bc50-837b44e26cb3",
    "disable_lets_encrypt_dns_records": false,
    "http_idle_timeout_seconds": 60,
    "domains": [
      {
        "name": "test-domain-1",
        "is_managed": true,
        "certificate_id": "test-cert-id-1",
        "status": "CREATING"
      },
      {
        "name": "test-domain-1-2",
        "is_managed": false,
        "certificate_id": "test-cert-id-2",
        "status": "CREATING"
      }
    ],
    "glb_settings": {
      "target_protocol": "HTTP",
      "target_port": 80,
      "cdn": {
        "is_enabled": true
      }
    },
    "target_load_balancer_ids": [
      "target-lb-id-1",
      "target-lb-id-2"
    ]
  }
}`
	glbCreateOutput = `
Notice: Load balancer created
ID                                      IP    Name           Status    Created At              Region    Size        Size Unit    VPC UUID    Tag    Droplet IDs    SSL      Sticky Sessions                                Health Check                                                                                                                   Forwarding Rules    Disable Lets Encrypt DNS Records
cf9f1aa1-e1f8-4f3a-ad71-124c45e204b8          my-glb-name    new       2024-04-09T16:10:11Z    <nil>     lb-small    1                                              false    type:none,cookie_name:,cookie_ttl_seconds:0    protocol:http,port:80,path:/,check_interval_seconds:10,response_timeout_seconds:5,healthy_threshold:5,unhealthy_threshold:3                        false
`
)
