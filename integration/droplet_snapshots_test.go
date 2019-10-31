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

var _ = suite("compute/droplet/snapshots", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/droplets/1111/snapshots":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(dropletSnapshotsResponse))
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
		it("lists droplet snapshots", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"snapshots",
				"1111",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletSnapshotsOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dropletSnapshotsOutput = `
ID      Name           Type        Distribution    Slug      Public    Min Disk
4444    magic          snapshot    Fedora          slimey    false     25
2222    other-magic    snapshot    Ubuntu          slimey    false     25
`
	dropletSnapshotsResponse = `
{"snapshots": [
  {
    "id": 4444,
    "name": "magic",
    "distribution": "Fedora",
    "type": "snapshot",
    "slug": "slimey",
    "min_disk_size": 25
  },
  {
    "id": 2222,
    "name": "other-magic",
    "distribution": "Ubuntu",
    "type": "snapshot",
    "slug": "slimey",
    "min_disk_size": 25
  }
]}
`
)
