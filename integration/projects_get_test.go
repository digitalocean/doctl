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

var _ = suite("projects/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/projects/test-project-1":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(projectsGetResponse))
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
		it("gets the specified project", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"projects",
				"get",
				"test-project-1",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(projectsGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing the format flag", func() {
		it("changes the output", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"projects",
				"get",
				"test-project-1",
				"--format", "Description",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(projectsGetFormattedOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	projectsGetOutput = `
ID                 Owner UUID         Owner ID    Name          Description       Purpose           Environment    Is Default?    Created At              Updated At
some-project-id    some-owner-uuid    2           my-web-api    My website API    Service or API    Production     false          2018-09-27T20:10:35Z    2018-09-27T20:10:35Z
`
	projectsGetFormattedOutput = `
Description
My website API
`
	projectsGetResponse = `
{
  "project": {
    "id": "some-project-id",
    "owner_uuid": "some-owner-uuid",
    "owner_id": 2,
    "name": "my-web-api",
    "description": "My website API",
    "purpose": "Service or API",
    "environment": "Production",
    "is_default": false,
    "created_at": "2018-09-27T20:10:35Z",
    "updated_at": "2018-09-27T20:10:35Z"
  }
}
`
)
