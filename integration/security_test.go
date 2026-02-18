package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("security/cspm", func(t *testing.T, when spec.G, it spec.S) {
	const scanID = "497dcba3-ecbf-4587-a2dd-5eb0665e6880"
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/security/scans":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := io.ReadAll(req.Body)
				expect.NoError(err)
				expect.JSONEq(securityScanCreateRequest, string(reqBody))

				w.Write([]byte(securityScanCreateResponse))
			case "/v2/security/scans/" + scanID:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(securityScanGetResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("the wait flag is passed", func() {
		it("creates a scan and waits for completion", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"security",
				"scans",
				"create",
				"--wait",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(securityScanWaitCreateOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	securityScanCreateRequest  = `{}`
	securityScanCreateResponse = `{
  "id": "497dcba3-ecbf-4587-a2dd-5eb0665e6880",
  "status": "in_progress",
  "created_at": "2025-12-04T00:00:00Z",
  "findings": []
}`
	securityScanGetResponse = `{
  "scan": {
    "id": "497dcba3-ecbf-4587-a2dd-5eb0665e6880",
    "status": "complete",
    "created_at": "2025-12-04T00:00:00Z",
    "findings": [
      {
        "rule_uuid": "rule-1",
        "name": "test",
        "found_at": "2025-12-04T00:00:00Z",
        "severity": "critical",
        "affected_resources_count": 2
      }
    ]
  }
}`
	securityScanWaitCreateOutput = `
Notice: Scan in progress, waiting for scan to complete
Notice: Scan completed
Rule ID    Name    Affected Resources    Found At                Severity
rule-1     test    2                     2025-12-04T00:00:00Z    critical
`
)
