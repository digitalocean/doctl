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

var _ = suite("compute/image/list-application", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				if req.URL.Query().Get("type") != "application" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				w.Write([]byte(imageListApplicationResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

	})

	when("passing no flags", func() {
		it("lists all application images", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"image",
				"list-application",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(imageListApplicationOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	imageListApplicationOutput = `
ID         Name                                        Type    Distribution    Slug             Public    Min Disk
6376601    Ruby on Rails on 14.04 (Nginx + Unicorn)            Ubuntu          ruby-on-rails    true      20
6376602    Ruby on Rails on 14.04 (Nginx + Unicorn)            Ubuntu          ruby-on-rails    false     20
	`
	imageListApplicationResponse = `{
  "images": [
    {
      "id": 6376601,
      "name": "Ruby on Rails on 14.04 (Nginx + Unicorn)",
      "distribution": "Ubuntu",
      "slug": "ruby-on-rails",
      "public": true,
      "regions": [ "nyc1", "ams1" ],
      "created_at": "2014-09-26T20:20:24Z",
      "min_disk_size": 20,
      "size_gigabytes": 2.34,
      "description": "",
      "tags": [],
      "status": "available",
      "error_message": ""
    },
    {
      "id": 6376602,
      "name": "Ruby on Rails on 14.04 (Nginx + Unicorn)",
      "distribution": "Ubuntu",
      "slug": "ruby-on-rails",
      "public": false,
      "regions": [ "nyc1", "ams1" ],
      "created_at": "2014-09-26T20:20:24Z",
      "min_disk_size": 20,
      "size_gigabytes": 2.34,
      "description": "",
      "tags": [],
      "status": "available",
      "error_message": ""
    }
  ],
  "links": {
    "pages": {}
  },
  "meta": {
    "total": 2
  }
}`
)
