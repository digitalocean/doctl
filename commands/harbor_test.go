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
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

func basicHarborDataset(t *testing.T) string {
	dir := filepath.Join(t.TempDir(), "backend-bugs")
	writeHarborFixture(t, dir, map[string]string{
		"dataset.toml":                               "[dataset]\nname = \"backend-bugs\"\n\n[[tasks]]\nname = \"fix-login-bug\"\n",
		"tasks/fix-login-bug/task.toml":              "[task]\nname = \"fix-login-bug\"\n",
		"tasks/fix-login-bug/instruction.md":         "Fix it.",
		"tasks/fix-login-bug/environment/Dockerfile": "FROM alpine\n",
		"tasks/fix-login-bug/tests/test.sh":          "#!/bin/sh\nexit 0\n",
		"tasks/fix-login-bug/notes.txt":              "not part of the task\n",
	})
	return dir
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

func TestHarborCommandTree(t *testing.T) {
	root := GenAI()
	for path, want := range map[string][]string{
		"harbor":         {"dataset", "job", "run"},
		"harbor dataset": {"delete", "get", "list", "publish", "pull"},
		"harbor job":     {"cancel", "delete", "download", "list"},
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

func TestRunHarborDatasetUpload(t *testing.T) {
	noHarborWaiting(t)

	t.Run("uploads one archive and reports ready", func(t *testing.T) {
		storage := newFakeHarborStorage(t)
		dir := basicHarborDataset(t)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			var announced *do.HarborDatasetUploadRequest
			tm.harbor.EXPECT().StartDatasetUpload("backend-bugs", gomock.Any()).DoAndReturn(
				func(_ string, req *do.HarborDatasetUploadRequest) (*do.HarborDatasetUploadStart, error) {
					announced = req
					return &do.HarborDatasetUploadStart{
						Upload:    &do.HarborDatasetUpload{UUID: "up-1", Status: do.HarborDatasetUploading},
						UploadURL: &do.HarborDatasetUploadURL{URL: storage.url("/blob")},
					}, nil
				})
			tm.harbor.EXPECT().CompleteDatasetUpload("backend-bugs", "up-1").Return(&do.HarborDatasetUpload{
				UUID: "up-1", Status: do.HarborDatasetValidating,
			}, nil)
			tm.harbor.EXPECT().GetDatasetUpload("backend-bugs", "up-1").Return(&do.HarborDatasetUpload{
				UUID: "up-1", Status: do.HarborDatasetReady, Digest: "sha256:ready",
				Findings: []do.HarborLintFinding{{Task: "fix-login-bug", File: "environment/Dockerfile", Rule: "unpinned-image", Severity: "warning", Message: "pin it"}},
			}, nil)
			tm.harbor.EXPECT().GetDataset("backend-bugs").Return(&do.HarborDataset{
				Name: "backend-bugs", Status: do.HarborDatasetReady, Digest: "sha256:ready",
				Tasks: []do.HarborDatasetTask{{Name: "fix-login-bug"}},
			}, nil)

			config.Out = &bytes.Buffer{}
			config.Args = []string{dir}
			config.Doit.Set(config.NS, doctl.ArgWaitTimeout, time.Minute)
			config.Doit.Set(config.NS, doctl.ArgHarborPollInterval, time.Millisecond)
			require.NoError(t, RunHarborDatasetPublish(config))

			require.NotNil(t, announced)
			assert.Regexp(t, `^sha256:[0-9a-f]{64}$`, announced.Digest)
			assert.Equal(t, int64(len(storage.puts["/blob"])), announced.SizeBytes)
			sum := sha256.Sum256(storage.puts["/blob"])
			assert.Equal(t, "sha256:"+hex.EncodeToString(sum[:]), announced.Digest)
			names := tarNames(t, storage.puts["/blob"])
			assert.Contains(t, names, "dataset.toml")
			assert.Contains(t, names, "tasks/fix-login-bug/task.toml")
			assert.NotContains(t, names, "tasks/fix-login-bug/notes.txt")
		})
	})

	t.Run("a failed upload fails the command", func(t *testing.T) {
		storage := newFakeHarborStorage(t)
		dir := basicHarborDataset(t)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().StartDatasetUpload(gomock.Any(), gomock.Any()).Return(&do.HarborDatasetUploadStart{
				Upload:    &do.HarborDatasetUpload{UUID: "up-2", Status: do.HarborDatasetUploading},
				UploadURL: &do.HarborDatasetUploadURL{URL: storage.url("/blob")},
			}, nil)
			tm.harbor.EXPECT().CompleteDatasetUpload("backend-bugs", "up-2").Return(&do.HarborDatasetUpload{
				UUID: "up-2", Status: do.HarborDatasetFailed, Error: "dataset.toml: dataset.name is required",
			}, nil)
			config.Args = []string{dir}
			config.Doit.Set(config.NS, doctl.ArgWaitTimeout, time.Minute)
			config.Doit.Set(config.NS, doctl.ArgHarborPollInterval, time.Millisecond)
			err := RunHarborDatasetPublish(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "dataset.name is required")
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

func TestBuildHarborJobRequest(t *testing.T) {
	tests := []struct {
		name         string
		set          map[string]any
		want         []do.HarborJobAgent
		wantDatasets []do.HarborJobDataset
		errMsg       string
	}{
		{name: "pairs by position", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"claude-code", "codex"}, doctl.ArgHarborModel: []string{"m1", "m2"}},
			want: []do.HarborJobAgent{{Name: "claude-code", ModelName: "m1"}, {Name: "codex", ModelName: "m2"}}},
		{name: "one model for every agent", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"claude-code", "aider"}, doctl.ArgHarborModel: []string{"m1"}},
			want: []do.HarborJobAgent{{Name: "claude-code", ModelName: "m1"}, {Name: "aider", ModelName: "m1"}}},
		{name: "one agent for every model", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"codex"}, doctl.ArgHarborModel: []string{"m1", "dedicated:my-di:llama"}},
			want: []do.HarborJobAgent{{Name: "codex", ModelName: "m1"}, {Name: "codex", ModelName: "dedicated:my-di:llama"}}},
		{name: "tasks narrow the dataset", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborTask: []string{"a", "b"}, doctl.ArgHarborAgent: []string{"codex"}, doctl.ArgHarborModel: []string{"m1"}},
			want: []do.HarborJobAgent{{Name: "codex", ModelName: "m1"}}, wantDatasets: []do.HarborJobDataset{{Name: "ds", TaskNames: []string{"a", "b"}}}},
		{name: "mismatched counts", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"a1", "a2"}, doctl.ArgHarborModel: []string{"m1", "m2", "m3"}}, errMsg: "one model per agent"},
		{name: "no dataset", set: map[string]any{doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}}, errMsg: "choose the dataset to run"},
		{name: "tasks without a dataset", set: map[string]any{doctl.ArgHarborTask: []string{"a"}, doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}}, errMsg: "name the dataset"},
		{name: "invalid dataset name", set: map[string]any{doctl.ArgHarborDataset: "My Set", doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}}, errMsg: "not a valid dataset name"},
		{name: "no agent", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborModel: []string{"m1"}}, errMsg: "at least one agent"},
		{name: "no model", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"a1"}}, errMsg: "--model"},
		{name: "attempts limit", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}, doctl.ArgHarborAttempts: 11}, errMsg: "between 1 and 10"},
		{name: "concurrency limit", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}, doctl.ArgHarborConcurrency: 51}, errMsg: "between 1 and 50"},
		{name: "invalid task name", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborTask: []string{"My Task"}, doctl.ArgHarborAgent: []string{"a1"}, doctl.ArgHarborModel: []string{"m1"}}, errMsg: "not a valid task name"},
		{name: "duplicate pair", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"a1", "a1"}, doctl.ArgHarborModel: []string{"m1"}}, errMsg: "more than once"},
		{name: "too many pairs", set: map[string]any{doctl.ArgHarborDataset: "ds", doctl.ArgHarborAgent: []string{"codex"}, doctl.ArgHarborModel: manyHarborModels(21)}, errMsg: "at most 20"},
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
				wantDatasets := tt.wantDatasets
				if wantDatasets == nil {
					wantDatasets = []do.HarborJobDataset{{Name: "ds"}}
				}
				assert.Equal(t, wantDatasets, req.Datasets)
			})
		})
	}
}

