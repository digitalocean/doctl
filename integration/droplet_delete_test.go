package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/stretchr/testify/require"
)

func TestDropletDelete(t *testing.T) {
	spec.Run(t, "compute/droplet/delete", testDropletDelete, spec.Report(report.Terminal{}))
}

func testDropletDelete(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/droplets":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				w.Write([]byte(`{"droplets":[{"name":"some-droplet-name", "id": 1337}]}`))
			case "/v2/droplets/1337":
				if req.Method != "DELETE" {
					w.WriteHeader(http.StatusTeapot)
					return
				}

				w.Write([]byte(`{}`))
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
			"droplet",
		}

		cases := []struct {
			desc string
			args []string
		}{
			{desc: "command is delete", args: append(base, []string{"delete", "some-droplet-name", "--force"}...)},
			{desc: "command is rm", args: append(base, []string{"rm", "some-droplet-name", "--force"}...)},
			{desc: "command is d", args: append(base, []string{"d", "some-droplet-name", "--force"}...)},
			{desc: "command is del", args: append(base, []string{"del", "some-droplet-name", "--force"}...)},
		}

		for _, c := range cases {
			commandArgs := c.args

			when(c.desc, func() {
				it("deletes a droplet", func() {
					finalArgs := append([]string{"-u", server.URL}, commandArgs...)
					cmd := exec.Command(builtBinaryPath, finalArgs...)

					output, err := cmd.CombinedOutput()
					expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				})
			})
		}
	})
}
