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

var _ = suite("genai/knowledge-base/get", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(knowledgeBaseResponse))
			case "/v2/gen-ai/knowledge_bases/99999999-9999-4999-8999-999999999999":
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

	when("valid knowledge base id is passed", func() {
		it("gets the knowledge base", func() {
			aliases := []string{"get", "g"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"knowledge-base",
					alias,
					"00000000-0000-4000-8000-000000000000",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(knowledgeBaseOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("knowledge base id is not passed", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"get",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "command is missing required arguments")
		})
	})

	when("knowledge base does not exist", func() {
		it("returns a not found error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"get",
				"99999999-9999-4999-8999-999999999999",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "Error")
		})
	})

	when("invalid knowledge base ID format is provided", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"get",
				"99999999-9999-4999-8999-999999999999",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "The resource you requested could not be found")
		})
	})
})

const (
	knowledgeBaseOutput = `
AddedToAgentAt    CreatedAt                        DatabaseId                              IsPublic    EmbeddingModelUuid                      LastIndexingJob                                                                                                                                                                                                                                Name              Region            ProjectId                               Tags                  UpdatedAt                        UserId    UUID
<nil>             2025-05-21 08:22:54 +0000 UTC    00000000-0000-4000-8000-000000000000    false       00000000-0000-4000-8000-000000000000    &{1 2025-05-21 08:24:09 +0000 UTC [] 2025-05-21 08:27:31 +0000 UTC 00000000-0000-4000-8000-000000000000 BATCH_JOB_PHASE_SUCCEEDED 2025-05-21 08:24:09 +0000 UTC 22222 1 2025-05-21 08:27:31 +0000 UTC 00000000-0000-4000-8000-000000000000}    marketplace-kb    marketplace-kb    00000000-0000-4000-8000-000000000000    [marketplaceagent]    2025-05-21 08:24:09 +0000 UTC              00000000-0000-4000-8000-000000000000
`

	knowledgeBaseResponse = `
{
	"knowledge_base": {
		"uuid": "00000000-0000-4000-8000-000000000000",
		"name": "marketplace-kb",
		"created_at": "2025-05-21T08:22:54Z",
		"updated_at": "2025-05-21T08:24:09Z",
		"tags": [
			"marketplaceagent"
		],
		"region": "tor1",
		"embedding_model_uuid": "00000000-0000-4000-8000-000000000000",
		"project_id": "00000000-0000-4000-8000-000000000000",
		"database_id": "00000000-0000-4000-8000-000000000000",
		"last_indexing_job": {
			"uuid": "00000000-0000-4000-8000-000000000000",
			"knowledge_base_uuid": "00000000-0000-4000-8000-000000000000",
			"created_at": "2025-05-21T08:24:09Z",
			"updated_at": "2025-05-21T08:27:31Z",
			"started_at": "2025-05-21T08:24:09Z",
			"finished_at": "2025-05-21T08:27:31Z",
			"phase": "BATCH_JOB_PHASE_SUCCEEDED",
			"total_datasources": 1,
			"completed_datasources": 1,
			"tokens": 22222
		}
	},
	"database_status": "ONLINE"
}
`
)