func TestBuildHarborJobRequestFromFile(t *testing.T) {
	writeJob := func(t *testing.T, body string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "job.yaml")
		require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
		return p
	}

	t.Run("a job file supplies the request", func(t *testing.T) {
		path := writeJob(t, `
job_name: nightly
n_attempts: 3
n_concurrent_trials: 2
dry_run: true
jobs_dir: jobs
retry:
  max_retries: 0
datasets:
  - name: bug-fixes
    task_names: [fix-login-bug]
    exclude_task_names: [skip-me]
    n_tasks: 5
agents:
  - name: claude-code
    model_name: anthropic-claude-sonnet-4
    kwargs:
      temperature: 0
`)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgHarborJobConfig, path)
			req, key, err := buildHarborJobRequest(config)
			require.NoError(t, err)
			assert.NotEmpty(t, key)
			assert.Equal(t, &do.HarborJobRequest{
				JobName:           "nightly",
				Datasets:          []do.HarborJobDataset{{Name: "bug-fixes", TaskNames: []string{"fix-login-bug"}, ExcludeTaskNames: []string{"skip-me"}, NTasks: 5}},
				Agents:            []do.HarborJobAgent{{Name: "claude-code", ModelName: "anthropic-claude-sonnet-4", Kwargs: map[string]any{"temperature": 0}}},
				NAttempts:         3,
				NConcurrentTrials: 2,
				DryRun:            true,
			}, req)
		})
	})

	t.Run("flags replace the matching fields", func(t *testing.T) {
		path := writeJob(t, `
job_name: from-file
datasets:
  - name: from-file
agents:
  - name: codex
    model_name: from-file
`)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgHarborJobConfig, path)
			config.Doit.Set(config.NS, doctl.ArgHarborDataset, "from-flag")
			config.Doit.Set(config.NS, doctl.ArgHarborAgent, []string{"claude-code"})
			config.Doit.Set(config.NS, doctl.ArgHarborModel, []string{"from-flag"})
			config.Doit.Set(config.NS, doctl.ArgHarborName, "from-flag")
			req, _, err := buildHarborJobRequest(config)
			require.NoError(t, err)
			assert.Equal(t, "from-flag", req.JobName)
			assert.Equal(t, []do.HarborJobDataset{{Name: "from-flag"}}, req.Datasets)
			assert.Equal(t, []do.HarborJobAgent{{Name: "claude-code", ModelName: "from-flag"}}, req.Agents)
			assert.Equal(t, 1, req.NAttempts)
			assert.Equal(t, 4, req.NConcurrentTrials)
		})
	})

	t.Run("a local task path is rejected", func(t *testing.T) {
		path := writeJob(t, `
tasks:
  - path: ssh-key-pair
agents:
  - name: oracle
environment:
  type: docker
`)
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgHarborJobConfig, path)
			_, _, err := buildHarborJobRequest(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "datasets[].name")
			assert.Contains(t, err.Error(), "environment")
		})
	})

	t.Run("json is accepted", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "job.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"datasets":[{"name":"bug-fixes"}],"agents":[{"name":"codex","model_name":"m1"}]}`), 0o644))
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgHarborJobConfig, path)
			req, _, err := buildHarborJobRequest(config)
			require.NoError(t, err)
			assert.Equal(t, "bug-fixes", req.Datasets[0].Name)
			assert.Equal(t, "codex", req.Agents[0].Name)
		})
	})
}

