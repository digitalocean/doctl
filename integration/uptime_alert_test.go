package integration

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("monitoring/uptime/alerts/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/uptime/checks/valid-check-uuid/alerts/valid-alert-uuid":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(uptimeAlertGetResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("returns the details of the uptime alert", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"monitoring",
			"uptime",
			"alert",
			"get",
			"valid-check-uuid",
			"valid-alert-uuid",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(uptimeAlertGetOutput), strings.TrimSpace(string(output)))
	})
})

const (
	uptimeAlertGetResponse = `
{
	"alert": {
		"id": "valid-alert-uuid",
		"name": "example.com is down",
		"type": "down",
		"threshold": 1,
		"comparison": "less_than",
		"notifications": {
		"email": [
			"sammy@digitalocean.com"
		],
		"slack": [
			{
			"channel": "#alerts",
			"url": "https://hooks.slack.com/services/ID"
			}
		]
		},
		"period": "2m"
	}
}`
	uptimeAlertGetOutput = `
ID                  Name                   Type    Threshold    Comparison    Period    Emails                    Slack Channels
valid-alert-uuid    example.com is down    down    1            less_than     2m        sammy@digitalocean.com    #alerts
`
)

var _ = suite("monitoring/uptime/alerts/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/uptime/checks/valid-check-uuid/alerts":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(uptimeAlertListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("lists all uptime alerts", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"monitoring",
			"uptime",
			"alert",
			"list",
			"valid-check-uuid",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))

		expect.Equal(strings.TrimSpace(uptimeAlertListOutput), strings.TrimSpace(string(output)))
	})
})

const (
	uptimeAlertListResponse = `
{
	"alerts":[
	{
		"id": "valid-alert-uuid",
		"name": "example.com is down",
		"type": "down",
		"threshold": 1,
		"comparison": "less_than",
		"notifications": {
		"email": [
			"sammy@digitalocean.com"
		],
		"slack": [
			{
			"channel": "#alerts",
			"url": "https://hooks.slack.com/services/ID"
			}
		]
		},
		"period": "2m"
	},
    {
		"id": "fee528f9-73a2-46a9-a248-84056c9a4488",
		"name": "example.com increased latency",
		"type": "latency",
		"threshold": 1000,
		"comparison": "greater_than",
		"notifications": {
		  "email": [
			"sammy@digitalocean.com"
		  ]
		},
		"period": "2m"
	  }
	]
}`
	uptimeAlertListOutput = `
ID                                      Name                             Type       Threshold    Comparison      Period    Emails                    Slack Channels
valid-alert-uuid                        example.com is down              down       1            less_than       2m        sammy@digitalocean.com    #alerts
fee528f9-73a2-46a9-a248-84056c9a4488    example.com increased latency    latency    1000         greater_than    2m        sammy@digitalocean.com
`
)

var _ = suite("monitoring/uptime/alerts/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/uptime/checks/valid-check-uuid/alerts":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(uptimeAlertPolicyCreateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("creates a new uptime alert", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"monitoring",
			"uptime",
			"alerts",
			"create",
			"--comparison", "less_than",
			"--type", "down",
			"--period", "2m",
			"--name", "example.com is down",
			"--emails", "sammy@digitalocean.com",
			"--threshold", "1",
			"--slack-channels", "#alerts",
			"--slack-urls", "https://hooks.slack.com/services/ID",
			"valid-check-uuid",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))

		expect.Equal(strings.TrimSpace(createUptimeAlertPolicyOutput), strings.TrimSpace(string(output)))
	})
})

const (
	uptimeAlertPolicyCreateResponse = `{
	"alert": {
		"id": "valid-alert-uuid",
		"name": "example.com is down",
		"type": "down",
		"threshold": 1,
		"comparison": "less_than",
		"notifications": {
		"email": [
			"sammy@digitalocean.com"
		],
		"slack": [
			{
			"channel": "#alerts",
			"url": "https://hooks.slack.com/services/ID"
			}
		]
		},
		"period": "2m"
	}
}`

	createUptimeAlertPolicyOutput = `
ID                  Name                   Type    Threshold    Comparison    Period    Emails                    Slack Channels
valid-alert-uuid    example.com is down    down    1            less_than     2m        sammy@digitalocean.com    #alerts`
)

var _ = suite("monitoring/uptime/alerts/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/uptime/checks/valid-check-uuid/alerts/valid-alert-uuid":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

	})

	when("required flags are passed", func() {
		it("deletes the uptime alert", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"monitoring",
				"uptime",
				"alert",
				"delete",
				"valid-check-uuid",
				"valid-alert-uuid",
			)
			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Empty(output)
		})
	})
})
