/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakeHarborStorage stands in for presigned Spaces URLs.
type fakeHarborStorage struct {
	server *httptest.Server
	mu     sync.Mutex
	puts   map[string][]byte
	files  map[string][]byte
	fail   map[string]int
	gets   map[string]int
}

func newFakeHarborStorage(t *testing.T) *fakeHarborStorage {
	s := &fakeHarborStorage{puts: map[string][]byte{}, files: map[string][]byte{}, fail: map[string]int{}, gets: map[string]int{}}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if n := s.fail[r.URL.Path]; n != 0 {
			if n > 0 {
				s.fail[r.URL.Path] = n - 1
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}
		switch r.Method {
		case http.MethodPut:
			b, _ := io.ReadAll(r.Body)
			s.puts[r.URL.Path] = b
		case http.MethodGet:
			s.gets[r.URL.Path]++
			b, ok := s.files[r.URL.Path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Write(b)
		}
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *fakeHarborStorage) url(path string) string {
	return s.server.URL + path + "?X-Amz-Signature=supersecret"
}

func noHarborWaiting(t *testing.T) {
	prevSleep, prevBackoff := harborSleep, harborRetryBackoff
	harborSleep = func(time.Duration) {}
	harborRetryBackoff = 0
	t.Cleanup(func() { harborSleep, harborRetryBackoff = prevSleep, prevBackoff })
}

func withHarborOutput(t *testing.T, format string) {
	prev := Output
	Output = format
	t.Cleanup(func() { Output = prev })
}

func basicHarborTask(t *testing.T) string {
	dir := filepath.Join(t.TempDir(), "fix-login-bug")
	writeHarborFixture(t, dir, map[string]string{
		"task.toml":              "[task]\nname = \"fix-login-bug\"\n",
		"instruction.md":         "Fix it.",
		"environment/Dockerfile": "FROM alpine\n",
		"tests/test.sh":          "#!/bin/sh\nexit 0\n",
	})
	return dir
}

func TestHarborCommandTree(t *testing.T) {
	root := GenAI()
	for path, want := range map[string][]string{
		"harbor":         {"agents", "dataset", "job", "run", "task"},
		"harbor task":    {"delete", "get", "list", "publish", "pull"},
		"harbor dataset": {"delete", "get", "list", "publish"},
		"harbor job":     {"cancel", "download", "get", "list", "trials"},
		"harbor agents":  {"list"},
	} {
		cmd, _, err := root.Find(strings.Fields(path))
		require.NoError(t, err, path)
		var names []string
		for _, c := range cmd.Commands() {
			names = append(names, c.Name())
		}
		sort.Strings(names)
		assert.Equal(t, want, names, path)
	}

	run, _, err := root.Find([]string{"harbor", "run"})
	require.NoError(t, err)
	assert.Equal(t, "T", run.Flags().Lookup(doctl.ArgHarborTask).Shorthand, "-t is the global --access-token")
	assert.Equal(t, "d", run.Flags().Lookup(doctl.ArgHarborDataset).Shorthand)
	assert.Equal(t, "a", run.Flags().Lookup(doctl.ArgHarborAgent).Shorthand)
	assert.Equal(t, "m", run.Flags().Lookup(doctl.ArgHarborModel).Shorthand)

	_, _, err = root.Find([]string{"knowledge-base"})
	assert.NoError(t, err, "the deprecated genai knowledge-base forwarder still resolves")
	assert.NotContains(t, GradientAI().Aliases, "genai")
}

func TestRunHarborTaskPublish(t *testing.T) {
	noHarborWaiting(t)
	withHarborOutput(t, "json")

	t.Run("uploads, completes, and waits for lint", func(t *testing.T) {
		storage := newFakeHarborStorage(t)
		dir := basicHarborTask(t)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			var announced *do.HarborTaskUploadRequest
			tm.harbor.EXPECT().StartTaskUpload("fix-login-bug", gomock.Any()).DoAndReturn(
				func(_ string, req *do.HarborTaskUploadRequest) (*do.HarborTaskUploadStart, error) {
					announced = req
					return &do.HarborTaskUploadStart{UploadUUID: "up-1", UploadURL: storage.url("/blob")}, nil
				})
			tm.harbor.EXPECT().CompleteTaskUpload("fix-login-bug", "up-1").Return(&do.HarborTaskUpload{UploadUUID: "up-1", State: do.HarborUploadValidating}, nil)
			passed := true
			tm.harbor.EXPECT().GetTaskUpload("fix-login-bug", "up-1").Return(&do.HarborTaskUpload{
				UploadUUID: "up-1",
				State:      do.HarborUploadAccepted,
				LintReport: &do.HarborLintReport{Passed: &passed, Findings: []do.HarborLintFinding{{Rule: "unpinned-image", Severity: "warning", Message: "pin it"}}},
			}, nil)

			out := &bytes.Buffer{}
			config.Out = out
			config.Args = []string{dir}
			config.Doit.Set(config.NS, doctl.ArgCommandWait, true)
			config.Doit.Set(config.NS, doctl.ArgWaitTimeout, time.Minute)
			config.Doit.Set(config.NS, doctl.ArgHarborPollInterval, time.Second)

			require.NoError(t, RunHarborTaskPublish(config))

			require.NotNil(t, announced)
			assert.Regexp(t, `^sha256:[0-9a-f]{64}$`, announced.Digest)
			assert.Equal(t, int64(len(storage.puts["/blob"])), announced.SizeBytes)
			assert.Len(t, announced.FileDigests, 4)
			assert.Equal(t, map[string]any{"name": "fix-login-bug"}, announced.Manifest["task"])

			sums, err := extractHarborArchive(writeTemp(t, storage.puts["/blob"]), t.TempDir())
			require.NoError(t, err)
			assert.Equal(t, announced.Digest, harborContentDigest(sums), "the uploaded archive holds the announced files")

			var got do.HarborPublishResult
			require.NoError(t, json.Unmarshal(out.Bytes(), &got))
			assert.Equal(t, do.HarborUploadAccepted, got.State)
			assert.Equal(t, announced.Digest, got.Digest)
			assert.Len(t, got.LintReport.Findings, 1)
		})
	})

	t.Run("unchanged tasks are not uploaded", func(t *testing.T) {
		dir := basicHarborTask(t)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().StartTaskUpload("renamed", gomock.Any()).Return(&do.HarborTaskUploadStart{Unchanged: true}, nil)
			config.Args = []string{dir}
			config.Doit.Set(config.NS, doctl.ArgHarborName, "renamed")
			require.NoError(t, RunHarborTaskPublish(config))
		})
	})

	t.Run("a rejected upload fails the command", func(t *testing.T) {
		storage := newFakeHarborStorage(t)
		dir := basicHarborTask(t)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().StartTaskUpload(gomock.Any(), gomock.Any()).Return(&do.HarborTaskUploadStart{UploadUUID: "up-2", UploadURL: storage.url("/blob")}, nil)
			tm.harbor.EXPECT().CompleteTaskUpload("fix-login-bug", "up-2").Return(&do.HarborTaskUpload{UploadUUID: "up-2", State: do.HarborUploadRejected, ErrorDescription: "private images must be on DOCR"}, nil)
			config.Args = []string{dir}
			config.Doit.Set(config.NS, doctl.ArgCommandWait, true)
			err := RunHarborTaskPublish(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "private images must be on DOCR")
		})
	})

	t.Run("an expired upload link is not retried and its signature is not shown", func(t *testing.T) {
		storage := newFakeHarborStorage(t)
		storage.fail["/blob"] = -1
		dir := basicHarborTask(t)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().StartTaskUpload(gomock.Any(), gomock.Any()).Return(&do.HarborTaskUploadStart{UploadUUID: "up-3", UploadURL: storage.url("/blob")}, nil)
			config.Args = []string{dir}
			err := RunHarborTaskPublish(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "403")
			assert.NotContains(t, err.Error(), "supersecret")
		})
	})
}

