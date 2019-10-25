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

var _ = suite("compute/snapshot/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/snapshots/53344211":
				if req.Method != "DELETE" {
					w.WriteHeader(http.StatusTeapot)
					return
				}

				if req.Method != "DELETE" {
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
		base := []string{
			"-t", "some-magic-token",
			"compute",
			"snapshot",
		}

		cases := []struct {
			desc string
			args []string
		}{
			{desc: "command is delete", args: append(base, []string{"delete", "53344211", "--force"}...)},
			{desc: "command is d", args: append(base, []string{"d", "53344211", "--force"}...)},
		}

		for _, c := range cases {
			commandArgs := c.args

			when(c.desc, func() {
				it("deletes a snapshot", func() {
					finalArgs := append([]string{"-u", server.URL}, commandArgs...)
					cmd := exec.Command(builtBinaryPath, finalArgs...)

					output, err := cmd.CombinedOutput()
					expect.NoError(err, fmt.Sprintf("received error output: %s", output))
					expect.Empty(output)
				})
			})
		}
	})
})
