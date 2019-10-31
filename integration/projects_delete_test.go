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

var _ = suite("projects/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/projects/test-project-1":
				fallthrough
			case "/v2/projects/test-project-2":
				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusNoContent)
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
		it("deletes the project", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"--trace",
				"projects",
				"delete",
				"test-project-1",
				"test-project-2",
				"-f",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))

			shared := []func(string){
				func(s string) { expect.Equal("doctl:", s) },
				func(s string) { expect.Regexp(`\d{4}/\d{2}/\d{2}`, s) },
				func(s string) { expect.Regexp(`\d{2}:\d{2}:\d{2}`, s) },
			}

			lines := strings.Split(string(output), "\n")

			line := lines[0]
			for i, content := range strings.SplitN(line, " ", 5) {
				matchers := append(shared,
					func(s string) { expect.Equal("->", s) },
					func(s string) { expect.Contains(s, "DELETE /v2/projects/test-project-1 HTTP/1.1") },
				)

				matchers[i](content)
			}

			line = lines[1]
			for i, content := range strings.SplitN(line, " ", 5) {
				matchers := append(shared,
					func(s string) { expect.Equal("<-", s) },
					func(s string) { expect.Contains(s, `HTTP/1.1 204 No Content\r\nDate:`) },
				)

				matchers[i](content)
			}

			line = lines[2]
			for i, content := range strings.SplitN(line, " ", 5) {
				matchers := append(shared,
					func(s string) { expect.Equal("->", s) },
					func(s string) { expect.Contains(s, "DELETE /v2/projects/test-project-2 HTTP/1.1") },
				)

				matchers[i](content)
			}
		})
	})
})
