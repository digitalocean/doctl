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
		case "/v1/eval-harness/jobs":
			fmt.Fprint(w, `{"job_uuid":"job-1","status":"queued","n_trials":3,"tasks":[{"name":"t","digest":"sha256:cd"}]}`)
		case "/v1/eval-harness/datasets/ds":
			fmt.Fprint(w, `{"dataset":{"name":"ds","tasks":["a",{"name":"b"}]}}`)
		case "/v1/eval-harness/jobs/job-1/artifacts":
			fmt.Fprint(w, `{"url":"https://spaces/job.zip","expires_at":"2026-11-20T12:00:00Z","size_bytes":42}`)
		}
	})

	var job HarborJob
	require.NoError(t, json.Unmarshal([]byte(`{"uuid":"job-1","status":"running","n_total":3,"n_completed":1}`), &job))
	assert.Equal(t, "job-1", job.JobUUID, "uuid is accepted for job_uuid")
	assert.Equal(t, 3, job.NTrials, "n_total is accepted for n_trials")

	var report HarborLintReport
	require.NoError(t, json.Unmarshal([]byte(`{"passed":false,"findings":[{"rule":"r","severity":"error"}],"harbor_extra":1}`), &report))
	require.NotNil(t, report.Passed)
	assert.False(t, *report.Passed)
	out, err := json.Marshal(report)
	require.NoError(t, err)
	assert.Contains(t, string(out), "harbor_extra", "the lint report is passed through unchanged")

	created, err := svc.CreateJob(&HarborJobRequest{JobName: "nightly", Datasets: []HarborJobDataset{{Name: "ds", TaskNames: []string{"t"}}}, Agents: []HarborJobAgent{{Name: "codex", ModelName: "m"}}, NAttempts: 3, NConcurrentTrials: 4, Retry: HarborJobRetry{MaxRetries: 1}}, "key-1")
	require.NoError(t, err)
	assert.Equal(t, "job-1", created.JobUUID)

	ds, err := svc.GetDataset("ds")
	require.NoError(t, err)
	assert.Equal(t, []HarborDatasetTask{{Name: "a"}, {Name: "b"}}, ds.Tasks)

	link, err := svc.GetJobArtifacts("job-1")
	require.NoError(t, err)
	assert.Equal(t, "https://spaces/job.zip", link.URL)
	assert.Equal(t, int64(42), link.SizeBytes)
	require.NotNil(t, link.ExpiresAt)

	assert.Equal(t, `POST /v1/eval-harness/jobs key-1 {"job_name":"nightly","datasets":[{"name":"ds","task_names":["t"]}],"agents":[{"name":"codex","model_name":"m"}],"n_attempts":3,"n_concurrent_trials":4,"retry":{"max_retries":1},"dry_run":false}`+"\n", got[0])
}

func TestHarborServiceListPaginates(t *testing.T) {
	svc := newHarborTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "1":
			fmt.Fprintf(w, `{"datasets":[{"name":"a"}],"links":{"pages":{"last":"http://x/v1/eval-harness/datasets?page=2"}},"meta":{"total":2}}`)
		case "2":
			fmt.Fprint(w, `{"datasets":[{"name":"b"}],"meta":{"total":2}}`)
		}
		assert.Equal(t, "login", r.URL.Query().Get("q"))
	})

	datasets, err := svc.ListDatasets("login")
	require.NoError(t, err)
	names := []string{}
	for _, dataset := range datasets {
		names = append(names, dataset.Name)
	}
	assert.ElementsMatch(t, []string{"a", "b"}, names)
}

func TestHarborServiceListJobs(t *testing.T) {
	var uris []string
	svc := newHarborTestService(t, func(w http.ResponseWriter, r *http.Request) {
		uris = append(uris, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page_token") {
		case "":
			fmt.Fprint(w, `{"jobs":[{"uuid":"job-a","name":"one","status":"successful","n_total":2,"n_completed":2,"created_at":"2026-11-19T00:00:00Z"}],"next_page_token":"next"}`)
		case "next":
			fmt.Fprint(w, `{"jobs":[{"uuid":"job-b","name":"two","status":"successful","n_total":1,"n_completed":1,"n_errored":1,"created_at":"2026-11-18T00:00:00Z"}]}`)
		default:
			t.Errorf("unexpected page_token %q", r.URL.Query().Get("page_token"))
		}
		assert.Equal(t, "100", r.URL.Query().Get("limit"))
		assert.Equal(t, "successful", r.URL.Query().Get("status"))
		assert.Empty(t, r.URL.Query().Get("page"))
		assert.Empty(t, r.URL.Query().Get("per_page"))
	})

	jobs, err := svc.ListJobs("successful")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "job-a", jobs[0].JobUUID)
	assert.Equal(t, 2, jobs[0].NTrials)
	assert.Equal(t, "job-b", jobs[1].JobUUID)
	assert.Equal(t, 1, jobs[1].NErrored)
	assert.Equal(t, []string{
		"/v1/eval-harness/jobs?limit=100&status=successful",
		"/v1/eval-harness/jobs?limit=100&page_token=next&status=successful",
	}, uris)
}

func TestHarborServiceDatasetListAndArchive(t *testing.T) {
	var got string
	svc := newHarborTestService(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/eval-harness/datasets":
			assert.Equal(t, "bugs", r.URL.Query().Get("q"))
			fmt.Fprint(w, `{"datasets":[{"name":"backend-bugs","digest":"sha256:ab"}]}`)
		case "/v1/eval-harness/datasets/backend-bugs/archive":
			fmt.Fprint(w, `{"url":"https://spaces/archive","expires_at":"2026-11-20T12:00:00Z"}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})

	datasets, err := svc.ListDatasets("bugs")
	require.NoError(t, err)
	require.Len(t, datasets, 1)
	assert.Equal(t, "backend-bugs", datasets[0].Name)
	assert.Contains(t, got, "q=bugs")

	archive, err := svc.GetDatasetArchive("backend-bugs")
	require.NoError(t, err)
	assert.Equal(t, "https://spaces/archive", archive.URL)
	require.NotNil(t, archive.ExpiresAt)
}

func TestHarborServiceDeleteJob(t *testing.T) {
	var got string
	svc := newHarborTestService(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	require.NoError(t, svc.DeleteJob("job-1"))
	assert.Equal(t, "DELETE /v1/eval-harness/jobs/job-1", got)
}

func TestHarborServiceErrors(t *testing.T) {
	svc := newHarborTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"id":"conflict","message":"dataset is in use"}`)
	})
	err := svc.DeleteDataset("ds")
	var apiErr *godo.ErrorResponse
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Response.StatusCode)
	assert.Contains(t, err.Error(), "dataset is in use")
}
