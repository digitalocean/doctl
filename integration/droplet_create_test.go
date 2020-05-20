package integration

import (
	"encoding/json"
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

type dropletRequest struct {
	Name string `json:"name"`
}

var _ = suite("compute/droplet/create", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				var err error
				reqBody, err = ioutil.ReadAll(req.Body)
				expect.NoError(err)

				var dr dropletRequest
				err = json.Unmarshal(reqBody, &dr)
				expect.NoError(err)

				if dr.Name == "waiting-on-name" {
					w.Write([]byte(dropletCreateWaitResponse))
					return
				}

				w.Write([]byte(dropletCreateResponse))
			case "/poll-for-droplet":
				w.Write([]byte(actionCompletedResponse))
			case "/v2/droplets/777":
				// we don't really need another fake droplet here
				// since we've successfully tested all the behavior
				// at this point
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
				"--vpc-uuid", "00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletCreateOutput), strings.TrimSpace(string(output)))

			request := &struct {
				Name    string
				Image   string
				Region  string
				Size    string
				VPCUUID string `json:"vpc_uuid"`
			}{}

			err = json.Unmarshal(reqBody, request)
			expect.NoError(err)

			expect.Equal("some-droplet-name", request.Name)
			expect.Equal("a-test-image", request.Image)
			expect.Equal("a-test-region", request.Region)
			expect.Equal("a-test-size", request.Size)
			expect.Equal("00000000-0000-4000-8000-000000000000", request.VPCUUID)
		})
	})

	when("the wait flag is passed", func() {
		it("polls until the droplet is created", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"create",
				"waiting-on-name",
				"--wait",
				"--image", "a-test-image",
				"--region", "a-test-region",
				"--size", "a-test-size",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
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

		baseErr := `Error: (droplet.create%s) command is missing required arguments`

		cases := []struct {
			desc string
			err  string
			args []string
		}{
			{desc: "missing all", err: fmt.Sprintf(baseErr, ""), args: base},
			{desc: "missing only name", err: fmt.Sprintf(baseErr, ""), args: append(base, []string{"--size", "test", "--region", "test", "--image", "test"}...)},
			{desc: "missing only region", err: fmt.Sprintf(baseErr, ".region"), args: append(base, []string{"some-name", "--size", "test", "--image", "test"}...)},
			{desc: "missing only size", err: fmt.Sprintf(baseErr, ".size"), args: append(base, []string{"some-name", "--image", "test", "--region", "test"}...)},
			{desc: "missing only image", err: fmt.Sprintf(baseErr, ".image"), args: append(base, []string{"some-name", "--size", "test", "--region", "test"}...)},
		}

		for _, c := range cases {
			commandArgs := c.args
			expectedErr := c.err

			when(c.desc, func() {
				it("returns an error", func() {
					cmd := exec.Command(builtBinaryPath, commandArgs...)

					output, err := cmd.CombinedOutput()
					expect.Error(err)
					expect.Contains(string(output), expectedErr)
				})
			})
		}
	})
})

const (
	dropletCreateResponse = `
{
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
	"vpc_uuid": "00000000-0000-4000-8000-000000000000",
    "tags": ["yes"],
    "features": ["remotes"],
    "volume_ids": ["some-volume-id"]

  }
}`
	dropletCreateWaitResponse = `
{"droplet": {"id": 777}, "links": {"actions": [{"id":1, "rel":"create", "href":"poll-for-droplet"}]}}
`
	actionCompletedResponse = `
{"action": "id": 1, "status": "completed"}
`
	dropletCreateOutput = `
ID      Name                 Public IPv4    Private IPv4    Public IPv6    Memory    VCPUs    Disk    Region              Image                          VPC UUID                                Status    Tags    Features    Volumes
1111    some-droplet-name    1.2.3.4        7.7.7.7                        12        13       15      some-region-slug    some-distro some-image-name    00000000-0000-4000-8000-000000000000    active    yes     remotes     some-volume-id
`
)
