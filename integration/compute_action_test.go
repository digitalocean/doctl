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

var _ = suite("compute/action", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect    *require.Assertions
		server    *httptest.Server
		callCount int
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/actions":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(computeActionListResponse))
			case "/v2/actions/10101":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				if callCount != 1 {
					w.Write([]byte(computeActionWaitInProgressResponse))
					callCount++
					return
				}

				w.Write([]byte(computeActionWaitCompletedResponse))
			case "/v2/actions/20202":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(computeActionGetResponse))
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
		it("gets the specified compute action", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"action",
				"get",
				"20202",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(computeActionGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is list", func() {
		it("lists compute actions", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"action",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(computeActionListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is wait", func() {
		it("waits for a specified compute action", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"action",
				"wait",
				"10101",
				"--poll-timeout", "1",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(computeActionWaitOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	computeActionGetOutput = `
ID       Status       Type      Started At                       Completed At                     Resource ID    Resource Type    Region
20202    completed    create    2014-11-14 16:29:21 +0000 UTC    2014-11-14 16:30:06 +0000 UTC    3164444        droplet          nyc3
	`
	computeActionGetResponse = `
{
  "action": {
    "id": 20202,
    "status": "completed",
    "type": "create",
    "started_at": "2014-11-14T16:29:21Z",
    "completed_at": "2014-11-14T16:30:06Z",
    "resource_id": 3164444,
    "resource_type": "droplet",
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
	computeActionListOutput = `
ID      Status       Type      Started At                       Completed At                     Resource ID    Resource Type    Region
4444    completed    create    2014-11-14 16:29:21 +0000 UTC    2014-11-14 16:30:06 +0000 UTC    3164444        droplet          nyc3
5555    completed    create    2014-11-14 16:29:21 +0000 UTC    2014-11-14 16:30:06 +0000 UTC    3164455        droplet          nyc3
	`
	computeActionListResponse = `
{
  "actions": [
    {
      "id": 4444,
      "status": "completed",
      "type": "create",
      "started_at": "2014-11-14T16:29:21Z",
      "completed_at": "2014-11-14T16:30:06Z",
      "resource_id": 3164444,
      "resource_type": "droplet",
      "region": {
        "name": "New York 3",
        "slug": "nyc3",
        "sizes": [ "s-24vcpu-128gb" ],
        "features": [ "image_transfer" ],
        "available": true
      },
      "region_slug": "nyc3"
    },
    {
      "id": 5555,
      "status": "completed",
      "type": "create",
      "started_at": "2014-11-14T16:29:21Z",
      "completed_at": "2014-11-14T16:30:06Z",
      "resource_id": 3164455,
      "resource_type": "droplet",
      "region": {
        "name": "New York 3",
        "slug": "nyc3",
        "sizes": [ "s-24vcpu-128gb" ],
        "features": [ "image_transfer" ],
        "available": true
      },
      "region_slug": "nyc3"
    }
  ],
  "links": {
    "pages": {}
  },
  "meta": {
    "total": 2
  }
}
`
	computeActionWaitOutput = `
ID       Status       Type      Started At                       Completed At                     Resource ID    Resource Type    Region
20202    completed    create    2014-11-14 16:29:21 +0000 UTC    2014-11-14 16:30:06 +0000 UTC    2222           droplet          nyc3
	`
	computeActionWaitInProgressResponse = `
{
  "action": {
    "id": 20202,
    "status": "in-progress",
    "type": "create",
    "started_at": "2014-11-14T16:29:21Z",
    "resource_id": 2222,
    "resource_type": "droplet",
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
	computeActionWaitCompletedResponse = `
{
  "action": {
    "id": 20202,
    "status": "completed",
    "type": "create",
    "started_at": "2014-11-14T16:29:21Z",
    "completed_at": "2014-11-14T16:30:06Z",
    "resource_id": 2222,
    "resource_type": "droplet",
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
