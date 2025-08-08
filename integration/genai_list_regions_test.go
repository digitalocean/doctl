package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("genai/list-regions", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v2/gen-ai/regions":
				auth := r.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if r.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("content-type", "application/json")
				fmt.Fprint(w, genaiListRegionsResponse)
			default:
				dump, err := httputil.DumpRequest(r, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("command is list-regions", func() {
		it("lists available datacenter regions", func() {
			aliases := []string{"list-regions", "lr"}
			for _, alias := range aliases {
				cmd := exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expectedOutput := `Inference URL                   Region    Serves Batch    Serves Inference    Stream Inference URL
https://inference.nyc1.do.ai    nyc1      false           true                https://stream.nyc1.do.ai
https://inference.fra1.do.ai    fra1      true            true                https://stream.fra1.do.ai
`
				expect.Equal(expectedOutput, string(output))
			}
		})
	})

	when("command is list-regions with json format", func() {
		it("lists available datacenter regions in json", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"list-regions",
				"--output", "json",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.JSONEq(genaiListRegionsJSONOutput, string(output))
		})
	})

	it.After(func() {
		if server != nil {
			server.Close()
		}
	})
})

const genaiListRegionsResponse = `{
  "regions": [
    {
      "region": "nyc1",
      "inference_url": "https://inference.nyc1.do.ai",
      "serves_batch": false,
      "serves_inference": true,
      "stream_inference_url": "https://stream.nyc1.do.ai"
    },
    {
      "region": "fra1",
      "inference_url": "https://inference.fra1.do.ai",
      "serves_batch": true,
      "serves_inference": true,
      "stream_inference_url": "https://stream.fra1.do.ai"
    }
  ]
}`

const genaiListRegionsJSONOutput = `[
  {
    "region": "nyc1",
    "inference_url": "https://inference.nyc1.do.ai",
    "serves_batch": false,
    "serves_inference": true,
    "stream_inference_url": "https://stream.nyc1.do.ai"
  },
  {
    "region": "fra1",
    "inference_url": "https://inference.fra1.do.ai",
    "serves_batch": true,
    "serves_inference": true,
    "stream_inference_url": "https://stream.fra1.do.ai"
  }
]`
