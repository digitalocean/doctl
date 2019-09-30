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

func testDropletNeighbors(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/droplets/1111/neighbors":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				w.Write([]byte(dropletNeighborsResponse))
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
		it("lists droplet kernels", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"neighbors",
				"1111",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletNeighborsOutput), strings.TrimSpace(string(output)))
		})
	})

	when("asking for particular headers", func() {
		it("only lists thoses headers", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"neighbors",
				"1111",
				"--format", "ID,Memory,VCPUs,Disk,Region",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletNeighborsHeadersOutput), strings.TrimSpace(string(output)))
		})
	})
}

const dropletNeighborsOutput = `
ID      Name    Public IPv4    Private IPv4    Public IPv6    Memory    VCPUs    Disk    Region       Image                          Status    Tags    Features    Volumes
2222                                                          0         0        0       some-slug    some-distro some-image-name    active    yes     remotes     some-volume-id
1440                                                          0         0        0       some-slug    some-distro some-image-name    active    yes     remotes     some-volume-id
`

const dropletNeighborsHeadersOutput = `
ID      Memory    VCPUs    Disk    Region
2222    0         0        0       some-slug
1440    0         0        0       some-slug
`

const dropletNeighborsResponse = `{
  "droplets": [{
    "id": 2222,
    "image": {
      "distribution": "some-distro",
      "name": "some-image-name"
    },
    "region": {
      "slug": "some-slug"
    },
    "status": "active",
    "tags": ["yes"],
    "features": ["remotes"],
    "volume_ids": ["some-volume-id"]
  },{
    "id": 1440,
    "image": {
      "distribution": "some-distro",
      "name": "some-image-name"
    },
    "region": {
      "slug": "some-slug"
    },
    "status": "active",
    "tags": ["yes"],
    "features": ["remotes"],
    "volume_ids": ["some-volume-id"]
  }]
}`
