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

var _ = suite("compute/snapshot/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/snapshots":
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
				resource := q.Get("resource_type")
				if resource == "droplet" {
					w.Write([]byte(snapshotListDropletResponse))
					return
				}

				if resource == "volume" {
					w.Write([]byte(snapshotListVolumeResponse))
					return
				}

				w.Write([]byte(snapshotListResponse))
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
		it("lists snapshots", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"snapshot",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(snapshotListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a format", func() {
		it("displays only those columns", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"snapshot",
				"list",
				"--format", "ID,ResourceType",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(snapshotListFormatOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing no-header", func() {
		it("displays only values, no headers", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"snapshot",
				"list",
				"--no-header",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(snapshotListNoHeaderOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing region", func() {
		it("displays only snapshots in the region", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"snapshot",
				"list",
				"--region",
				"nyc1",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(snapshotListRegionOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing droplet as resource type", func() {
		it("displays only droplet snapshots", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"snapshot",
				"list",
				"--resource",
				"droplet",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(snapshotListDropletOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing volume as resource type", func() {
		it("displays only volume snapshots", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"snapshot",
				"list",
				"--resource",
				"volume",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(snapshotListVolumeOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing region and resource type together", func() {
		it("displays only droplet snapshots in the region", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"snapshot",
				"list",
				"--resource",
				"droplet",
				"--region",
				"nyc1",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(snapshotListDropletRegionOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	snapshotListResponse = `
{
  "snapshots": [
    {
      "id": "0a343fac-eacf-11e9-b96b-0a58ac144633",
      "name": "volume-nyc1-01-1570651053836",
      "regions": [
        "nyc1"
      ],
      "created_at": "2019-10-09T19:57:36Z",
      "resource_id": "e2068b37-eace-11e9-85ad-0a58ac14430f",
      "resource_type": "volume",
      "min_disk_size": 100,
      "size_gigabytes": 0,
      "tags": []
    },
    {
      "id": "0e0adfa4-eacf-11e9-9e75-0a58ac14c13b",
      "name": "volume-lon1-01-1570651061232",
      "regions": [
        "lon1"
      ],
      "created_at": "2019-10-09T19:57:42Z",
      "resource_id": "fcaf04e4-eace-11e9-a09f-0a58ac14c0f4",
      "resource_type": "volume",
      "min_disk_size": 100,
      "size_gigabytes": 0,
      "tags": []
    },
    {
      "id": "53344211",
      "name": "ubuntu-s-1vcpu-1gb-nyc1-01-1570651077842",
      "regions": [
        "nyc1"
      ],
      "created_at": "2019-10-09T19:57:59Z",
      "resource_id": "162347943",
      "resource_type": "droplet",
      "min_disk_size": 25,
      "size_gigabytes": 1.01,
      "tags": []
    },
    {
      "id": "53344231",
      "name": "ubuntu-s-1vcpu-1gb-lon1-01-1570651124450",
      "regions": [
        "lon1"
      ],
      "created_at": "2019-10-09T19:58:50Z",
      "resource_id": "162348013",
      "resource_type": "droplet",
      "min_disk_size": 25,
      "size_gigabytes": 1.01,
      "tags": []
    }
  ],
  "links": {},
  "meta": {
    "total": 4
  }
}
`
	snapshotListDropletResponse = `
{
  "snapshots": [
    {
      "id": "53344211",
      "name": "ubuntu-s-1vcpu-1gb-nyc1-01-1570651077842",
      "regions": [
        "nyc1"
      ],
      "created_at": "2019-10-09T19:57:59Z",
      "resource_id": "162347943",
      "resource_type": "droplet",
      "min_disk_size": 25,
      "size_gigabytes": 1.01,
      "tags": []
    },
    {
      "id": "53344231",
      "name": "ubuntu-s-1vcpu-1gb-lon1-01-1570651124450",
      "regions": [
        "lon1"
      ],
      "created_at": "2019-10-09T19:58:50Z",
      "resource_id": "162348013",
      "resource_type": "droplet",
      "min_disk_size": 25,
      "size_gigabytes": 1.01,
      "tags": []
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}
`
	snapshotListVolumeResponse = `
{
  "snapshots": [
    {
      "id": "0a343fac-eacf-11e9-b96b-0a58ac144633",
      "name": "volume-nyc1-01-1570651053836",
      "regions": [
        "nyc1"
      ],
      "created_at": "2019-10-09T19:57:36Z",
      "resource_id": "e2068b37-eace-11e9-85ad-0a58ac14430f",
      "resource_type": "volume",
      "min_disk_size": 100,
      "size_gigabytes": 0,
      "tags": []
    },
    {
      "id": "0e0adfa4-eacf-11e9-9e75-0a58ac14c13b",
      "name": "volume-lon1-01-1570651061232",
      "regions": [
        "lon1"
      ],
      "created_at": "2019-10-09T19:57:42Z",
      "resource_id": "fcaf04e4-eace-11e9-a09f-0a58ac14c0f4",
      "resource_type": "volume",
      "min_disk_size": 100,
      "size_gigabytes": 0,
      "tags": []
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}
`
	snapshotListOutput = `
ID                                      Name                                        Created at              Regions    Resource ID                             Resource Type    Min Disk Size    Size        Tags
0a343fac-eacf-11e9-b96b-0a58ac144633    volume-nyc1-01-1570651053836                2019-10-09T19:57:36Z    [nyc1]     e2068b37-eace-11e9-85ad-0a58ac14430f    volume           100              0.00 GiB    
0e0adfa4-eacf-11e9-9e75-0a58ac14c13b    volume-lon1-01-1570651061232                2019-10-09T19:57:42Z    [lon1]     fcaf04e4-eace-11e9-a09f-0a58ac14c0f4    volume           100              0.00 GiB    
53344211                                ubuntu-s-1vcpu-1gb-nyc1-01-1570651077842    2019-10-09T19:57:59Z    [nyc1]     162347943                               droplet          25               1.01 GiB    
53344231                                ubuntu-s-1vcpu-1gb-lon1-01-1570651124450    2019-10-09T19:58:50Z    [lon1]     162348013                               droplet          25               1.01 GiB
`
	snapshotListFormatOutput = `
ID                                      Resource Type
0a343fac-eacf-11e9-b96b-0a58ac144633    volume
0e0adfa4-eacf-11e9-9e75-0a58ac14c13b    volume
53344211                                droplet
53344231                                droplet
`
	snapshotListNoHeaderOutput = `
0a343fac-eacf-11e9-b96b-0a58ac144633    volume-nyc1-01-1570651053836                2019-10-09T19:57:36Z    [nyc1]    e2068b37-eace-11e9-85ad-0a58ac14430f    volume     100    0.00 GiB    
0e0adfa4-eacf-11e9-9e75-0a58ac14c13b    volume-lon1-01-1570651061232                2019-10-09T19:57:42Z    [lon1]    fcaf04e4-eace-11e9-a09f-0a58ac14c0f4    volume     100    0.00 GiB    
53344211                                ubuntu-s-1vcpu-1gb-nyc1-01-1570651077842    2019-10-09T19:57:59Z    [nyc1]    162347943                               droplet    25     1.01 GiB    
53344231                                ubuntu-s-1vcpu-1gb-lon1-01-1570651124450    2019-10-09T19:58:50Z    [lon1]    162348013                               droplet    25     1.01 GiB
`
	snapshotListRegionOutput = `
ID                                      Name                                        Created at              Regions    Resource ID                             Resource Type    Min Disk Size    Size        Tags
0a343fac-eacf-11e9-b96b-0a58ac144633    volume-nyc1-01-1570651053836                2019-10-09T19:57:36Z    [nyc1]     e2068b37-eace-11e9-85ad-0a58ac14430f    volume           100              0.00 GiB    
53344211                                ubuntu-s-1vcpu-1gb-nyc1-01-1570651077842    2019-10-09T19:57:59Z    [nyc1]     162347943                               droplet          25               1.01 GiB
`
	snapshotListDropletOutput = `
ID          Name                                        Created at              Regions    Resource ID    Resource Type    Min Disk Size    Size        Tags
53344211    ubuntu-s-1vcpu-1gb-nyc1-01-1570651077842    2019-10-09T19:57:59Z    [nyc1]     162347943      droplet          25               1.01 GiB    
53344231    ubuntu-s-1vcpu-1gb-lon1-01-1570651124450    2019-10-09T19:58:50Z    [lon1]     162348013      droplet          25               1.01 GiB
`
	snapshotListVolumeOutput = `
ID                                      Name                            Created at              Regions    Resource ID                             Resource Type    Min Disk Size    Size        Tags
0a343fac-eacf-11e9-b96b-0a58ac144633    volume-nyc1-01-1570651053836    2019-10-09T19:57:36Z    [nyc1]     e2068b37-eace-11e9-85ad-0a58ac14430f    volume           100              0.00 GiB    
0e0adfa4-eacf-11e9-9e75-0a58ac14c13b    volume-lon1-01-1570651061232    2019-10-09T19:57:42Z    [lon1]     fcaf04e4-eace-11e9-a09f-0a58ac14c0f4    volume           100              0.00 GiB
`
	snapshotListDropletRegionOutput = `
ID          Name                                        Created at              Regions    Resource ID    Resource Type    Min Disk Size    Size        Tags
53344211    ubuntu-s-1vcpu-1gb-nyc1-01-1570651077842    2019-10-09T19:57:59Z    [nyc1]     162347943      droplet          25               1.01 GiB
`
)
