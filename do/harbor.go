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
const harborBasePath = "/v2/gen-ai/harbor"

// Upload states.
const (
	HarborUploadUploading  = "uploading"
	HarborUploadValidating = "validating"
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

// HarborLintFinding is one result of server-side lint.
type HarborLintFinding struct {
	Rule     string `json:"rule,omitempty"`
	Severity string `json:"severity,omitempty"`
	Path     string `json:"path,omitempty"`
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

// HarborTask is a published task: its current archive and lint report.
type HarborTask struct {
	Name                 string            `json:"name"`
	Description          string            `json:"description,omitempty"`
	Keywords             []string          `json:"keywords,omitempty"`
	Digest               string            `json:"digest,omitempty"`
	ArchiveSizeBytes     int64             `json:"archive_size_bytes,omitempty"`
	ImageSource          string            `json:"image_source,omitempty"`
	ImageRef             string            `json:"image_ref,omitempty"`
	Manifest             json.RawMessage   `json:"manifest,omitempty"`
	LintReport           *HarborLintReport `json:"lint_report,omitempty"`
	HarborVersion        string            `json:"harbor_version,omitempty"`
	PublishedByUserEmail string            `json:"published_by_user_email,omitempty"`
	PublishedAt          *time.Time        `json:"published_at,omitempty"`
	CreatedAt            *time.Time        `json:"created_at,omitempty"`
	UpdatedAt            *time.Time        `json:"updated_at,omitempty"`
}

// HarborTaskUploadRequest announces an archive before it is uploaded.
type HarborTaskUploadRequest struct {
	Digest      string            `json:"digest"`
	SizeBytes   int64             `json:"size_bytes"`
	FileDigests map[string]string `json:"file_digests"`
	Manifest    map[string]any    `json:"manifest"`
}

// HarborTaskUploadStart is the reply to an upload request: either the digest
// is already current (Unchanged), or the archive goes to UploadURL.
type HarborTaskUploadStart struct {
	Unchanged  bool   `json:"unchanged"`
	UploadUUID string `json:"upload_uuid,omitempty"`
	UploadURL  string `json:"upload_url,omitempty"`
}

// HarborTaskUpload is the state of one publish attempt.
type HarborTaskUpload struct {
	UploadUUID       string            `json:"upload_uuid"`
	TaskName         string            `json:"task_name,omitempty"`
	Digest           string            `json:"digest,omitempty"`
	State            string            `json:"state"`
	LintReport       *HarborLintReport `json:"lint_report,omitempty"`
	ErrorDescription string            `json:"error_description,omitempty"`
	CreatedAt        *time.Time        `json:"created_at,omitempty"`
	UpdatedAt        *time.Time        `json:"updated_at,omitempty"`
}

// HarborPublishUnchanged is the publish state reported when the task's
// current digest already matches the local folder.
const HarborPublishUnchanged = "unchanged"

// HarborPublishResult is what doctl reports for one published task folder.
type HarborPublishResult struct {
	Name             string            `json:"name"`
	Digest           string            `json:"digest"`
	State            string            `json:"state"`
	UploadUUID       string            `json:"upload_uuid,omitempty"`
	Files            int               `json:"files"`
	ArchiveSizeBytes int64             `json:"archive_size_bytes"`
	LintReport       *HarborLintReport `json:"lint_report,omitempty"`
	ErrorDescription string            `json:"error_description,omitempty"`
}

// HarborPullResult is what doctl reports for a downloaded task.
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

// HarborDatasetTask is a task referenced by a dataset.
type HarborDatasetTask struct {
	Name   string `json:"name"`
	Digest string `json:"digest,omitempty"`
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

// HarborDataset is a named list of the team's tasks.
type HarborDataset struct {
	Name                 string              `json:"name"`
	Description          string              `json:"description,omitempty"`
	Tasks                []HarborDatasetTask `json:"tasks,omitempty"`
	HasMetric            bool                `json:"has_metric,omitempty"`
	PublishedByUserEmail string              `json:"published_by_user_email,omitempty"`
	PublishedAt          *time.Time          `json:"published_at,omitempty"`
	CreatedAt            *time.Time          `json:"created_at,omitempty"`
	UpdatedAt            *time.Time          `json:"updated_at,omitempty"`
}

// HarborDatasetRequest creates or replaces a dataset.
type HarborDatasetRequest struct {
	Description string   `json:"description,omitempty"`
	Tasks       []string `json:"tasks"`
}

// HarborJobAgent is one agent and model pair in a job.
type HarborJobAgent struct {
	Name    string         `json:"name"`
	Version string         `json:"version,omitempty"`
	Model   string         `json:"model"`
	Kwargs  map[string]any `json:"kwargs,omitempty"`
}

// HarborJobRetry is the trial retry policy.
type HarborJobRetry struct {
	MaxRetries int `json:"max_retries"`
}

// HarborJobRequest submits a job.
type HarborJobRequest struct {
	Name              string           `json:"name,omitempty"`
	Tasks             []string         `json:"tasks,omitempty"`
	Dataset           string           `json:"dataset,omitempty"`
	Agents            []HarborJobAgent `json:"agents"`
	NAttempts         int              `json:"n_attempts"`
	NConcurrentTrials int              `json:"n_concurrent_trials"`
	Retry             HarborJobRetry   `json:"retry"`
	DryRun            bool             `json:"dry_run"`
	TriggerSource     string           `json:"trigger_source,omitempty"`
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

// HarborTrial is one agent attempt at one task.
type HarborTrial struct {
	TrialName        string             `json:"trial_name"`
	TaskName         string             `json:"task_name"`
	TaskDigest       string             `json:"task_digest,omitempty"`
	AgentName        string             `json:"agent_name"`
	AgentVersion     string             `json:"agent_version,omitempty"`
	ModelSlug        string             `json:"model_slug"`
	Attempt          int                `json:"attempt"`
	Status           string             `json:"status"`
	Rewards          map[string]float64 `json:"rewards,omitempty"`
	NInputTokens     *int64             `json:"n_input_tokens,omitempty"`
	NOutputTokens    *int64             `json:"n_output_tokens,omitempty"`
	ExceptionType    string             `json:"exception_type,omitempty"`
	ExceptionMessage string             `json:"exception_message,omitempty"`
	Timing           json.RawMessage    `json:"timing,omitempty"`
	CreatedAt        *time.Time         `json:"created_at,omitempty"`
	UpdatedAt        *time.Time         `json:"updated_at,omitempty"`
}

// HarborArtifact is one file of a job's results with its download link.
type HarborArtifact struct {
	Path      string     `json:"path"`
	URL       string     `json:"url"`
	SizeBytes int64      `json:"size_bytes,omitempty"`
	SHA256    string     `json:"sha256,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (a *HarborArtifact) UnmarshalJSON(b []byte) error {
	type plain HarborArtifact
	var v struct {
		plain
		Name        string `json:"name"`
		Key         string `json:"key"`
		DownloadURL string `json:"download_url"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*a = HarborArtifact(v.plain)
	a.Path = harborFirstNonEmpty(a.Path, v.Name, v.Key)
	a.URL = harborFirstNonEmpty(a.URL, v.DownloadURL)
	return nil
}

// HarborAgent is a supported agent with its pinned version.
type HarborAgent struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Description string `json:"description,omitempty"`
}

// HarborService is the Managed Harbor API.
type HarborService interface {
	ListTasks(query string) ([]HarborTask, error)
	GetTask(name string) (*HarborTask, error)
	DeleteTask(name string) error
	GetTaskArchive(name string) (*HarborPresignedURL, error)
	StartTaskUpload(name string, req *HarborTaskUploadRequest) (*HarborTaskUploadStart, error)
	CompleteTaskUpload(name, uploadUUID string) (*HarborTaskUpload, error)
	GetTaskUpload(name, uploadUUID string) (*HarborTaskUpload, error)

	PutDataset(name string, req *HarborDatasetRequest) (*HarborDataset, error)
	ListDatasets() ([]HarborDataset, error)
	GetDataset(name string) (*HarborDataset, error)
	DeleteDataset(name string) error

	CreateJob(req *HarborJobRequest, idempotencyKey string) (*HarborJob, error)
	ListJobs() ([]HarborJob, error)
	GetJob(uuid string) (*HarborJob, error)
	ListTrials(uuid string) ([]HarborTrial, error)
	ListArtifacts(uuid string) ([]HarborArtifact, error)
	CancelJob(uuid string) (*HarborJob, error)

	ListAgents() ([]HarborAgent, error)
}

type harborService struct {
	client *godo.Client
}

var _ HarborService = &harborService{}

// NewHarborService builds a HarborService on the shared API client, so it uses
// the same token, base URL, and retry policy as every other command.
func NewHarborService(client *godo.Client) HarborService {
	return &harborService{client: client}
}

func (s *harborService) do(method, path string, body any, header http.Header, out any) (*godo.Response, error) {
	ctx := context.TODO()
	req, err := s.client.NewRequest(ctx, method, harborBasePath+path, body)
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

func harborTaskPath(name string) string { return "/tasks/" + url.PathEscape(name) }

func harborJobPath(uuid string) string { return "/jobs/" + url.PathEscape(uuid) }

func (s *harborService) ListTasks(query string) ([]HarborTask, error) {
	q := url.Values{}
	if query != "" {
		q.Set("q", query)
	}
	return harborList[HarborTask](s, "/tasks", q, "tasks")
}

func (s *harborService) GetTask(name string) (*HarborTask, error) {
	return harborOne[HarborTask](s, http.MethodGet, harborTaskPath(name), nil, nil, "task")
}

func (s *harborService) DeleteTask(name string) error {
	_, err := s.do(http.MethodDelete, harborTaskPath(name), nil, nil, nil)
	return err
}

func (s *harborService) GetTaskArchive(name string) (*HarborPresignedURL, error) {
	return harborOne[HarborPresignedURL](s, http.MethodGet, harborTaskPath(name)+"/archive", nil, nil, "archive")
}

func (s *harborService) StartTaskUpload(name string, req *HarborTaskUploadRequest) (*HarborTaskUploadStart, error) {
	return harborOne[HarborTaskUploadStart](s, http.MethodPost, harborTaskPath(name)+"/uploads", req, nil, "upload")
}

func (s *harborService) CompleteTaskUpload(name, uploadUUID string) (*HarborTaskUpload, error) {
	up, err := harborOne[HarborTaskUpload](s, http.MethodPost, harborTaskPath(name)+"/uploads/"+url.PathEscape(uploadUUID)+"/complete", nil, nil, "upload")
	if err == nil && up.UploadUUID == "" {
		up.UploadUUID = uploadUUID
	}
	return up, err
}

func (s *harborService) GetTaskUpload(name, uploadUUID string) (*HarborTaskUpload, error) {
	up, err := harborOne[HarborTaskUpload](s, http.MethodGet, harborTaskPath(name)+"/uploads/"+url.PathEscape(uploadUUID), nil, nil, "upload")
	if err == nil && up.UploadUUID == "" {
		up.UploadUUID = uploadUUID
	}
	return up, err
}

func (s *harborService) PutDataset(name string, req *HarborDatasetRequest) (*HarborDataset, error) {
	ds, err := harborOne[HarborDataset](s, http.MethodPut, "/datasets/"+url.PathEscape(name), req, nil, "dataset")
	if err == nil && ds.Name == "" {
		ds.Name = name
	}
	return ds, err
}

func (s *harborService) ListDatasets() ([]HarborDataset, error) {
	return harborList[HarborDataset](s, "/datasets", nil, "datasets")
}

func (s *harborService) GetDataset(name string) (*HarborDataset, error) {
	return harborOne[HarborDataset](s, http.MethodGet, "/datasets/"+url.PathEscape(name), nil, nil, "dataset")
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

func (s *harborService) ListJobs() ([]HarborJob, error) {
	return harborList[HarborJob](s, "/jobs", nil, "jobs")
}

func (s *harborService) GetJob(uuid string) (*HarborJob, error) {
	return harborOne[HarborJob](s, http.MethodGet, harborJobPath(uuid), nil, nil, "job")
}

func (s *harborService) ListTrials(uuid string) ([]HarborTrial, error) {
	return harborList[HarborTrial](s, harborJobPath(uuid)+"/trials", nil, "trials")
}

func (s *harborService) ListArtifacts(uuid string) ([]HarborArtifact, error) {
	return harborList[HarborArtifact](s, harborJobPath(uuid)+"/artifacts", nil, "artifacts")
}

func (s *harborService) CancelJob(uuid string) (*HarborJob, error) {
	job, err := harborOne[HarborJob](s, http.MethodPost, harborJobPath(uuid)+"/cancel", nil, nil, "job")
	if err == nil && job.JobUUID == "" {
		job.JobUUID = uuid
	}
	return job, err
}

func (s *harborService) ListAgents() ([]HarborAgent, error) {
	return harborList[HarborAgent](s, "/agents", nil, "agents")
}

func harborFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
