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
  "partner_attachment": {
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
ID       Name         State     Connection Bandwidth (MBPS)    Region    NaaS Provider    VPC IDs                                 Created At                       BGP Local ASN    BGP Local Router IP    BGP Peer ASN    BGP Peer Router IP    Redundancy Zone    Parent UUID    Children UUIDs
12345    doctl-pia    active    50                             stage2    MEGAPORT         d35e5cb7-7957-4643-8e3a-1ab4eb3a494c    2025-01-30 12:00:00 +0000 UTC    0                                       0
`

	paListRoutesOutput = `
ID                                      Cidr
a0eb6eb0-fa38-41a8-a5de-1a75524667fe    169.250.0.0/29
`

	paListRoutesResponse = `
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

	paRegenerateServiceKeyOutput   = ``
	paRegenerateServiceKeyResponse = `{}`

	paGetBgpAuthKeyOutput = `
Value
test-bgp-auth-key
	`
	paGetBgpAuthKeyResponse = `
{
	"bgp_auth_key": {
		"value": "test-bgp-auth-key"
	}
}`
	paGetServiceKeyOutput = `
Value               State     CreatedAt
test-service-key    active    2025-01-30 12:00:00 +0000 UTC	
	`

	paGetServiceKeyResponse = `
{
	"service_key": {
		"created_at": "2025-01-30T12:00:00Z",
		"value": "test-service-key",
		"state": "active"
	}
}`

	// Knowledge Base Operations
	kbListDataSourcesOutput = `
ID                                      Name           Type    Status    Created At
a0eb6eb0-fa38-41a8-a5de-1a75524667fe    docs-source    file    active    2025-01-30 12:00:00 +0000 UTC
b1fc7fc1-gb49-52b9-b6ef-2b86535778gf    api-source     url     pending   2025-01-30 12:05:00 +0000 UTC
`

	kbListDataSourcesResponse = `
{
  "data_sources": [
    {
      "id": "a0eb6eb0-fa38-41a8-a5de-1a75524667fe",
      "name": "docs-source",
      "type": "file",
      "status": "active",
      "created_at": "2025-01-30T12:00:00Z"
    },
    {
      "id": "b1fc7fc1-gb49-52b9-b6ef-2b86535778gf",
      "name": "api-source", 
      "type": "url",
      "status": "pending",
      "created_at": "2025-01-30T12:05:00Z"
    }
  ],
  "meta": {
    "total": 2
  }
}`

	kbGetIndexingJobStatusOutput = `
ID                                      Status       Progress    Started At                       Completed At                     Error
c2gd8gd2-hc60-63c0-c7fg-3c97646889hg    completed    100         2025-01-30 12:00:00 +0000 UTC   2025-01-30 12:15:00 +0000 UTC   
`

	kbGetIndexingJobStatusResponse = `
{
  "indexing_job": {
    "id": "c2gd8gd2-hc60-63c0-c7fg-3c97646889hg",
    "status": "completed",
    "progress": 100,
    "started_at": "2025-01-30T12:00:00Z",
    "completed_at": "2025-01-30T12:15:00Z",
    "error": ""
  }
}`

	kbCancelIndexingJobOutput = `
ID                                      Status       Progress    Started At                       Completed At    Error
c2gd8gd2-hc60-63c0-c7fg-3c97646889hg    cancelled    50          2025-01-30 12:00:00 +0000 UTC                   
`

	kbCancelIndexingJobResponse = `
{
  "indexing_job": {
    "id": "c2gd8gd2-hc60-63c0-c7fg-3c97646889hg", 
    "status": "cancelled",
    "progress": 50,
    "started_at": "2025-01-30T12:00:00Z",
    "completed_at": null,
    "error": ""
  }
}`
)

var _ = suite("partner_network_connect/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/partner_network_connect/attachments":
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
				"attachment",
				"create",
				"--name", "doctl-pia",
				"--connection-bandwidth-in-mbps", "50",
				"--naas-provider", "MEGAPORT",
				"--region", "stage2",
				"--vpc-ids", "d35e5cb7-7957-4643-8e3a-1ab4eb3a494c",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			t.Log("printing output ", string(output))
			t.Log("printing partnerAttachmentCreateOutput ", string(partnerAttachmentCreateOutput))
			expect.Equal(strings.TrimSpace(partnerAttachmentCreateOutput), strings.TrimSpace(string(output)))
		})
	})
})

