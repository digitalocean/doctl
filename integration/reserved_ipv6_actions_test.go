package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("compute/reserved-ipv6-action/assign", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/reserved_ipv6/fd53:616d:6d60::1071:5001/actions":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := io.ReadAll(req.Body)
				expect.NoError(err)

				reqJson := struct {
					Type string `json:"type"`
				}{}
				err = json.NewDecoder(strings.NewReader(string(reqBody))).Decode(&reqJson)
				expect.NoError(err)

				var matchedRequest, responseJSON string
				if reqJson.Type == "assign" {
					matchedRequest = reservedIPv6AssignActionRequest
					responseJSON = reservedIPv6AssignActionResponse
				} else if reqJson.Type == "unassign" {
					matchedRequest = reservedIPv6UnassignActionRequest
					responseJSON = reservedIPv6UnassignActionResponse
				} else {
					t.Fatalf("received unknown request: %s", reqJson.Type)
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

	when("assign action is executed", func() {
		it("assigns reserved ipv6 to the droplet", func() {
			aliases := []string{"assign"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"reserved-ipv6-action",
					alias,
					"fd53:616d:6d60::1071:5001",
					"1212",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(reservedIPv6AssignActionOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("unassign action is executed", func() {
		it("unassigns reserved ipv6 from the droplet", func() {
			aliases := []string{"unassign"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"compute",
					"reserved-ipv6-action",
					alias,
					"fd53:616d:6d60::1071:5001",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(reservedIPv6UnassignActionOutput), strings.TrimSpace(string(output)))
			}
		})
	})

})

const (
	reservedIPv6AssignActionOutput = `
ID            Status         Type         Started At                       Completed At    Resource ID    Resource Type    Region
2208110886    in-progress    assign_ip    2021-10-01 01:00:00 +0000 UTC    <nil>           0              reserved_ipv6    
`
	reservedIPv6AssignActionResponse = `
{
  "action": {
    "id": 2208110886,
    "status": "in-progress",
	"type": "assign_ip",
	"started_at": "2021-10-01T01:00:00Z",
	"resource_type": "reserved_ipv6",
	"region_slug": "nyc3"
  }
}
`

	reservedIPv6AssignActionRequest = `
{"type":"assign", "droplet_id": 1212}
`

	reservedIPv6UnassignActionOutput = `
ID            Status         Type         Started At                       Completed At    Resource ID    Resource Type    Region
2208110887    in-progress    remove_ip    2021-10-01 01:00:00 +0000 UTC    <nil>           0              reserved_ipv6    
`
	reservedIPv6UnassignActionResponse = `
{
  "action": {
    "id": 2208110887,
    "status": "in-progress",
	"type": "remove_ip",
	"started_at": "2021-10-01T01:00:00Z",
	"resource_type": "reserved_ipv6",
	"region_slug": "nyc3"
  }
}
`

	reservedIPv6UnassignActionRequest = `
{"type":"unassign"}
`
)
