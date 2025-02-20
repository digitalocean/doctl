package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
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

var _ = suite("database/pool/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	type poolReq struct {
		Name string `json:"name"`
		Mode string `json:"mode"`
		Size int    `json:"size"`
		DB   string `json:"db"`
		User string `json:"user"`
	}

	render := func(pr *poolReq) string {
		t, err := template.New("response").Parse(databaseConnectionPoolTpl)
		expect.NoError(err)

		buf := &bytes.Buffer{}
		err = t.Execute(buf, pr)
		expect.NoError(err)
		return buf.String()
	}

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/some-database-id/pools":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
				}

				switch req.Method {
				case http.MethodGet:
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(databaseConnectionPoolList))

				case http.MethodPost:
					reqBody, err := io.ReadAll(req.Body)
					expect.NoError(err)

					request := &poolReq{}

					err = json.Unmarshal(reqBody, request)
					expect.NoError(err)

					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(render(request)))

				default:
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

			case "/v2/databases/some-database-id/pools/reporting-pool":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
				}

				switch req.Method {
				case http.MethodGet:
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(databaseConnectionPoolGet))

				case http.MethodPut:
					reqBody, err := io.ReadAll(req.Body)
					expect.NoError(err)

					request := &poolReq{}

					err = json.Unmarshal(reqBody, request)
					expect.NoError(err)

					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(render(request)))

				case http.MethodDelete:

				default:
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is list", func() {
		it("lists the database pools", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"pool",
				"list",
				"some-database-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseConnectionPoolListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is get", func() {
		it("gets a database pool", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"pool",
				"get",
				"some-database-id",
				"reporting-pool",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseConnectionPoolGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is create", func() {
		it("creates a database pool", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"pool",
				"create",
				"some-database-id",
				"reporting-pool",
				"--user", "doadmin",
				"--size", "10",
				"--db", "defaultdb",
				"--mode", "session",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseConnectionPoolGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is update", func() {
		it("updates a database pool", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"pool",
				"update",
				"some-database-id",
				"reporting-pool",
				"--size", "20",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		})
	})

	when("command is delete", func() {
		it("deletes a database pool", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"pool",
				"delete",
				"some-database-id",
				"reporting-pool",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		})
	})
})

const (
	databaseConnectionPoolListOutput = `
User       Name              Size    Database     Mode           URI
doadmin    reporting-pool    10      defaultdb    session        postgres://doadmin:wv78n3zpz42xezdk@backend-do-user-19081923-0.db.ondigitalocean.com:25061/foo?sslmode=require
doadmin    backend-pool      10      defaultdb    transaction    postgres://doadmin:wv78n3zpz42xezdk@backend-do-user-19081923-0.db.ondigitalocean.com:25061/backend-pool?sslmode=require
`

	databaseConnectionPoolGetOutput = `
User       Name              Size    Database     Mode       URI
doadmin    reporting-pool    10      defaultdb    session    postgres://doadmin:wv78n3zpz42xezdk@backend-do-user-19081923-0.db.ondigitalocean.com:25061/foo?sslmode=require
`

	databaseConnectionPoolList = `{
  "pools":[
    {
      "user":"doadmin",
      "name":"reporting-pool",
      "size":10,
      "db":"defaultdb",
      "mode":"session",
      "connection":{
        "uri":"postgres://doadmin:wv78n3zpz42xezdk@backend-do-user-19081923-0.db.ondigitalocean.com:25061/foo?sslmode=require",
        "database":"foo",
        "host":"backend-do-user-19081923-0.db.ondigitalocean.com",
        "port":25061,
        "user":"doadmin",
        "password":"wv78n3zpz42xezdk",
        "ssl":true
      }
    },
    {
      "user":"doadmin",
      "name":"backend-pool",
      "size":10,
      "db":"defaultdb",
      "mode":"transaction",
      "connection":{
        "uri":"postgres://doadmin:wv78n3zpz42xezdk@backend-do-user-19081923-0.db.ondigitalocean.com:25061/backend-pool?sslmode=require",
        "database":"backend-pool",
        "host":"backend-do-user-19081923-0.db.ondigitalocean.com",
        "port":25061,
        "user":"doadmin",
        "password":"wv78n3zpz42xezdk",
        "ssl":true
      }
    }
  ]
}
`

	databaseConnectionPoolGet = `{
  "pool":{
    "user":"doadmin",
    "name":"reporting-pool",
    "size":10,
    "db":"defaultdb",
    "mode":"session",
    "connection":{
      "uri":"postgres://doadmin:wv78n3zpz42xezdk@backend-do-user-19081923-0.db.ondigitalocean.com:25061/foo?sslmode=require",
      "database":"foo",
      "host":"backend-do-user-19081923-0.db.ondigitalocean.com",
      "port":25061,
      "user":"doadmin",
      "password":"wv78n3zpz42xezdk",
      "ssl":true
    }
  }
}
`

	databaseConnectionPoolTpl = `{
  "pool":{
    "user": "{{ .User }}",
    "name": "{{ .Name }}",
    "size": {{ .Size }},
    "db": "{{ .DB }}",
    "mode": "{{ .Mode }}",
    "connection":{
      "uri":"postgres://doadmin:wv78n3zpz42xezdk@backend-do-user-19081923-0.db.ondigitalocean.com:25061/foo?sslmode=require",
      "database":"foo",
      "host":"backend-do-user-19081923-0.db.ondigitalocean.com",
      "port":25061,
      "user":"doadmin",
      "password":"wv78n3zpz42xezdk",
      "ssl":true
    }
  }
}`
)