func writeTemp(t *testing.T, b []byte) string {
	p := filepath.Join(t.TempDir(), "blob")
	require.NoError(t, os.WriteFile(p, b, 0o644))
	return p
}

func TestPutHarborArchiveRetries(t *testing.T) {
	noHarborWaiting(t)
	storage := newFakeHarborStorage(t)
	storage.fail["/blob"] = 2
	archive := writeTemp(t, []byte("archive bytes"))

	require.NoError(t, putHarborArchive(storage.url("/blob"), archive, 13))
	assert.Equal(t, []byte("archive bytes"), storage.puts["/blob"])

	storage.fail["/blob"] = 3
	err := putHarborArchive(storage.url("/blob"), archive, 13)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "after 3 attempts")

	err = putHarborArchive("http://127.0.0.1:1/blob?X-Amz-Signature=supersecret", archive, 13)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "supersecret")
}

func TestRunHarborTaskPull(t *testing.T) {
	noHarborWaiting(t)
	pkg, err := packHarborTask(basicHarborTask(t), "")
	require.NoError(t, err)
	defer pkg.Close()
	archive, err := os.ReadFile(pkg.ArchivePath)
	require.NoError(t, err)

	storage := newFakeHarborStorage(t)
	storage.files["/archive"] = archive

	t.Run("unpacks and checks the digest", func(t *testing.T) {
		out := t.TempDir()
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().GetTaskArchive("fix-login-bug").Return(&do.HarborPresignedURL{URL: storage.url("/archive"), Digest: pkg.Digest}, nil)
			config.Args = []string{"fix-login-bug"}
			config.Doit.Set(config.NS, doctl.ArgHarborOut, out)
			require.NoError(t, RunHarborTaskPull(config))
		})
		b, err := os.ReadFile(filepath.Join(out, "fix-login-bug", "tests", "test.sh"))
		require.NoError(t, err)
		assert.Equal(t, "#!/bin/sh\nexit 0\n", string(b))
		info, err := os.Stat(filepath.Join(out, "fix-login-bug", "tests", "test.sh"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())

		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Args = []string{"fix-login-bug"}
			config.Doit.Set(config.NS, doctl.ArgHarborOut, out)
			err := RunHarborTaskPull(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "--force")
		})
	})

	t.Run("refuses files that do not match the digest", func(t *testing.T) {
		out := t.TempDir()
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().GetTaskArchive("fix-login-bug").Return(&do.HarborPresignedURL{URL: storage.url("/archive")}, nil)
			tm.harbor.EXPECT().GetTask("fix-login-bug").Return(&do.HarborTask{Name: "fix-login-bug", Digest: "sha256:" + strings.Repeat("0", 64)}, nil)
			config.Args = []string{"fix-login-bug"}
			config.Doit.Set(config.NS, doctl.ArgHarborOut, out)
			err := RunHarborTaskPull(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "digest")
		})
		_, err := os.Stat(filepath.Join(out, "fix-login-bug"))
		assert.True(t, os.IsNotExist(err), "nothing is left behind")
	})
}

