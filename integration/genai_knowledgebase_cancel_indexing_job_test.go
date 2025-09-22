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

var _ = suite("genai/knowledgebase/cancel-indexing-job", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/indexing_jobs/12345678-1234-1234-1234-123456789012/cancel":
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
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(cancelIndexingJobResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("all required flags are passed", func() {
		it("cancels the specified indexing job", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"cancel-indexing-job",
				"12345678-1234-1234-1234-123456789012",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(cancelIndexingJobOutput), strings.TrimSpace(string(output)))
		})
	})

	when("format and no-header flags are passed", func() {
		it("cancels the specified indexing job with custom format", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"cancel-indexing-job",
				"12345678-1234-1234-1234-123456789012",
				"--format", "UUID,Status",
				"--no-header",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal("12345678-1234-1234-1234-123456789012    INDEX_JOB_STATUS_CANCELLED", strings.TrimSpace(string(output)))
		})
	})
})

const (
	cancelIndexingJobResponse = `{
  "job": {
    "uuid": "12345678-1234-1234-1234-123456789012",
    "knowledge_base_uuid": "kb-12345678-1234-1234-1234-123456789012",
    "phase": "BATCH_JOB_PHASE_CANCELLED",
    "status": "INDEX_JOB_STATUS_CANCELLED",
    "completed_datasources": 1,
    "total_datasources": 2,
    "tokens": 750,
    "total_items_indexed": "75",
    "total_items_failed": "0",
    "total_items_skipped": "5",
    "created_at": "2025-09-12T10:00:00Z",
    "started_at": "2025-09-12T10:00:30Z",
    "finished_at": "2025-09-12T10:02:15Z",
    "updated_at": "2025-09-12T10:02:15Z",
    "data_source_uuids": ["ds-1", "ds-2"]
  }
}`

	cancelIndexingJobOutput = `UUID                                    Knowledge Base UUID                        Phase                        Status                        Completed Datasources    Total Datasources    Tokens    Total Items Indexed    Total Items Failed    Total Items Skipped    Created At                       Started At                       Finished At                      Updated At
12345678-1234-1234-1234-123456789012    kb-12345678-1234-1234-1234-123456789012    BATCH_JOB_PHASE_CANCELLED    INDEX_JOB_STATUS_CANCELLED    1                        2                    750       75                     0                     5                      2025-09-12 10:00:00 +0000 UTC    2025-09-12 10:00:30 +0000 UTC    2025-09-12 10:02:15 +0000 UTC    2025-09-12 10:02:15 +0000 UTC`
)