func manyHarborModels(n int) []string {
	models := make([]string, n)
	for i := range models {
		models[i] = fmt.Sprintf("model-%d", i)
	}
	return models
}

func TestRunHarborRun(t *testing.T) {
	noHarborWaiting(t)

	setRun := func(config *CmdConfig) {
		config.Doit.Set(config.NS, doctl.ArgHarborDataset, "bug-fixes")
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
				Datasets:          []do.HarborJobDataset{{Name: "bug-fixes", TaskNames: []string{"fix-login-bug"}}},
				Agents:            []do.HarborJobAgent{{Name: "claude-code", ModelName: "anthropic-claude-sonnet-4"}},
				NAttempts:         3,
				NConcurrentTrials: 4,
				Retry:             do.HarborJobRetry{MaxRetries: 1},
			}
			tm.harbor.EXPECT().CreateJob(want, "ci-run-42").Return(&do.HarborJob{JobUUID: "job-1", Status: do.HarborJobQueued, NTrials: 3}, nil)
			setRun(config)
			config.Doit.Set(config.NS, doctl.ArgHarborIdempotencyKey, "ci-run-42")
			require.NoError(t, RunHarborRun(config))
		})
	})

}

func TestRunHarborJobDownload(t *testing.T) {
	noHarborWaiting(t)
	storage := newFakeHarborStorage(t)
	archive := harborTestZip(t, map[string]string{
		"result.json": `{"reward":1}`,
		"fix-login-bug__a1b2/agent/trajectory.json": `{"steps":[]}`,
	})
	storage.files["/job.zip"] = archive
	link := &do.HarborPresignedURL{URL: storage.url("/job.zip"), SizeBytes: int64(len(archive))}
	out := t.TempDir()

	for round := 0; round < 2; round++ {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().GetJobArtifacts("job-1").Return(link, nil)
			config.Args = []string{"job-1"}
			config.Doit.Set(config.NS, doctl.ArgHarborOut, out)
			require.NoError(t, RunHarborJobDownload(config))
		})
	}
	b, err := os.ReadFile(filepath.Join(out, "job-1", "fix-login-bug__a1b2", "agent", "trajectory.json"))
	require.NoError(t, err)
	assert.Equal(t, `{"steps":[]}`, string(b))
	assert.Equal(t, 2, storage.gets["/job.zip"], "each run downloads the one results archive")

	bad := harborTestZip(t, map[string]string{"../../escape": "nope"})
	storage.files["/bad.zip"] = bad
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.harbor.EXPECT().GetJobArtifacts("job-1").Return(&do.HarborPresignedURL{URL: storage.url("/bad.zip"), SizeBytes: int64(len(bad))}, nil)
		config.Args = []string{"job-1"}
		config.Doit.Set(config.NS, doctl.ArgHarborOut, out)
		err := RunHarborJobDownload(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsafe path")
	})
	_, err = os.Stat(filepath.Join(filepath.Dir(out), "escape"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func harborTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = io.WriteString(w, body)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestRunHarborDatasetPublish(t *testing.T) {
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
		config.Args = []string{dir}
		err := RunHarborDatasetPublish(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already-published")
	})

	plan, err := planHarborDataset(dir, "")
	require.NoError(t, err)
	assert.Len(t, plan.Warnings, 4, "org prefix of the dataset, of a task, ignored digests, and dataset files")
}

func TestRunHarborDatasetList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.harbor.EXPECT().ListDatasets("bugs").Return([]do.HarborDataset{{Name: "backend-bugs"}}, nil)
		config.Doit.Set(config.NS, doctl.ArgHarborQuery, " bugs ")
		require.NoError(t, RunHarborDatasetList(config))
	})
}

