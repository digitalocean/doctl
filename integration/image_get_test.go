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

var _ = suite("compute/image/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
	)

	it.Before(func() {
		expect = require.New(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/images/ubuntu-16-04-x64":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(imageGetResponse))
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
			"image",
		)

	})

	when("when image slug is provided", func() {
		it("gets the specified image", func() {
			baseArgs := []string{"ubuntu-16-04-x64"}
			args := append([]string{"get"}, baseArgs...)
			cmd.Args = append(cmd.Args, args...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(imageGetOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	imageGetOutput = `
ID         Name         Type    Distribution    Slug                Public    Min Disk
6918990    14.04 x64            Ubuntu          ubuntu-16-04-x64    true      20`
	imageGetResponse = `{
  "image": {
    "id": 6918990,
    "name": "14.04 x64",
    "distribution": "Ubuntu",
    "slug": "ubuntu-16-04-x64",
    "public": true,
    "regions": [ "ams3", "nyc3" ],
    "created_at": "2014-10-17T20:24:33Z",
    "min_disk_size": 20,
    "size_gigabytes": 2.34,
    "description": "",
    "tags": [],
    "status": "available",
    "error_message": ""
  }
}`
)
