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

var _ = suite("dedicated-inference/get-sizes", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/dedicated-inferences/sizes":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(dedicatedInferenceGetSizesResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is invoked", func() {
		it("lists available sizes", func() {
			aliases := []string{"get-sizes", "gs"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"dedicated-inference",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output for alias %q: %s", alias, output))
				expect.Equal(strings.TrimSpace(dedicatedInferenceGetSizesOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("passing a format flag", func() {
		it("displays only those columns", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"get-sizes",
				"--format", "GPUSlug,PricePerHour,Regions",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceGetSizesFormatOutput), strings.TrimSpace(string(output)))
		})
	})

	when("using the di alias", func() {
		it("lists available sizes", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"di",
				"get-sizes",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceGetSizesOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dedicatedInferenceGetSizesResponse = `
{
  "enabled_regions": ["nyc2", "sfo3"],
  "sizes": [
    {
      "gpu_slug": "gpu-mi300x1-192gb",
      "price_per_hour": "3.59",
      "regions": ["nyc2", "sfo3"],
      "currency": "USD",
      "cpu": 24,
      "memory": 98304,
      "gpu": {
        "count": 1,
        "vram_gb": 192,
        "slug": "mi300x"
      },
      "size_category": {
        "name": "GPU Optimized",
        "fleet_name": "gpu-mi300x"
      },
      "disks": [
        {
          "type": "local",
          "size_gb": 960
        }
      ]
    },
    {
      "gpu_slug": "gpu-h100x1-80gb",
      "price_per_hour": "4.25",
      "regions": ["nyc2"],
      "currency": "USD",
      "cpu": 16,
      "memory": 65536,
      "gpu": {
        "count": 1,
        "vram_gb": 80,
        "slug": "h100"
      },
      "size_category": {
        "name": "GPU Optimized",
        "fleet_name": "gpu-h100"
      },
      "disks": [
        {
          "type": "local",
          "size_gb": 480
        }
      ]
    }
  ]
}
`

	// NOTE: Column spacing must exactly match doctl's table formatter.
	dedicatedInferenceGetSizesOutput = `
GPU Slug             Price/Hour    Currency    CPU    Memory (MB)    GPU Count    GPU VRAM (GB)    GPU Model    Regions
gpu-mi300x1-192gb    3.59 USD      USD         24     98304          1            192              mi300x       nyc2,sfo3
gpu-h100x1-80gb      4.25 USD      USD         16     65536          1            80               h100         nyc2
`
	dedicatedInferenceGetSizesFormatOutput = `
GPU Slug             Price/Hour    Regions
gpu-mi300x1-192gb    3.59 USD      nyc2,sfo3
gpu-h100x1-80gb      4.25 USD      nyc2
`
)
