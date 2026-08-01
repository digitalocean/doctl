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

var _ = suite("gradient/anthropic-key/create", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/gen-ai/anthropic/keys":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(anthropicKeyResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("valid anthropic api key and name is passed", func() {
		it("creates the anthropic key", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"anthropic-key",
				"create",
				"--name", "api-key",
				"--api-key", "sk-ant-prodajkbcsdovub",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Contains(strings.TrimSpace(string(output)), strings.TrimSpace(string(anthropicKeyOutput)))
		})
	})

	when("valid anthropic api key not passed", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"gradient",
				"anthropic-key",
				"create",
				"--api-key", "sk-ant-prodajkbcsdovub",
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
				"anthropic-key",
				"create",
				"--name", "api-key",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(strings.TrimSpace(string(output)), "missing required arguments")
		})
	})
})
