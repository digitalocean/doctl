package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/tags/remove", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)
		expectedJSONSingle := `{
  "resources": [
    {
      "resource_id": "123456",
      "resource_type": "droplet"
    }
  ]
}`
		expectedJSONMulti := `{
  "resources": [
    {
      "resource_id": "123456",
      "resource_type": "droplet"
    },
    {
      "resource_id": "64f051b2-a702-11ec-bae8-6b11363a9137",
      "resource_type": "kubernetes"
    }
  ]
}`

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/tags/foo/resources":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				request := godo.UntagResourcesRequest{}
				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)
				expect.JSONEq(expectedJSONSingle, string(reqBody))

				w.WriteHeader(http.StatusNoContent)

			case "/v2/tags/bar/resources":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				request := godo.UntagResourcesRequest{}
				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)
				expect.JSONEq(expectedJSONMulti, string(reqBody))

				w.WriteHeader(http.StatusNoContent)

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
		it("removes the right tag", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"tag",
				"remove",
				"foo",
				"--resource", "do:droplet:123456",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(output)
		})
	})

	when("multiple resources are provided and all required flags are passed", func() {
		it("removes the right tag", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"tag",
				"remove",
				"bar",
				"--resource", "do:droplet:123456",
				"--resource", "do:kubernetes:64f051b2-a702-11ec-bae8-6b11363a9137",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Empty(output)
		})
	})

})
