package integration

import (
	"archive/tar"
	"archive/zip"
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
	jobZip    []byte
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
		case "/spaces/job.zip":
			w.Header().Set("Content-Type", "application/zip")
			w.Write(f.jobArchive())
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
	base := "/v1/eval-harness"
	switch {
	case r.Method == http.MethodPost && r.URL.Path == base+"/datasets/bug-fixes/uploads":
		require.NoError(f.t, json.NewDecoder(r.Body).Decode(&f.announced))
		fmt.Fprintf(w, `{"unchanged":false,"upload":{"uuid":"up-1","dataset":"bug-fixes","status":"uploading","digest":%q},"upload_url":{"method":"PUT","url":%q,"expires_at":"2026-11-19T00:00:00Z"}}`, f.announced["digest"], signed("blob"))

	case r.Method == http.MethodPost && r.URL.Path == base+"/datasets/bug-fixes/uploads/up-1/complete":
		// The dataset digest is the SHA-256 of the archive bytes.
		sum := sha256.Sum256(f.blob)
		require.Equal(f.t, "sha256:"+hex.EncodeToString(sum[:]), f.announced["digest"])
		fmt.Fprintf(w, `{"uuid":"up-1","dataset":"bug-fixes","status":"ready","digest":%q,"findings":[{"task":"fix-login-bug","file":"environment/Dockerfile","rule":"unpinned-image","severity":"warning","message":"Pin the base image by digest"}]}`, f.announced["digest"])

	case r.Method == http.MethodGet && r.URL.Path == base+"/datasets/bug-fixes":
		fmt.Fprintf(w, `{"name":"bug-fixes","status":"ready","digest":%q,"tasks":[{"name":"fix-login-bug","digest":"sha256:ab","size_bytes":10}]}`, f.announced["digest"])

	case r.Method == http.MethodGet && r.URL.Path == base+"/jobs":
		require.Equal(f.t, "successful", r.URL.Query().Get("status"))
		require.Equal(f.t, "100", r.URL.Query().Get("limit"))
		require.Empty(f.t, r.URL.Query().Get("page"))
		if r.URL.Query().Get("page_token") == "" {
			fmt.Fprint(w, `{"jobs":[{"uuid":"job-a","name":"one","status":"successful","n_total":2,"n_completed":2,"n_errored":0,"created_at":"2026-11-19T00:00:00Z"}],"next_page_token":"next"}`)
		} else {
			require.Equal(f.t, "next", r.URL.Query().Get("page_token"))
			fmt.Fprintf(w, `{"jobs":[{"uuid":%q,"name":"two","status":"successful","n_total":1,"n_completed":1,"n_errored":0,"created_at":"2026-11-18T00:00:00Z"}]}`, harborJobUUID)
		}

	case r.Method == http.MethodDelete && r.URL.Path == base+"/jobs/"+harborJobUUID:
		w.WriteHeader(http.StatusNoContent)

	case r.Method == http.MethodPost && r.URL.Path == base+"/jobs":
		f.idemKey = r.Header.Get("Idempotency-Key")
		require.NoError(f.t, json.NewDecoder(r.Body).Decode(&f.jobReq))
		fmt.Fprintf(w, `{"uuid":%q,"name":"nightly","status":"queued","dry_run":false,"n_total":3,"dataset":"bug-fixes","tasks":[{"name":"fix-login-bug","digest":%q}],"agents":[{"name":"claude-code","version":"1.0.0","model_name":"anthropic-claude-sonnet-4"}]}`, harborJobUUID, f.announced["digest"])

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
		fmt.Fprintf(w, `{"url":%q,"expires_at":"2026-11-20T12:00:00Z","size_bytes":%d}`, signed("job.zip"), len(f.jobArchive()))

	default:
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
		w.WriteHeader(http.StatusNotFound)
	}
}

