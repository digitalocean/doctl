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

var _ = suite("1-click/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
		cmd    *exec.Cmd
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/1-clicks":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(oneClickResp))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"1-click",
		)

	})

	when("command is list", func() {
		it("lists all 1-Click applications", func() {
			cmd.Args = append(cmd.Args, []string{"list"}...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(oneClickListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is ls", func() {
		it("lists all 1-Click Applications", func() {
			cmd.Args = append(cmd.Args, []string{"ls"}...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(oneClickListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	oneClickListOutput = `
SLUG               TYPE
discourse-18-04    droplet
django-18-04       droplet
`
	oneClickResp = `
{
  "1_clicks": [
    {
      "slug": "discourse-18-04",
      "type": "droplet"
    },
    {
      "slug": "django-18-04",
      "type": "droplet"
    }
  ]
}
`
)
