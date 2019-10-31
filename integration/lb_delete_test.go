package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/load-balancer/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect   *require.Assertions
		server   *httptest.Server
		cmd      *exec.Cmd
		baseArgs []string
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/load_balancers/that-lb-id":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

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

		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"compute",
			"load-balancer",
		)

		baseArgs = []string{
			"that-lb-id",
			"--force",
		}
	})

	when("required flags are passed", func() {
		cases := []struct {
			desc    string
			command string
		}{
			{desc: "command is delete", command: "delete"},
			{desc: "command is rm", command: "rm"},
			{desc: "command is d", command: "d"},
		}

		for _, c := range cases {
			when(c.desc, func() {
				command := c.command

				it("deletes the specified load balancer", func() {
					args := append([]string{command}, baseArgs...)
					cmd.Args = append(cmd.Args, args...)

					output, err := cmd.CombinedOutput()
					expect.NoError(err, fmt.Sprintf("received error output: %s", output))
					expect.Empty(output)
				})
			})
		}
	})
})
