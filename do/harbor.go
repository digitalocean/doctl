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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
)

// harborBasePath is the root of the Managed Harbor API. It has no godo client
// yet, so every route is built from this prefix.
const harborBasePath = "/v1/eval-harness"

// Upload states.
const (
	HarborUploadUploading  = "uploading"
	HarborUploadValidating = "validating"
	HarborUploadBuilding   = "building"
	HarborUploadAccepted   = "accepted"
	HarborUploadRejected   = "rejected"
	HarborUploadSuperseded = "superseded"
)

// HarborUploadTerminal reports whether an upload in state will not change again.
func HarborUploadTerminal(state string) bool {
	switch state {
	case HarborUploadAccepted, HarborUploadRejected, HarborUploadSuperseded:
		return true
	}
	return false
}

// Job statuses.
const (
	HarborJobQueued              = "queued"
	HarborJobRunning             = "running"
	HarborJobCancelling          = "cancelling"
	HarborJobCancelled           = "cancelled"
	HarborJobSuccessful          = "successful"
	HarborJobPartiallySuccessful = "partially_successful"
	HarborJobFailed              = "failed"
)

// HarborJobTerminal reports whether a job in status will not change again.
func HarborJobTerminal(status string) bool {
	switch status {
	case HarborJobCancelled, HarborJobSuccessful, HarborJobPartiallySuccessful, HarborJobFailed:
		return true
	}
	return false
}

// HarborLintFinding is one result of server-side lint. Dataset findings use
// file and task; task findings use path.
type HarborLintFinding struct {
	Rule     string `json:"rule,omitempty"`
	Severity string `json:"severity,omitempty"`
	Path     string `json:"path,omitempty"`
	File     string `json:"file,omitempty"`
	Task     string `json:"task,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message,omitempty"`
}

// HarborLintReport is the lint result stored on an upload and its task. The
// report is passed through unchanged in JSON output; Findings and Passed are
// read from it on a best-effort basis for tables.
type HarborLintReport struct {
	Passed   *bool
	Findings []HarborLintFinding
	raw      json.RawMessage
}

func (r *HarborLintReport) UnmarshalJSON(b []byte) error {
	r.raw = append(json.RawMessage(nil), b...)
	var known struct {
		Passed   *bool               `json:"passed"`
		Findings []HarborLintFinding `json:"findings"`
	}
	if err := json.Unmarshal(b, &known); err == nil {
		r.Passed, r.Findings = known.Passed, known.Findings
	}
	return nil
}

func (r HarborLintReport) MarshalJSON() ([]byte, error) {
	if len(r.raw) > 0 {
		return r.raw, nil
	}
	return json.Marshal(struct {
		Passed   *bool               `json:"passed,omitempty"`
		Findings []HarborLintFinding `json:"findings"`
	}{r.Passed, r.Findings})
}

// HarborPullResult is what doctl reports for a downloaded dataset.
type HarborPullResult struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
	Path   string `json:"path"`
	Files  int    `json:"files"`
}

// HarborDownloadResult is what doctl reports for a downloaded job.
type HarborDownloadResult struct {
	JobUUID    string `json:"job_uuid"`
	Path       string `json:"path"`
	Downloaded int    `json:"downloaded"`
	Skipped    int    `json:"skipped"`
	Bytes      int64  `json:"bytes"`
}

// HarborPresignedURL is a short-lived download link.
type HarborPresignedURL struct {
	URL       string     `json:"url"`
	Digest    string     `json:"digest,omitempty"`
	SizeBytes int64      `json:"size_bytes,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (p *HarborPresignedURL) UnmarshalJSON(b []byte) error {
	type plain HarborPresignedURL
	var v struct {
		plain
		DownloadURL string `json:"download_url"`
		ArchiveURL  string `json:"archive_url"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*p = HarborPresignedURL(v.plain)
	p.URL = harborFirstNonEmpty(p.URL, v.DownloadURL, v.ArchiveURL)
	return nil
}

