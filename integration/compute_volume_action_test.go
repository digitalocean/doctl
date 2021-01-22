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

var _ = suite("compute/volume-action", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/volumes/1/actions":
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

				expect.JSONEq(`{"region":"moonbase","size_gigabytes":100,"type":"resize"}`, string(reqBody))

				w.Write([]byte(volumeActionResponse))
			case "/v2/volumes/22/actions":
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

				expect.JSONEq(`{"droplet_id":11,"type":"attach"}`, string(reqBody))

				w.Write([]byte(volumeActionResponse))
			case "/v2/volumes/13/actions":
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

				expect.JSONEq(`{"droplet_id":14,"type":"detach"}`, string(reqBody))

				w.Write([]byte(volumeActionResponse))
			case "/v2/volumes/1213/actions/22":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(volumeActionResponse))
			case "/v2/volumes/1213/actions":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(volumeActionsResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is attach", func() {
		it("attaches a volume", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"volume-action",
				"attach",
				"22",
				"11",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(volumeActionOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is detach", func() {
		it("detaches the volume", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"volume-action",
				"detach",
				"13",
				"14",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(volumeActionOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is resize", func() {
		it("resizes the particular volume", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"volume-action",
				"resize",
				"1",
				"--region", "moonbase",
				"--size", "100",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(volumeActionOutput), strings.TrimSpace(string(output)))
		})
	})
	when("command is list", func() {
		it("Retrieve a list of actions taken on a volume", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"volume-action",
				"list",
				"1213",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(volumeActionOutput), strings.TrimSpace(string(output)))
		})
	})
	when("command is get", func() {
		it("Retrieve the status of the particular volume action", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"volume-action",
				"get",
				"1213",
				"--action-id",
				"22",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(volumeActionOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	volumeActionOutput = `
ID          Status         Type             Started At                       Completed At    Resource ID    Resource Type    Region
68212773    in-progress    detach_volume    2015-10-15 17:46:15 +0000 UTC    <nil>           0              backend          nyc1
	`
	volumeActionResponse = `
{
  "action": {
    "id": 68212773,
    "status": "in-progress",
    "type": "detach_volume",
    "started_at": "2015-10-15T17:46:15Z",
    "completed_at": null,
    "resource_id": null,
    "resource_type": "backend",
    "region": {
      "name": "New York 1",
      "slug": "nyc1",
      "sizes": [ "s-32vcpu-192gb" ],
      "features": [ "metadata" ],
      "available": true
    },
    "region_slug": "nyc1"
  }
}
`

	volumeActionsResponse = `
	{
		"actions": [{
		  "id": 68212773,
		  "status": "in-progress",
		  "type": "detach_volume",
		  "started_at": "2015-10-15T17:46:15Z",
		  "completed_at": null,
		  "resource_id": null,
		  "resource_type": "backend",
		  "region": {
			"name": "New York 1",
			"slug": "nyc1",
			"sizes": [ "s-32vcpu-192gb" ],
			"features": [ "metadata" ],
			"available": true
		  },
		  "region_slug": "nyc1"
		}]
	  }
`
)
