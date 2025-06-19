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

var _ = suite("genai/knowledge-bases", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/knowledge_bases/00000000-0000-4000-8000-000000000000":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPut {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(knowledgeBasesUpdateResponse))
			case "/v2/gen-ai/knowledge_bases/99999999-9999-4999-8999-999999999999":
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
				w.Write([]byte(`{"id":"not_found","message":"failed to get knowledge base"}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("valid knowledge base ID and update fields are provided", func() {
		it("updates the knowledge base", func() {
			aliases := []string{"update", "u"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"knowledge-base",
					alias,
					"00000000-0000-4000-8000-000000000000",
					"--name", "updated-agent",
					"--tags", "tag1,tag2",
					"--embedding-model-uuid", "00000000-0000-4000-8000-000000000000",
					"--project-id", "00000000-0000-4000-8000-000000000000",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", string(output)))
				expect.Equal(strings.TrimSpace(knowledgeBasesUpdateOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("all update fields are provided", func() {
		it("updates the knowledge base with all fields", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"update",
				"00000000-0000-4000-8000-000000000000",
				"--name", "updated-agent",
				"--tags", "updated,description",
				"--embedding-model-uuid", "00000000-0000-4000-8000-000000000000",
				"--project-id", "00000000-0000-4000-8000-000000000000",
				"--database-id", "00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(knowledgeBasesUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("only name is updated", func() {
		it("updates only the agent name", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"update",
				"00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(knowledgeBasesUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("agent ID is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
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
				"knowledge-base",
				"update",
				"99999999-9999-4999-8999-999999999999",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "failed to get knowledge base")
		})
	})

	when("authentication fails", func() {
		it("returns an unauthorized error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "invalid-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"update",
				"00000000-0000-4000-8000-000000000000",
				"--name", "updated-agent",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "401")
		})
	})
})

const (
	knowledgeBasesUpdateOutput = `
AddedToAgentAt    CreatedAt                        DatabaseId                              IsPublic    EmbeddingModelUuid                      LastIndexingJob                                                                                                                                                                                                                               Name                 Region               ProjectId                               Tags                UpdatedAt                        UserId    UUID
<nil>             2025-05-29 09:07:59 +0000 UTC    00000000-0000-4000-8000-000000000000    false       00000000-0000-4000-8000-000000000000    &{1 2025-05-29 09:12:33 +0000 UTC [] 2025-05-29 09:13:00 +0000 UTC 00000000-0000-4000-8000-000000000000 BATCH_JOB_PHASE_SUCCEEDED 2025-05-29 09:12:33 +0000 UTC 1750 1 2025-05-29 09:13:13 +0000 UTC 00000000-0000-4000-8000-000000000000}    My Knowledge Base    My Knowledge Base    00000000-0000-4000-8000-000000000000    [example string]    2025-05-29 14:27:15 +0000 UTC              00000000-0000-4000-8000-000000000000
`
	knowledgeBasesUpdateResponse = `
{
	"knowledge_base": {
		"uuid": "00000000-0000-4000-8000-000000000000",
		"name": "My Knowledge Base",
		"created_at": "2025-05-29T09:07:59Z",
		"updated_at": "2025-05-29T14:27:15Z",
		"tags": [
			"example string"
		],
		"region": "tor1",
		"embedding_model_uuid": "00000000-0000-4000-8000-000000000000",
		"project_id": "00000000-0000-4000-8000-000000000000",
		"database_id": "00000000-0000-4000-8000-000000000000",
		"last_indexing_job": {
			"uuid": "00000000-0000-4000-8000-000000000000",
			"knowledge_base_uuid": "00000000-0000-4000-8000-000000000000",
			"created_at": "2025-05-29T09:12:33Z",
			"updated_at": "2025-05-29T09:13:13Z",
			"started_at": "2025-05-29T09:12:33Z",
			"finished_at": "2025-05-29T09:13:00Z",
			"phase": "BATCH_JOB_PHASE_SUCCEEDED",
			"total_datasources": 1,
			"completed_datasources": 1,
			"tokens": 1750
		}
	}
}
`
)
