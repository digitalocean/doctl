package integration

import (
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

var _ = suite("compute/image-action", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/images/1212/actions/4444":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(imageActionResponse))
			case "/v2/images/1313/actions":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				expect.JSONEq(`{"region":"saturn","type":"transfer"}`, string(reqBody))

				w.Write([]byte(imageActionResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is get", func() {
		it("gets the specified image action", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"image-action",
				"get",
				"1212",
				"--action-id",
				"4444",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(imageActionOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is transfer", func() {
		it("transfers the image", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"image-action",
				"transfer",
				"1313",
				"--region",
				"saturn",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(imageActionOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	imageActionOutput = `
ID          Status         Type        Started At                       Completed At    Resource ID    Resource Type    Region
36805527    in-progress    transfer    2014-11-14 16:42:45 +0000 UTC    <nil>           7938269        image            nyc3
	`
	imageActionResponse = `
{
  "action": {
    "id": 36805527,
    "status": "in-progress",
    "type": "transfer",
    "started_at": "2014-11-14T16:42:45Z",
    "completed_at": null,
    "resource_id": 7938269,
    "resource_type": "image",
    "region": {
      "name": "New York 3",
      "slug": "nyc3",
      "sizes": [ "s-24vcpu-128gb" ],
      "features": [ "image_transfer" ],
      "available": true
    },
    "region_slug": "nyc3"
  }
}
`
)
