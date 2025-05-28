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

var _ = suite("genai/agent/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/genai/agents/00000000-0000-4000-8000-000000000000":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(agentGetResponse))
			case "/v2/genai/agents/99999999-9999-4999-8999-999999999999":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

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

	when("valid agent ID is provided", func() {
		it("gets the agent", func() {
			aliases := []string{"get", "g"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"agent",
					alias,
					"00000000-0000-4000-8000-000000000000",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(agentGetOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("agent ID is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"get",
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
				"get",
				"99999999-9999-4999-8999-999999999999",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "404")
		})
	})

	when("invalid agent ID format is provided", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"get",
				"invalid-uuid",
			)

			_, err := cmd.CombinedOutput()
			expect.Error(err)
		})
	})
})

const (
	agentGetOutput = `
ID                                     Name         Region    Project ID                             Model ID                               Created At                   User ID
00000000-0000-4000-8000-000000000000   test-agent   tor1      00000000-0000-4000-8000-000000000000   00000000-0000-4000-8000-000000000000   2023-01-01T00:00:00Z         user1
`
	agentGetResponse = `
{
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
  "max_tokens": 100,
  "temperature": 0.7,
  "retrieval_method": "RETRIEVAL_METHOD_UNKNOWN",
  "tags": ["test", "integration"]
}
`
)
