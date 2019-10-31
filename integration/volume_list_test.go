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

var _ = suite("compute/volume/list", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(volumeListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

	})

	when("required flags are passed", func() {
		it("lists all volumes", func() {
			aliases := []string{"ls", "list"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"volume",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(volumeListOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	volumeListOutput = `
ID                  Name       Size      Region    Filesystem Type    Filesystem Label    Droplet IDs    Tags
some-volume-id-1    example    10 GiB    nyc1                                             [1]            aninterestingtag
some-volume-id-2    example    40 GiB    nyc1                                             [2]            adifferenttag
`
	volumeListResponse = `
{
  "volumes": [
    {
      "id": "some-volume-id-1",
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
    },
    {
      "id": "some-volume-id-2",
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
      "droplet_ids": [2],
      "name": "example",
      "description": "Block store for examples",
      "size_gigabytes": 40,
      "created_at": "2016-03-02T17:10:49Z",
      "tags": ["adifferenttag"]
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}
`
)
