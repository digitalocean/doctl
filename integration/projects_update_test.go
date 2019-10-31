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

var _ = suite("projects/update", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/projects/some-project-to-update":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				request := struct {
					Name        string `json:"name"`
					Env         string `json:"environment"`
					Description string `json:"description"`
					Purpose     string `json:"purpose"`
					IsDefault   bool   `json:"is_default"`
				}{}

				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)

				t, err := template.New("response").Parse(projectsUpdateResponse)
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

	when("all required flags are passed", func() {
		it("updates the project", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"projects",
				"update",
				"some-project-to-update",
				"--description", "yes",
				"--name", "some-name",
				"--purpose", "some-purpose",
				"--environment", "mars",
				"--is_default",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(projectsUpdateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing the format flag", func() {
		it("changes the output", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"projects",
				"update",
				"some-project-to-update",
				"--description", "yes",
				"--name", "some-name",
				"--purpose", "some-purpose",
				"--environment", "mars",
				"--is_default",
				"--format", "ID,Name",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(projectsUpdateFormattedOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	projectsUpdateOutput = `
ID                        Owner UUID    Owner ID    Name         Description    Purpose         Environment    Is Default?    Created At              Updated At
some-project-to-update    owner-uuid    2           some-name    yes            some-purpose    mars           true           2018-09-27T15:52:48Z    2018-09-27T15:52:48Z
`
	projectsUpdateFormattedOutput = `
ID                        Name
some-project-to-update    some-name
`
	projectsUpdateResponse = `
{
  "project": {
    "id": "some-project-to-update",
    "owner_uuid": "owner-uuid",
    "owner_id": 2,
    "name": "{{.Name}}",
    "description": "{{.Description}}",
    "purpose": "{{.Purpose}}",
    "environment": "{{.Env}}",
    "is_default": {{.IsDefault}},
    "created_at": "2018-09-27T15:52:48Z",
    "updated_at": "2018-09-27T15:52:48Z"
  }
}
`
)
