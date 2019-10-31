package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("database/create", func(t *testing.T, when spec.G, it spec.S) {
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
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				request := struct {
					Name    string `json:"name"`
					Engine  string `json:"engine"`
					Version string `json:"version"`
					Region  string `json:"region"`
					Nodes   int    `json:"num_nodes"`
				}{}

				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)

				t, err := template.New("response").Parse(databaseCreateResponse)
				expect.NoError(err)

				var b []byte
				buffer := bytes.NewBuffer(b)
				err = t.Execute(buffer, request)
				expect.NoError(err)

				w.Write(buffer.Bytes())
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("all flags are passed", func() {
		it("creates the databases", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"databases",
				"create",
				"my-database-name",
				"--engine", "mysql",
				"--num-nodes", "100",
				"--private-network-uuid", "some-uuid",
				"--region", "nyc3",
				"--size", "biggest",
				"--version", "what-version",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databasesCreateOutput), strings.TrimSpace(string(output)))

		})
	})

})

const (
	databasesCreateOutput = `
ID         Name                Engine    Version         Number of Nodes    Region    Status      Size       URI    Created At
some-id    my-database-name    mysql     what-version    100                nyc3      creating    biggest           2019-01-11 18:37:36 +0000 UTC
`
	databaseCreateResponse = `
{
  "database": {
    "id": "some-id",
    "name": "{{.Name}}",
    "engine": "{{.Engine}}",
    "version": "{{.Version}}",
    "connection": {},
    "private_connection": {},
    "users": null,
    "db_names": null,
    "num_nodes": {{.Nodes}},
    "region": "{{.Region}}",
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
