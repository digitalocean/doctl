package integration

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

const harborJobUUID = "3f2a5c1e-9b7d-4c6a-8e1f-2d4b6a8c0e12"

// fakeManagedHarbor is the Managed Harbor API of RFC section 8.5 plus the
// presigned Spaces URLs it hands out.
type fakeManagedHarbor struct {
	t      *testing.T
	server *httptest.Server

	mu        sync.Mutex
	announced map[string]any
	blob      []byte
	jobReq    map[string]any
	idemKey   string
	jobPolls  int
}

func newFakeManagedHarbor(t *testing.T) *fakeManagedHarbor {
	f := &fakeManagedHarbor{t: t}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	return f
}

func (f *fakeManagedHarbor) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")

	if strings.HasPrefix(r.URL.Path, "/spaces/") {
		require.Equal(f.t, "supersecret", r.URL.Query().Get("X-Amz-Signature"))
		switch r.URL.Path {
		case "/spaces/blob":
			b, _ := io.ReadAll(r.Body)
			f.blob = b
		case "/spaces/result.json":
			w.Write([]byte(`{"reward":1.0}`))
		case "/spaces/trajectory.json":
			w.Write([]byte(`{"steps":[{"role":"agent"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}

	if r.Header.Get("Authorization") != "Bearer some-magic-token" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"id":"unauthorized","message":"Unable to authenticate you"}`))
		return
	}

	signed := func(name string) string { return f.server.URL + "/spaces/" + name + "?X-Amz-Signature=supersecret" }
	base := "/v2/gen-ai/harbor"
	switch {
	case r.Method == http.MethodPost && r.URL.Path == base+"/tasks/fix-login-bug/uploads":
		require.NoError(f.t, json.NewDecoder(r.Body).Decode(&f.announced))
		fmt.Fprintf(w, `{"upload_uuid":"up-1","upload_url":%q}`, signed("blob"))

	case r.Method == http.MethodPost && r.URL.Path == base+"/tasks/fix-login-bug/uploads/up-1/complete":
		// The server re-hashes the files inside the archive, not its bytes.
		require.Equal(f.t, f.announced["digest"], harborArchiveDigest(f.t, f.blob))
		fmt.Fprint(w, `{"state":"validating"}`)

	case r.Method == http.MethodGet && r.URL.Path == base+"/tasks/fix-login-bug/uploads/up-1":
		fmt.Fprint(w, `{"upload":{"upload_uuid":"up-1","state":"accepted","lint_report":{"passed":true,"findings":[{"rule":"unpinned-image","severity":"warning","path":"environment/Dockerfile","message":"Pin the base image by digest"}]}}}`)

	case r.Method == http.MethodPost && r.URL.Path == base+"/jobs":
		f.idemKey = r.Header.Get("Idempotency-Key")
		require.NoError(f.t, json.NewDecoder(r.Body).Decode(&f.jobReq))
		fmt.Fprintf(w, `{"job_uuid":%q,"name":"nightly","status":"queued","n_trials":3,"tasks":[{"name":"fix-login-bug","digest":%q}],"agents":[{"name":"claude-code","version":"1.0.0","model":"anthropic-claude-sonnet-4"}]}`, harborJobUUID, f.announced["digest"])

	case r.Method == http.MethodGet && r.URL.Path == base+"/jobs/"+harborJobUUID:
		f.jobPolls++
		status, done := "running", 1
		if f.jobPolls > 1 {
			status, done = "successful", 3
		}
		fmt.Fprintf(w, `{"job":{"uuid":%q,"name":"nightly","status":%q,"n_total":3,"n_completed":%d,"n_errored":0}}`, harborJobUUID, status, done)

	case r.Method == http.MethodGet && r.URL.Path == base+"/jobs/"+harborJobUUID+"/trials":
		fmt.Fprint(w, `{"trials":[{"trial_name":"fix-login-bug__a1b2","task_name":"fix-login-bug","agent_name":"claude-code","model_slug":"anthropic-claude-sonnet-4","attempt":1,"status":"completed","rewards":{"reward":1},"n_input_tokens":1200,"n_output_tokens":340}],"meta":{"total":1}}`)

	case r.Method == http.MethodGet && r.URL.Path == base+"/jobs/"+harborJobUUID+"/artifacts":
		fmt.Fprintf(w, `{"artifacts":[{"path":"result.json","url":%q},{"path":"fix-login-bug__a1b2/agent/trajectory.json","url":%q}]}`, signed("result.json"), signed("trajectory.json"))

	default:
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
		w.WriteHeader(http.StatusNotFound)
	}
}

