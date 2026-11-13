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

package do

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newHarborTestService(t *testing.T, handler http.HandlerFunc) HarborService {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := godo.New(http.DefaultClient, godo.SetBaseURL(server.URL+"/"))
	require.NoError(t, err)
	return NewHarborService(client)
}

func TestHarborServiceRequests(t *testing.T) {
	var got []string
	svc := newHarborTestService(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, fmt.Sprintf("%s %s %s %s", r.Method, r.URL.RequestURI(), r.Header.Get("Idempotency-Key"), body))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/gen-ai/harbor/tasks/fix-login-bug/uploads":
			fmt.Fprint(w, `{"upload_uuid":"up-1","upload_url":"https://spaces/blob"}`)
		case "/v2/gen-ai/harbor/tasks/fix-login-bug/uploads/up-1/complete":
			fmt.Fprint(w, `{"state":"validating"}`)
		case "/v2/gen-ai/harbor/tasks/fix-login-bug":
			fmt.Fprint(w, `{"task":{"name":"fix-login-bug","digest":"sha256:ab","lint_report":{"passed":false,"findings":[{"rule":"r","severity":"error"}],"harbor_extra":1}}}`)
		case "/v2/gen-ai/harbor/jobs":
			fmt.Fprint(w, `{"job_uuid":"job-1","status":"queued","n_trials":3,"tasks":[{"name":"t","digest":"sha256:cd"}]}`)
		case "/v2/gen-ai/harbor/jobs/job-1":
			fmt.Fprint(w, `{"job":{"uuid":"job-1","status":"running","n_total":3,"n_completed":1}}`)
		case "/v2/gen-ai/harbor/datasets/ds":
			fmt.Fprint(w, `{"dataset":{"name":"ds","tasks":["a",{"name":"b"}]}}`)
		case "/v2/gen-ai/harbor/jobs/job-1/artifacts":
			fmt.Fprint(w, `{"artifacts":[{"name":"result.json","download_url":"https://spaces/r"}]}`)
		}
	})

	start, err := svc.StartTaskUpload("fix-login-bug", &HarborTaskUploadRequest{Digest: "sha256:ab", SizeBytes: 10, FileDigests: map[string]string{"task.toml": "sha256:ef"}, Manifest: map[string]any{}})
	require.NoError(t, err)
	assert.Equal(t, &HarborTaskUploadStart{UploadUUID: "up-1", UploadURL: "https://spaces/blob"}, start)

	up, err := svc.CompleteTaskUpload("fix-login-bug", "up-1")
	require.NoError(t, err)
	assert.Equal(t, "up-1", up.UploadUUID)
	assert.Equal(t, HarborUploadValidating, up.State)

	task, err := svc.GetTask("fix-login-bug")
	require.NoError(t, err)
	assert.Equal(t, "sha256:ab", task.Digest)
	require.NotNil(t, task.LintReport.Passed)
	assert.False(t, *task.LintReport.Passed)
	assert.Len(t, task.LintReport.Findings, 1)
	out, err := json.Marshal(task.LintReport)
	require.NoError(t, err)
	assert.Contains(t, string(out), "harbor_extra", "the lint report is passed through unchanged")

	job, err := svc.CreateJob(&HarborJobRequest{Tasks: []string{"t"}, Agents: []HarborJobAgent{{Name: "codex", Model: "m"}}, NAttempts: 3, NConcurrentTrials: 4, Retry: HarborJobRetry{MaxRetries: 1}, TriggerSource: "cli"}, "key-1")
	require.NoError(t, err)
	assert.Equal(t, "job-1", job.JobUUID)

	job, err = svc.GetJob("job-1")
	require.NoError(t, err)
	assert.Equal(t, "job-1", job.JobUUID, "uuid is accepted for job_uuid")
	assert.Equal(t, 3, job.NTrials, "n_total is accepted for n_trials")

	ds, err := svc.GetDataset("ds")
	require.NoError(t, err)
	assert.Equal(t, []HarborDatasetTask{{Name: "a"}, {Name: "b"}}, ds.Tasks)

	artifacts, err := svc.ListArtifacts("job-1")
	require.NoError(t, err)
	assert.Equal(t, []HarborArtifact{{Path: "result.json", URL: "https://spaces/r"}}, artifacts)

	assert.Equal(t, `POST /v2/gen-ai/harbor/tasks/fix-login-bug/uploads  {"digest":"sha256:ab","size_bytes":10,"file_digests":{"task.toml":"sha256:ef"},"manifest":{}}`+"\n", got[0])
	assert.Equal(t, `POST /v2/gen-ai/harbor/jobs key-1 {"tasks":["t"],"agents":[{"name":"codex","model":"m"}],"n_attempts":3,"n_concurrent_trials":4,"retry":{"max_retries":1},"dry_run":false,"trigger_source":"cli"}`+"\n", got[3])
}

func TestHarborServiceListPaginates(t *testing.T) {
	svc := newHarborTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "1":
			fmt.Fprintf(w, `{"tasks":[{"name":"a"}],"links":{"pages":{"last":"http://x/v2/gen-ai/harbor/tasks?page=2"}},"meta":{"total":2}}`)
		case "2":
			fmt.Fprint(w, `{"tasks":[{"name":"b"}],"meta":{"total":2}}`)
		}
		assert.Equal(t, "login", r.URL.Query().Get("q"))
	})

	tasks, err := svc.ListTasks("login")
	require.NoError(t, err)
	names := []string{}
	for _, task := range tasks {
		names = append(names, task.Name)
	}
	assert.ElementsMatch(t, []string{"a", "b"}, names)
}

func TestHarborServiceErrors(t *testing.T) {
	svc := newHarborTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"id":"conflict","message":"task is used by dataset ds"}`)
	})
	err := svc.DeleteTask("t")
	var apiErr *godo.ErrorResponse
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Response.StatusCode)
	assert.Contains(t, err.Error(), "task is used by dataset ds")
}
