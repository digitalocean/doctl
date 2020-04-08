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

var _ = suite("vpcs/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
		cmd    *exec.Cmd
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/vpcs":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(vpcsListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))

		cmd = exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"vpcs",
		)

	})

	when("command is list", func() {
		it("lists all VPCs", func() {
			cmd.Args = append(cmd.Args, []string{"list"}...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(vpcsListOutput), strings.TrimSpace(string(output)))
		})
	})

	when("command is ls", func() {
		it("lists all VPCs", func() {
			cmd.Args = append(cmd.Args, []string{"ls"}...)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(vpcsListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	vpcsListOutput = `
ID                                      URN                                            Name            Description    IP Range         Region    Created At                       Default
5a4981aa-9653-4bd1-bef5-d6bff52042e4    do:vpc:5a4981aa-9653-4bd1-bef5-d6bff52042e4    my-new-vpc                     10.10.10.0/24    nyc1      2020-03-13 19:20:47 +0000 UTC    false
e0fe0f4d-596a-465e-a902-571ce57b79fa    do:vpc:e0fe0f4d-596a-465e-a902-571ce57b79fa    default-nyc1                   10.102.0.0/20    nyc1      2020-03-13 19:29:20 +0000 UTC    true
d455e75d-4858-4eec-8c95-da2f0a5f93a7    do:vpc:d455e75d-4858-4eec-8c95-da2f0a5f93a7    default-nyc3                   10.100.0.0/20    nyc3      2019-11-19 22:19:35 +0000 UTC    true
`
	vpcsListResponse = `
{
  "vpcs": [
    {
      "id": "5a4981aa-9653-4bd1-bef5-d6bff52042e4",
      "urn": "do:vpc:5a4981aa-9653-4bd1-bef5-d6bff52042e4",
      "name": "my-new-vpc",
      "description": "",
      "region": "nyc1",
      "ip_range": "10.10.10.0/24",
      "created_at": "2020-03-13T19:20:47Z",
      "default": false
    },
    {
      "id": "e0fe0f4d-596a-465e-a902-571ce57b79fa",
      "urn": "do:vpc:e0fe0f4d-596a-465e-a902-571ce57b79fa",
      "name": "default-nyc1",
      "description": "",
      "region": "nyc1",
      "ip_range": "10.102.0.0/20",
      "created_at": "2020-03-13T19:29:20Z",
      "default": true
    },
    {
      "id": "d455e75d-4858-4eec-8c95-da2f0a5f93a7",
      "urn": "do:vpc:d455e75d-4858-4eec-8c95-da2f0a5f93a7",
      "name": "default-nyc3",
      "description": "",
      "region": "nyc3",
      "ip_range": "10.100.0.0/20",
      "created_at": "2019-11-19T22:19:35Z",
      "default": true
    }
  ],
  "links": {
  },
  "meta": {
    "total": 3
  }
}
`
)
