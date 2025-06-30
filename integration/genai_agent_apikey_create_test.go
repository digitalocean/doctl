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

var _ = suite("genai/agent/apikey/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents/00000000-0000-4000-8000-000000000000/api_keys":
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
		it("creates an api key", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"agent",
					"apikeys",
					alias,
					"--name", "API Key One",
					"--agent-uuid", "00000000-0000-4000-8000-000000000000",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(agentApiKeyCreateOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("required flags are missing", func() {
		it("returns an error when name is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"apikeys",
				"create",
				"--name", "test-key",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "required arguments")
		})

		it("returns an error when agent-uuid is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"apikeys",
				"create",
				"--name", "test-key",
			)
			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "required arguments")
		})
	})
})

const (
	agentApiKeyCreateOutput = `
ID                                      Name              Created At                             Created By  			Secret Key                               Deleted At                
123e4567-e89b-12d3-a456-426614174000    Key one          2023-01-01 00:00:00 +0000 UTC    			12345    		Test Secret Key					2023-01-01 00:00:00 +0000 UTC   
`
	agentApiKeyCreateResponse = `

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
