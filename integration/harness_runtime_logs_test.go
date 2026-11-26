package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

const harnessLogsSSE = "" +
	"id: e1\ndata: {\"event_id\":\"e1\",\"run_id\":\"run_1\",\"session_id\":\"sess_x\",\"timestamp\":\"2026-01-02T03:04:05Z\",\"seq\":1,\"type\":\"run.started\",\"data\":{}}\n\n" +
	"id: e2\ndata: {\"event_id\":\"e2\",\"run_id\":\"run_1\",\"session_id\":\"sess_x\",\"timestamp\":\"2026-01-02T03:04:06Z\",\"seq\":2,\"type\":\"run.completed\",\"data\":{}}\n\n"

var _ = suite("harness-runtime/logs", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/harness-runtime/sessions/sess_x/events":
				w.Header().Set("Content-Type", "text/event-stream")
				w.Write([]byte(harnessLogsSSE))
			case "/v2/harness-runtime/sessions/sess_gone/events":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"id":"not_found","message":"session not found"}`))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	it.After(func() { server.Close() })

	it("prints JSON when the config file sets output: json", func() {
		config := filepath.Join(t.TempDir(), "config.yaml")
		expect.NoError(os.WriteFile(config, []byte("output: json\n"), 0600))

		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"--config", config,
			"harness-runtime", "logs", "sess_x",
		)

		output, err := cmd.Output()
		expect.NoError(err)
		lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
		expect.Len(lines, 2, string(output))
		for _, l := range lines {
			expect.True(json.Valid([]byte(l)), l)
		}
	})

	it("prints the -o json error envelope on stdout when nothing was streamed", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"-o", "json",
			"harness-runtime", "logs", "sess_gone",
		)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		output, err := cmd.Output()
		expect.Error(err)
		expect.Empty(stderr.String())
		var got struct {
			Errors []struct {
				Detail string `json:"detail"`
				Status int    `json:"status"`
			} `json:"errors"`
		}
		expect.NoError(json.Unmarshal(output, &got), string(output))
		expect.Len(got.Errors, 1)
		expect.Equal(http.StatusNotFound, got.Errors[0].Status)
		expect.Contains(got.Errors[0].Detail, "session not found")
	})
})