var _ = suite("partner_network_connect/list-routes", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/partner_network_connect/attachments/c5537207-ebf0-47cb-bc10-6fac717cd672/remote_routes":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(paListRoutesResponse))
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
				"attachment",
				"list-routes",
				"c5537207-ebf0-47cb-bc10-6fac717cd672",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(paListRoutesOutput), strings.TrimSpace(string(output)))
		})
	})

	when("format and no-header flags are passed", func() {
		it("gets the specified VPC", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"network",
				"attachment",
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

var _ = suite("partner_network_connect/regenerate-service-key", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/partner_network_connect/attachments/c5537207-ebf0-47cb-bc10-6fac717cd672/service_key":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(paRegenerateServiceKeyResponse))
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
				"attachment",
				"regenerate-service-key",
				"c5537207-ebf0-47cb-bc10-6fac717cd672",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(paRegenerateServiceKeyOutput), strings.TrimSpace(string(output)))
		})
	})
})

var _ = suite("partner_network_connect/get-bgp-auth-key", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/partner_network_connect/attachments/c5537207-ebf0-47cb-bc10-6fac717cd672/bgp_auth_key":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(paGetBgpAuthKeyResponse))
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
		it("gets the specified bgp auth key", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"network",
				"attachment",
				"get-bgp-auth-key",
				"c5537207-ebf0-47cb-bc10-6fac717cd672",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(paGetBgpAuthKeyOutput), strings.TrimSpace(string(output)))
		})
	})

	when("format and no-header flags are passed", func() {
		it("gets the specified bgp auth key", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"network",
				"attachment",
				"get-bgp-auth-key",
				"--format", "Value",
				"--no-header",
				"c5537207-ebf0-47cb-bc10-6fac717cd672",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal("test-bgp-auth-key", strings.TrimSpace(string(output)))
		})
	})
})

var _ = suite("partner_network_connect/get-service-key", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/partner_network_connect/attachments/c5537207-ebf0-47cb-bc10-6fac717cd672/service_key":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(paGetServiceKeyResponse))
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
				"attachment",
				"get-service-key",
				"c5537207-ebf0-47cb-bc10-6fac717cd672",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(paGetServiceKeyOutput), strings.TrimSpace(string(output)))
		})
	})

	when("format and no-header flags are passed", func() {
		it("gets the specified service key", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"network",
				"attachment",
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

var _ = suite("knowledge_base_operations/list-data-sources", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/knowledge_base/data_sources":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(kbListDataSourcesResponse))
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
		it("lists all data sources", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"knowledge_base",
				"data_sources",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(kbListDataSourcesOutput), strings.TrimSpace(string(output)))
		})
	})

	when("format and no-header flags are passed", func() {
		it("lists all data sources", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"knowledge_base",
				"data_sources",
				"list",
				"--format", "Name,Type",
				"--no-header",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal("docs-source file\napi-source url", strings.TrimSpace(string(output)))
		})
	})
})

var _ = suite("knowledge_base_operations/get-indexing-job-status", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/knowledge_base/indexing_jobs/c2gd8gd2-hc60-63c0-c7fg-3c97646889hg":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(kbGetIndexingJobStatusResponse))
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
		it("gets the specified indexing job status", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"knowledge_base",
				"indexing_jobs",
				"get-status",
				"c2gd8gd2-hc60-63c0-c7fg-3c97646889hg",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(kbGetIndexingJobStatusOutput), strings.TrimSpace(string(output)))
		})
	})
})

var _ = suite("knowledge_base_operations/cancel-indexing-job", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/knowledge_base/indexing_jobs/c2gd8gd2-hc60-63c0-c7fg-3c97646889hg/cancel":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(kbCancelIndexingJobResponse))
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
		it("cancels the specified indexing job", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"knowledge_base",
				"indexing_jobs",
				"cancel",
				"c2gd8gd2-hc60-63c0-c7fg-3c97646889hg",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(kbCancelIndexingJobOutput), strings.TrimSpace(string(output)))
		})
	})
})
