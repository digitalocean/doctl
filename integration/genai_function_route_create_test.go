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

var _ = suite("genai/agent/functionroute/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents/0f0e928f-4649-11f0-bf8f-4e013e2ddde4/functions":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(functionRouteCreateResponse))
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
		it("creates a function route", func() {
			aliases := []string{"create", "c"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"agent",
					"functionroute",
					alias,
					"--agent-id", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
					"--name", "get-weather",
					"--description", "Creates a weather-lookup route",
					"--faas-name", "default/testing",
					"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
					"--input-schema", `{"parameters":[{"name":"zipCode","in":"query","schema":{"type":"string"},"required":false,"description":"Zip description in input"},{"name":"measurement","in":"query","schema":{"type":"string","enum":["F","C"]},"required":false,"description":"Temperature unit (F or C)"}]}`,
					"--output-schema", `{"properties":{"temperature":{"type":"number","description":"Temperature for the specified location"},"measurement":{"type":"string","description":"Unit used (F or C)"},"conditions":{"type":"string","description":"Weather conditions (Sunny, Cloudy, etc.)"}}}`,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(functionRouteCreateOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("functionroute alias is used", func() {
		it("creates a function route with fr alias", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"fr", // Testing the "fr" alias
				"create",
				"--agent-id", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--name", "get-weather",
				"--description", "Creates a weather-lookup route",
				"--faas-name", "default/testing",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--input-schema", `{"parameters":[{"name":"zipCode","in":"query","schema":{"type":"string"},"required":false,"description":"Zip description in input"},{"name":"measurement","in":"query","schema":{"type":"string","enum":["F","C"]},"required":false,"description":"Temperature unit (F or C)"}]}`,
				"--output-schema", `{"properties":{"temperature":{"type":"number","description":"Temperature for the specified location"},"measurement":{"type":"string","description":"Unit used (F or C)"},"conditions":{"type":"string","description":"Weather conditions (Sunny, Cloudy, etc.)"}}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(functionRouteCreateOutput), strings.TrimSpace(string(output)))
		})
	})

	when("required flags are missing", func() {
		it("returns an error when agent-id is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"create",
				"--name", "get-weather",
				"--description", "Creates a weather-lookup route",
				"--faas-name", "default/testing",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--input-schema", `{"parameters":[]}`,
				"--output-schema", `{"properties":{}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})

		it("returns an error when name is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"create",
				"--agent-id", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--description", "Creates a weather-lookup route",
				"--faas-name", "default/testing",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--input-schema", `{"parameters":[]}`,
				"--output-schema", `{"properties":{}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})

		it("returns an error when description is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"create",
				"--agent-id", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--name", "get-weather",
				"--faas-name", "default/testing",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--input-schema", `{"parameters":[]}`,
				"--output-schema", `{"properties":{}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})

		it("returns an error when faas-name is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"create",
				"--agent-id", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--name", "get-weather",
				"--description", "Creates a weather-lookup route",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--input-schema", `{"parameters":[]}`,
				"--output-schema", `{"properties":{}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})

		it("returns an error when faas-namespace is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"create",
				"--agent-id", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--name", "get-weather",
				"--description", "Creates a weather-lookup route",
				"--faas-name", "default/testing",
				"--input-schema", `{"parameters":[]}`,
				"--output-schema", `{"properties":{}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})

		it("returns an error when input-schema is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"create",
				"--agent-id", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--name", "get-weather",
				"--description", "Creates a weather-lookup route",
				"--faas-name", "default/testing",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--output-schema", `{"properties":{}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})

		it("returns an error when output-schema is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"create",
				"--agent-id", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--name", "get-weather",
				"--description", "Creates a weather-lookup route",
				"--faas-name", "default/testing",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--input-schema", `{"parameters":[]}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})
	})

	when("invalid JSON schemas are provided", func() {
		it("returns an error when input-schema is invalid JSON", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"create",
				"--agent-id", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--name", "get-weather",
				"--description", "Creates a weather-lookup route",
				"--faas-name", "default/testing",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--input-schema", `{"parameters":[invalid json}`,
				"--output-schema", `{"properties":{}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "input_schema must be valid JSON")
		})

		it("returns an error when output-schema is invalid JSON", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"create",
				"--agent-id", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--name", "get-weather",
				"--description", "Creates a weather-lookup route",
				"--faas-name", "default/testing",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--input-schema", `{"parameters":[]}`,
				"--output-schema", `{"properties":invalid json}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "output_schema must be valid JSON")
		})
	})
})

const (
	functionRouteCreateOutput = `
ID                                      Name           Region    Project ID                              Model ID                                Created At                       User ID
0f0e928f-4649-11f0-bf8f-4e013e2ddde4    get-weather    tor1      00000000-0000-4000-8000-000000000000    00000000-0000-4000-8000-000000000000    2023-01-01 00:00:00 +0000 UTC    user1
`
	functionRouteCreateResponse = `
{
  "agent": {
    "uuid": "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
    "name": "get-weather",
    "region": "tor1",
    "project_id": "00000000-0000-4000-8000-000000000000",
    "model": {
      "uuid": "00000000-0000-4000-8000-000000000000"
    },
    "instruction": "You are a weather assistant",
    "description": "Creates a weather-lookup route",
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z",
    "user_id": "user1",
    "retrieval_method": "RETRIEVAL_METHOD_UNKNOWN",
    "function_routes": [
      {
        "uuid": "e40dc785-5e69-11f0-bf8f-4e013e2ddde4",
        "function_name": "get-weather",
        "description": "Creates a weather-lookup route",
        "faas_name": "default/testing",
        "faas_namespace": "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
        "input_schema": {
          "parameters": [
            {
              "name": "zipCode",
              "in": "query",
              "schema": {
                "type": "string"
              },
              "required": false,
              "description": "Zip description in input"
            },
            {
              "name": "measurement",
              "in": "query",
              "schema": {
                "type": "string",
                "enum": ["F", "C"]
              },
              "required": false,
              "description": "Temperature unit (F or C)"
            }
          ]
        },
        "output_schema": {
          "properties": {
            "temperature": {
              "type": "number",
              "description": "Temperature for the specified location"
            },
            "measurement": {
              "type": "string",
              "description": "Unit used (F or C)"
            },
            "conditions": {
              "type": "string",
              "description": "Weather conditions (Sunny, Cloudy, etc.)"
            }
          }
        },
        "created_at": "2023-01-01T00:00:00Z",
        "updated_at": "2023-01-01T00:00:00Z"
      }
    ]
  }
}
`
)