func TestRunHarborDatasetPull(t *testing.T) {
	plan, err := planHarborDataset(basicHarborDataset(t), "")
	require.NoError(t, err)
	arch, err := packHarborDataset(plan)
	require.NoError(t, err)
	t.Cleanup(arch.Close)
	blob, err := os.ReadFile(arch.Path)
	require.NoError(t, err)

	storage := newFakeHarborStorage(t)
	storage.files["/archive"] = blob

	t.Run("unpacks and checks the archive digest", func(t *testing.T) {
		out := t.TempDir()
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().GetDataset("backend-bugs").Return(&do.HarborDataset{Name: "backend-bugs", Digest: arch.Digest, SizeBytes: arch.Size}, nil)
			tm.harbor.EXPECT().GetDatasetArchive("backend-bugs").Return(&do.HarborPresignedURL{URL: storage.url("/archive")}, nil)
			config.Args = []string{"backend-bugs"}
			config.Doit.Set(config.NS, doctl.ArgHarborOut, out)
			require.NoError(t, RunHarborDatasetPull(config))
		})
		b, err := os.ReadFile(filepath.Join(out, "backend-bugs", "dataset.toml"))
		require.NoError(t, err)
		assert.Contains(t, string(b), `name = "backend-bugs"`)
		_, err = os.Stat(filepath.Join(out, "backend-bugs", "tasks", "fix-login-bug", "task.toml"))
		require.NoError(t, err)
	})

	t.Run("refuses an archive that does not match the digest", func(t *testing.T) {
		out := t.TempDir()
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.harbor.EXPECT().GetDataset("backend-bugs").Return(&do.HarborDataset{Name: "backend-bugs", Digest: "sha256:" + strings.Repeat("0", 64), SizeBytes: arch.Size}, nil)
			tm.harbor.EXPECT().GetDatasetArchive("backend-bugs").Return(&do.HarborPresignedURL{URL: storage.url("/archive")}, nil)
			config.Args = []string{"backend-bugs"}
			config.Doit.Set(config.NS, doctl.ArgHarborOut, out)
			err := RunHarborDatasetPull(config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "checksum mismatch")
		})
		_, err := os.Stat(filepath.Join(out, "backend-bugs"))
		assert.True(t, os.IsNotExist(err))
	})
}

func TestRunHarborJobList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.harbor.EXPECT().ListJobs("running").Return([]do.HarborJob{{JobUUID: "job-1", Name: "nightly", Status: do.HarborJobRunning, NTrials: 2, NCompleted: 1}}, nil)
		config.Doit.Set(config.NS, doctl.ArgHarborStatus, " running ")
		require.NoError(t, RunHarborJobList(config))
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgHarborStatus, "done")
		err := RunHarborJobList(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown job status")
	})
}

func TestRunHarborJobDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.harbor.EXPECT().DeleteJob("job-1").Return(nil)
		config.Args = []string{"job-1"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		require.NoError(t, RunHarborJobDelete(config))
	})
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
