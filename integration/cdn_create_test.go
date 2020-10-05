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

var _ = suite("compute/cdn/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/cdn/endpoints":
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
					Origin        string `json:"origin"`
					CertificateID string `json:"certificate_id"`
					Domain        string `json:"custom_domain"`
					TTL           int    `json:"ttl"`
				}{}

				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)

				t, err := template.New("response").Parse(cdnCreateResponse)
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

	when("all flags are passed", func() {
		it("creates the cdn", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"compute",
				"cdn",
				"create",
				"magic-origin",
				"--certificate-id", "some-cert-id",
				"--domain", "example.com",
				"--ttl", "60",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(cdnCreateOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	cdnCreateOutput = `
ID                                      Origin          Endpoint                                         TTL    CustomDomain    CertificateID    CreatedAt
19f06b6a-3ace-4315-b086-499a0e521b76    magic-origin    static-images.nyc3.cdn.digitaloceanspaces.com    60     example.com     some-cert-id     2018-07-19 15:04:16 +0000 UTC
`
	cdnCreateResponse = `
{
  "endpoint": {
    "id": "19f06b6a-3ace-4315-b086-499a0e521b76",
    "origin": "{{.Origin}}",
    "endpoint": "static-images.nyc3.cdn.digitaloceanspaces.com",
    "created_at": "2018-07-19T15:04:16Z",
    "certificate_id": "{{.CertificateID}}",
    "custom_domain": "{{.Domain}}",
    "ttl": {{.TTL}}
  }
}`
)
