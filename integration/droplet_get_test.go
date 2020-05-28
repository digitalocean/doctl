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

var _ = suite("compute/droplet/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect     *require.Assertions
		server     *httptest.Server
		configPath string
	)

	it.Before(func() {
		expect = require.New(t)

		dir, err := ioutil.TempDir("", "doct-integratio-tests")
		expect.NoError(err)

		configPath = filepath.Join(dir, "config.yaml")

		err = ioutil.WriteFile(configPath, []byte(dropletGetConfig), 0644)
		expect.NoError(err)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/droplets/5555":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer special-broken" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(dropletGetResponse))
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
		it("gets the specified droplet ID", func() {
			cmd := exec.Command(builtBinaryPath,
				"-c", configPath,
				"-u", server.URL,
				"compute",
				"droplet",
				"get",
				"5555",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a format", func() {
		it("displays only those columns", func() {
			cmd := exec.Command(builtBinaryPath,
				"-c", configPath,
				"-u", server.URL,
				"compute",
				"droplet",
				"get",
				"5555",
				"--format", "ID,Name",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletGetFormatOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a template", func() {
		it("renders the template with the values", func() {
			cmd := exec.Command(builtBinaryPath,
				"-c", configPath,
				"-u", server.URL,
				"compute",
				"droplet",
				"get",
				"5555",
				"--template", "this is magic {{.ID}} can be shown with {{.Region.Slug}}",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dropletGetTemplateOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dropletGetConfig = `
---
access-token: special-broken
`
	dropletGetOutput = `
ID      Name                 Public IPv4    Private IPv4    Public IPv6    Memory    VCPUs    Disk    Region              Image                          VPC UUID    Status    Tags    Features    Volumes
5555    some-droplet-name                                                  0         0        0       some-region-slug    some-distro some-image-name                active    yes     remotes     some-volume-id
`
	dropletGetFormatOutput = `
ID      Name
5555    some-droplet-name
`

	dropletGetTemplateOutput = `
this is magic 5555 can be shown with some-region-slug
`
	dropletGetResponse = `
{
  "droplet": {
    "id": 5555,
    "name": "some-droplet-name",
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
)
