package integration

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("genai/openai-key/delete", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/openai/keys/00000000-0000-4000-8000-000000000000":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(openAIKeyResponse))
			case "/v2/gen-ai/openai/keys/99999999-9999-4999-8999-999999999999":
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
				w.Write([]byte(`{
					"id": "not_found",
					"message": "failed to get openai api key"
				}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("valid openai key id is passed", func() {
		it("deletes the openai key", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"openai-key",
				"delete",
				"00000000-0000-4000-8000-000000000000",
				"-f",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Contains(strings.TrimSpace(string(output)), "OpenAI API Key deleted successfully")
		})
	})

	when("invalid openai key is passed", func() {
		it("return not found error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"openai-key",
				"delete",
				"99999999-9999-4999-8999-999999999999",
				"-f",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(strings.TrimSpace(string(output)), "failed to get openai api key")
		})
	})
})
