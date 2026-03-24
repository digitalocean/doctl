package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("dedicated-inference/create-token", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		cmd    *exec.Cmd
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/dedicated-inferences/00000000-0000-4000-8000-000000000000/tokens":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal("failed to read request body")
				}

				var createReq map[string]any
				err = json.Unmarshal(body, &createReq)
				if err != nil {
					t.Fatalf("failed to parse request body: %s", err)
				}

				name, ok := createReq["name"]
				if !ok || name == "" {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"id":"bad_request","message":"name is required"}`))
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(dedicatedInferenceCreateTokenResponse))
			case "/v2/dedicated-inferences/99999999-9999-4999-8999-999999999999/tokens":
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

	when("valid dedicated inference ID and token name are provided", func() {
		it("creates an auth token", func() {
			aliases := []string{"create-token", "ct"}

			for _, alias := range aliases {
				cmd = exec.Command(builtBinaryPath,
					"-t", "some-magic-token",
					"-u", server.URL,
					"dedicated-inference",
					alias,
					"00000000-0000-4000-8000-000000000000",
					"--token-name", "my-token",
				)

				output, err := cmd.CombinedOutput()
				expect.NoError(err, fmt.Sprintf("received error output for alias %q: %s", alias, output))
				expect.Equal(strings.TrimSpace(dedicatedInferenceCreateTokenOutput), strings.TrimSpace(string(output)))
			}
		})
	})

	when("dedicated inference ID is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"create-token",
				"--token-name", "my-token",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "missing")
		})
	})

	when("dedicated inference does not exist", func() {
		it("returns a not found error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"create-token",
				"99999999-9999-4999-8999-999999999999",
				"--token-name", "my-token",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "404")
		})
	})

	when("token-name flag is missing", func() {
		it("returns an error", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"create-token",
				"00000000-0000-4000-8000-000000000000",
			)

			output, err := cmd.CombinedOutput()
			expect.Error(err)
			expect.Contains(string(output), "token-name")
		})
	})

	when("using the di alias", func() {
		it("creates an auth token", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"di",
				"create-token",
				"00000000-0000-4000-8000-000000000000",
				"--token-name", "my-token",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceCreateTokenOutput), strings.TrimSpace(string(output)))
		})
	})

	when("passing a format flag", func() {
		it("displays only those columns", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"dedicated-inference",
				"create-token",
				"00000000-0000-4000-8000-000000000000",
				"--token-name", "my-token",
				"--format", "ID,Name",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(dedicatedInferenceCreateTokenFormatOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	dedicatedInferenceCreateTokenOutput = `
ID       Name        Value               Created At
tok-1    my-token    secret-token-123    2023-01-01 00:00:00 +0000 UTC
`
	dedicatedInferenceCreateTokenFormatOutput = `
ID       Name
tok-1    my-token
`

	dedicatedInferenceCreateTokenResponse = `
{
  "token": {
    "id": "tok-1",
    "name": "my-token",
    "value": "secret-token-123",
    "created_at": "2023-01-01T00:00:00Z"
  }
}
`
)
