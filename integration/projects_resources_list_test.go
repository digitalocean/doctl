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

var _ = suite("projects/resources/list", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(projectsResourcesListResponse))
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
		it("list resources for the project", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"projects",
				"resources",
				"list",
				"some-project-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(projectsResourcesListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("no header is passed", func() {
		it("returns outputs data with no headers", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"projects",
				"resources",
				"list",
				"some-project-id",
				"--no-header",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(projectsResourcesListNoHeaderOutput), strings.TrimSpace(string(output)))
		})
	})

	when("format is passed", func() {
		it("gives you the columns requested", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"projects",
				"resources",
				"list",
				"some-project-id",
				"--format", "URN,Status",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(projectsResourcesListFormatOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	projectsResourcesListOutput = `
URN                Assigned At             Status
do:droplet:1       2018-09-28T19:26:37Z    ok
do:floatingip:1    2018-09-28T19:26:38Z    ok
`
	projectsResourcesListNoHeaderOutput = `
do:droplet:1       2018-09-28T19:26:37Z    ok
do:floatingip:1    2018-09-28T19:26:38Z    ok
`
	projectsResourcesListFormatOutput = `
URN                Status
do:droplet:1       ok
do:floatingip:1    ok
`
	projectsResourcesListResponse = `
{
  "resources": [
    {
      "urn": "do:droplet:1",
      "assigned_at": "2018-09-28T19:26:37Z",
      "links": {
        "self": "https://api.digitalocean.com/v2/droplets/1"
      },
      "status": "ok"
    },
    {
      "urn": "do:floatingip:1",
      "assigned_at": "2018-09-28T19:26:38Z",
      "links": {
        "self": "https://api.digitalocean.com/v2/floating_ips/1"
      },
      "status": "ok"
    }
  ],
  "links": {},
  "meta": {
    "total": 1
  }
}
`
)