// harborArchiveDigest recomputes Harbor's content digest from the files in a
// tar.gz, as the server does when an upload completes.
func harborArchiveDigest(t *testing.T, blob []byte) string {
	gz, err := gzip.NewReader(bytes.NewReader(blob))
	require.NoError(t, err)
	tr := tar.NewReader(gz)
	sums := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		h := sha256.New()
		_, err = io.Copy(h, tr)
		require.NoError(t, err)
		sums[hdr.Name] = hex.EncodeToString(h.Sum(nil))
	}
	paths := make([]string, 0, len(sums))
	for p := range sums {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	outer := sha256.New()
	for _, p := range paths {
		fmt.Fprintf(outer, "%s\x00%s\n", p, sums[p])
	}
	return "sha256:" + hex.EncodeToString(outer.Sum(nil))
}

var _ = suite("genai/harbor", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect  *require.Assertions
		fake    *fakeManagedHarbor
		taskDir string
	)

	it.Before(func() {
		expect = require.New(t)
		fake = newFakeManagedHarbor(t)

		taskDir = filepath.Join(t.TempDir(), "fix-login-bug")
		for rel, content := range map[string]string{
			"task.toml":              "[task]\nname = \"my-org/fix-login-bug\"\n",
			"instruction.md":         "The login page crashes. Fix it.\n",
			"environment/Dockerfile": "FROM python:3.12\n",
			"tests/test.sh":          "#!/bin/sh\nexit 0\n",
			"notes.txt":              "not part of the task\n",
		} {
			p := filepath.Join(taskDir, filepath.FromSlash(rel))
			expect.NoError(os.MkdirAll(filepath.Dir(p), 0o755))
			expect.NoError(os.WriteFile(p, []byte(content), 0o755))
		}
	})

	it.After(func() {
		fake.server.Close()
	})

	doctl := func(args ...string) *exec.Cmd {
		return exec.Command(builtBinaryPath, append([]string{"-t", "some-magic-token", "-u", fake.server.URL}, args...)...)
	}

	it("publishes a task, runs it, and downloads the results", func() {
		output, err := doctl("genai", "harbor", "task", "publish", taskDir, "--wait", "--poll-interval", "10ms").CombinedOutput()
		expect.NoError(err, string(output))
		expect.Contains(string(output), "accepted")
		expect.Contains(string(output), "Pin the base image by digest")
		expect.Contains(string(output), "the org prefix is dropped")
		expect.NotContains(string(output), "supersecret")
		expect.Len(fake.announced["file_digests"], 4, "notes.txt is not part of a Harbor task")

		output, err = doctl("genai", "harbor", "run",
			"-T", "fix-login-bug", "-a", "claude-code", "-m", "anthropic-claude-sonnet-4",
			"--attempts", "3", "--concurrency", "4", "--name", "nightly",
			"--wait", "--poll-interval", "10ms", "-o", "json").Output()
		expect.NoError(err, string(output))
		var job map[string]any
		expect.NoError(json.Unmarshal(output, &job))
		expect.Equal("successful", job["status"])
		expect.Equal(map[string]any{
			"name":                "nightly",
			"tasks":               []any{"fix-login-bug"},
			"agents":              []any{map[string]any{"name": "claude-code", "model": "anthropic-claude-sonnet-4"}},
			"n_attempts":          float64(3),
			"n_concurrent_trials": float64(4),
			"retry":               map[string]any{"max_retries": float64(1)},
			"dry_run":             false,
			"trigger_source":      "cli",
		}, fake.jobReq)
		expect.NotEmpty(fake.idemKey)

		output, err = doctl("genai", "harbor", "job", "trials", harborJobUUID).CombinedOutput()
		expect.NoError(err, string(output))
		expect.Contains(string(output), "fix-login-bug__a1b2")
		expect.Contains(string(output), "1200/340")

		out := t.TempDir()
		output, err = doctl("genai", "harbor", "job", "download", harborJobUUID, "--out", out).CombinedOutput()
		expect.NoError(err, string(output))
		b, err := os.ReadFile(filepath.Join(out, harborJobUUID, "fix-login-bug__a1b2", "agent", "trajectory.json"))
		expect.NoError(err)
		expect.JSONEq(`{"steps":[{"role":"agent"}]}`, string(b))
	})

	it("fails clearly with an invalid token", func() {
		output, err := exec.Command(builtBinaryPath, "-t", "wrong-token", "-u", fake.server.URL, "genai", "harbor", "job", "list").CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), "401")
	})

	it("validates attempts before calling the API", func() {
		output, err := doctl("genai", "harbor", "run", "-T", "fix-login-bug", "-a", "claude-code", "--attempts", "0", "-m", "x").CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), "between 1 and 10")
	})
})
