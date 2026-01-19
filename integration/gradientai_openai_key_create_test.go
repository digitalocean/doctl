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

var _ = suite("gradient/openai-key/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/openai/keys":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(openAIKeyResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("valid openai api key and name is passed", func() {
		it("creates the openai key", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"openai-key",
				"create",
				"--name", "api-key",
				"--api-key", "sk-prodajkbcsdovub",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Contains(strings.TrimSpace(string(output)), strings.TrimSpace(string(openAIKeyOutput)))
		})
	})

	when("valid openai api key not passed", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"openai-key",
				"create",
				"--api-key", "sk-prodajkbcsdovub",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(strings.TrimSpace(string(output)), "missing required arguments")
		})
	})

	when("valid name not passed", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"openai-key",
				"create",
				"--name", "api-key",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(strings.TrimSpace(string(output)), "missing required arguments")
		})
	})
})