func TestRunHarborTaskDeleteConflict(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		conflict := &godo.ErrorResponse{Response: &http.Response{StatusCode: http.StatusConflict, Request: &http.Request{}}, Message: "task is referenced"}
		tm.harbor.EXPECT().DeleteTask("fix-login-bug").Return(conflict)
		config.Args = []string{"fix-login-bug"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		err := RunHarborTaskDelete(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "used by a dataset")
	})
}

func TestBuildHarborJobRequest(t *testing.T) {
	tests := []struct {
		name   string
		set    map[string]any
		want   []do.HarborJobAgent
		errMsg string
	}{
		{name: "pairs by position", set: map[string]any{doctl.ArgHarborTask: []string{"a"}, doctl.ArgHarborAgent: []string{"claude-code", "codex"}, doctl.ArgHarborModel: []string{"m1", "m2"}},
			want: []do.HarborJobAgent{{Name: "claude-code", Model: "m1"}, {Name: "codex", Model: "m2"}}},
		{name: "one model for every agent", set: map[string]any{doctl.ArgHarborTask: []string{"a"}, doctl.ArgHarborAgent: []string{"claude-code", "aider"}, doctl.ArgHarborModel: []string{"m1"}},
			want: []do.HarborJobAgent{{Name: "claude-code", Model: "m1"}, {Name: "aider", Model: "m1"}}},
		{name: "one agent for every model", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"codex"}, doctl.ArgHarborModel: []string{"m1", "dedicated:my-di:llama"}},
			want: []do.HarborJobAgent{{Name: "codex", Model: "m1"}, {Name: "codex", Model: "dedicated:my-di:llama"}}},
		{name: "mismatched counts", set: map[string]any{doctl.ArgHarborTask: []string{"a"}, doctl.ArgHarborAgent: []string{"a1", "a2"}, doctl.ArgHarborModel: []string{"m1", "m2", "m3"}}, errMsg: "one model per agent"},
		{name: "no tasks", set: map[string]any{doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}}, errMsg: "choose what to run"},
		{name: "tasks and dataset", set: map[string]any{doctl.ArgHarborTask: []string{"a"}, doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}}, errMsg: "not both"},
		{name: "no agent", set: map[string]any{doctl.ArgHarborTask: []string{"a"}, doctl.ArgHarborModel: []string{"m1"}}, errMsg: "at least one agent"},
		{name: "no model", set: map[string]any{doctl.ArgHarborTask: []string{"a"}, doctl.ArgHarborAgent: []string{"a1"}}, errMsg: "--model"},
		{name: "attempts limit", set: map[string]any{doctl.ArgHarborTask: []string{"a"}, doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}, doctl.ArgHarborAttempts: 11}, errMsg: "between 1 and 10"},
		{name: "concurrency limit", set: map[string]any{doctl.ArgHarborTask: []string{"a"}, doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}, doctl.ArgHarborConcurrency: 51}, errMsg: "between 1 and 50"},
		{name: "invalid task name", set: map[string]any{doctl.ArgHarborTask: []string{"My Task"}, doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}}, errMsg: "not a valid task name"},
		{name: "duplicate pair", set: map[string]any{doctl.ArgHarborTask: []string{"a"}, doctl.ArgHarborAgent: []string{"a1", "a1"}, doctl.ArgHarborModel: []string{"m1"}}, errMsg: "more than once"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgHarborAttempts, 1)
				config.Doit.Set(config.NS, doctl.ArgHarborConcurrency, 4)
				config.Doit.Set(config.NS, doctl.ArgHarborMaxRetries, 1)
				for k, v := range tt.set {
					config.Doit.Set(config.NS, k, v)
				}
				req, key, err := buildHarborJobRequest(config)
				if tt.errMsg != "" {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.errMsg)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want, req.Agents)
				assert.NotEmpty(t, key)
				assert.Equal(t, "cli", req.TriggerSource)
			})
		})
	}
}

