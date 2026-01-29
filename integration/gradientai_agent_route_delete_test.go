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

var _ = suite("gen-ai/agent/route/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
		cmd    *exec.Cmd
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
				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNoContent)
			case "/v2/gen-ai/agents/99999999-9999-4999-8999-999999999999/child_agents/00000000-0000-4000-9000-000000000000":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"id":"not_found","message":"The resource you requested could not be found."}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("valid agent parent-id and child-id is provided with force flag", func() {
		it("deletes the agent route", func() {
			aliases := []string{"delete", "del", "rm"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"gradient",
					"agent",
					"route",
					alias,
					"--parent-agent-id", "00000000-0000-4000-8000-000000000000",
					"--child-agent-id", "00000000-0000-4000-9000-000000000000",
					"--force",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			}
		})
	})

	when("agent route does not exist", func() {
		it("returns not found error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"agent",
				"route",
				"delete",
				"--parent-agent-id", "99999999-9999-4999-8999-999999999999",
				"--child-agent-id", "00000000-0000-4000-9000-000000000000",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)

			outputLower := strings.ToLower(string(output))
			expect.Contains(outputLower, "could not be found")
		})
	})

	when("invalid authentication token is provided", func() {
		it("returns unauthorized error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "invalid-token",
				"-u", server.URL,
				"gradient",
				"agent",
				"route",
				"delete",
				"--parent-agent-id", "00000000-0000-4000-8000-000000000000",
				"--child-agent-id", "00000000-0000-4000-9000-000000000000",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			// Check for common unauthorized indicators
			outputStr := strings.ToLower(string(output))
			expect.True(
				strings.Contains(outputStr, "401") ||
					strings.Contains(outputStr, "unauthorized") ||
					strings.Contains(outputStr, "auth") ||
					strings.Contains(outputStr, "authentication"),
				"Expected unauthorized error message not found in output: %s", output,
			)
		})
	})

	when("required flags are missing", func() {
		it("returns error when parent-agent-id is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"agent",
				"route",
				"delete",
				"--child-agent-id", "00000000-0000-4000-9000-000000000000",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "parent-agent-id")
		})

		it("returns error when child-agent-id is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"agent",
				"route",
				"delete",
				"--parent-agent-id", "00000000-0000-4000-8000-000000000000",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "child-agent-id")
		})
	})

	when("force flag is not provided", func() {
		it("prompts for confirmation", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"agent",
				"route",
				"delete",
				"--parent-agent-id", "00000000-0000-4000-8000-000000000000",
				"--child-agent-id", "00000000-0000-4000-9000-000000000000",
				"--no-interactive",
			)

			output, err := cmd.CombinedOutput()
			// The command should require the force flag in non-interactive mode
			expect.Error(err, "Command should require force flag in non-interactive mode")
			outputStr := strings.ToLower(string(output))
			expect.True(
				strings.Contains(outputStr, "force") ||
					strings.Contains(outputStr, "confirm") ||
					strings.Contains(outputStr, "required"),
				"Expected message about confirmation or force flag not found: %s", output,
			)
		})
	})
})
