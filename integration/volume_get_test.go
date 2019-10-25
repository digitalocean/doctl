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

var _ = suite("compute/volume/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect   *require.Assertions
		cmd      *exec.Cmd
		baseArgs = []string{"some-volume-id"}
	)

	it.Before(func() {
		expect = require.New(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/volumes/some-volume-id":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != "GET" {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(volumeGetResponse))
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
			"volume",
		)
	})

	when("command is get", func() {
		it("gets the specified volume", func() {
			args := append([]string{"get"}, baseArgs...)
			cmd.Args = append(cmd.Args, args...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(volumeGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is g", func() {
		it("gets the specified volume", func() {
			args := append([]string{"g"}, baseArgs...)
			cmd.Args = append(cmd.Args, args...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(volumeGetOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	volumeGetOutput = `
ID                                      Name       Size      Region    Filesystem Type    Filesystem Label    Droplet IDs    Tags
506f78a4-e098-11e5-ad9f-000f53306ae1    example    10 GiB    nyc1                                             [1]            aninterestingtag
`
	volumeGetResponse = `
{
  "volume": {
    "id": "506f78a4-e098-11e5-ad9f-000f53306ae1",
    "region": {
      "name": "New York 1",
      "slug": "nyc1",
      "sizes": ["s-1vcpu-1gb"],
      "features": [
        "private_networking",
        "backups",
        "ipv6",
        "metadata"
      ],
      "available": true
    },
    "droplet_ids": [1],
    "name": "example",
    "description": "Block store for examples",
    "size_gigabytes": 10,
    "created_at": "2016-03-02T17:00:49Z",
    "tags": ["aninterestingtag"]
  }
}`
)
