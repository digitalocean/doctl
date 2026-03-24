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

var _ = suite("dedicated-inference/get-gpu-model-config", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/dedicated-inferences/gpu-model-config":
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
				w.Write([]byte(dedicatedInferenceGetGPUModelConfigResponse))
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
		it("lists GPU model configurations", func() {
			aliases := []string{"get-gpu-model-config", "ggmc"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"dedicated-inference",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output for alias %q: %s", alias, output))
				expect.Equal(strings.TrimSpace(dedicatedInferenceGetGPUModelConfigOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("passing a format flag", func() {
		it("displays only those columns", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"get-gpu-model-config",
				"--format", "ModelSlug,IsModelGated",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceGetGPUModelConfigFormatOutput), strings.TrimSpace(string(output)))
		})
	})

	when("using the di alias", func() {
		it("lists GPU model configurations", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"di",
				"get-gpu-model-config",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceGetGPUModelConfigOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dedicatedInferenceGetGPUModelConfigOutput = `
Model Slug                        Model Name                Gated    GPU Slugs
mistral/mistral-7b-instruct-v3    Mistral 7B Instruct v3    false    gpu-mi300x1-192gb,gpu-h100x1-80gb
meta-llama/llama-3-70b            Llama 3 70B               true     gpu-mi300x1-192gb
`
	dedicatedInferenceGetGPUModelConfigFormatOutput = `
Model Slug                        Gated
mistral/mistral-7b-instruct-v3    false
meta-llama/llama-3-70b            true
`

	dedicatedInferenceGetGPUModelConfigResponse = `
{
  "gpu_model_configs": [
    {
      "model_slug": "mistral/mistral-7b-instruct-v3",
      "model_name": "Mistral 7B Instruct v3",
      "is_model_gated": false,
      "gpu_slugs": ["gpu-mi300x1-192gb", "gpu-h100x1-80gb"]
    },
    {
      "model_slug": "meta-llama/llama-3-70b",
      "model_name": "Llama 3 70B",
      "is_model_gated": true,
      "gpu_slugs": ["gpu-mi300x1-192gb"]
    }
  ]
}
`
)
