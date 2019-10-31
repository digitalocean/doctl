package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

type tagRequest struct {
	Resources []struct {
		ResourceID   string `json:"resource_id"`
		ResourceType string `json:"resource_type"`
	} `json:"resources"`
}

var _ = suite("compute/droplet/tag", func(t *testing.T, when spec.G, it spec.S) {
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

				w.Write([]byte(`{"droplets":[{"name":"some-droplet-name", "id": 1337}]}`))
			case "/v2/tags/my-tag/resources":
				body, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				var tagRequest tagRequest
				err = json.Unmarshal(body, &tagRequest)
				expect.NoError(err)

				if req.Method == http.MethodPost || req.Method == http.MethodDelete {
					if tagRequest.Resources[0].ResourceID == "1444" {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte(`{"message": "tag not found"}`))
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
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
		base := []string{
			"-t", "some-magic-token",
			"compute",
			"droplet",
		}

		cases := []struct {
			desc string
			args []string
		}{
			{desc: "when tagging", args: append(base, []string{"tag", "some-droplet-name", "--tag-name", "my-tag"}...)},
			{desc: "when untagging", args: append(base, []string{"untag", "some-droplet-name", "--tag-name", "my-tag"}...)},
		}

		for _, c := range cases {
			commandArgs := c.args

			when(c.desc, func() {
				it("completes successfully", func() {
					finalArgs := append([]string{"-u", server.URL}, commandArgs...)
					cmd := exec.Command(builtBinaryPath, finalArgs...)

					output, err := cmd.CombinedOutput()
					expect.NoError(err, fmt.Sprintf("received error output: %s", output))
					expect.Empty(output)
				})
			})
		}
	})

	when("an error occurs", func() {
		base := []string{
			"-t", "some-magic-token",
			"compute",
			"droplet",
		}

		cases := []struct {
			desc string
			args []string
			err  string
		}{
			{
				desc: "when tagging and droplet id is missing",
				args: append(base, []string{"tag", "1444", "--tag-name", "my-tag"}...),
				err:  "^Error: POST http.*: 404 tag not found",
			},
			{
				desc: "when untagging and droplet id is missing",
				args: append(base, []string{"untag", "1444", "--tag-name", "my-tag"}...),
				err:  "^Error: DELETE http.*: 404 tag not found",
			},
			{
				desc: "when tagging and droplet name is missing",
				args: append(base, []string{"untag", "bad-droplet-name", "--tag-name", "my-tag"}...),
				err:  `^Error:.*\".*\" could not be found`,
			},
			{
				desc: "when untagging and droplet name is missing",
				args: append(base, []string{"untag", "bad-droplet-name", "--tag-name", "my-tag"}...),
				err:  `^Error:.*\".*\" could not be found`,
			},
		}

		for _, c := range cases {
			commandArgs := c.args
			errRegex := c.err

			when(c.desc, func() {
				it("completes successfully", func() {
					finalArgs := append([]string{"-u", server.URL}, commandArgs...)
					cmd := exec.Command(builtBinaryPath, finalArgs...)

					output, err := cmd.CombinedOutput()
					expect.Error(err)
					expect.Regexp(regexp.MustCompile(errRegex), strings.TrimSpace(string(output)))

				})
			})
		}
	})
})
