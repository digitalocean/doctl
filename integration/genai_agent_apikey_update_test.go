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
			case "/v2/gen-ai/agents/00000000-0000-4000-8000-000000000001/api_keys/00000000-0000-4000-8000-000000000000":
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
				w.Write([]byte(agentApiKeyUpdateResponse))
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
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"apikeys",
				"update",
				"00000000-0000-4000-8000-000000000000",
				"--agent-id", "00000000-0000-4000-8000-000000000001",
				"--name", "updated-apikey",
			)
			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(agentApiKeyUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("apikey ID is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"apikeys",
				"update",
				"--agent-id", "00000000-0000-4000-8000-000000000001",
				"--name", "updated-apikey",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing")
		})
	})

	when("authentication fails", func() {
		it("returns an unauthorized error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "invalid-token",
				"-u", server.URL,
				"genai",
				"agent",
				"apikeys",
				"update",
				"00000000-0000-4000-8000-000000000000",
				"--agent-id", "00000000-0000-4000-8000-000000000001",
				"--name", "updated-apikey",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "401")
		})
	})
})

const (
	agentApiKeyUpdateOutput = `
ID                                      Name       Created By    Secret Key         Deleted At                       Created At
123e4567-e89b-12d3-a456-426614174000    Key One    12345         Test Secret Key    2023-01-01 00:00:00 +0000 UTC    2023-01-01 00:00:00 +0000 UTC
`
	agentApiKeyUpdateResponse = `
{
"api_key_info": {
"created_at": "2023-01-01T00:00:00Z",
"created_by": "12345",
"deleted_at": "2023-01-01T00:00:00Z",
"name": "Key One",
"secret_key": "Test Secret Key",
"uuid": "123e4567-e89b-12d3-a456-426614174000"
}
}
`
)
