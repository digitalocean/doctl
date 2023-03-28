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

var _ = suite("compute/reserved-ip/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/reserved_ips":
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

				responseJSON := reservedIPCreateResponse
				matchedRequest := reservedIPCreateRequest
				if !strings.Contains(string(reqBody), "droplet_id") {
					matchedRequest = reservedIPRegionCreateRequest
					responseJSON = reservedIPCreateRegionResponse
				}

				expect.JSONEq(matchedRequest, string(reqBody))

				w.Write([]byte(responseJSON))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

	})

	when("the droplet-id flag is provided", func() {
		it("creates the reserved-ip", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"reserved-ip",
					alias,
					"--droplet-id", "1212",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(reservedIPCreateOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("the region and project-id flags are provided", func() {
		it("creates the reserved-ip", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"reserved-ip",
					alias,
					"--region", "nyc3",
					"--project-id", "c98374fa-35e2-11ed-870f-c7de97c5d5ed",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(reservedIPCreateRegionOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("the region and droplet-id flags are provided", func() {
		it("returns the expected error", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"reserved-ip",
					alias,
					"--region", "nyc3",
					"--droplet-id", "1234",
				)

				output, err := cmd.CombinedOutput()
				expect.Error(err)
				expect.Equal(strings.TrimSpace("Error: Only one of `--region` or `--droplet-id` may be specified when creating a reserved IP address."), strings.TrimSpace(string(output)))
			}
		})
	})

	when("the region and project-id flags are provided", func() {
		it("returns the expected error", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"reserved-ip",
					alias,
					"--droplet-id", "1234",
					"--project-id", "c98374fa-35e2-11ed-870f-c7de97c5d5ed",
				)

				output, err := cmd.CombinedOutput()
				expect.Error(err)
				expect.Equal(strings.TrimSpace("Error: Only one of `--project-id` or `--droplet-id` may be specified when creating a reserved IP address."), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	reservedIPCreateOutput = `
IP             Region    Droplet ID    Droplet Name    Project ID
45.55.96.47    nyc3      1212          magic-name      c98374fa-35e2-11ed-870f-c7de97c5d5ed
`
	reservedIPCreateRegionOutput = `
IP             Region    Droplet ID    Droplet Name    Project ID
45.55.96.47    nyc3                                    c98374fa-35e2-11ed-870f-c7de97c5d5ed
`
	reservedIPCreateResponse = `
{
  "reserved_ip": {
    "ip": "45.55.96.47",
    "droplet": {
      "id": 1212,
      "name": "magic-name"
    },
    "region": {
      "name": "New York 3",
      "slug": "nyc3",
      "sizes": [ "s-32vcpu-192gb" ],
      "features": [ "metadata" ],
      "available": true
    },
    "locked": false,
	"project_id": "c98374fa-35e2-11ed-870f-c7de97c5d5ed"
  },
  "links": {}
}
`
	reservedIPCreateRegionResponse = `
{
  "reserved_ip": {
    "ip": "45.55.96.47",
    "droplet": null,
    "region": {
      "name": "New York 3",
      "slug": "nyc3",
      "sizes": [ "s-32vcpu-192gb" ],
      "features": [ "metadata" ],
      "available": true
    },
    "locked": false,
	"project_id": "c98374fa-35e2-11ed-870f-c7de97c5d5ed"
  },
  "links": {}
}
`
	reservedIPCreateRequest = `
{"droplet_id":1212}
`
	reservedIPRegionCreateRequest = `
{"region":"nyc3","project_id":"c98374fa-35e2-11ed-870f-c7de97c5d5ed"}
`
)
