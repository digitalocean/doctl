package integration

import (
	"encoding/json"
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

func testDropletCreate(t *testing.T, when spec.G, it spec.S) {
	var (
		expect  *require.Assertions
		server  *httptest.Server
		reqBody []byte
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

				var err error
				reqBody, err = ioutil.ReadAll(req.Body)
				expect.NoError(err)

				w.Write([]byte(dropletCreateResponse))
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
		it("creates a droplet", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"create",
				"some-droplet-name",
				"--image", "a-test-image",
				"--region", "a-test-region",
				"--size", "a-test-size",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Equal(strings.TrimSpace(dropletCreateOutput), strings.TrimSpace(string(output)))

			request := &struct {
				Name   string
				Image  string
				Region string
				Size   string
			}{}

			err = json.Unmarshal(reqBody, request)
			expect.NoError(err)

			expect.Equal("some-droplet-name", request.Name)
			expect.Equal("a-test-image", request.Image)
			expect.Equal("a-test-region", request.Region)
			expect.Equal("a-test-size", request.Size)
		})
	})

	when("missing required arguments", func() {
		base := []string{
			"-t", "some-magic-token",
			"-u", "https://www.example.com",
			"compute",
			"droplet",
			"create",
		}

		cases := []struct {
			desc string
			args []string
		}{
			{desc: "missing all", args: base},
			{desc: "missing only name", args: append(base, []string{"--size", "test", "--region", "test", "--image", "test"}...)},
			{desc: "missing only region", args: append(base, []string{"some-name", "--size", "test", "--image", "test"}...)},
			{desc: "missing only size", args: append(base, []string{"some-name", "--image", "test", "--region", "test"}...)},
			{desc: "missing only image", args: append(base, []string{"some-name", "--image", "test", "--region", "test"}...)},
		}

		for _, c := range cases {
			when(c.desc, func() {
				it("returns an error", func() {
					cmd := exec.Command(builtBinaryPath, c.args...)

					output, err := cmd.CombinedOutput()
					expect.Error(err)
					expect.Contains(string(output), "Error: (droplet.create.size) command is missing required arguments")
				})
			})
		}
	})
}

const dropletCreateResponse = `{
  "droplet": {
    "id": 1111,
    "memory": 12,
    "vcpus": 13,
    "disk": 15,
    "name": "some-droplet-name",
    "networks": {
      "v4": [
        {"type": "public", "ip_address": "1.2.3.4"},
        {"type": "private", "ip_address": "7.7.7.7"}
      ]
    },
    "image": {
      "distribution": "some-distro",
      "name": "some-image-name"
    },
    "region": {
      "slug": "some-region-slug"
    },
    "status": "active",
    "tags": ["yes"],
    "features": ["remotes"],
    "volume_ids": ["some-volume-id"]
  }
}`

const dropletCreateOutput = `
ID      Name                 Public IPv4    Private IPv4    Public IPv6    Memory    VCPUs    Disk    Region              Image                          Status    Tags    Features    Volumes
1111    some-droplet-name    1.2.3.4        7.7.7.7                        12        13       15      some-region-slug    some-distro some-image-name    active    yes     remotes     some-volume-id
`
