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

type tagRequest struct {
	Resources []struct {
		ResourceID   string `json:"resource_id"`
		ResourceType string `json:"resource_type"`
	} `json:"resources"`
}

func testDropletTag(t *testing.T, when spec.G, it spec.S) {
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

				w.Write([]byte(`{"droplets":[{"name":"some-droplet-name", "id": 1337}]}`))
			case "/v2/tags/my-tag/resources":
				body, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				var tagRequest tagRequest
				err = json.Unmarshal(body, &tagRequest)
				expect.NoError(err)

				if req.Method == "POST" {
					if tagRequest.Resources[0].ResourceID == "1444" {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte(`{"message": "tag not found"}`))
						return
					}

					w.Write([]byte(`{}`))
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
		it("tags the droplet", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"tag",
				"some-droplet-name",
				"--tag-name", "my-tag",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
		})
	})

	when("the droplet-id cannot be found", func() {
		it("returns no error", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"tag",
				"1444",
				"--tag-name", "my-tag",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(string(output)), fmt.Sprintf("Error: POST %s/v2/tags/my-tag/resources: 404 tag not found", server.URL))
		})
	})

	when("the droplet-name cannot be found", func() {
		it("returns no error", func() {
			dropletName := "missing-droplet"
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"droplet",
				"tag",
				dropletName,
				"--tag-name", "my-tag",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Equal(strings.TrimSpace(string(output)), fmt.Sprintf("Error: droplet with name %q could not be found", dropletName))
		})
	})
}
