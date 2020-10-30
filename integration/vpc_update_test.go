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

var _ = suite("vpcs/update", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/vpcs/vpc-uuid":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPatch {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				reqBody, err := ioutil.ReadAll(req.Body)
				expect.NoError(err)

				request := struct {
					Name        string `json:"name"`
					Description string `json:"description"`
				}{}

				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)

				t, err := template.New("response").Parse(vpcsUpdateResponse)
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
		it("updates the VPC", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"vpcs",
				"update",
				"vpc-uuid",
				"--name", "renamed-new-vpc",
				"--description", "A new description",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(vpcsUpdateOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	vpcsUpdateOutput = `
ID                                      URN                                            Name               Description          IP Range         Region    Created At                       Default
5a4981aa-9653-4bd1-bef5-d6bff52042e4    do:vpc:5a4981aa-9653-4bd1-bef5-d6bff52042e4    renamed-new-vpc    A new description    10.10.10.0/24    nyc1      2020-03-13 19:20:47 +0000 UTC    false
`
	vpcsUpdateResponse = `
{
  "vpc": {
    "id": "5a4981aa-9653-4bd1-bef5-d6bff52042e4",
    "urn": "do:vpc:5a4981aa-9653-4bd1-bef5-d6bff52042e4",
    "name": "renamed-new-vpc",
    "description": "A new description",
    "region": "nyc1",
    "ip_range": "10.10.10.0/24",
    "created_at": "2020-03-13T19:20:47Z",
    "default": false
  }
}
`
)
