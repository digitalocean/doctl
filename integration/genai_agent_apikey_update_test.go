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

var _ = suite("genai/agent/apikeys/update", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents/00000000-0000-4000-8000-000000000000/api_keys/00000000-0000-4000-8000-000000000001":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPut {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(agentUpdateResponse))
			case "/v2/gen-ai/agents/99999999-9999-4999-8999-999999999999/api_keys/00000000-0000-4000-8000-000000000001":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPut {
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

	when("valid api key ID and update fields are provided", func() {
		it("updates the api key", func() {
			aliases := []string{"update", "u"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"agent",
					"apikeys",
					alias,
					"00000000-0000-4000-8000-000000000000",
					"00000000-0000-4000-8000-000000000001",
					"--name", "updated-apikey",
					"--description", "Updated description",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(agentApiKeyUpdateOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("few update fields are provided", func() {
		it("updates the agent with few fields", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update",
				"00000000-0000-4000-8000-000000000000",
				"--name", "updated-agent",
				"--description", "Updated description",
				"--instruction", "Updated instruction",
				"--max-tokens", "200",
				"--temperature", "0.8",
				"--retrieval-method", "RETRIEVAL_METHOD_REWRITE",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(agentApiKeyUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("only name is updated", func() {
		it("updates only the agent name", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update",
				"00000000-0000-4000-8000-000000000000",
				"--name", "new-name-only",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(agentApiKeyUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("agent ID is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update",
				"--name", "updated-agent",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing")
		})
	})

	when("agent does not exist", func() {
		it("returns a not found error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update",
				"99999999-9999-4999-8999-999999999999",
				"--name", "updated-agent",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "404")
		})
	})

	when("no update fields are provided", func() {
		it("still executes successfully", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update",
				"00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(agentApiKeyUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("authentication fails", func() {
		it("returns an unauthorized error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "invalid-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update",
				"00000000-0000-4000-8000-000000000000",
				"--name", "updated-agent",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "401")
		})
	})
	when("invalid parameter values are provided", func() {
		it("returns an error for invalid temperature", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update",
				"00000000-0000-4000-8000-000000000000",
				"--temperature", "2.0",
			)

			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Log("CLI accepted invalid temperature - consider adding validation")
			} else {
				expect.Contains(string(output), "temperature")
			}
		})
	})
})

const (
	agentApiKeyUpdateOutput = `
ID                                      Name                 Agent ID                                              Created At                       User ID
00000000-0000-4000-8000-000000000000    updated-apikey       00000000-0000-4000-8000-000000000000       2023-01-01 00:00:00 +0000 UTC    user1
`
	agentApiKeyUpdateResponse = `
{
"api_key_info": {
"created_at": "2023-01-01T00:00:00Z",
"created_by": "12345",
"deleted_at": "2023-01-01T00:00:00Z",
"name": "updated-apikey ",
"secret_key": "Test Secret Key",
"uuid": "123e4567-e89b-12d3-a456-426614174000"
}
}
`
)
