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

var _ = suite("monitoring/alerts/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/monitoring/alerts/938f5546-508d-9956-a27f-99bdf31a75b9":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(alertPolicyGetResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("returns the details of my alert policy", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"monitoring",
			"alerts",
			"get",
			"938f5546-508d-9956-a27f-99bdf31a75b9",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(getAlertPolicyOutput), strings.TrimSpace(string(output)))
	})
})

const (
	alertPolicyGetResponse = `
{
 "policy": {
    "uuid": "938f5546-508d-9956-a27f-99bdf31a75b9",
    "type": "v1/insights/droplet/cpu",
    "description": "CPU Utilization is running high",
    "compare": "GreaterThan",
    "value": 99,
    "window": "5m",
    "entities": [
    ],
    "tags": [],
    "alerts": {
      "slack": [],
      "email": [
        "bob@example.com"
      ]
    },
    "enabled": true
  }
}`
	getAlertPolicyOutput = `
UUID                                    Type                       Description                        Compare        Value    Window    Entities    Tags    Emails             Slack Channels    Enabled
938f5546-508d-9956-a27f-99bdf31a75b9    v1/insights/droplet/cpu    CPU Utilization is running high    GreaterThan    99       5m        []          []      bob@example.com                      true
`
)

var _ = suite("monitoring/alerts/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/monitoring/alerts":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(alertPolicyCreateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("creates a new alert policy", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"monitoring",
			"alerts",
			"create",
			"--compare",
			"GreaterThan",
			"--value",
			"50",
			"--type",
			"v1/insights/droplet/cpu",
			"--description",
			"Test alert",
			"--enabled",
			"--tags",
			"mytag",
			"--emails",
			"fake@example.com",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(createAlertPolicyOutput), strings.TrimSpace(string(output)))
	})
})

const (
	alertPolicyCreateResponse = `
{
 "policy": {
    "uuid": "938f5546-508d-9956-a27f-99bdf31a75b9",
    "type": "v1/insights/droplet/cpu",
    "description": "Test alert",
    "compare": "GreaterThan",
    "value": 50,
    "window": "5m",
    "entities": [
    ],
    "tags": ["mytag"],
    "alerts": {
      "slack": [],
      "email": [
        "fake@example.com"
      ]
    },
    "enabled": true
  }
}`
	createAlertPolicyOutput = `
UUID                                    Type                       Description    Compare        Value    Window    Entities    Tags       Emails              Slack Channels    Enabled
938f5546-508d-9956-a27f-99bdf31a75b9    v1/insights/droplet/cpu    Test alert     GreaterThan    50       5m        []          [mytag]    fake@example.com                      true
`
)

var _ = suite("monitoring/alerts/update", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/monitoring/alerts/938f5546-508d-9956-a27f-99bdf31a75b9":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPut {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(alertPolicyUpdateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("updates an existing alert policy", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"monitoring",
			"alerts",
			"update",
			"938f5546-508d-9956-a27f-99bdf31a75b9",
			"--compare",
			"LessThan",
			"--value",
			"82",
			"--type",
			"v1/insights/droplet/cpu",
			"--description",
			"TEST ALERT",
			"--tags",
			"abcde",
			"--emails",
			"abcd@example.com",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)

		expect.Equal(strings.TrimSpace(updateAlertPolicyOutput), strings.TrimSpace(string(output)))
	})
})

const (
	alertPolicyUpdateResponse = `
{
 "policy": {
    "uuid": "938f5546-508d-9956-a27f-99bdf31a75b9",
    "type": "v1/insights/droplet/cpu",
    "description": "Test ALERT",
    "compare": "LessThan",
    "value": 82,
    "window": "5m",
    "entities": [
    ],
    "tags": ["abcde"],
    "alerts": {
      "slack": [],
      "email": [
        "abcd@example.com"
      ]
    },
    "enabled": false
  }
}`
	updateAlertPolicyOutput = `
UUID                                    Type                       Description    Compare     Value    Window    Entities    Tags       Emails              Slack Channels    Enabled
938f5546-508d-9956-a27f-99bdf31a75b9    v1/insights/droplet/cpu    Test ALERT     LessThan    82       5m        []          [abcde]    abcd@example.com                      false
`
)