// jobArchive is the results zip the artifacts route signs. Files sit at the
// archive root, which is where harbor view reads them after unpacking.
func (f *fakeManagedHarbor) jobArchive() []byte {
	if f.jobZip != nil {
		return f.jobZip
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("fix-login-bug__a1b2/agent/trajectory.json")
	require.NoError(f.t, err)
	_, err = io.WriteString(w, `{"steps":[{"role":"agent"}]}`)
	require.NoError(f.t, err)
	require.NoError(f.t, zw.Close())
	f.jobZip = buf.Bytes()
	return f.jobZip
}

func tarNames(t *testing.T, blob []byte) []string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(blob))
	require.NoError(t, err)
	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, hdr.Name)
	}
	return names
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

		taskDir = filepath.Join(t.TempDir(), "bug-fixes")
		for rel, content := range map[string]string{
			"dataset.toml":                               "[dataset]\nname = \"my-org/bug-fixes\"\n\n[[tasks]]\nname = \"my-org/fix-login-bug\"\n",
			"tasks/fix-login-bug/task.toml":              "[task]\nname = \"my-org/fix-login-bug\"\n",
			"tasks/fix-login-bug/instruction.md":         "The login page crashes. Fix it.\n",
			"tasks/fix-login-bug/environment/Dockerfile": "FROM python:3.12\n",
			"tasks/fix-login-bug/tests/test.sh":          "#!/bin/sh\nexit 0\n",
			"tasks/fix-login-bug/notes.txt":              "not part of the task\n",
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

	it("publishes a dataset, runs it, and downloads the results", func() {
		output, err := doctl("genai", "harbor", "dataset", "publish", taskDir, "--poll-interval", "10ms").CombinedOutput()
		expect.NoError(err, string(output))
		expect.Contains(string(output), "ready")
		expect.Contains(string(output), "Pin the base image by digest")
		expect.Contains(string(output), "org prefix")
		expect.NotContains(string(output), "supersecret")
		expect.NotContains(strings.Join(tarNames(t, fake.blob), "\n"), "notes.txt")

		output, err = doctl("genai", "harbor", "run",
			"-d", "bug-fixes", "-T", "fix-login-bug", "-a", "claude-code", "-m", "anthropic-claude-sonnet-4",
			"--attempts", "3", "--concurrency", "4", "--name", "nightly",
			"-o", "json").Output()
		expect.NoError(err, string(output))
		var job map[string]any
		expect.NoError(json.Unmarshal(output, &job))
		expect.Equal("queued", job["status"])
		expect.Equal(map[string]any{
			"job_name":            "nightly",
			"datasets":            []any{map[string]any{"name": "bug-fixes", "task_names": []any{"fix-login-bug"}}},
			"agents":              []any{map[string]any{"name": "claude-code", "model_name": "anthropic-claude-sonnet-4"}},
			"n_attempts":          float64(3),
			"n_concurrent_trials": float64(4),
			"retry":               map[string]any{"max_retries": float64(1)},
			"dry_run":             false,
		}, fake.jobReq)
		expect.NotEmpty(fake.idemKey)

		out := t.TempDir()
		output, err = doctl("genai", "harbor", "job", "download", harborJobUUID, "--out", out).CombinedOutput()
		expect.NoError(err, string(output))
		b, err := os.ReadFile(filepath.Join(out, harborJobUUID, "fix-login-bug__a1b2", "agent", "trajectory.json"))
		expect.NoError(err)
		expect.JSONEq(`{"steps":[{"role":"agent"}]}`, string(b))
	})

	it("lists every page of jobs and deletes a finished job", func() {
		output, err := doctl("genai", "harbor", "job", "list", "--status", "successful").CombinedOutput()
		expect.NoError(err, string(output))
		expect.Contains(string(output), "job-a")
		expect.Contains(string(output), harborJobUUID)
		expect.Contains(string(output), "one")
		expect.Contains(string(output), "two")

		output, err = doctl("genai", "harbor", "job", "delete", harborJobUUID, "--force").CombinedOutput()
		expect.NoError(err, string(output))
	})

	it("fails clearly with an invalid token", func() {
		output, err := exec.Command(builtBinaryPath, "-t", "wrong-token", "-u", fake.server.URL, "genai", "harbor", "job", "list").CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), "401")
	})

	it("validates attempts before calling the API", func() {
		output, err := doctl("genai", "harbor", "run", "-d", "bug-fixes", "-a", "claude-code", "--attempts", "0", "-m", "x").CombinedOutput()
		expect.Error(err)
		expect.Contains(string(output), "between 1 and 10")
	})
})
