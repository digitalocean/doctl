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

var _ = suite("genai/list-models", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/models":
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
				w.Write([]byte(modelsListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("required flags are passed", func() {
		it("lists all models", func() {
			aliases := []string{"list-models", "models", "lm"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					alias,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(modelsListOutput), strings.TrimSpace(string(output)))
			}
		})
	})
})

const (
	modelsListOutput = `
ID         Name            Agreement                   Created At                       Updated At                       Is Foundational    Parent ID    Upload Complete    URL                                             Version
model-1    GPT-4 Turbo     OpenAI Terms of Service     2024-05-01 00:00:00 +0000 UTC    2024-06-01 00:00:00 +0000 UTC    true                            true               https://api.openai.com/v1/models/gpt-4-turbo    4.0.0
model-2    Claude 3.5      Anthropic Service Terms     2024-05-15 00:00:00 +0000 UTC    2024-06-15 00:00:00 +0000 UTC    true                            true               https://api.anthropic.com/v1/models/claude-3    3.5.0
model-3    Custom Model    DigitalOcean GenAI Terms    2024-06-20 00:00:00 +0000 UTC    2024-06-24 00:00:00 +0000 UTC    false              model-1      false                                                              1.0.0
`
	modelsListResponse = `
{
  "models": [
    {
      "uuid": "model-1",
      "name": "GPT-4 Turbo",
      "agreement": {
        "name": "OpenAI Terms of Service",
        "description": "Standard OpenAI API terms and conditions"
      },
      "created_at": "2024-05-01T00:00:00Z",
      "updated_at": "2024-06-01T00:00:00Z",
      "is_foundational": true,
      "inference_name": "gpt-4-turbo",
      "inference_version": "2024-04-09",
      "upload_complete": true,
      "url": "https://api.openai.com/v1/models/gpt-4-turbo",
      "usecases": ["text-generation", "chat"],
      "version": {
        "major": 4,
        "minor": 0,
        "patch": 0
      }
    },
    {
      "uuid": "model-2",
      "name": "Claude 3.5",
      "agreement": {
        "name": "Anthropic Service Terms",
        "description": "Anthropic API service agreement"
      },
      "created_at": "2024-05-15T00:00:00Z",
      "updated_at": "2024-06-15T00:00:00Z",
      "is_foundational": true,
      "inference_name": "claude-3-5-sonnet",
      "inference_version": "20240620",
      "upload_complete": true,
      "url": "https://api.anthropic.com/v1/models/claude-3",
      "usecases": ["text-generation", "analysis", "coding"],
      "version": {
        "major": 3,
        "minor": 5,
        "patch": 0
      }
    },
    {
      "uuid": "model-3",
      "name": "Custom Model",
      "agreement": {
        "name": "DigitalOcean GenAI Terms",
        "description": "DigitalOcean custom model agreement"
      },
      "created_at": "2024-06-20T00:00:00Z",
      "updated_at": "2024-06-24T00:00:00Z",
      "is_foundational": false,
      "parent_uuid": "model-1",
      "inference_name": "custom-gpt-4",
      "upload_complete": false,
      "usecases": ["domain-specific"],
      "version": {
        "major": 1,
        "minor": 0,
        "patch": 0
      }
    }
  ],
  "links": {},
  "meta": {
    "total": 3
  }
}
`
)
