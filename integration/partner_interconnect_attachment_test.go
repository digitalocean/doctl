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

var (
	partnerAttachmentCreateResponse = `
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

	partnerAttachmentCreateOutput = `
ID       Name         State     Connection Bandwidth (MBPS)    Region    NaaS Provider    VPC IDs                                 Created At                       BGP Local ASN    BGP Local Router IP    BGP Peer ASN    BGP Peer Router IP
12345    doctl-pia    active    50                             stage2    MEGAPORT         d35e5cb7-7957-4643-8e3a-1ab4eb3a494c    2025-01-30 12:00:00 +0000 UTC    0                                       0
`

	interconnectListRoutesOutput = `
ID                                      Cidr
a0eb6eb0-fa38-41a8-a5de-1a75524667fe    169.250.0.0/29
`

	interconnectListRoutesResponse = `
{
  "remote_routes": [
	{"id": "a0eb6eb0-fa38-41a8-a5de-1a75524667fe", "cidr": "169.250.0.0/29"}
  ],
  "links": {
    "pages": {
      "last": "http://localhost/v2/partner_interconnect/attachments?page=1&per_page=1",
      "next": "http://localhost/v2/partner_interconnect/attachments?page=2&per_page=1"
    }
  },
  "links": {
    "pages": {}
  },
  "meta": {
    "total": 1
  }
}
`
	interconnectGetServiceKeyOutput = `
Value               State     CreatedAt
test-service-key    active    2025-01-30 12:00:00 +0000 UTC	
	`

	interconnectGetServiceKeyResponse = `
{
	"service_key": {
		"created_at": "2025-01-30T12:00:00Z",
		"value": "test-service-key",
		"state": "active"
	}
}`
)

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

var _ = suite("partner_interconnect_attachments/list-routes", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/partner_interconnect/attachments/c5537207-ebf0-47cb-bc10-6fac717cd672/remote_routes":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(interconnectListRoutesResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("no flags are passed", func() {
		it("gets the specified VPC", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"network",
				"interconnect-attachment",
				"list-routes",
				"c5537207-ebf0-47cb-bc10-6fac717cd672",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(interconnectListRoutesOutput), strings.TrimSpace(string(output)))
		})
	})

	when("format and no-header flags are passed", func() {
		it("gets the specified VPC", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"network",
				"interconnect-attachment",
				"list-routes",
				"--format", "Cidr",
				"--no-header",
				"c5537207-ebf0-47cb-bc10-6fac717cd672",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal("169.250.0.0/29", strings.TrimSpace(string(output)))
		})
	})
})

var _ = suite("partner_interconnect_attachments/get-service-key", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/partner_interconnect/attachments/c5537207-ebf0-47cb-bc10-6fac717cd672/service_key":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(interconnectGetServiceKeyResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("no flags are passed", func() {
		it("gets the specified service key", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"network",
				"interconnect-attachment",
				"get-service-key",
				"c5537207-ebf0-47cb-bc10-6fac717cd672",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(interconnectGetServiceKeyOutput), strings.TrimSpace(string(output)))
		})
	})

	when("format and no-header flags are passed", func() {
		it("gets the specified service key", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"network",
				"interconnect-attachment",
				"get-service-key",
				"--format", "Value",
				"--no-header",
				"c5537207-ebf0-47cb-bc10-6fac717cd672",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal("test-service-key", strings.TrimSpace(string(output)))
		})
	})
})
