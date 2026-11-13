package godo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

// ADSTarballUpload is an upload grant returned when a named tarball object is absent.
type ADSTarballUpload struct {
	URL       string `json:"url,omitempty"`
	ObjectKey string `json:"object_key,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// ADSTarballStatus is the tarball digests on a data flow.
type ADSTarballStatus struct {
	Requested string `json:"requested,omitempty"`
	Attached  string `json:"attached,omitempty"`
}

// ADSDataFlow is one data flow resource.
type ADSDataFlow struct {
	UUID            string            `json:"uuid,omitempty"`
	Name            string            `json:"name,omitempty"`
	State           string            `json:"state,omitempty"`
	Workspace       string            `json:"workspace,omitempty"`
	FchainID        string            `json:"fchain_id,omitempty"`
	ConfigJSON      string            `json:"config_json,omitempty"`
	DesiredReplicas int32             `json:"desired_replicas,omitempty"`
	MaxReplicas     int32             `json:"max_replicas,omitempty"`
	CreatedAt       string            `json:"created_at,omitempty"`
	UpdatedAt       string            `json:"updated_at,omitempty"`
	ImageRef        string            `json:"image_ref,omitempty"`
	Spec            json.RawMessage   `json:"spec,omitempty"`
	Tarball         *ADSTarballStatus `json:"tarball,omitempty"`
	DesiredState    string            `json:"desired_state,omitempty"`
	LastError       string            `json:"last_error,omitempty"`
}

// ADSDataFlowCreateRequest is the body for CreateDataFlow.
// Region is path-only and is not sent in the JSON body. Workspace is required
// and used as a path segment; it is also sent in the body.
type ADSDataFlowCreateRequest struct {
	Region          string          `json:"-"`
	Name            string          `json:"name,omitempty"`
	DesiredReplicas int32           `json:"desired_replicas,omitempty"`
	MaxReplicas     int32           `json:"max_replicas,omitempty"`
	Workspace       string          `json:"workspace"`
	ImageRef        string          `json:"image_ref,omitempty"`
	Manifest        json.RawMessage `json:"manifest,omitempty"`
}

// ADSDataFlowCreateResponse is the body of CreateDataFlow.
type ADSDataFlowCreateResponse struct {
	UUID   string            `json:"uuid,omitempty"`
	State  string            `json:"state,omitempty"`
	Upload *ADSTarballUpload `json:"upload,omitempty"`
}

// ADSDataFlowListRequest holds path and query args for ListDataFlows.
// Region and Workspace are path-only.
type ADSDataFlowListRequest struct {
	Region    string `url:"-"`
	Workspace string `url:"-"`
	PageSize  int    `url:"page_size,omitempty"`
	PageToken string `url:"page_token,omitempty"`
}

// ADSListDataFlowsResponse is the body of ListDataFlows.
type ADSListDataFlowsResponse struct {
	DataFlows     []ADSDataFlow `json:"data_flows"`
	NextPageToken string        `json:"next_page_token,omitempty"`
}

// ADSDataFlowUpdateRequest is the body for UpdateDataFlow.
// Region is path-only and is not sent in the JSON body. Workspace and ID are
// required path segments and are also sent in the body.
type ADSDataFlowUpdateRequest struct {
	Region          string          `json:"-"`
	ID              string          `json:"id"`
	Workspace       string          `json:"workspace"`
	DesiredReplicas int32           `json:"desired_replicas,omitempty"`
	MaxReplicas     int32           `json:"max_replicas,omitempty"`
	SecretRefs      []string        `json:"secret_refs,omitempty"`
	ClearSecretRefs bool            `json:"clear_secret_refs,omitempty"`
	Manifest        json.RawMessage `json:"manifest,omitempty"`
}

// ADSDataFlowUpdateResponse is the body of UpdateDataFlow.
type ADSDataFlowUpdateResponse struct {
	UUID   string            `json:"uuid,omitempty"`
	State  string            `json:"state,omitempty"`
	Upload *ADSTarballUpload `json:"upload,omitempty"`
}

// ADSDataFlowRewindRequest is the body for RewindDataFlow.
// Region is path-only and is not sent in the JSON body.
type ADSDataFlowRewindRequest struct {
	Region    string `json:"-"`
	ID        string `json:"id"`
	Workspace string `json:"workspace"`
	RewindTo  string `json:"rewind_to,omitempty"`
}

// ADSFchainUploadURLRequest is the body for CreateFchainUploadURL.
// Region is path-only and is not sent in the JSON body.
type ADSFchainUploadURLRequest struct {
	Region    string `json:"-"`
	SHA256    string `json:"sha256,omitempty"`
	Workspace string `json:"workspace"`
}

// ADSFchainUploadURLResponse is the body of CreateFchainUploadURL.
type ADSFchainUploadURLResponse struct {
	UploadURL string `json:"upload_url,omitempty"`
	ObjectKey string `json:"object_key,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// ADSAttachDataFlowFchainRequest is the body for AttachDataFlowFchain.
// Region is path-only and is not sent in the JSON body.
type ADSAttachDataFlowFchainRequest struct {
	Region    string `json:"-"`
	ID        string `json:"id"`
	Workspace string `json:"workspace"`
	ObjectKey string `json:"object_key,omitempty"`
}

// ADSWorkspaceMetricsRequest is the body for GetWorkspaceMetrics.
// Region is path-only and is not sent in the JSON body.
type ADSWorkspaceMetricsRequest struct {
	Region      string   `json:"-"`
	Workspace   string   `json:"workspace"`
	MetricNames []string `json:"metric_names,omitempty"`
	DataFlowID  string   `json:"data_flow_id,omitempty"`
	From        string   `json:"from,omitempty"`
	To          string   `json:"to,omitempty"`
	GroupBy     []string `json:"group_by,omitempty"`
	Limit       int32    `json:"limit,omitempty"`
	MetricType  string   `json:"metric_type,omitempty"`
}

// ADSWorkspaceMetricsResponse is the body of GetWorkspaceMetrics.
type ADSWorkspaceMetricsResponse struct {
	Count   int32            `json:"count,omitempty"`
	Columns []string         `json:"columns,omitempty"`
	Rows    []map[string]any `json:"rows,omitempty"`
}

func adsDataFlowsPath(region, workspace string) (string, error) {
	if region == "" {
		return "", errors.New("ads: region is required")
	}
	if workspace == "" {
		return "", errors.New("ads: workspace is required")
	}
	return adsPrefix + "/" + url.PathEscape(region) + "/workspaces/" + url.PathEscape(workspace) + "/data-flows", nil
}

func adsDataFlowPath(region, workspace, id string) (string, error) {
	if id == "" {
		return "", errors.New("ads: data flow id is required")
	}
	base, err := adsDataFlowsPath(region, workspace)
	if err != nil {
		return "", err
	}
	return base + "/" + url.PathEscape(id), nil
}

func adsDataFlowActionPath(region, workspace, id, action string) (string, error) {
	base, err := adsDataFlowPath(region, workspace, id)
	if err != nil {
		return "", err
	}
	return base + "/" + action, nil
}

func adsFchainUploadURLPath(region, workspace string) (string, error) {
	if region == "" {
		return "", errors.New("ads: region is required")
	}
	if workspace == "" {
		return "", errors.New("ads: workspace is required")
	}
	return adsPrefix + "/" + url.PathEscape(region) + "/workspaces/" + url.PathEscape(workspace) + "/fchain/upload-url", nil
}

func adsWorkspaceMetricsPath(region, workspace string) (string, error) {
	if region == "" {
		return "", errors.New("ads: region is required")
	}
	if workspace == "" {
		return "", errors.New("ads: workspace is required")
	}
	return adsPrefix + "/" + url.PathEscape(region) + "/workspaces/" + url.PathEscape(workspace) + "/metrics", nil
}

// CreateDataFlow creates a data flow in the workspace on create.
// Region comes from create and is used as the path segment only.
func (s *ADSServiceOp) CreateDataFlow(ctx context.Context, create *ADSDataFlowCreateRequest) (*ADSDataFlowCreateResponse, *Response, error) {
	if create == nil {
		return nil, nil, errors.New("ads: create request is required")
	}
	path, err := adsDataFlowsPath(create.Region, create.Workspace)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, create)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSDataFlowCreateResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// ListDataFlows lists data flows in the workspace on list.
// Region comes from list and is used as the path segment only.
func (s *ADSServiceOp) ListDataFlows(ctx context.Context, list *ADSDataFlowListRequest) (*ADSListDataFlowsResponse, *Response, error) {
	if list == nil {
		return nil, nil, errors.New("ads: list request is required")
	}
	path, err := adsDataFlowsPath(list.Region, list.Workspace)
	if err != nil {
		return nil, nil, err
	}
	path, err = addOptions(path, list)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSListDataFlowsResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetDataFlow returns one data flow by region, workspace, and id.
func (s *ADSServiceOp) GetDataFlow(ctx context.Context, region, workspace, id string) (*ADSDataFlow, *Response, error) {
	path, err := adsDataFlowPath(region, workspace, id)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSDataFlow)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// UpdateDataFlow patches a data flow.
// Region comes from update and is used as the path segment only.
func (s *ADSServiceOp) UpdateDataFlow(ctx context.Context, update *ADSDataFlowUpdateRequest) (*ADSDataFlowUpdateResponse, *Response, error) {
	if update == nil {
		return nil, nil, errors.New("ads: update request is required")
	}
	path, err := adsDataFlowPath(update.Region, update.Workspace, update.ID)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPatch, path, update)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSDataFlowUpdateResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// DeleteDataFlow begins deleting a data flow.
func (s *ADSServiceOp) DeleteDataFlow(ctx context.Context, region, workspace, id string) (*Response, error) {
	path, err := adsDataFlowPath(region, workspace, id)
	if err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// PauseDataFlow pauses an active data flow.
func (s *ADSServiceOp) PauseDataFlow(ctx context.Context, region, workspace, id string) (*Response, error) {
	path, err := adsDataFlowActionPath(region, workspace, id, "pause")
	if err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// ResumeDataFlow resumes a paused data flow.
func (s *ADSServiceOp) ResumeDataFlow(ctx context.Context, region, workspace, id string) (*Response, error) {
	path, err := adsDataFlowActionPath(region, workspace, id, "resume")
	if err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// RestartDataFlow restarts a data flow.
func (s *ADSServiceOp) RestartDataFlow(ctx context.Context, region, workspace, id string) (*Response, error) {
	path, err := adsDataFlowActionPath(region, workspace, id, "restart")
	if err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// RewindDataFlow requests rewind. The server may respond that rewind is unsupported.
// Region comes from rewind and is used as the path segment only.
func (s *ADSServiceOp) RewindDataFlow(ctx context.Context, rewind *ADSDataFlowRewindRequest) (*Response, error) {
	if rewind == nil {
		return nil, errors.New("ads: rewind request is required")
	}
	path, err := adsDataFlowActionPath(rewind.Region, rewind.Workspace, rewind.ID, "rewind")
	if err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, rewind)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// CreateFchainUploadURL creates a temporary upload URL for a function-chain archive.
// Region comes from create and is used as the path segment only.
func (s *ADSServiceOp) CreateFchainUploadURL(ctx context.Context, create *ADSFchainUploadURLRequest) (*ADSFchainUploadURLResponse, *Response, error) {
	if create == nil {
		return nil, nil, errors.New("ads: create request is required")
	}
	path, err := adsFchainUploadURLPath(create.Region, create.Workspace)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, create)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSFchainUploadURLResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// AttachDataFlowFchain attaches an uploaded function-chain archive to a data flow.
// Region comes from attach and is used as the path segment only.
func (s *ADSServiceOp) AttachDataFlowFchain(ctx context.Context, attach *ADSAttachDataFlowFchainRequest) (*Response, error) {
	if attach == nil {
		return nil, errors.New("ads: attach request is required")
	}
	path, err := adsDataFlowActionPath(attach.Region, attach.Workspace, attach.ID, "fchain")
	if err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, attach)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// GetWorkspaceMetrics queries data flow metrics for a workspace.
// Region comes from metrics and is used as the path segment only.
func (s *ADSServiceOp) GetWorkspaceMetrics(ctx context.Context, metrics *ADSWorkspaceMetricsRequest) (*ADSWorkspaceMetricsResponse, *Response, error) {
	if metrics == nil {
		return nil, nil, errors.New("ads: metrics request is required")
	}
	path, err := adsWorkspaceMetricsPath(metrics.Region, metrics.Workspace)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, metrics)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSWorkspaceMetricsResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}
