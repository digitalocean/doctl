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

var _ = suite("genai/knowledge-base/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/knowledge_bases":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(knowledgeBaseCreateResponse))
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
		it("creates an knowledge base", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"create",
				"--name", "test-knowledge-base",
				"--region", "tor1",
				"--project-id", "00000000-0000-4000-8000-000000000000",
				"--embedding-model-uuid", "00000000-0000-4000-8000-000000000000",
				"--data-sources", `[{"web_crawler_data_source":{"base_url":"https://example.com","crawling_option":"Unknown","embed_media":true}}]`,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(knowledgeBaseCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("optional flags are passed", func() {
		it("creates an agent with optional fields", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"create",
				"--name", "test-knowledge-base",
				"--region", "tor1",
				"--project-id", "00000000-0000-4000-8000-000000000000",
				"--embedding-model-uuid", "00000000-0000-4000-8000-000000000000",
				"--data-sources", `[{"web_crawler_data_source":{"base_url":"https://example.com","crawling_option":"Unknown","embed_media":true}}]`,
				"--tags", "field1",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(knowledgeBaseCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("required flags are missing", func() {
		it("returns an error when name is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"create",
				"--region", "tor1",
				"--project-id", "00000000-0000-4000-8000-000000000000",
				"--embedding-model-uuid", "00000000-0000-4000-8000-000000000000",
				"--data-sources", `[{"web_crawler_data_source":{"base_url":"https://example.com","crawling_option":"Unknown","embed_media":true}}]`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required")
		})

		it("returns an error when region is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"create",
				"--name", "test-knowledge-base",
				"--project-id", "00000000-0000-4000-8000-000000000000",
				"--embedding-model-uuid", "00000000-0000-4000-8000-000000000000",
				"--data-sources", `[{"web_crawler_data_source":{"base_url":"https://example.com","crawling_option":"Unknown","embed_media":true}}]`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required")
		})

		it("returns an error when project-id is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"create",
				"--name", "test-knowledge-base",
				"--region", "tor1",
				"--embedding-model-uuid", "00000000-0000-4000-8000-000000000000",
				"--data-sources", `[{"web_crawler_data_source":{"base_url":"https://example.com","crawling_option":"Unknown","embed_media":true}}]`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required")
		})
	})
})

var _ = suite("genai/knowledge-base/add-datasource", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/knowledge_bases/00000000-0000-4000-8000-000000000000/data_sources":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(knowledgeBaseAddDataSourceResponse))
			case "/v2/gen-ai/knowledge_bases/99999999-9999-4999-8999-999999999999/data_sources":
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
				w.Write([]byte(`{"id":"invalid_argument","message":"failed to get authorized KB"}`))
			case "/v2/gen-ai/knowledge_bases/99999999-9999-4999-8999-999999999998/data_sources":
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
				w.Write([]byte(`{"id":"invalid_argument","message":"failed to validate knowledge base datasource input"}`))
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
		it("creates a data source for knowledge base", func() {
			aliases := []string{"add-datasource", "add-ds"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"knowledge-base",
					alias,
					"00000000-0000-4000-8000-000000000000",
					"--base-url", "https://example.com",
					"--crawling-option", "DOMAIN",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(knowledgeBaseAddDataSourceOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("optional flags are passed", func() {
		it("creates a data source for knowledge base with optional fields", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"add-datasource",
				"00000000-0000-4000-8000-000000000000",
				"--base-url", "https://example.com",
				"--crawling-option", "DOMAIN",
				"--embed-media", "true",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(knowledgeBaseAddDataSourceOutput), strings.TrimSpace(string(output)))
		})
	})

	when("when invalid knowledge base id is passed", func() {
		it("return an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"add-datasource",
				"99999999-9999-4999-8999-999999999999",
				"--base-url", "https://example.com",
				"--crawling-option", "DOMAIN",
				"--embed-media", "true",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "failed to get authorized KB")
		})
	})

	when("required parameters are missing", func() {
		it("returns an error when base url is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"add-datasource",
				"99999999-9999-4999-8999-999999999998",
				"--crawling-option", "DOMAIN",
				"--embed-media", "true",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "either --bucket-name and --region or --base-url must be provided")
		})
	})
})

const (
	knowledgeBaseCreateOutput = `
Added To Agent At    Created At                       Database Id                             Is Public    Embedding Model Uuid                    Last Indexing Job    Name                   Region    Project Id                              Tags        Updated At                       User Id    UUID
<nil>                2025-05-23 11:23:24 +0000 UTC    00000000-0000-4000-8000-000000000000    false        00000000-0000-4000-8000-000000000000    <nil>                test-knowledge_base    tor1      00000000-0000-4000-8000-000000000000    [field1]    2025-05-23 11:23:24 +0000 UTC               00000000-0000-4000-8000-000000000000
`

	knowledgeBaseCreateResponse = `
{
	"knowledge_base": {
		"uuid": "00000000-0000-4000-8000-000000000000",
		"name": "test-knowledge_base",
		"created_at": "2025-05-23T11:23:24Z",
		"updated_at": "2025-05-23T11:23:24Z",
		"tags": [
			"field1"
		],
		"region": "tor1",
		"embedding_model_uuid": "00000000-0000-4000-8000-000000000000",
		"project_id": "00000000-0000-4000-8000-000000000000",
		"database_id": "00000000-0000-4000-8000-000000000000"
	}
}
`

	knowledgeBaseAddDataSourceOutput = `
Created At                       File Upload Datasource    Last Indexing Job    Spaces Datasource    Updated At                       UUID                                    Web Crawler Datasource
2025-05-29 12:17:56 +0000 UTC    <nil>                     <nil>                <nil>                2025-05-29 12:17:56 +0000 UTC    00000000-0000-4000-8000-000000000000    &{https://example.com DOMAIN true}
`

	knowledgeBaseAddDataSourceResponse = `
{
	"knowledge_base_data_source": {
		"uuid": "00000000-0000-4000-8000-000000000000",
		"created_at": "2025-05-29T12:17:56Z",
		"updated_at": "2025-05-29T12:17:56Z",
		"web_crawler_data_source": {
			"base_url": "https://example.com",
			"crawling_option": "DOMAIN",
            "embed_media": true
		}
	}
}
`
)
