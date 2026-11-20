package integration

import (
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
})