func TestRunHarborRun(t *testing.T) {
	noHarborWaiting(t)

	setRun := func(config *CmdConfig) {
		config.Doit.Set(config.NS, doctl.ArgHarborTask, []string{"fix-login-bug"})
		config.Doit.Set(config.NS, doctl.ArgHarborAgent, []string{"claude-code"})
		config.Doit.Set(config.NS, doctl.ArgHarborModel, []string{"anthropic-claude-sonnet-4"})
		config.Doit.Set(config.NS, doctl.ArgHarborAttempts, 3)
		config.Doit.Set(config.NS, doctl.ArgHarborConcurrency, 4)
		config.Doit.Set(config.NS, doctl.ArgHarborMaxRetries, 1)
	}

	t.Run("submits the RFC request with the idempotency key", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			want := &do.HarborJobRequest{
				Tasks:             []string{"fix-login-bug"},
				Agents:            []do.HarborJobAgent{{Name: "claude-code", Model: "anthropic-claude-sonnet-4"}},
				NAttempts:         3,
				NConcurrentTrials: 4,
				Retry:             do.HarborJobRetry{MaxRetries: 1},
				TriggerSource:     "cli",
			}
			tm.harbor.EXPECT().CreateJob(want, "ci-run-42").Return(&do.HarborJob{JobUUID: "job-1", Status: do.HarborJobQueued, NTrials: 3}, nil)
			setRun(config)
			config.Doit.Set(config.NS, doctl.ArgHarborIdempotencyKey, "ci-run-42")
			require.NoError(t, RunHarborRun(config))
		})
	})

	t.Run("--wait follows the job and fails on a failed job", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().CreateJob(gomock.Any(), gomock.Any()).Return(&do.HarborJob{JobUUID: "job-2", Status: do.HarborJobQueued, NTrials: 3}, nil)
			gomock.InOrder(
				tm.harbor.EXPECT().GetJob("job-2").Return(&do.HarborJob{JobUUID: "job-2", Status: do.HarborJobRunning, NTrials: 3, NCompleted: 1}, nil),
				tm.harbor.EXPECT().GetJob("job-2").Return(&do.HarborJob{JobUUID: "job-2", Status: do.HarborJobFailed, NTrials: 3, NCompleted: 3, NErrored: 3, ErrorDescription: "sandbox quota"}, nil),
			)
			setRun(config)
			config.Doit.Set(config.NS, doctl.ArgCommandWait, true)
			config.Doit.Set(config.NS, doctl.ArgWaitTimeout, time.Hour)
			config.Doit.Set(config.NS, doctl.ArgHarborPollInterval, time.Second)
			err := RunHarborRun(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed: sandbox quota")
		})
	})

	t.Run("--wait succeeds on a successful job", func(t *testing.T) {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().GetJob("job-3").Return(&do.HarborJob{JobUUID: "job-3", Status: do.HarborJobSuccessful, NTrials: 3, NCompleted: 3}, nil)
			config.Args = []string{"job-3"}
			config.Doit.Set(config.NS, doctl.ArgCommandWait, true)
			config.Doit.Set(config.NS, doctl.ArgWaitTimeout, time.Hour)
			config.Doit.Set(config.NS, doctl.ArgHarborPollInterval, time.Second)
			require.NoError(t, RunHarborJobGet(config))
		})
	})
}

