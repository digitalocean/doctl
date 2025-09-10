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

var _ = suite("genai/knowledgebase/list", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(knowledgeBaseListResponse))
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
		it("lists all knowledge bases", func() {
			aliases := []string{"list", "ls"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"knowledge-base",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(knowledgeBaseListOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

var _ = suite("genai/knowledgebase/list-datasource", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(knowledgeBaseListDataSourceResponse))
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
		it("lists all knowledge bases", func() {
			aliases := []string{"list-datasources", "ls-ds"}

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
				expect.Equal(strings.TrimSpace(knowledgeBaseListDataSourceOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	knowledgeBaseListOutput = `
Added To Agent At    Created At                       Database Id                             Is Public    Embedding Model Uuid                    Last Indexing Job                                                                                                                                                                                                                           Name                   Region    Project Id                              Tags    Updated At                       User Id    UUID
<nil>                2025-05-29 09:07:59 +0000 UTC    00000000-0000-4000-8000-000000000000    false        00000000-0000-4000-8000-000000000000    &{0 2025-05-29 09:12:33 +0000 UTC [] 2025-05-29 09:13:00 +0000 UTC 00000000-0000-4000-8000-000000000000 BATCH_JOB_PHASE_SUCCEEDED 2025-05-29 09:12:33 +0000 UTC  0 0 2025-05-29 09:13:13 +0000 UTC 00000000-0000-4000-8000-000000000000}    deka-knowledge_base    tor1      00000000-0000-4000-8000-000000000000    []      2025-05-29 09:12:33 +0000 UTC               00000000-0000-4000-8000-000000000000
`

	knowledgeBaseListResponse = `
{
	"knowledge_bases": [
		{
			"uuid": "00000000-0000-4000-8000-000000000000",
			"name": "deka-knowledge_base",
			"created_at": "2025-05-29T09:07:59Z",
			"updated_at": "2025-05-29T09:12:33Z",
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
				"phase": "BATCH_JOB_PHASE_SUCCEEDED"
			}
		}
	]
}
`
	knowledgeBaseListDataSourceOutput = `
Created At                       File Upload Datasource    Last Indexing Job    Spaces Datasource    Updated At                       UUID                                    Web Crawler Datasource
2025-05-29 10:49:50 +0000 UTC    <nil>                     <nil>                <nil>                2025-05-29 10:49:50 +0000 UTC    00000000-0000-4000-8000-000000000000    &{https://docs.digitalocean.com/data_source DOMAIN false}
`

	knowledgeBaseListDataSourceResponse = `
{
	"knowledge_base_data_sources": [
		{
			"uuid": "00000000-0000-4000-8000-000000000000",
			"created_at": "2025-05-29T10:49:50Z",
			"updated_at": "2025-05-29T10:49:50Z",
			"web_crawler_data_source": {
				"base_url": "https://docs.digitalocean.com/data_source",
				"crawling_option": "DOMAIN",
				"embed_media":false
			}
		}
	]
}
`
)
