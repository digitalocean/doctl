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

var _ = suite("genai/agent/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents":
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
				w.Write([]byte(agentCreateResponse))
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
		it("creates an agent", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"agent",
					alias,
					"--name", "test-agent",
					"--region", "tor1",
					"--project-id", "00000000-0000-4000-8000-000000000000",
					"--model-id", "00000000-0000-4000-8000-000000000000",
					"--instruction", "You are a helpful assistant",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(agentCreateOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("optional flags are passed", func() {
		it("creates an agent with optional fields", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"create",
				"--name", "test-agent",
				"--region", "tor1",
				"--project-id", "00000000-0000-4000-8000-000000000000",
				"--model-id", "00000000-0000-4000-8000-000000000000",
				"--instruction", "You are a helpful assistant",
				"--description", "A test agent for integration testing",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(agentCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("required flags are missing", func() {
		it("returns an error when name is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"create",
				"--region", "tor1",
				"--project-id", "00000000-0000-4000-8000-000000000000",
				"--model-id", "00000000-0000-4000-8000-000000000000",
				"--instruction", "You are a helpful assistant",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "required arguments")
		})

		it("returns an error when instruction is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"create",
				"--name", "test-agent",
				"--region", "tor1",
				"--project-id", "00000000-0000-4000-8000-000000000000",
				"--model-id", "00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})

		it("returns an error when model-id is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"create",
				"--name", "test-agent",
				"--region", "tor1",
				"--project-id", "00000000-0000-4000-8000-000000000000",
				"--instruction", "You are a helpful assistant",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})
	})
})

const (
	agentCreateOutput = `
ID                                      Name          Region    Project ID                              Model ID                                Created At                       User ID
00000000-0000-4000-8000-000000000000    test-agent    tor1      00000000-0000-4000-8000-000000000000    00000000-0000-4000-8000-000000000000    2023-01-01 00:00:00 +0000 UTC    user1
`
	agentCreateResponse = `
{
 "agent": {
  "uuid": "00000000-0000-4000-8000-000000000000",
  "name": "test-agent",
  "region": "tor1",
  "project_id": "00000000-0000-4000-8000-000000000000",
  "model": {
    "uuid": "00000000-0000-4000-8000-000000000000"
  },
  "instruction": "You are a helpful assistant",
  "description": "A test agent for integration testing",
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z",
  "user_id": "user1",
  "retrieval_method": "RETRIEVAL_METHOD_UNKNOWN"
}
}
`
)
