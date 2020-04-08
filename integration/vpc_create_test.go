package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
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

var _ = suite("vpcs/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
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

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				request := struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					RegionSlug  string `json:"region"`
					IPRange     string `json:"ip_range"`
				}{}

				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)

				t, err := template.New("response").Parse(vpcsCreateResponse)
				expect.NoError(err)

				var b []byte
				buffer := bytes.NewBuffer(b)
				err = t.Execute(buffer, request)
				expect.NoError(err)

				w.Write(buffer.Bytes())
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
		it("creates new VPC", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"vpcs",
				"create",
				"--name", "some-vpc",
				"--region", "nyc3",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(vpcsCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("missing required arguments", func() {
		base := []string{
			"-t", "some-magic-token",
			"-u", "https://www.example.com",
			"vpcs",
			"create",
		}

		baseErr := `Error: (vpcs.create%s) command is missing required arguments`

		cases := []struct {
			desc string
			err  string
			args []string
		}{
			{desc: "missing all", err: fmt.Sprintf(baseErr, ".name"), args: base},
			{desc: "missing only name", err: fmt.Sprintf(baseErr, ".name"), args: append(base, []string{"--region", "not missing"}...)},
			{desc: "missing only region", err: fmt.Sprintf(baseErr, ".region"), args: append(base, []string{"--name", "not missing"}...)},
		}

		for _, c := range cases {
			commandArgs := c.args
			expectedErr := c.err

			when(c.desc, func() {
				it("returns an error", func() {
					cmd := exec.Command(builtBinaryPath, commandArgs...)

					output, err := cmd.CombinedOutput()
					expect.Error(err)
					expect.Contains(string(output), expectedErr)
				})
			})
		}
	})
})

const (
	vpcsCreateOutput = `
ID                                      URN                                            Name          Description        IP Range         Region    Created At                       Default
5a4981aa-9653-4bd1-bef5-d6bff52042e4    do:vpc:5a4981aa-9653-4bd1-bef5-d6bff52042e4    my-new-vpc    vpc description    10.10.10.0/24    nyc1      2020-03-13 19:20:47 +0000 UTC    false`
	vpcsCreateResponse = `
{
  "vpc": {
    "id": "5a4981aa-9653-4bd1-bef5-d6bff52042e4",
    "urn": "do:vpc:5a4981aa-9653-4bd1-bef5-d6bff52042e4",
    "name": "my-new-vpc",
    "description": "vpc description",
    "region": "nyc1",
    "ip_range": "10.10.10.0/24",
    "created_at": "2020-03-13T19:20:47Z",
    "default": false
  }
}
`
)