// HarborDatasetTask is a task inside a dataset.
type HarborDatasetTask struct {
	Name      string `json:"name"`
	Digest    string `json:"digest,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

// UnmarshalJSON accepts a bare task name as well as an object.
func (t *HarborDatasetTask) UnmarshalJSON(b []byte) error {
	var name string
	if err := json.Unmarshal(b, &name); err == nil {
		*t = HarborDatasetTask{Name: name}
		return nil
	}
	type plain HarborDatasetTask
	return json.Unmarshal(b, (*plain)(t))
}

// Dataset upload states. Validation finishes inside the complete call, which
// returns ready or failed.
const (
	HarborDatasetUploading  = "uploading"
	HarborDatasetValidating = "validating"
	HarborDatasetReady      = "ready"
	HarborDatasetFailed     = "failed"
	HarborDatasetArchived   = "archived"
)

// HarborDatasetTerminal reports whether a dataset upload will not change again.
func HarborDatasetTerminal(status string) bool {
	switch status {
	case HarborDatasetReady, HarborDatasetFailed, HarborDatasetArchived:
		return true
	}
	return false
}

// HarborDataset is the ready version of a dataset.
type HarborDataset struct {
	UUID                 string              `json:"uuid,omitempty"`
	Name                 string              `json:"name"`
	Description          string              `json:"description,omitempty"`
	Digest               string              `json:"digest,omitempty"`
	SizeBytes            int64               `json:"size_bytes,omitempty"`
	Status               string              `json:"status,omitempty"`
	Tasks                []HarborDatasetTask `json:"tasks,omitempty"`
	HasMetric            bool                `json:"has_metric,omitempty"`
	PublishedByUserEmail string              `json:"published_by_user_email,omitempty"`
	PublishedAt          *time.Time          `json:"published_at,omitempty"`
	CreatedAt            *time.Time          `json:"created_at,omitempty"`
	UpdatedAt            *time.Time          `json:"updated_at,omitempty"`
}

// HarborDatasetUploadRequest announces a dataset archive. Digest is the
// SHA-256 of the .tar.gz bytes, not the hash of the files inside it.
type HarborDatasetUploadRequest struct {
	Digest    string `json:"digest"`
	SizeBytes int64  `json:"size_bytes"`
}

// HarborDatasetUploadURL is the presigned PUT for a dataset archive.
type HarborDatasetUploadURL struct {
	Method    string     `json:"method"`
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// HarborDatasetUpload is one upload of a dataset archive.
type HarborDatasetUpload struct {
	UUID      string              `json:"uuid"`
	Dataset   string              `json:"dataset,omitempty"`
	Status    string              `json:"status"`
	Digest    string              `json:"digest,omitempty"`
	SizeBytes int64               `json:"size_bytes,omitempty"`
	Error     string              `json:"error,omitempty"`
	Findings  []HarborLintFinding `json:"findings,omitempty"`
	CreatedAt *time.Time          `json:"created_at,omitempty"`
	UpdatedAt *time.Time          `json:"updated_at,omitempty"`
}

// HarborDatasetUploadStart is the reply to a dataset upload request.
type HarborDatasetUploadStart struct {
	Unchanged bool                    `json:"unchanged"`
	Dataset   *HarborDataset          `json:"dataset,omitempty"`
	Upload    *HarborDatasetUpload    `json:"upload,omitempty"`
	UploadURL *HarborDatasetUploadURL `json:"upload_url,omitempty"`
}

// HarborJobAgent is one agent and model pair in a job.
type HarborJobAgent struct {
	Name      string         `json:"name"`
	Version   string         `json:"version,omitempty"`
	ModelName string         `json:"model_name"`
	Kwargs    map[string]any `json:"kwargs,omitempty"`
}

// HarborJobDataset is the dataset a job runs, optionally narrowed to some of
// its tasks. A job runs exactly one dataset.
type HarborJobDataset struct {
	Name             string   `json:"name"`
	TaskNames        []string `json:"task_names,omitempty"`
	ExcludeTaskNames []string `json:"exclude_task_names,omitempty"`
	NTasks           int      `json:"n_tasks,omitempty"`
}

// HarborJobRetry is the trial retry policy.
type HarborJobRetry struct {
	MaxRetries int `json:"max_retries"`
}

// HarborJobRequest submits a job.
type HarborJobRequest struct {
	JobName           string             `json:"job_name,omitempty"`
	Datasets          []HarborJobDataset `json:"datasets"`
	Agents            []HarborJobAgent   `json:"agents"`
	NAttempts         int                `json:"n_attempts"`
	NConcurrentTrials int                `json:"n_concurrent_trials"`
	Retry             HarborJobRetry     `json:"retry"`
	DryRun            bool               `json:"dry_run"`
}

// HarborJob is a submitted job and its progress.
type HarborJob struct {
	JobUUID            string              `json:"job_uuid"`
	Name               string              `json:"name,omitempty"`
	Status             string              `json:"status"`
	Dataset            string              `json:"dataset,omitempty"`
	NTrials            int                 `json:"n_trials"`
	NCompleted         int                 `json:"n_completed"`
	NErrored           int                 `json:"n_errored"`
	Tasks              []HarborDatasetTask `json:"tasks,omitempty"`
	Agents             []HarborJobAgent    `json:"agents,omitempty"`
	Metrics            json.RawMessage     `json:"metrics,omitempty"`
	ErrorDescription   string              `json:"error_description,omitempty"`
	CreatedByUserEmail string              `json:"created_by_user_email,omitempty"`
	CreatedAt          *time.Time          `json:"created_at,omitempty"`
	StartedAt          *time.Time          `json:"started_at,omitempty"`
	CompletedAt        *time.Time          `json:"completed_at,omitempty"`
	DeadlineAt         *time.Time          `json:"deadline_at,omitempty"`
}

// UnmarshalJSON accepts the column names of the jobs table (uuid, n_total) as
// well as the submit response names (job_uuid, n_trials).
func (j *HarborJob) UnmarshalJSON(b []byte) error {
	type plain HarborJob
	var v struct {
		plain
		UUID   string `json:"uuid"`
		NTotal int    `json:"n_total"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*j = HarborJob(v.plain)
	j.JobUUID = harborFirstNonEmpty(j.JobUUID, v.UUID)
	if j.NTrials == 0 {
		j.NTrials = v.NTotal
	}
	return nil
}

