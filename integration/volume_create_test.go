package integration

import (
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

var _ = suite("compute/volume/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/volumes":
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

				expect.JSONEq(volumeCreateRequest, string(reqBody))

				w.Write([]byte(volumeCreateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

	})

	when("passing almost all the flags", func() {
		it("creates the volume", func() {
			aliases := []string{"create", "c"}
			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"volume",
					alias,
					"my-volume",
					"--fs-label", "some-fs-label",
					"--fs-type", "xfs",
					"--region", "mars",
					"--size", "4TiB",
					"--tag", "yes",
					"--tag", "again",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(volumeCreateOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	volumeCreateOutput = `
ID                   Name         Size        Region    Filesystem Type    Filesystem Label    Droplet IDs    Tags
some-generated-id    my-volume    4000 GiB    mars1     xfs                some-fs-label       [1 2]          yes,again
`
	volumeCreateResponse = `
{
  "volume": {
    "id": "some-generated-id",
    "region": {
      "name": "mars",
      "slug": "mars1",
      "sizes": [
        "s-1vcpu-1gb",
        "s-1vcpu-2gb"
      ],
      "features": [
        "private_networking",
        "backups",
        "ipv6",
        "metadata"
      ],
      "available": true
    },
    "droplet_ids": [1,2],
    "filesystem_type": "xfs",
    "filesystem_label": "some-fs-label",
    "name": "my-volume",
    "description": "Block store for examples",
    "size_gigabytes": 4000,
    "created_at": "2016-03-02T17:00:49Z",
    "tags": ["yes","again"]
  }
}
`
	volumeCreateRequest = `
{
  "region":"mars",
  "name": "my-volume",
  "description":"",
  "size_gigabytes":4096,
  "snapshot_id":"",
  "filesystem_type":"xfs",
  "filesystem_label":"some-fs-label",
  "tags":["yes","again"]
}`
)
