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

var _ = suite("database/index/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/some-database-id/indexes":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(databaseOpenSearchIndexListResponse))
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
		it("lists indexes for an opensearch cluster", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"indexes",
				"list",
				"some-database-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseOpenSearchIndexListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	databaseOpenSearchIndexListOutput = `
Index Name         Status    Health    Size    Docs    Create At               Number of Shards    Number of Replica
sql-datasources    open      green     624     0       2024-05-24T07:44:45Z    1                   0`
	databaseOpenSearchIndexListResponse = `
{
  "indexes": [
    {
		"index_name": "sql-datasources",
		"status": "open",
		"health": "green",
		"size": 624,
		"docs": 0,
		"create_time": "2024-05-24T07:44:45Z",
		"number_of_shards": 1,
		"number_of_replica": 0
    }
  ]
}
`
)
