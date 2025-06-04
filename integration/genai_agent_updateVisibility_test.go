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

var _ = suite("genai/agent/update-visibility", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents/00000000-0000-4000-8000-000000000000/deployment_visibility":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPut {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				// Check the request body for invalid visibility values
				body := make([]byte, req.ContentLength)
				req.Body.Read(body)
				if strings.Contains(string(body), "INVALID_VISIBILITY") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"id":"bad_request","message":"Invalid visibility value provided."}`))
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(agentUpdateVisibilityResponse))
			case "/v2/gen-ai/agents/99999999-9999-4999-8999-999999999999/deployment_visibility":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPut {
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

	when("valid agent ID and visibility are provided", func() {
		it("updates agent visibility to public", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update-visibility",
				"00000000-0000-4000-8000-000000000000",
				"--visibility", "VISIBILITY_PUBLIC",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(agentUpdateVisibilityOutput), strings.TrimSpace(string(output)))
		})

		it("updates agent visibility to private", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update-visibility",
				"00000000-0000-4000-8000-000000000000",
				"--visibility", "VISIBILITY_PRIVATE",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(agentUpdateVisibilityOutput), strings.TrimSpace(string(output)))
		})
	})

	when("agent ID is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update-visibility",
				"--visibility", "VISIBILITY_PUBLIC",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})
	})

	when("visibility flag is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update-visibility",
				"00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})
	})

	when("invalid visibility value is provided", func() {
		it("returns an error for invalid visibility", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update-visibility",
				"00000000-0000-4000-8000-000000000000",
				"--visibility", "INVALID_VISIBILITY",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "Invalid visibility value provided")
		})
	})

	when("edge case visibility values", func() {
		it("handles lowercase visibility values", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"update-visibility",
				"00000000-0000-4000-8000-000000000000",
				"--visibility", "visibility_public",
			)

			output, err := cmd.CombinedOutput()
			if err != nil {
				expect.Contains(string(output), "invalid")
			} else {
				expect.Equal(strings.TrimSpace(agentUpdateVisibilityOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	agentUpdateVisibilityOutput = `
ID                                      Name          Region    Project ID                              Model ID                                Created At                       User ID
00000000-0000-4000-8000-000000000000    test-agent    tor1      00000000-0000-4000-8000-000000000000    00000000-0000-4000-8000-000000000000    2023-01-01 00:00:00 +0000 UTC    100000000
`
	agentUpdateVisibilityResponse = `
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
  "updated_at": "2023-01-01T01:00:00Z",
  "user_id": "100000000",
  "max_tokens": 100,
  "temperature": 0.7,
  "retrieval_method": "RETRIEVAL_METHOD_UNKNOWN",
  "visibility": "VISIBILITY_PUBLIC"
}
}
`
)