// HarborService is the Managed Harbor API.
type HarborService interface {
	StartDatasetUpload(name string, req *HarborDatasetUploadRequest) (*HarborDatasetUploadStart, error)
	CompleteDatasetUpload(name, uploadUUID string) (*HarborDatasetUpload, error)
	GetDatasetUpload(name, uploadUUID string) (*HarborDatasetUpload, error)
	ListDatasets(query string) ([]HarborDataset, error)
	GetDataset(name string) (*HarborDataset, error)
	GetDatasetArchive(name string) (*HarborPresignedURL, error)
	DeleteDataset(name string) error

	CreateJob(req *HarborJobRequest, idempotencyKey string) (*HarborJob, error)
	ListJobs(status string) ([]HarborJob, error)
	GetJobArtifacts(uuid string) (*HarborPresignedURL, error)
	CancelJob(uuid string) (*HarborJob, error)
	DeleteJob(uuid string) error
}

type harborService struct {
	client   *godo.Client
	basePath string
}

var _ HarborService = &harborService{}

// NewHarborService builds a HarborService on the shared API client, so it uses
// the same token, base URL, and retry policy as every other command. To test
// against a preview deployment, point the client at it with --api-url or
// DIGITALOCEAN_API_URL.
func NewHarborService(client *godo.Client) HarborService {
	return &harborService{client: client, basePath: harborBasePath}
}