func TestRunHarborJobDownload(t *testing.T) {
	noHarborWaiting(t)
	storage := newFakeHarborStorage(t)
	storage.files["/result.json"] = []byte(`{"reward":1}`)
	storage.files["/trajectory.json"] = []byte(`{"steps":[]}`)
	sum := sha256.Sum256(storage.files["/result.json"])

	artifacts := []do.HarborArtifact{
		{Path: "result.json", URL: storage.url("/result.json"), SizeBytes: 12, SHA256: hex.EncodeToString(sum[:])},
		{Path: "fix-login-bug__a1b2/agent/trajectory.json", URL: storage.url("/trajectory.json"), SizeBytes: 12},
	}
	out := t.TempDir()

	for round := 0; round < 2; round++ {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().ListArtifacts("job-1").Return(artifacts, nil)
			config.Args = []string{"job-1"}
			config.Doit.Set(config.NS, doctl.ArgHarborOut, out)
			require.NoError(t, RunHarborJobDownload(config))
		})
	}
	b, err := os.ReadFile(filepath.Join(out, "job-1", "fix-login-bug__a1b2", "agent", "trajectory.json"))
	require.NoError(t, err)
	assert.Equal(t, `{"steps":[]}`, string(b))
	assert.Equal(t, 1, storage.gets["/result.json"], "a second run skips files already downloaded")

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.harbor.EXPECT().ListArtifacts("job-1").Return([]do.HarborArtifact{{Path: "../../escape", URL: storage.url("/result.json")}}, nil)
		config.Args = []string{"job-1"}
		config.Doit.Set(config.NS, doctl.ArgHarborOut, out)
		err := RunHarborJobDownload(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsafe path")
	})
}

func TestRunHarborDatasetPublish(t *testing.T) {
	noHarborWaiting(t)
	storage := newFakeHarborStorage(t)

	dir := filepath.Join(t.TempDir(), "backend-bugs")
	writeHarborFixture(t, dir, map[string]string{
		"dataset.toml": `[dataset]
name = "my-org/backend-bugs"
description = "Backend bugs"

[[tasks]]
name = "my-org/fix-login-bug"
digest = "sha256:` + strings.Repeat("a", 64) + `"

[[tasks]]
name = "already-published"

[[files]]
path = "metric.py"
`,
		"metric.py":                             "def metric(): pass\n",
		"tasks/fix-login-bug/task.toml":         "[task]\nname = \"my-org/fix-login-bug\"\n",
		"tasks/fix-login-bug/instruction.md":    "Fix it.",
		"tasks/fix-login-bug/tests/test.sh":     "#!/bin/sh\n",
		"tasks/unlisted/task.toml":              "[task]\nname = \"unlisted\"\n",
		"tasks/unlisted/environment/Dockerfile": "FROM alpine\n",
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.harbor.EXPECT().StartTaskUpload("fix-login-bug", gomock.Any()).Return(&do.HarborTaskUploadStart{UploadUUID: "up-1", UploadURL: storage.url("/blob")}, nil)
		tm.harbor.EXPECT().CompleteTaskUpload("fix-login-bug", "up-1").Return(&do.HarborTaskUpload{UploadUUID: "up-1", State: do.HarborUploadValidating}, nil)
		tm.harbor.EXPECT().GetTaskUpload("fix-login-bug", "up-1").Return(&do.HarborTaskUpload{UploadUUID: "up-1", State: do.HarborUploadAccepted}, nil)
		tm.harbor.EXPECT().PutDataset("backend-bugs", &do.HarborDatasetRequest{
			Description: "Backend bugs",
			Tasks:       []string{"fix-login-bug", "already-published"},
		}).Return(&do.HarborDataset{Name: "backend-bugs"}, nil)

		config.Args = []string{dir}
		config.Doit.Set(config.NS, doctl.ArgWaitTimeout, time.Minute)
		config.Doit.Set(config.NS, doctl.ArgHarborPollInterval, time.Second)
		require.NoError(t, RunHarborDatasetPublish(config))
	})

	plan, err := planHarborDataset(dir, "")
	require.NoError(t, err)
	assert.Len(t, plan.Warnings, 4, "org prefix of the dataset, of a task, ignored digests, and dataset files")
}

func TestHarborSafePath(t *testing.T) {
	dir := t.TempDir()
	for _, bad := range []string{"", ".", "..", "../x", "/etc/passwd", "a/../../x"} {
		_, err := harborSafePath(dir, bad)
		assert.Error(t, err, bad)
	}
	got, err := harborSafePath(dir, "a/b/../c")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "a", "c"), filepath.Clean(got))
}
