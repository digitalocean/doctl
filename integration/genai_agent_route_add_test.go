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

var _ = suite("gen-ai/agent/route/add", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect  *require.Assertions
		server  *httptest.Server
		cmd     *exec.Cmd
		baseURL string
	)

	it.Before(func() {
		expect = require.New(t)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents/00000000-0000-4000-8000-000000000000/child_agents/00000000-0000-4000-9000-000000000000":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{
                    "parent_agent_uuid": "00000000-0000-4000-8000-000000000000",
                    "child_agent_uuid": "00000000-0000-4000-9000-000000000000"
                }`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

		u, err := url.Parse(server.URL)
		expect.NoError(err)
		baseURL = u.String()
	})

	it("creates an agent route", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai",
			"agent",
			"route",
			"add",
			"--parent-agent-id", "00000000-0000-4000-8000-000000000000",
			"--child-agent-id", "00000000-0000-4000-9000-000000000000",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		expect.Contains(string(output), "00000000-0000-4000-8000-000000000000")
		expect.Contains(string(output), "00000000-0000-4000-9000-000000000000")
	})

	it("creates an agent route with JSON output", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai",
			"agent",
			"route",
			"add",
			"--parent-agent-id", "00000000-0000-4000-8000-000000000000",
			"--child-agent-id", "00000000-0000-4000-9000-000000000000",
			"--output", "json",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		expect.Contains(string(output), `"parent_agent_uuid": "00000000-0000-4000-8000-000000000000"`)
		expect.Contains(string(output), `"child_agent_uuid": "00000000-0000-4000-9000-000000000000"`)
	})

	it("returns an error when parent-agent-id is missing", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai",
			"agent",
			"route",
			"add",
			"--child-agent-id", "00000000-0000-4000-9000-000000000000",
		)

		output, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), "missing required arguments")
	})

	it("returns an error when child-agent-id is missing", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai",
			"agent",
			"route",
			"add",
			"--parent-agent-id", "00000000-0000-4000-8000-000000000000",
		)

		output, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), "missing required arguments")
	})

	it("returns an error when both agent IDs are missing", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", baseURL,
			"genai",
			"agent",
			"route",
			"add",
		)

		output, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), "missing required arguments")
	})

	it("returns an authentication error", func() {
		cmd = exec.Command(builtBinaryPath,
			"-t", "invalid-token",
			"-u", baseURL,
			"genai",
			"agent",
			"route",
			"add",
			"--parent-agent-id", "00000000-0000-4000-8000-000000000000",
			"--child-agent-id", "00000000-0000-4000-9000-000000000000",
		)

		output, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), "401")
	})

	it("shows help information", func() {
		cmd = exec.Command(builtBinaryPath,
			"genai",
			"agent",
			"route",
			"add",
			"--help",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err)
		expect.Contains(string(output), "Use this command to add an agent route to an agent")
		expect.Contains(string(output), "--parent-agent-id")
		expect.Contains(string(output), "--child-agent-id")
	})

	it.After(func() {
		if server != nil {
			server.Close()
		}
	})
})
