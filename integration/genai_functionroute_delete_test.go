// integration/functionroute_delete_test.go
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

var _ = suite("gen-ai/agent/functionroute/delete", func(t *testing.T, when spec.G, it spec.S) {
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
			"functions": []
		}`, agentID)
	)

	it.Before(func() {
		expect = require.New(t)

		// Mock API server
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents/" + agentID + "/functions/" + functionID:
				if req.Header.Get("Authorization") != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if req.Method != http.MethodDelete {
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

	// ──────────────────────────────────────────────────────────────────────
	it("deletes a function route", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "delete",
			"--agentid", agentID,
			"--functionid", functionID,
		)

		out, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("unexpected error: %s", out))
		expect.Contains(string(out), agentID)
	})

	it("deletes a function route with JSON output", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "delete",
			"--agentid", agentID,
			"--functionid", functionID,
			"--output", "json",
		)

		out, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("unexpected error: %s", out))
		expect.Contains(string(out), `"uuid": "`+agentID+`"`)
	})

	it("errors when agentid is missing", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "delete",
			"--functionid", functionID,
		)
		out, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(out), "missing required arguments")
	})

	it("errors when functionid is missing", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "delete",
			"--agentid", agentID,
		)
		out, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(out), "missing required arguments")
	})

	it("returns an authentication error", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "bad-token",
			"-u", baseURL,
			"genai", "agent", "functionroute", "delete",
			"--agentid", agentID,
			"--functionid", functionID,
		)
		out, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(out), "401")
	})

	it("shows help information", func() {
		cmd = exec.Command(builtBinaryPath,
			"genai", "agent", "functionroute", "delete", "--help",
		)
		out, err := cmd.CombinedOutput()
		expect.NoError(err)
		expect.Contains(string(out), "Delete a function route")
		expect.Contains(string(out), "--agentid")
		expect.Contains(string(out), "--functionid")
	})

	it.After(func() {
		server.Close()
	})
})
