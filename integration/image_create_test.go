package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

func testImageCreate(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
	)

	it.Before(func() {
		expect = require.New(t)
		httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			auth := req.Header.Get("Authorization")
			if auth != "Bearer some-magic-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			var err error
			_, err = ioutil.ReadAll(req.Body)
			expect.NoError(err)

			w.Write([]byte(imageCreateResponse))
			return
		}))
	})

	when("all the required flags are passed", func() {
		cmd := exec.Command(builtBinaryPath,
			"--image-name", "ubuntu-18.04-minimal",
			"--image-url", "http://cloud-images.ubuntu.com/minimal/releases/bionic/release/ubuntu-18.04-minimal-cloudimg-amd64.img",
			"--region", "nyc3",
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, fmt.Sprintf("received unexpected error: %s", output))
		expect.Equal(strings.TrimSpace(imageCreateResponse), strings.TrimSpace(string(output)))
	})

	when("required arguments are missing", func() {
		baseErr := `Error: (images.create%s) command is missing required arguments`

		var cases = []struct {
			desc string
			err  string
			args []string
		}{
			{
				"missing all",
				fmt.Sprintf(baseErr, ""),
				[]string{
					"--image-description", "an ubuntu custom minimal image",
				},
			},
			{
				"missing name",
				fmt.Sprintf(baseErr, ""),
				[]string{
					"--image-description", "an ubuntu custom minimal image",
					"--image-url", "http://cloud-images.ubuntu.com/minimal/releases/bionic/release/ubuntu-18.04-minimal-cloudimg-amd64.img",
					"--region", "nyc3",
				},
			},
			{
				"missing region",
				fmt.Sprintf(baseErr, ""),
				[]string{
					"--image-description", "an ubuntu custom minimal image",
					"--image-name", "ubuntu-18.04-minimal",
					"--image-url", "http://cloud-images.ubuntu.com/minimal/releases/bionic/release/ubuntu-18.04-minimal-cloudimg-amd64.img",
				},
			},
			{
				"missing image url",
				fmt.Sprintf(baseErr, ""),
				[]string{
					"--image-description", "an ubuntu custom minimal image",
					"--image-name", "ubuntu-18.04-minimal",
					"--region", "nyc3",
				},
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
}

const imageCreateResponse = `{
	"name": "ubuntu-18.04-minimal",
	"url": "http://cloud-images.ubuntu.com/minimal/releases/bionic/release/ubuntu-18.04-minimal-cloudimg-amd64.img",
	"distribution": "Ubuntu",
	"region": "nyc3",
	"description": "Cloud-optimized image w/ small footprint",
	"tags": [
	  "base-image",
	  "prod"
	]
  }`
