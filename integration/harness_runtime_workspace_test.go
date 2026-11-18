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

var _ = suite("harness-runtime/workspace", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	const notEnrolledMsg = "workspaces are not enabled for this team: persistent workspaces are in limited availability. Contact DigitalOcean support to request access"

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch req.URL.Path {
			case "/v2/harness-runtime/workspaces":
				w.WriteHeader(http.StatusForbidden)
				_, _ = fmt.Fprintf(w, `{"id":"err","message":%q}`, notEnrolledMsg)
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}
				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it("explains limited availability when workspaces are not enabled", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"harness-runtime", "workspace", "list",
		)

		output, err := cmd.CombinedOutput()
		expect.Error(err, "list should fail when the team is not enrolled")

		out := string(output)
		expect.Contains(out, "Limited availability")
		expect.Contains(out, notEnrolledMsg)
		expect.Contains(out, "403")
		expect.Contains(out, strings.TrimSpace("Request access at https://cloudsupport.digitalocean.com"))
	})
})