func (s *harborService) do(method, path string, body any, header http.Header, out any) (*godo.Response, error) {
	ctx := context.TODO()
	req, err := s.client.NewRequest(ctx, method, s.basePath+path, body)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header[k] = v
	}
	var raw json.RawMessage
	resp, err := s.client.Do(ctx, req, &raw)
	if err != nil {
		return resp, err
	}
	if out != nil && len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp, fmt.Errorf("decoding the %s %s response: %w", method, path, err)
		}
	}
	return resp, nil
}

// harborOne decodes a single resource, which may be wrapped as {"<key>": {...}}
// in the DO style or returned bare.
func harborOne[T any](s *harborService, method, path string, body any, header http.Header, key string) (*T, error) {
	var raw json.RawMessage
	if _, err := s.do(method, path, body, header, &raw); err != nil {
		return nil, err
	}
	v := new(T)
	if len(raw) == 0 {
		return v, nil
	}
	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		if inner, ok := wrapped[key]; ok && len(inner) > 0 && inner[0] == '{' {
			raw = inner
		}
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return nil, fmt.Errorf("decoding the %s %s response: %w", method, path, err)
	}
	return v, nil
}

// harborList fetches every page of a collection returned as {"<key>": [...],
// "links": ..., "meta": ...}, or as a bare array.
func harborList[T any](s *harborService, path string, query url.Values, key string) ([]T, error) {
	si, err := PaginateResp(func(lo *godo.ListOptions) ([]any, *godo.Response, error) {
		q := url.Values{}
		for k, v := range query {
			q[k] = v
		}
		q.Set("page", strconv.Itoa(lo.Page))
		q.Set("per_page", strconv.Itoa(lo.PerPage))

		var raw json.RawMessage
		resp, err := s.do(http.MethodGet, path+"?"+q.Encode(), nil, nil, &raw)
		if err != nil {
			return nil, nil, err
		}

		var items []T
		if len(raw) > 0 && raw[0] == '[' {
			if err := json.Unmarshal(raw, &items); err != nil {
				return nil, nil, fmt.Errorf("decoding the GET %s response: %w", path, err)
			}
		} else if len(raw) > 0 {
			var page map[string]json.RawMessage
			if err := json.Unmarshal(raw, &page); err != nil {
				return nil, nil, fmt.Errorf("decoding the GET %s response: %w", path, err)
			}
			if inner := page[key]; len(inner) > 0 {
				if err := json.Unmarshal(inner, &items); err != nil {
					return nil, nil, fmt.Errorf("decoding the GET %s response: %w", path, err)
				}
			}
			if l := page["links"]; len(l) > 0 {
				resp.Links = &godo.Links{}
				_ = json.Unmarshal(l, resp.Links)
			}
			if m := page["meta"]; len(m) > 0 {
				resp.Meta = &godo.Meta{}
				_ = json.Unmarshal(m, resp.Meta)
			}
		}

		out := make([]any, len(items))
		for i := range items {
			out[i] = items[i]
		}
		return out, resp, nil
	})
	if err != nil {
		return nil, err
	}
	items := make([]T, len(si))
	for i := range si {
		items[i] = si[i].(T)
	}
	return items, nil
}

func harborJobPath(uuid string) string { return "/jobs/" + url.PathEscape(uuid) }

func (s *harborService) StartDatasetUpload(name string, req *HarborDatasetUploadRequest) (*HarborDatasetUploadStart, error) {
	var out HarborDatasetUploadStart
	_, err := s.do(http.MethodPost, "/datasets/"+url.PathEscape(name)+"/uploads", req, nil, &out)
	if err == nil && out.Upload != nil && out.Upload.Dataset == "" {
		out.Upload.Dataset = name
	}
	return &out, err
}

