// integration/functionroute_update_test.go
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

var _ = suite("gen-ai/agent/functionroute/update", func(t *testing.T, when spec.G, it spec.S) {
	const (
		agentID    = "00000000-0000-4000-8000-000000000000"
		functionID = "11111111-2222-3333-4444-555555555555"
	)

	var (
		expect  *require.Assertions
		server  *httptest.Server
		cmd     *exec.Cmd
		baseURL string

		successBody = fmt.Sprintf(`{
			"uuid": "%s",
			"functions": [{
				"uuid": "%s",
				"name": "get-weather",
				"description": "updated-desc"
			}]
		}`, agentID, functionID)
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents/" + agentID + "/functions/" + functionID:
				if req.Header.Get("Authorization") != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				// accept either PUT or PATCH depending on backend
				if req.Method != http.MethodPut && req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
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
	it("updates a function route", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "update",
			"--agentid", agentID,
			"--functionid", functionID,
			"--description", "updated-desc",
		)

		out, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("unexpected error: %s", out))
		expect.Contains(string(out), "updated-desc")
		expect.Contains(string(out), functionID)
	})

	it("updates a function route with JSON output", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "update",
			"--agentid", agentID,
			"--functionid", functionID,
			"--description", "updated-desc",
			"--output", "json",
		)

		out, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("unexpected error: %s", out))
		expect.Contains(string(out), `"description": "updated-desc"`)
		expect.Contains(string(out), `"uuid": "`+functionID+`"`)
	})

	it("errors when agentid is missing", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "update",
			"--functionid", functionID,
			"--description", "updated-desc",
		)
		out, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(out), "missing required arguments")
	})

	it("errors when functionid is missing", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "update",
			"--agentid", agentID,
			"--description", "updated-desc",
		)
		out, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(out), "missing required arguments")
	})

	it("returns an authentication error", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "bad-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "update",
			"--agentid", agentID,
			"--functionid", functionID,
			"--description", "updated-desc",
		)
		out, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(out), "401")
	})

	it("shows help information", func() {
		cmd = exec.Command(builtBinaryPath,
			"genai", "agent", "functionroute", "update", "--help",
		)
		out, err := cmd.CombinedOutput()
		expect.NoError(err)
		expect.Contains(string(out), "Updates a function route")
		expect.Contains(string(out), "--agentid")
		expect.Contains(string(out), "--functionid")
	})

	it.After(func() {
		server.Close()
	})
})
