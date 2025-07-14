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

var _ = suite("genai/agent/functionroute/update", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/agents/0f0e928f-4649-11f0-bf8f-4e013e2ddde4/functions/e40dc785-5e69-11f0-bf8f-4e013e2ddde4":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPut {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(functionRouteUpdateResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it.After(func() {
		server.Close()
	})

	when("all update flags are passed", func() {
		it("updates a function route", func() {
			aliases := []string{"update", "u"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"agent",
					"functionroute",
					alias,
					"--agentid", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
					"--functionid", "e40dc785-5e69-11f0-bf8f-4e013e2ddde4",
					"--name", "terraform-testing-1",
					"--description", "Creating via doctl again",
					"--faas-name", "default/testing",
					"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
					"--input-schema", `{"parameters":[{"name":"zipCode","in":"query","schema":{"type":"string"},"required":false,"description":"Zip description in input"},{"name":"measurement","in":"query","schema":{"type":"string","enum":["F","C"]},"required":false,"description":"Temperature unit (F or C)"}]}`,
					"--output-schema", `{"properties":{"temperature":{"type":"number","description":"Temperature for the specified location"},"measurement":{"type":"string","description":"Unit used (F or C)"},"conditions":{"type":"string","description":"Weather conditions (Sunny, Cloudy, etc.)"}}}`,
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Equal(strings.TrimSpace(functionRouteUpdateOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("required flags are missing", func() {
		it("returns an error when agentid is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"update",
				"--functionid", "e40dc785-5e69-11f0-bf8f-4e013e2ddde4",
				"--name", "terraform-testing-1",
				"--description", "Creating via doctl again",
				"--faas-name", "default/testing",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--input-schema", `{"parameters":[]}`,
				"--output-schema", `{"properties":{}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})

		it("returns an error when functionid is missing", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"update",
				"--agentid", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--name", "terraform-testing-1",
				"--description", "Creating via doctl again",
				"--faas-name", "default/testing",
				"--faas-namespace", "fn-b90faf52-2b42-49c2-9792-75edfbb6f397",
				"--input-schema", `{"parameters":[]}`,
				"--output-schema", `{"properties":{}}`,
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing required arguments")
		})

		it("returns an error when no update fields are provided", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"agent",
				"functionroute",
				"update",
				"--agentid", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--functionid", "e40dc785-5e69-11f0-bf8f-4e013e2ddde4",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "at least one field to update must be supplied")
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
				"update",
				"--agentid", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--functionid", "e40dc785-5e69-11f0-bf8f-4e013e2ddde4",
				"--name", "terraform-testing-1",
				"--description", "Creating via doctl again",
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
				"update",
				"--agentid", "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
				"--functionid", "e40dc785-5e69-11f0-bf8f-4e013e2ddde4",
				"--name", "terraform-testing-1",
				"--description", "Creating via doctl again",
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
	functionRouteUpdateOutput = `
ID                                      Name                   Region    Project ID                              Model ID    Created At                       User ID
0f0e928f-4649-11f0-bf8f-4e013e2ddde4    terraform-testing-1    tor1      84e1e297-ee40-41ac-95ff-1067cf2206e9                2025-06-10 22:20:04 +0000 UTC    18697494
`
	functionRouteUpdateResponse = `
{
  "agent": {
    "uuid": "0f0e928f-4649-11f0-bf8f-4e013e2ddde4",
    "name": "terraform-testing-1",
    "region": "tor1",
    "project_id": "84e1e297-ee40-41ac-95ff-1067cf2206e9",
    "model": {
      "uuid": ""
    },
    "instruction": "You are a weather assistant",
    "description": "Creating via doctl again",
    "created_at": "2025-06-10T22:20:04Z",
    "updated_at": "2025-06-10T22:20:04Z",
    "user_id": "18697494",
    "retrieval_method": "RETRIEVAL_METHOD_UNKNOWN",
    "function_routes": [
      {
        "uuid": "e40dc785-5e69-11f0-bf8f-4e013e2ddde4",
        "function_name": "terraform-testing-1",
        "description": "Creating via doctl again",
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
        "created_at": "2025-06-10T22:20:04Z",
        "updated_at": "2025-06-10T22:20:04Z"
      }
    ]
  }
}
`
)
