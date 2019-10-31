package integration

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("projects/list", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				pageNum := req.URL.Query().Get("page")
				response := projectsListPageOneResponse
				if pageNum == "2" {
					response = projectsListPageTwoResponse
				}

				reqURL := struct {
					Host string
				}{
					Host: req.RemoteAddr,
				}

				t, err := template.New("response").Parse(response)
				expect.NoError(err)

				var b []byte
				buffer := bytes.NewBuffer(b)
				err = t.Execute(buffer, reqURL)
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
		it("lists the projects", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"projects",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(projectsListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	projectsListOutput = `
ID                 Owner UUID    Owner ID    Name     Description    Purpose    Environment    Is Default?    Created At              Updated At
some-project-id    some-uuid     2           magic    hello          none       mars           false          2018-09-27T20:10:35Z    2018-09-27T20:10:35Z
other-id           some-uuid     2           yes      no             other      venus          true           2018-09-27T20:10:35Z    2018-09-27T20:10:35Z
`
	projectsListPageOneResponse = `
{
  "projects": [
    {
      "id": "some-project-id",
      "owner_uuid": "some-uuid",
      "owner_id": 2,
      "name": "magic",
      "description": "hello",
      "purpose": "none",
      "environment": "mars",
      "is_default": false,
      "created_at": "2018-09-27T20:10:35Z",
      "updated_at": "2018-09-27T20:10:35Z"
    }
  ],
  "links": {
    "pages": {
      "first": "https://{{.Host}}/v2/projects?page=1",
      "next": "https://{{.Host}}/v2/projects?page=2",
      "last": "https://{{.Host}}/v2/projects?page=2"
    }
  },
  "meta": {
    "total": 1
  }
}
`
	projectsListPageTwoResponse = `
{
  "projects": [
    {
      "id": "other-id",
      "owner_uuid": "some-uuid",
      "owner_id": 2,
      "name": "yes",
      "description": "no",
      "purpose": "other",
      "environment": "venus",
      "is_default": true,
      "created_at": "2018-09-27T20:10:35Z",
      "updated_at": "2018-09-27T20:10:35Z"
    }
  ],
  "links": {
    "pages": {
      "first": "https://{{.Host}}/v2/projects?page=1",
      "last": "https://{{.Host}}/v2/projects?page=2"
    }
  },
  "meta": {
    "total": 1
  }
}
`
)
