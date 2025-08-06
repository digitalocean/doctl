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

var _ = suite("genai/openai-key/list", func(t *testing.T, when spec.G, it spec.S) {
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

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(openAIKeyListResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("valid list openai keys", func() {
		it("gets the list of openai key", func() {
			cmd = exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"genai",
				"openai-key",
				"list",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err)
			expect.Contains(strings.TrimSpace(string(openAIKeyListOutput)), strings.TrimSpace(string(output)))
		})
	})
})

const (
	openAIKeyListResponse = `
	{
	"api_key_infos": [
		{
			"name": "new key",
			"uuid": "11f037b5-0000-0000-0000-4e013e2ddde4",
			"created_by": "18919793",
			"models": [],
			"created_at": "2025-05-23T09:09:25Z",
			"updated_at": "2025-07-08T05:33:17Z"
		}
	]
}`

	openAIKeyListOutput = `
Name       UUID                                    Created At                       Created By    Updated At                       Deleted At
new key    11f037b5-0000-0000-0000-4e013e2ddde4    2025-05-23 09:09:25 +0000 UTC    18919793      2025-07-08 05:33:17 +0000 UTC    <nil>`
)
