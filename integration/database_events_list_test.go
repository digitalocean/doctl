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

var _ = suite("database/events", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/some-database-id/events":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(databaseListEventsResponse))
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
		it("lists users for the database", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"events",
				"list",
				"some-database-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseListEventsOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	databaseListEventsOutput = `
ID           Cluster Name       Type of Event     Create Time
pe8u2huh     customer-events    cluster_create    2020-10-29T15:57:38Z
pe8ufefuh    customer-events    cluster_update    2023-10-30T15:57:38Z
`
	databaseListEventsResponse = `
{
  "events": [
    {
		"id": "pe8u2huh",
		"cluster_name": "customer-events",
		"event_type": "cluster_create",
		"create_time": "2020-10-29T15:57:38Z"
    },
    {
		"id": "pe8ufefuh",
		"cluster_name": "customer-events",
		"event_type": "cluster_update",
		"create_time": "2023-10-30T15:57:38Z"
    }
  ]
}
`
)