func (s *harborService) CompleteDatasetUpload(name, uploadUUID string) (*HarborDatasetUpload, error) {
	up, err := harborOne[HarborDatasetUpload](s, http.MethodPost, "/datasets/"+url.PathEscape(name)+"/uploads/"+url.PathEscape(uploadUUID)+"/complete", nil, nil, "upload")
	if err == nil && up.UUID == "" {
		up.UUID = uploadUUID
	}
	return up, err
}

func (s *harborService) GetDatasetUpload(name, uploadUUID string) (*HarborDatasetUpload, error) {
	up, err := harborOne[HarborDatasetUpload](s, http.MethodGet, "/datasets/"+url.PathEscape(name)+"/uploads/"+url.PathEscape(uploadUUID), nil, nil, "upload")
	if err == nil && up.UUID == "" {
		up.UUID = uploadUUID
	}
	return up, err
}

func (s *harborService) ListDatasets(query string) ([]HarborDataset, error) {
	q := url.Values{}
	if query != "" {
		q.Set("q", query)
	}
	return harborList[HarborDataset](s, "/datasets", q, "datasets")
}

func (s *harborService) GetDataset(name string) (*HarborDataset, error) {
	return harborOne[HarborDataset](s, http.MethodGet, "/datasets/"+url.PathEscape(name), nil, nil, "dataset")
}

func (s *harborService) GetDatasetArchive(name string) (*HarborPresignedURL, error) {
	return harborOne[HarborPresignedURL](s, http.MethodGet, "/datasets/"+url.PathEscape(name)+"/archive", nil, nil, "archive")
}

func (s *harborService) DeleteDataset(name string) error {
	_, err := s.do(http.MethodDelete, "/datasets/"+url.PathEscape(name), nil, nil, nil)
	return err
}

func (s *harborService) CreateJob(req *HarborJobRequest, idempotencyKey string) (*HarborJob, error) {
	header := http.Header{}
	if idempotencyKey != "" {
		header.Set("Idempotency-Key", idempotencyKey)
	}
	return harborOne[HarborJob](s, http.MethodPost, "/jobs", req, header, "job")
}

// harborJobListLimit is the largest page GET /jobs accepts. ListJobs asks for
// that many and follows next_page_token until the last page.
const harborJobListLimit = 100

func (s *harborService) ListJobs(status string) ([]HarborJob, error) {
	jobs := []HarborJob{}
	token := ""
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(harborJobListLimit))
		if status != "" {
			q.Set("status", status)
		}
		if token != "" {
			q.Set("page_token", token)
		}
		var page struct {
			Jobs          []HarborJob `json:"jobs"`
			NextPageToken string      `json:"next_page_token"`
		}
		if _, err := s.do(http.MethodGet, "/jobs?"+q.Encode(), nil, nil, &page); err != nil {
			return nil, err
		}
		if len(page.Jobs) > 0 {
			jobs = append(jobs, page.Jobs...)
		}
		if page.NextPageToken == "" || page.NextPageToken == token {
			return jobs, nil
		}
		token = page.NextPageToken
	}
}

// GetJobArtifacts returns the presigned URL for a job's results archive. The
// API signs job.zip, whose files sit at the archive root.
func (s *harborService) GetJobArtifacts(uuid string) (*HarborPresignedURL, error) {
	return harborOne[HarborPresignedURL](s, http.MethodGet, harborJobPath(uuid)+"/artifacts", nil, nil, "artifacts")
}

func (s *harborService) CancelJob(uuid string) (*HarborJob, error) {
	job, err := harborOne[HarborJob](s, http.MethodPost, harborJobPath(uuid)+"/cancel", nil, nil, "job")
	if err == nil && job.JobUUID == "" {
		job.JobUUID = uuid
	}
	return job, err
}

func (s *harborService) DeleteJob(uuid string) error {
	_, err := s.do(http.MethodDelete, harborJobPath(uuid), nil, nil, nil)
	return err
}

func harborFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
