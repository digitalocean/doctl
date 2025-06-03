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

var _ = suite("genai/knowledge-base/attach", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents/00000000-0000-4000-8000-000000000000/knowledge_bases/00000000-0000-4000-8000-000000000000":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(agentAttachResponse))
			case "/v2/gen-ai/agents/99999999-9999-4999-8999-999999999999/knowledge_bases/99999999-9999-4999-8999-999999999999":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"id":"invalid_argument","message":"failed to link knowledge base"}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("valid agent id and knowledge base id is passed", func() {
		it("attaches the knowledge base to an agent", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"attach",
				"00000000-0000-4000-8000-000000000000",
				"00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(agentCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("invalid agent id or knowledge base id is passed", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"attach",
				"99999999-9999-4999-8999-999999999999",
				"99999999-9999-4999-8999-999999999999",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "failed to link knowledge base")
		})
	})
})

const (
	agentCreateOutput = `
Name          Region    ProjectID                               CreatedAt                        UserId
test-agent    tor1      00000000-0000-4000-8000-000000000000    2023-01-01 00:00:00 +0000 UTC    user1
`

	agentAttachResponse = `
{
  "agent": {
    "uuid": "00000000-0000-4000-8000-000000000000",
    "name": "test-agent",
    "region": "tor1",
    "project_id": "00000000-0000-4000-8000-000000000000",
    "embedding_model_uuid": "00000000-0000-4000-8000-000000000000",
    "created_at": "2023-01-01T00:00:00Z",
    "user_id": "user1"
  }
}
`
)
