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

var _ = suite("genai/knowledgebase/list-indexing-jobs", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/indexing_jobs":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(indexingJobsListResponse))
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
		it("lists all indexing jobs", func() {
			aliases := []string{"list-indexing-jobs", "ls-jobs"}

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
				expect.Equal(strings.TrimSpace(indexingJobsListOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const indexingJobsListResponse = `{
  "jobs": [
    {
      "completed_datasources": 1,
      "created_at": "2023-01-01T00:00:00Z",
      "data_source_uuids": [
        "data-source-uuid-1",
        "data-source-uuid-2"
      ],
      "finished_at": "2023-01-01T01:00:00Z",
      "knowledge_base_uuid": "kb-uuid-123",
      "phase": "BATCH_JOB_PHASE_SUCCEEDED",
      "started_at": "2023-01-01T00:30:00Z",
      "status": "INDEX_JOB_STATUS_COMPLETED",
      "tokens": 1000,
      "total_datasources": 2,
      "total_items_failed": "0",
      "total_items_indexed": "100",
      "total_items_skipped": "5",
      "updated_at": "2023-01-01T01:00:00Z",
      "uuid": "indexing-job-uuid-123"
    },
    {
      "completed_datasources": 0,
      "created_at": "2023-01-01T02:00:00Z",
      "data_source_uuids": [
        "data-source-uuid-3"
      ],
      "finished_at": null,
      "knowledge_base_uuid": "kb-uuid-456",
      "phase": "BATCH_JOB_PHASE_RUNNING",
      "started_at": "2023-01-01T02:30:00Z",
      "status": "INDEX_JOB_STATUS_RUNNING",
      "tokens": 0,
      "total_datasources": 1,
      "total_items_failed": "0",
      "total_items_indexed": "0",
      "total_items_skipped": "0",
      "updated_at": "2023-01-01T02:30:00Z",
      "uuid": "indexing-job-uuid-456"
    }
  ],
  "links": {
    "pages": {
      "first": "",
      "last": "",
      "next": "",
      "previous": ""
    }
  },
  "meta": {
    "page": 1,
    "pages": 1,
    "total": 2
  }
}`

const indexingJobsListOutput = `UUID                     Knowledge Base UUID    Phase                        Status                        Completed Datasources    Total Datasources    Tokens    Total Items Indexed    Total Items Failed    Total Items Skipped    Created At                       Started At                       Finished At                      Updated At
indexing-job-uuid-123    kb-uuid-123            BATCH_JOB_PHASE_SUCCEEDED    INDEX_JOB_STATUS_COMPLETED    1                        2                    1000      100                    0                     5                      2023-01-01 00:00:00 +0000 UTC    2023-01-01 00:30:00 +0000 UTC    2023-01-01 01:00:00 +0000 UTC    2023-01-01 01:00:00 +0000 UTC
indexing-job-uuid-456    kb-uuid-456            BATCH_JOB_PHASE_RUNNING      INDEX_JOB_STATUS_RUNNING      0                        1                    0         0                      0                     0                      2023-01-01 02:00:00 +0000 UTC    2023-01-01 02:30:00 +0000 UTC    <nil>                            2023-01-01 02:30:00 +0000 UTC`
