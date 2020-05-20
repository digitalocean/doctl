package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/droplet/neighbors", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect     *require.Assertions
		server     *httptest.Server
		configPath string
	)

	it.Before(func() {
		expect = require.New(t)

		dir, err := ioutil.TempDir("", "doct-integration-tests")
		expect.NoError(err)

		configPath = filepath.Join(dir, "config.yaml")

		err = ioutil.WriteFile(configPath, []byte(dropletNeighborsConfig), 0644)
		expect.NoError(err)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/droplets/1111/neighbors":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-extra-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
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

	it.After(func() {
		err := os.RemoveAll(configPath)
		expect.NoError(err)
	})

	when("all required flags are passed", func() {
		it("lists droplet kernels", func() {
			cmd := exec.Command(builtBinaryPath,
				"compute",
				"droplet",
				"neighbors",
				"1111",
			)

			cmd.Env = append(os.Environ(),
				"DIGITALOCEAN_ACCESS_TOKEN=some-extra-token",
				fmt.Sprintf("DIGITALOCEAN_API_URL=%s", server.URL),
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletNeighborsOutput), strings.TrimSpace(string(output)))
		})
	})

	when("asking for particular headers", func() {
		it("only lists thoses headers", func() {
			cmd := exec.Command(builtBinaryPath,
				"compute",
				"droplet",
				"neighbors",
				"1111",
				"--format", "ID,Memory,VCPUs,Disk,Region",
			)

			cmd.Env = append(os.Environ(),
				fmt.Sprintf("DIGITALOCEAN_API_URL=%s", server.URL),
				fmt.Sprintf("DIGITALOCEAN_CONFIG=%s", configPath),
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletNeighborsHeadersOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dropletNeighborsConfig = `
---
access-token: some-extra-token
`
	dropletNeighborsOutput = `
ID      Name    Public IPv4    Private IPv4    Public IPv6    Memory    VCPUs    Disk    Region       Image                          VPC UUID    Status    Tags    Features    Volumes
2222                                                          0         0        0       some-slug    some-distro some-image-name                active    yes     remotes     some-volume-id
1440                                                          0         0        0       some-slug    some-distro some-image-name                active    yes     remotes     some-volume-id
`

	dropletNeighborsHeadersOutput = `
ID      Memory    VCPUs    Disk    Region
2222    0         0        0       some-slug
1440    0         0        0       some-slug
`
	dropletNeighborsResponse = `
{
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
)
