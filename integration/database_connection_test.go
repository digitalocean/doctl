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

var _ = suite("database/connection", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	const testUUID = "aaa-bbb-111-222-ccc-333"

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/" + testUUID:
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(databaseGetResponseWithConnection))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("private flag not passed", func() {
		it("public connection details are returned", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"connection",
				testUUID,
				"--no-header",
				"--format", "Host",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal("db-postgresql-nyc3-test.db.ondigitalocean.com", strings.TrimSpace(string(output)))

		})
	})

	when("private flag is passed", func() {
		it("private connection details are returned", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"connection",
				testUUID,
				"--private",
				"--no-header",
				"--format", "Host",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal("private-db-postgresql-nyc3-test.db.ondigitalocean.com", strings.TrimSpace(string(output)))

		})
	})

})

const (
	databaseGetResponseWithConnection = `
{
  "database": {
    "id": "aaa-bbb-111-222-ccc-333",
    "name": "test",
    "engine": "pg",
    "version": "13",
    "connection": {
        "protocol": "postgresql",
        "uri": "postgresql://doadmin:secret@db-postgresql-nyc3-test.db.ondigitalocean.com:25060/defaultdb?sslmode=require",
        "database": "defaultdb",
        "host": "db-postgresql-nyc3-test.db.ondigitalocean.com",
        "port": 25060,
        "user": "doadmin",
        "password": "secret",
        "ssl": true
    },
    "private_connection": {
        "protocol": "postgresql",
        "uri": "postgresql://doadmin:secret@private-db-postgresql-nyc3-test.db.ondigitalocean.com:25060/defaultdb?sslmode=require",
        "database": "defaultdb",
        "host": "private-db-postgresql-nyc3-test.db.ondigitalocean.com",
        "port": 25060,
        "user": "doadmin",
        "password": "secret",
        "ssl": true
    },
    "users": null,
    "db_names": null,
    "num_nodes": 1,
    "region": "nyc3",
    "status": "creating",
    "created_at": "2019-01-11T18:37:36Z",
    "maintenance_window": null,
    "size": "biggest",
    "tags": [
      "production"
    ]
  }
}`
)
