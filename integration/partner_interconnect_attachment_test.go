package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
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

var partnerAttachmentCreateResponse = `
{
  "partner_interconnect_attachment": {
    "id": "12345",
    "name": "{{.Name}}",
    "state": "active",
    "connection_bandwidth_in_mbps": {{.ConnectionBandwidthInMbps}},
    "region": "{{.Region}}",
    "naas_provider": "{{.NaaSProvider}}",
    "vpc_ids": ["{{index .VPCIDs 0}}"],
    "created_at": "2025-01-30T12:00:00Z",
    "bgp": {
      "local_asn": 0,
      "local_router_ip": "",
      "peer_asn": 0,
      "peer_router_ip": ""
    }
  }
}
`

var partnerAttachmentCreateOutput = `
ID       Name         State     Connection Bandwidth (MBPS)    Region    NaaS Provider    VPC IDs                                 Created At                       BGP Local ASN    BGP Local Router IP    BGP Peer ASN    BGP Peer Router IP
12345    doctl-pia    active    50                             stage2    MEGAPORT         d35e5cb7-7957-4643-8e3a-1ab4eb3a494c    2025-01-30 12:00:00 +0000 UTC    0                                       0
`

var _ = suite("partner_interconnect_attachments/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/partner_interconnect/attachments":
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

				var request struct {
					Name                      string   `json:"name"`
					ConnectionBandwidthInMbps int      `json:"connection_bandwidth_in_mbps"`
					Region                    string   `json:"region"`
					NaaSProvider              string   `json:"naas_provider"`
					VPCIDs                    []string `json:"vpc_ids"`
				}
				err = json.Unmarshal(reqBody, &request)
				expect.NoError(err)

				t, err := template.New("response").Parse(partnerAttachmentCreateResponse)
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
		it("creates new Partner Interconnect Attachment", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"network",
				"interconnect-attachment",
				"create",
				"--name", "doctl-pia",
				"--connection-bandwidth-in-mbps", "50",
				"--naas-provider", "MEGAPORT",
				"--region", "stage2",
				"--vpc-ids", "d35e5cb7-7957-4643-8e3a-1ab4eb3a494c",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(partnerAttachmentCreateOutput), strings.TrimSpace(string(output)))
		})
	})
})
