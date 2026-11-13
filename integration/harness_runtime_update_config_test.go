package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

// `harness-runtime update --spec` sends a partial manifest as manifest_yaml,
// takes the 202 → UPDATING → APPLIED path by polling GET, and stops early on
// the 200 UNCHANGED answer.
var _ = suite("harness-runtime/update-config", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect   *require.Assertions
		server   *httptest.Server
		specPath string
		// outcome is what the PATCH handler answers with.
		outcome string
		gets    int32
		gotBody map[string]any
	)

	const manifest = "tools:\n  - do.actions:\n      tools: [toolbelt:read-only@1]\n"

	it.Before(func() {
		expect = require.New(t)
		atomic.StoreInt32(&gets, 0)
		gotBody = nil
		outcome = "ACCEPTED"

		dir := t.TempDir()
		specPath = filepath.Join(dir, "tools.yaml")
		expect.NoError(os.WriteFile(specPath, []byte(manifest), 0o644))

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("content-type", "application/json")

			switch {
			case req.URL.Path == "/v2/harness-runtime/sessions/sess_cfg" && req.Method == http.MethodPatch:
				raw, _ := io.ReadAll(req.Body)
				_ = json.Unmarshal(raw, &gotBody)
				if outcome == "UNCHANGED" {
					w.Write([]byte(`{"session":{"session_id":"sess_cfg","name":"cfg","agent_kind":"AGENT_KIND_OPENCODE","status":"SESSION_STATUS_READY"},"outcome":"UNCHANGED"}`))
					return
				}
				w.WriteHeader(http.StatusAccepted)
				w.Write([]byte(`{"session":{"session_id":"sess_cfg","name":"cfg","agent_kind":"AGENT_KIND_OPENCODE","status":"SESSION_STATUS_UPDATING","update":{"update_id":"u1","state":"APPLYING","changes":{"tools":{"added":["do.actions"]}}}},"outcome":"ACCEPTED"}`))
			case req.URL.Path == "/v2/harness-runtime/sessions/sess_cfg" && req.Method == http.MethodGet:
				if atomic.AddInt32(&gets, 1) < 2 {
					w.Write([]byte(`{"session":{"session_id":"sess_cfg","name":"cfg","agent_kind":"AGENT_KIND_OPENCODE","status":"SESSION_STATUS_UPDATING","update":{"update_id":"u1","state":"APPLYING"}}}`))
					return
				}
				w.Write([]byte(`{"session":{"session_id":"sess_cfg","name":"cfg","agent_kind":"AGENT_KIND_OPENCODE","status":"SESSION_STATUS_READY","config_drifted":true,"update":{"update_id":"u1","state":"APPLIED","changes":{"tools":{"added":["do.actions"]}}}}}`))
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

	it("patches manifest_yaml, waits for UPDATING to settle and reports the change", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"harness-runtime", "update", "sess_cfg", "--spec", specPath,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))
		expect.Equal(manifest, gotBody["manifest_yaml"])
		expect.Len(gotBody, 1)
		expect.Contains(string(output), "updated")
		expect.Contains(string(output), "tools +do.actions")
		expect.GreaterOrEqual(atomic.LoadInt32(&gets), int32(2))
	})

	it("accepts --file as an alias of --spec", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"harness-runtime", "update", "sess_cfg", "--file", specPath,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))
		expect.Equal(manifest, gotBody["manifest_yaml"])
	})

	it("returns the PATCH result without polling for -o json", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"-o", "json",
			"harness-runtime", "update", "sess_cfg", "--spec", specPath,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))
		var got map[string]any
		expect.NoError(json.Unmarshal(output, &got))
		expect.Equal("SESSION_STATUS_UPDATING", got["status"])
		expect.Equal(int32(0), atomic.LoadInt32(&gets))
	})

	it("reports UNCHANGED without polling", func() {
		outcome = "UNCHANGED"
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"harness-runtime", "update", "sess_cfg", "--spec", specPath,
		)

		output, err := cmd.CombinedOutput()
		expect.NoError(err, string(output))
		expect.Contains(string(output), "nothing to change")
		expect.Equal(int32(0), atomic.LoadInt32(&gets))
	})

	it("rejects --spec together with --resume-on-topoff", func() {
		cmd := exec.Command(builtBinaryPath,
			"-t", "some-magic-token",
			"-u", server.URL,
			"harness-runtime", "update", "sess_cfg", "--spec", specPath, "--resume-on-topoff",
		)

		output, err := cmd.CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), "cannot be combined with --resume-on-topoff")
	})
})
