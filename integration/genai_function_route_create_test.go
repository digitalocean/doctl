// integration/functionroute_create_test.go
package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("gen-ai/agent/functionroute/create", func(t *testing.T, when spec.G, it spec.S) {
	const (
		agentID = "00000000-0000-4000-8000-000000000000"
		fnUUID  = "11111111-2222-3333-4444-555555555555"
	)

	var (
		expect  *require.Assertions
		server  *httptest.Server
		cmd     *exec.Cmd
		baseURL string

		// keep the payload terse – just enough for the CLI to show something
		successBody = fmt.Sprintf(`{
			"uuid": "%s",
			"functions": [{
				"uuid": "%s",
				"name": "get-weather",
				"description": "integration-test fn route"
			}]
		}`, agentID, fnUUID)
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents/" + agentID + "/functions":
				// Auth
				if req.Header.Get("Authorization") != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				// Method
				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(successBody))
			default:
				dump, _ := httputil.DumpRequest(req, true)
				t.Fatalf("received unknown request:\n%s", dump)
			}
		}))

		u, _ := url.Parse(server.URL)
		baseURL = u.String()
	})

	// ─────────────────────────────────────────────────────────────────────────────
	it("creates a function route", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "create",
			"--agentid", agentID,
			"--name", "get-weather",
			"--description", "integration-test fn route",
			"--faas-name", "default/testing",
			"--faas-namespace", "fn-ns",
			"--input-schema", `{"parameters":[]}`,
			"--output-schema", `{"properties":[]}`,
		)

		out, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("unexpected error: %s", out))
		expect.Contains(string(out), "get-weather")
		expect.Contains(string(out), fnUUID)
	})

	it("creates a function route with JSON output", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "create",
			"--agentid", agentID,
			"--name", "get-weather",
			"--description", "integration-test fn route",
			"--faas-name", "default/testing",
			"--faas-namespace", "fn-ns",
			"--input-schema", `{"parameters":[]}`,
			"--output-schema", `{"properties":[]}`,
			"--output", "json",
		)

		out, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("unexpected error: %s", out))
		expect.Contains(string(out), `"name": "get-weather"`)
		expect.Contains(string(out), `"uuid": "`+fnUUID+`"`)
	})

	it("errors when required flags are missing", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "create",
			// missing --agentid
			"--name", "get-weather",
		)
		out, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(out), "missing required arguments")
	})

	it("returns an authentication error", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "bad-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "create",
			"--agentid", agentID,
			"--name", "get-weather",
			"--description", "integration-test fn route",
			"--faas-name", "default/testing",
			"--faas-namespace", "fn-ns",
			"--input-schema", `{"parameters":[]}`,
			"--output-schema", `{"properties":[]}`,
		)
		out, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(out), "401")
	})

	it("shows help information", func() {
		cmd = exec.Command(builtBinaryPath,
			"genai", "agent", "functionroute", "create", "--help",
		)
		out, err := cmd.CombinedOutput()
		expect.NoError(err)
		expect.Contains(string(out), "Creates a function route")
		expect.Contains(string(out), "--agentid")
		expect.Contains(string(out), "--faas-name")
	})

	it.After(func() {
		server.Close()
	})
})
