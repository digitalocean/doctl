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

var _ = suite("projects/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/projects":
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
					Name        string `json:"name"`
					Env         string `json:"environment"`
					Description string `json:"description"`
					Purpose     string `json:"purpose"`
				}{}

				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)

				t, err := template.New("response").Parse(projectsCreateResponse)
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
		it("creates a project", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"projects",
				"create",
				"--name", "some-project",
				"--purpose", "to-organize",
				"--description", "just magic",
				"--environment", "Staging",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(projectsCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("missing required arguments", func() {
		base := []string{
			"-t", "some-magic-token",
			"-u", "https://www.example.com",
			"projects",
			"create",
		}

		baseErr := `Error: (projects.create%s) command is missing required arguments`

		cases := []struct {
			desc string
			err  string
			args []string
		}{
			{desc: "missing all", err: fmt.Sprintf(baseErr, ".name"), args: base},
			{desc: "missing only name", err: fmt.Sprintf(baseErr, ".name"), args: append(base, []string{"--purpose", "not missing"}...)},
			{desc: "missing only purpose", err: fmt.Sprintf(baseErr, ".purpose"), args: append(base, []string{"--name", "where are you purpose"}...)},
		}

		for _, c := range cases {
			commandArgs := c.args
			expectedErr := c.err

			when(c.desc, func() {
				it("returns an error", func() {
					cmd := exec.Command(builtBinaryPath, commandArgs...)

					output, err := cmd.CombinedOutput()
					expect.Error(err)
					expect.Contains(string(output), expectedErr)
				})
			})
		}
	})
})

const (
	projectsCreateOutput = `
ID         Owner UUID         Owner ID    Name            Description    Purpose        Environment    Is Default?    Created At              Updated At
some-id    some-owner-uuid    2           some-project    just magic     to-organize    Staging        false          2018-09-27T15:52:48Z    2018-09-27T15:52:48Z
`
	projectsCreateResponse = `
{
  "project": {
    "id": "some-id",
    "owner_uuid": "some-owner-uuid",
    "owner_id": 2,
    "name": "{{.Name}}",
    "description": "{{.Description}}",
    "purpose": "{{.Purpose}}",
    "environment": "{{.Env}}",
    "is_default": false,
    "created_at": "2018-09-27T15:52:48Z",
    "updated_at": "2018-09-27T15:52:48Z"
  }
}
`
)
