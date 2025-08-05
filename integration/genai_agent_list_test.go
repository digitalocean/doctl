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

var _ = suite("genai/agent/list", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(agentListResponse))
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
		it("lists all agents", func() {
			aliases := []string{"list", "ls"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"agent",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(agentListOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	agentListOutput = `
ID                                      Name      Region    Project ID                              Model ID                                Created At                       User ID
00000000-0000-4000-8000-000000000000    Agent1    tor1      00000000-0000-4000-8000-000000000000    00000000-0000-4000-8000-000000000000    2023-01-01 00:00:00 +0000 UTC    user1
`
	agentListResponse = `
{
  "agents": [
    {
      "uuid": "00000000-0000-4000-8000-000000000000",
      "name": "Agent1",
      "region": "tor1",
      "project_id": "00000000-0000-4000-8000-000000000000",
      "model": {
        "uuid": "00000000-0000-4000-8000-000000000000"
      },
      "instruction": "You are an agent who thinks deeply about the world",
      "created_at": "2023-01-01T00:00:00Z",
      "user_id": "user1"
    }
  ],
  "links": {},
  "meta": {
    "total": 1
  }
}
`
)
