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

var _ = suite("gradient/anthropic-key/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/anthropic/keys/00000000-0000-4000-8000-000000000000":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(anthropicKeyResponse))
			case "/v2/gen-ai/anthropic/keys/99999999-9999-4999-8999-999999999999":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"id":"not_found","message":"The resource you requested could not be found."}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("anthropic key id is not passed", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"anthropic-key",
				"get",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "command is missing required arguments")
		})
	})

	when("anthropic key does not exist", func() {
		it("returns a not found error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"anthropic-key",
				"get",
				"99999999-9999-4999-8999-999999999999",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "The resource you requested could not be found.")
		})
	})

	when("valid anthropic key id is passed", func() {
		it("gets the anthropic key", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"anthropic-key",
				"get",
				"00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Contains(strings.TrimSpace(string(anthropicKeyOutput)), strings.TrimSpace(string(output)))
		})
	})
})

var _ = suite("gradient/anthropic-key/get-agents", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/anthropic/keys/00000000-0000-4000-8000-000000000000/agents":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(agentListResponse))
			case "/v2/gen-ai/anthropic/keys/99999999-9999-4999-8999-999999999999/agents":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"id":"not_found","message":"The resource you requested could not be found."}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("anthropic key id is not passed", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"anthropic-key",
				"get-agents",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "command is missing required arguments")
		})
	})

	when("anthropic key does not exist", func() {
		it("returns a not found error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"anthropic-key",
				"get-agents",
				"99999999-9999-4999-8999-999999999999",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "The resource you requested could not be found.")
		})
	})

	when("valid anthropic key id is passed", func() {
		it("gets the agents using the anthropic key", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"anthropic-key",
				"get-agents",
				"00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(strings.TrimSpace(string(agentListOutput)), strings.TrimSpace(string(output)))
		})
	})
})

const (
	anthropicKeyResponse = `
	{
		"api_key_info": {
			"name": "api-key",
			"uuid": "11f066ea-0000-0000-0000-4e013e2ddde4",
			"created_by": "18919793",
			"created_at": "2025-07-22T10:57:42Z",
			"updated_at": "2025-07-22T10:57:42Z"
		}
	}`

	anthropicKeyOutput = `
Name       UUID                                    Created At                       Created By    Updated At                       Deleted At
api-key    11f066ea-0000-0000-0000-4e013e2ddde4    2025-07-22 10:57:42 +0000 UTC    18919793      2025-07-22 10:57:42 +0000 UTC    <nil>`
)
