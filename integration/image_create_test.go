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

var _ = suite("compute/image/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/images":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(imageCreateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("all the required flags are passed", func() {
		it("creates an image", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"image",
				"create", "ubuntu-18.04-minimal",
				"--image-url", "http://cloud-images.ubuntu.com/minimal/releases/bionic/release/ubuntu-18.04-minimal-cloudimg-amd64.img",
				"--region", "nyc3",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received unexpected error: %s", output))
			expect.Equal(strings.TrimSpace(imageCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("required arguments are missing", func() {
		base := []string{
			"-t", "some-magic-token",
			"-u", "https://www.example.com",
			"compute",
			"image",
			"create",
		}

		baseErr := `Error: (image.create%s) command is missing required arguments`

		var cases = []struct {
			desc string
			err  string
			args []string
		}{
			{
				"missing all",
				fmt.Sprintf(baseErr, ".image-name"),
				base,
			},
			{
				"missing all flags",
				fmt.Sprintf(baseErr, ".image-url"),
				append(base, []string{
					"ubuntu-18.04-minimal",
				}...),
			},
			{
				"missing region",
				fmt.Sprintf(baseErr, ".region"),
				append(base, []string{
					"ubuntu-18.04-minimal",
					"--image-description", "an ubuntu custom minimal image",
					"--image-url", "http://cloud-images.ubuntu.com/minimal/releases/bionic/release/ubuntu-18.04-minimal-cloudimg-amd64.img",
				}...),
			},
			{
				"missing image url",
				fmt.Sprintf(baseErr, ".image-url"),
				append(base, []string{
					"ubuntu-18.04-minimal",
					"--image-description", "an ubuntu custom minimal image",
					"--region", "nyc3",
				}...),
			},
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

const imageCreateResponse = `{
	"image": {
	  "created_at": "2018-09-20T19:28:00Z",
	  "description": "Cloud-optimized image w/ small footprint",
	  "distribution": "Ubuntu",
	  "error_message": "",
	  "id": 38413969,
	  "name": "ubuntu-18.04-minimal",
	  "regions": [],
	  "type": "custom",
	  "tags": [
		"base-image",
		"prod"
	  ],
	  "status": "NEW"
	}
  }`

const imageCreateOutput = `
ID          Name                    Type      Distribution    Slug    Public    Min Disk
38413969    ubuntu-18.04-minimal    custom    Ubuntu                  false     0
`
