package integration

import (
	"encoding/json"
	"fmt"
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

var _ = suite("projects/resources/assign", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/projects/some-project-id/resources":
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
					Resources []string `json:"resources"`
				}{}

				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)

				expect.ElementsMatch(request.Resources, []string{"some-urn-1", "some-urn-2"})

				w.Write([]byte(projectsResourcesAssignResponse))
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
		it("assigns resources to the project", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"projects",
				"resources",
				"assign",
				"some-project-id",
				"--resource", "some-urn-1",
				"--resource", "some-urn-2",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(projectsResourcesAssignOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	projectsResourcesAssignOutput = `
URN           Assigned At             Status
some-urn-1    2018-09-28T19:26:37Z    assigned
some-urn-2    2018-09-28T19:26:37Z    assigned
`
	projectsResourcesAssignResponse = `
{
  "resources": [
    {
      "urn": "some-urn-1",
      "assigned_at": "2018-09-28T19:26:37Z",
      "status": "assigned"
    },
    {
      "urn": "some-urn-2",
      "assigned_at": "2018-09-28T19:26:37Z",
      "status": "assigned"
    }
  ]
}
`
)
