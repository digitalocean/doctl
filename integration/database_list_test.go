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

var _ = suite("database/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusForbidden)
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(databaseListResponse))
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
		it("lists all databases", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("all format flag is passed", func() {
		it("lists contains correct fields", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"list",
				"--format",
				"ID,Name,URI,Created",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseFormattedListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	databaseListOutput = `
ID                                      Name                  Engine    Version    Number of Nodes    Region    Status    Size              Storage (MiB)
1df3bb24-d2a5-11ef-ae5c-ff567b29ed0b    db-mysql-test         mysql     8          1                  nyc3      online    db-s-1vcpu-1gb    10240
6ee6ac26-d2a5-11ef-b1f3-d74f7709225a    db-postgresql-test    pg        16         1                  nyc3      online    db-s-1vcpu-1gb    10240
`
	databaseFormattedListOutput = `
ID                                      Name                  URI                                                                                                                    Created At
1df3bb24-d2a5-11ef-ae5c-ff567b29ed0b    db-mysql-test         mysql://doadmin:nope@db-mysql-test-do-user-123-0.b.db.ondigitalocean.com:25060/defaultdb?ssl-mode=REQUIRED             2024-12-12 15:41:34 +0000 UTC
6ee6ac26-d2a5-11ef-b1f3-d74f7709225a    db-postgresql-test    postgresql://doadmin:shhhh@db-postgresql-test-do-user-123-0.l.db.ondigitalocean.com:25060/defaultdb?sslmode=require    2024-12-17 22:11:32 +0000 UTC
`
	databaseListResponse = `
{
  "databases": [
    {
      "id": "1df3bb24-d2a5-11ef-ae5c-ff567b29ed0b",
      "name": "db-mysql-test",
      "engine": "mysql",
      "version": "8",
      "semantic_version": "8.0.30",
      "connection": {
        "protocol": "mysql",
        "uri": "mysql://doadmin:nope@db-mysql-test-do-user-123-0.b.db.ondigitalocean.com:25060/defaultdb?ssl-mode=REQUIRED",
        "database": "defaultdb",
        "host": "db-mysql-test-do-user-123-0.b.db.ondigitalocean.com",
        "port": 25060,
        "user": "doadmin",
        "password": "nope",
        "ssl": true
      },
      "private_connection": {
        "protocol": "mysql",
        "uri": "mysql://doadmin:nope@private-db-mysql-test-do-user-123-0.b.db.ondigitalocean.com:25060/defaultdb?ssl-mode=REQUIRED",
        "database": "defaultdb",
        "host": "private-db-mysql-test-do-user-123-0.b.db.ondigitalocean.com",
        "port": 25060,
        "user": "doadmin",
        "password": "nope",
        "ssl": true
      },
      "metrics_endpoints": [
        {
          "host": "db-mysql-test-do-user-123-0.b.db.ondigitalocean.com",
          "port": 9273
        }
      ],
      "users": [
        {
          "name": "doadmin",
          "role": "primary",
          "password": "nope"
        }
      ],
      "db_names": [
        "defaultdb"
      ],
      "num_nodes": 1,
      "region": "nyc3",
      "status": "online",
      "created_at": "2024-12-12T15:41:34Z",
      "maintenance_window": {
        "day": "monday",
        "hour": "10:23:13",
        "pending": true
      },
      "size": "db-s-1vcpu-1gb",
      "tags": null,
      "private_network_uuid": "5b41fa22-d2a5-11ef-b99d-23e1382dd662",
      "project_id": "6393e064-d2a5-11ef-b030-9bd9b52e0f58",
      "read_only": false,
      "storage_size_mib": 10240
    },
    {
      "id": "6ee6ac26-d2a5-11ef-b1f3-d74f7709225a",
      "name": "db-postgresql-test",
      "engine": "pg",
      "version": "16",
      "semantic_version": "16.6",
      "connection": {
        "protocol": "postgresql",
        "uri": "postgresql://doadmin:shhhh@db-postgresql-test-do-user-123-0.l.db.ondigitalocean.com:25060/defaultdb?sslmode=require",
        "database": "defaultdb",
        "host": "db-postgresql-test-do-user-123-0.l.db.ondigitalocean.com",
        "port": 25060,
        "user": "doadmin",
        "password": "shhhh",
        "ssl": true
      },
      "private_connection": {
        "protocol": "postgresql",
        "uri": "postgresql://doadmin:shhhh@private-db-postgresql-test-do-user-123-0.l.db.ondigitalocean.com:25060/defaultdb?sslmode=require",
        "database": "defaultdb",
        "host": "private-db-postgresql-test-do-user-123-0.l.db.ondigitalocean.com",
        "port": 25060,
        "user": "doadmin",
        "password": "shhhh",
        "ssl": true
      },
      "metrics_endpoints": [
        {
          "host": "db-postgresql-test-do-user-123-0.l.db.ondigitalocean.com",
          "port": 9273
        }
      ],
      "users": [
        {
          "name": "doadmin",
          "role": "primary",
          "password": "shhhh"
        }
      ],
      "db_names": [
        "defaultdb"
      ],
      "num_nodes": 1,
      "region": "nyc3",
      "status": "online",
      "created_at": "2024-12-17T22:11:32Z",
      "maintenance_window": {
        "day": "saturday",
        "hour": "14:00:00",
        "pending": false
      },
      "size": "db-s-1vcpu-1gb",
      "tags": null,
      "private_network_uuid": "5b41fa22-d2a5-11ef-b99d-23e1382dd662",
      "project_id": "6393e064-d2a5-11ef-b030-9bd9b52e0f58",
      "read_only": false,
      "version_end_of_life": "2028-11-09T00:00:00Z",
      "version_end_of_availability": "2028-05-09T00:00:00Z",
      "storage_size_mib": 10240
    }
  ]
}
`
)
