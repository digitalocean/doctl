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

var _ = suite("compute/droplet/list", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				q := req.URL.Query()
				tag := q.Get("tag_name")
				if tag == "some-tag" {
					w.Write([]byte(`{}`))
					return
				}

				if tag == "regions" {
					w.Write([]byte(dropletListRegionResponse))
					return
				}

				w.Write([]byte(dropletListResponse))
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
		it("lists droplets", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("a region is provided", func() {
		it("filters the returned droplets by region", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"list",
				"--tag-name", "regions",
				"--region", "my-region",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletListRegionOutput), strings.TrimSpace(string(output)))
		})
	})

	when("there are no droplets", func() {
		it("lists only headers", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"list",
				"--tag-name", "some-tag",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletListEmptyOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dropletListResponse = `
{
  "droplets": [{
    "id": 1111,
    "name": "some-droplet-name",
    "image": {
      "distribution": "some-distro",
      "name": "some-image-name"
    },
    "region": {
      "slug": "some-region-slug"
    },
    "status": "active",
    "tags": ["yes", "test"],
    "features": ["remotes"],
    "volume_ids": ["some-volume-id"]
  }]
}`

	dropletListRegionResponse = `
{
  "droplets": [{
    "id": 1111,
    "image": {
      "distribution": "some-distro",
      "name": "some-image-name"
    },
    "region": {
      "slug": "not-regions"
    },
    "status": "active",
    "tags": ["yes", "test"],
    "features": ["remotes"],
    "volume_ids": ["some-volume-id"]
  },{
    "id": 1440,
    "image": {
      "distribution": "some-distro",
      "name": "some-image-name"
    },
    "region": {
      "slug": "my-region"
    },
    "status": "active",
    "tags": ["yes", "test"],
    "features": ["remotes"],
    "volume_ids": ["some-volume-id"]
  }]
}`

	dropletListOutput = `
ID      Name                 Public IPv4    Private IPv4    Public IPv6    Memory    VCPUs    Disk    Region              Image                          VPC UUID    Status    Tags        Features    Volumes
1111    some-droplet-name                                                  0         0        0       some-region-slug    some-distro some-image-name                active    test,yes    remotes     some-volume-id
`

	dropletListRegionOutput = `
ID      Name    Public IPv4    Private IPv4    Public IPv6    Memory    VCPUs    Disk    Region       Image                          VPC UUID    Status    Tags        Features    Volumes
1440                                                          0         0        0       my-region    some-distro some-image-name                active    test,yes    remotes     some-volume-id
`

	dropletListEmptyOutput = `
ID    Name    Public IPv4    Private IPv4    Public IPv6    Memory    VCPUs    Disk    Region    Image    VPC UUID    Status    Tags    Features    Volumes
`
)
