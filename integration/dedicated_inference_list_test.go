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

var _ = suite("dedicated-inference/list", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/dedicated-inferences":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				regionFilter := req.URL.Query().Get("region")
				nameFilter := req.URL.Query().Get("name")

				if regionFilter == "nyc2" {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(dedicatedInferenceListFilteredByRegionResponse))
					return
				}

				if nameFilter == "test-di-1" {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(dedicatedInferenceListFilteredByNameResponse))
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(dedicatedInferenceListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("no filters are provided", func() {
		it("lists all dedicated inference endpoints", func() {
			aliases := []string{"list", "ls"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"dedicated-inference",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output for alias %q: %s", alias, output))
				expect.Equal(strings.TrimSpace(dedicatedInferenceListOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("region filter is provided", func() {
		it("lists only endpoints in that region", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"list",
				"--region", "nyc2",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceListFilteredByRegionOutput), strings.TrimSpace(string(output)))
		})
	})

	when("name filter is provided", func() {
		it("lists only endpoints with that name", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"list",
				"--name", "test-di-1",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceListFilteredByNameOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a format flag", func() {
		it("displays only those columns", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"list",
				"--format", "ID,Name,Status",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceListFormatOutput), strings.TrimSpace(string(output)))
		})
	})

	when("using the di alias", func() {
		it("lists all dedicated inference endpoints", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"di",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceListOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dedicatedInferenceListOutput = `
ID                                      Name         Region    Status          VPC UUID                                Public Endpoint            Private Endpoint            Created At                       Updated At
00000000-0000-4000-8000-000000000000    test-di-1    nyc2      ACTIVE          00000000-0000-4000-8000-000000000001    public.di-1.example.com    private.di-1.example.com    2023-01-01 00:00:00 +0000 UTC    2023-01-01 00:00:00 +0000 UTC
11111111-1111-4111-8111-111111111111    test-di-2    sfo3      PROVISIONING    11111111-1111-4111-8111-111111111112    public.di-2.example.com    private.di-2.example.com    2023-01-02 00:00:00 +0000 UTC    2023-01-02 00:00:00 +0000 UTC
`
	dedicatedInferenceListFilteredByRegionOutput = `
ID                                      Name         Region    Status    VPC UUID                                Public Endpoint            Private Endpoint            Created At                       Updated At
00000000-0000-4000-8000-000000000000    test-di-1    nyc2      ACTIVE    00000000-0000-4000-8000-000000000001    public.di-1.example.com    private.di-1.example.com    2023-01-01 00:00:00 +0000 UTC    2023-01-01 00:00:00 +0000 UTC
`
	dedicatedInferenceListFilteredByNameOutput = `
ID                                      Name         Region    Status    VPC UUID                                Public Endpoint            Private Endpoint            Created At                       Updated At
00000000-0000-4000-8000-000000000000    test-di-1    nyc2      ACTIVE    00000000-0000-4000-8000-000000000001    public.di-1.example.com    private.di-1.example.com    2023-01-01 00:00:00 +0000 UTC    2023-01-01 00:00:00 +0000 UTC
`
	dedicatedInferenceListFormatOutput = `
ID                                      Name         Status
00000000-0000-4000-8000-000000000000    test-di-1    ACTIVE
11111111-1111-4111-8111-111111111111    test-di-2    PROVISIONING
`

	dedicatedInferenceListResponse = `
{
  "dedicated_inferences": [
    {
      "id": "00000000-0000-4000-8000-000000000000",
      "name": "test-di-1",
      "region": "nyc2",
      "status": "ACTIVE",
	  "provider_model_id": ["mistralai/Mistral-7B-Instruct-v0.3"],
      "vpc_uuid": "00000000-0000-4000-8000-000000000001",
      "endpoints": {
        "public_endpoint_fqdn": "public.di-1.example.com",
        "private_endpoint_fqdn": "private.di-1.example.com"
      },
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    },
    {
      "id": "11111111-1111-4111-8111-111111111111",
      "name": "test-di-2",
      "region": "sfo3",
      "status": "PROVISIONING",
	  "provider_model_id": ["meta-llama/Meta-Llama-3-8B-Instruct"],
      "vpc_uuid": "11111111-1111-4111-8111-111111111112",
      "endpoints": {
        "public_endpoint_fqdn": "public.di-2.example.com",
        "private_endpoint_fqdn": "private.di-2.example.com"
      },
      "created_at": "2023-01-02T00:00:00Z",
      "updated_at": "2023-01-02T00:00:00Z"
    }
  ],
  "links": {},
  "meta": {
    "total": 2
  }
}
`
	dedicatedInferenceListFilteredByRegionResponse = `
{
  "dedicated_inferences": [
    {
      "id": "00000000-0000-4000-8000-000000000000",
      "name": "test-di-1",
      "region": "nyc2",
      "status": "ACTIVE",
	  "provider_model_id": ["mistralai/Mistral-7B-Instruct-v0.3"],
      "vpc_uuid": "00000000-0000-4000-8000-000000000001",
      "endpoints": {
        "public_endpoint_fqdn": "public.di-1.example.com",
        "private_endpoint_fqdn": "private.di-1.example.com"
      },
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    }
  ],
  "links": {},
  "meta": {
    "total": 1
  }
}
`
	dedicatedInferenceListFilteredByNameResponse = `
{
  "dedicated_inferences": [
    {
      "id": "00000000-0000-4000-8000-000000000000",
      "name": "test-di-1",
      "region": "nyc2",
      "status": "ACTIVE",
	  "provider_model_id": ["mistralai/Mistral-7B-Instruct-v0.3"],
      "vpc_uuid": "00000000-0000-4000-8000-000000000001",
      "endpoints": {
        "public_endpoint_fqdn": "public.di-1.example.com",
        "private_endpoint_fqdn": "private.di-1.example.com"
      },
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    }
  ],
  "links": {},
  "meta": {
    "total": 1
  }
}
`
)
