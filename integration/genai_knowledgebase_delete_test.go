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

var _ = suite("genai/knowledge-base/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/knowledge_bases/00000000-0000-4000-8000-000000000000":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusNoContent)
				w.Write([]byte(`{"message":"Knowledge Base deleted successfully"}`))
			case "/v2/gen-ai/knowledge_bases/99999999-9999-4999-8999-999999999999":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
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

	when("valid knowledge base ID is provided with force flag", func() {
		it("deletes the knowledge-base", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"delete",
				"00000000-0000-4000-8000-000000000000",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "deleted successfully")
		})
	})

	when("Knowledge base ID is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"delete",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing")
		})
	})

	when("Knowledge base does not exist", func() {
		it("returns a not found error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"delete",
				"99999999-9999-4999-8999-999999999999",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "The resource you requested could not be found")
		})
	})

	when("force flag is not provided", func() {
		it("prompts for confirmation and aborts", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"delete",
				"00000000-0000-4000-8000-000000000000",
			)

			// Since we can't easily provide interactive input, the command should abort
			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "operation aborted")
		})
	})

	when("using short flag for force", func() {
		it("deletes the Knowledge base with -f flag", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"delete",
				"00000000-0000-4000-8000-000000000000",
				"-f",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "Knowledge Base deleted successfully")
		})
	})

	when("network connectivity issues", func() {
		it("handles network errors", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", "http://nonexistent-server.example.com",
				"genai",
				"knowledge-base",
				"delete",
				"00000000-0000-4000-8000-000000000000",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "no such host")
		})
	})
})

var _ = suite("genai/knowledge-base/delete-datasource", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/knowledge_bases/00000000-0000-4000-8000-000000000000/data_sources/00000000-0000-4000-8000-000000000000":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			case "/v2/gen-ai/knowledge_bases/99999999-9999-4999-8999-999999999999/data_sources/99999999-9999-4999-8999-999999999999":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
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

	when("valid knowledge bases and data source ID is provided with force flag", func() {
		it("deletes the data source from a knowledge base", func() {
			aliases := []string{"delete-datasource", "d-ds"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"genai",
					"knowledge-base",
					alias,
					"00000000-0000-4000-8000-000000000000",
					"00000000-0000-4000-8000-000000000000",
					"--force",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output: %s", output))
				expect.Contains(string(output), "DataSource of Knowledge Base deleted successfully")
			}
		})
	})

	when("data source ID is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"delete-datasource",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing")
		})
	})

	when("agent does not exist", func() {
		it("returns a not found error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"delete-datasource",
				"99999999-9999-4999-8999-999999999999",
				"99999999-9999-4999-8999-999999999999",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "The resource you requested could not be found")
		})
	})

	when("force flag is not provided", func() {
		it("prompts for confirmation and aborts", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"delete-datasource",
				"00000000-0000-4000-8000-000000000000",
				"00000000-0000-4000-8000-000000000000",
			)

			// Since we can't easily provide interactive input, the command should abort
			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "operation aborted")
		})
	})

	when("using short flag for force", func() {
		it("deletes the agent with -f flag", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"knowledge-base",
				"delete-datasource",
				"00000000-0000-4000-8000-000000000000",
				"00000000-0000-4000-8000-000000000000",
				"-f",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Contains(string(output), "DataSource of Knowledge Base deleted successfully")
		})
	})

	when("network connectivity issues", func() {
		it("handles network errors", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", "http://nonexistent-server.example.com",
				"genai",
				"knowledge-base",
				"delete-datasource",
				"00000000-0000-4000-8000-000000000000",
				"--force",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "error")
		})
	})
})
