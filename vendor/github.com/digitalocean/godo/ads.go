package godo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

const adsPrefix = "/v2/data-cloud-platform"

// ADSService is the Agent Data Services client for workspace,
// token, secret, query-credential, and data-flow routes under
// /v2/data-cloud-platform.
type ADSService interface {
	// ListPlans returns the regions and query engine sizes a team can use
	// when creating a workspace. The list is not paginated.
	ListPlans(context.Context) (*ADSPlansResponse, *Response, error)
	// CreateWorkspace creates a workspace in region. Region is the path
	// segment. The body carries name and query_instance_size only.
	CreateWorkspace(context.Context, string, *ADSWorkspaceCreateRequest) (*ADSWorkspace, *Response, error)
	// ListWorkspaces returns the workspaces in region. The list is not paginated.
	ListWorkspaces(context.Context, string) (*ADSWorkspacesResponse, *Response, error)
	// GetWorkspace returns one workspace by region and uuid.
	GetWorkspace(context.Context, string, string) (*ADSWorkspace, *Response, error)
	// DeleteWorkspace deletes one workspace. The API returns HTTP 200 and an
	// empty object.
	DeleteWorkspace(context.Context, string, string) (*Response, error)
	// CreateToken creates a token in the region carried on the request body.
	// Region is also the path segment.
	CreateToken(context.Context, *ADSTokenCreateRequest) (*ADSTokenCreateResponse, *Response, error)
	// ListTokens returns tokens in the region carried on the request.
	// Region is the path segment. Optional filters are query parameters.
	ListTokens(context.Context, *ADSTokenListRequest) (*ADSListTokensResponse, *Response, error)
	// PutWorkspaceSecret writes a workspace secret. Region, UUID, and SecretKey
	// on the request are path segments and are also sent in the body. Data must
	// contain at least one string key-value pair.
	PutWorkspaceSecret(context.Context, *ADSPutWorkspaceSecretRequest) (*ADSPutWorkspaceSecretResponse, *Response, error)
	// GetWorkspaceSecret returns secret metadata and property names. Values are never returned.
	GetWorkspaceSecret(context.Context, string, string, string) (*ADSGetWorkspaceSecretResponse, *Response, error)
	// ListWorkspaceSecrets returns secret key names for a workspace. Values are never returned.
	ListWorkspaceSecrets(context.Context, string, string) (*ADSListWorkspaceSecretsResponse, *Response, error)
	// DeleteWorkspaceSecret permanently deletes a workspace secret.
	DeleteWorkspaceSecret(context.Context, string, string, string) (*Response, error)
	// GetWorkspaceQueryCredentials returns the query user and password for a workspace.
	// Do not log the response; it contains a password.
	GetWorkspaceQueryCredentials(context.Context, string, string) (*ADSQueryCredentials, *Response, error)

	// CreateDataFlow creates a data flow in a workspace.
	// Region on the request is path-only and is not sent in the body.
	CreateDataFlow(context.Context, *ADSDataFlowCreateRequest) (*ADSDataFlowCreateResponse, *Response, error)
	// ListDataFlows lists data flows in a workspace.
	// Region on the request is path-only.
	ListDataFlows(context.Context, *ADSDataFlowListRequest) (*ADSListDataFlowsResponse, *Response, error)
	// GetDataFlow returns one data flow by region, workspace, and id.
	GetDataFlow(context.Context, string, string, string) (*ADSDataFlow, *Response, error)
	// UpdateDataFlow patches a data flow's replica settings, secret refs, or manifest.
	// Region on the request is path-only and is not sent in the body.
	UpdateDataFlow(context.Context, *ADSDataFlowUpdateRequest) (*ADSDataFlowUpdateResponse, *Response, error)
	// DeleteDataFlow begins deleting a data flow.
	DeleteDataFlow(context.Context, string, string, string) (*Response, error)
	// PauseDataFlow pauses an active data flow.
	PauseDataFlow(context.Context, string, string, string) (*Response, error)
	// ResumeDataFlow resumes a paused data flow.
	ResumeDataFlow(context.Context, string, string, string) (*Response, error)
	// RestartDataFlow restarts a data flow.
	RestartDataFlow(context.Context, string, string, string) (*Response, error)
	// RewindDataFlow requests rewind to an earlier time. The server currently
	// reports rewind as unsupported; still call the endpoint and surface that response.
	// Region on the request is path-only and is not sent in the body.
	RewindDataFlow(context.Context, *ADSDataFlowRewindRequest) (*Response, error)
	// CreateFchainUploadURL creates a temporary upload URL for a function-chain archive.
	// Region on the request is path-only and is not sent in the body.
	CreateFchainUploadURL(context.Context, *ADSFchainUploadURLRequest) (*ADSFchainUploadURLResponse, *Response, error)
	// AttachDataFlowFchain attaches an uploaded function-chain archive to a data flow.
	// Region on the request is path-only and is not sent in the body.
	AttachDataFlowFchain(context.Context, *ADSAttachDataFlowFchainRequest) (*Response, error)
	// GetWorkspaceMetrics queries data flow metrics for a workspace.
	// Region on the request is path-only and is not sent in the body.
	GetWorkspaceMetrics(context.Context, *ADSWorkspaceMetricsRequest) (*ADSWorkspaceMetricsResponse, *Response, error)
}

// ADSServiceOp handles communication with Agent Data Services
// workspace and plan methods.
type ADSServiceOp struct {
	client *Client
}

var _ ADSService = &ADSServiceOp{}

// ADSQueryEngineSize is one selectable query engine size inside a plan.
// Size is a plain string. Advertised values are small, medium, and large.
type ADSQueryEngineSize struct {
	Size        string `json:"size,omitempty"`
	DropletSlug string `json:"droplet_slug,omitempty"`
	CPU         int32  `json:"cpu,omitempty"`
	MemoryGi    int32  `json:"memory_gi,omitempty"`
}

// ADSPlan is a region offering returned by ListPlans.
type ADSPlan struct {
	RegionSlug         string               `json:"region_slug,omitempty"`
	AvailableForCreate bool                 `json:"available_for_create"`
	QueryEngineSizes   []ADSQueryEngineSize `json:"query_engine_sizes,omitempty"`
}

// ADSPlansResponse is the body of GET /v2/data-cloud-platform/plans.
type ADSPlansResponse struct {
	Plans []ADSPlan `json:"plans"`
}

// ADSWorkspace is a workspace in one region.
// Status is a plain string, for example pending, provisioning, active, or errored.
// QueryInstanceSize is a plain string. Advertised values are small, medium, and large.
type ADSWorkspace struct {
	UUID              string `json:"uuid"`
	OwnerUUID         string `json:"owner_uuid,omitempty"`
	Name              string `json:"name,omitempty"`
	Region            string `json:"region,omitempty"`
	Status            string `json:"status,omitempty"`
	QueryEndpoint     string `json:"query_endpoint,omitempty"`
	QueryInstanceSize string `json:"query_instance_size,omitempty"`
}

// ADSWorkspaceCreateRequest is the body for CreateWorkspace.
// Region is not a body field. Pass it as the region argument. Name and
// QueryInstanceSize are optional. An empty name leaves the server default.
// An empty QueryInstanceSize leaves the server default, small.
type ADSWorkspaceCreateRequest struct {
	Name              string `json:"name,omitempty"`
	QueryInstanceSize string `json:"query_instance_size,omitempty"`
}

// ADSTokenPurpose selects how a data-plane token can be used.
type ADSTokenPurpose string

const (
	ADSTokenPurposeUnknown ADSTokenPurpose = "TOKEN_PURPOSE_UNKNOWN"
	// ADSTokenPurposeQuery grants query access to the selected workspaces.
	ADSTokenPurposeQuery ADSTokenPurpose = "TOKEN_PURPOSE_QUERY"
)

// ADSTokenCreateRequest is the body for CreateToken.
// Region is required. The other fields are optional and omitted when empty.
// WorkspaceUUIDs is the workspace binding list.
// ExpiresInSeconds is written as a JSON string. Zero means the server default and is omitted.
type ADSTokenCreateRequest struct {
	Region           string          `json:"region"`
	Purpose          ADSTokenPurpose `json:"purpose,omitempty"`
	WorkspaceUUIDs   []string        `json:"workspace_uuids,omitempty"`
	Name             string          `json:"name,omitempty"`
	ExpiresInSeconds int64           `json:"-"`
	Permissions      []string        `json:"permissions,omitempty"`
}

// MarshalJSON writes expires_in_seconds as a decimal string. Proto JSON represents int64 that way.
func (r ADSTokenCreateRequest) MarshalJSON() ([]byte, error) {
	payload := struct {
		Region           string          `json:"region"`
		Purpose          ADSTokenPurpose `json:"purpose,omitempty"`
		WorkspaceUUIDs   []string        `json:"workspace_uuids,omitempty"`
		Name             string          `json:"name,omitempty"`
		ExpiresInSeconds string          `json:"expires_in_seconds,omitempty"`
		Permissions      []string        `json:"permissions,omitempty"`
	}{
		Region:         r.Region,
		Purpose:        r.Purpose,
		WorkspaceUUIDs: r.WorkspaceUUIDs,
		Name:           r.Name,
		Permissions:    r.Permissions,
	}
	if r.ExpiresInSeconds != 0 {
		payload.ExpiresInSeconds = strconv.FormatInt(r.ExpiresInSeconds, 10)
	}
	return json.Marshal(payload)
}

// ADSTokenCreateResponse is the body returned by CreateToken.
// AccessToken is returned once, on create. Do not log this struct with %v or %+v.
type ADSTokenCreateResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// ADSTokenListRequest holds path and query args for ListTokens.
// Region is required and used as the path segment. The other fields are
// optional query filters and omitted when empty.
type ADSTokenListRequest struct {
	Region        string          `url:"-"`
	Purpose       ADSTokenPurpose `url:"purpose,omitempty"`
	Page          int64           `url:"page,omitempty"`
	PerPage       int64           `url:"per_page,omitempty"`
	WorkspaceUUID string          `url:"workspace_uuid,omitempty"`
}

// ADSTokenSummary is one token in a ListTokens response.
// It never includes an access token.
type ADSTokenSummary struct {
	ID             string          `json:"id"`
	Name           string          `json:"name,omitempty"`
	CreatedAt      string          `json:"created_at,omitempty"`
	ExpiresAt      string          `json:"expires_at,omitempty"`
	Purpose        ADSTokenPurpose `json:"purpose,omitempty"`
	QueryHosts     []string        `json:"query_hosts,omitempty"`
	WorkspaceUUIDs []string        `json:"workspace_uuids,omitempty"`
}

// ADSListTokensResponse is the body of ListTokens.
// Tokens and Total are siblings. Total is a decimal string (proto JSON int64).
type ADSListTokensResponse struct {
	Tokens []ADSTokenSummary `json:"tokens"`
	Total  string            `json:"total,omitempty"`
}

// ADSSecretOptions configures a secret write.
// CAS is currently ignored by the server because secret version checks are not supported.
type ADSSecretOptions struct {
	CAS int32 `json:"cas,omitempty"`
}

// ADSPutWorkspaceSecretRequest is the body for PutWorkspaceSecret.
// Region, UUID, and SecretKey are required path segments and are also sent in the body.
// Data must contain at least one string key-value pair.
type ADSPutWorkspaceSecretRequest struct {
	Region    string            `json:"region"`
	UUID      string            `json:"uuid"`
	SecretKey string            `json:"secret_key"`
	Data      map[string]string `json:"data,omitempty"`
	Options   *ADSSecretOptions `json:"options,omitempty"`
}

// ADSSecretMetadata describes a stored secret version.
type ADSSecretMetadata struct {
	CreatedTime    string         `json:"created_time,omitempty"`
	CustomMetadata map[string]any `json:"custom_metadata,omitempty"`
	DeletionTime   string         `json:"deletion_time,omitempty"`
	Destroyed      bool           `json:"destroyed"`
	Version        int32          `json:"version,omitempty"`
}

// ADSPutWorkspaceSecretResponse is the body of PutWorkspaceSecret.
type ADSPutWorkspaceSecretResponse struct {
	Data *ADSSecretMetadata `json:"data,omitempty"`
}

// ADSGetSecretData is the nested payload of GetWorkspaceSecret.
// Keys lists property names. Values are never returned.
type ADSGetSecretData struct {
	Data     map[string]any     `json:"data,omitempty"`
	Metadata *ADSSecretMetadata `json:"metadata,omitempty"`
	Keys     []string           `json:"keys,omitempty"`
}

// ADSGetWorkspaceSecretResponse is the body of GetWorkspaceSecret.
type ADSGetWorkspaceSecretResponse struct {
	Data *ADSGetSecretData `json:"data,omitempty"`
}

// ADSListWorkspaceSecretsResponse is the body of ListWorkspaceSecrets.
type ADSListWorkspaceSecretsResponse struct {
	SecretKeys []string `json:"secret_keys"`
}

// ADSQueryCredentials is the body of GetWorkspaceQueryCredentials.
// Password is a secret. Do not log this struct with %v or %+v.
type ADSQueryCredentials struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

// ADSWorkspacesResponse is the body of ListWorkspaces.
type ADSWorkspacesResponse struct {
	Workspaces []ADSWorkspace `json:"workspaces"`
}

type adsWorkspaceRoot struct {
	Workspace *ADSWorkspace `json:"workspace"`
}

func adsWorkspacesPath(region string) (string, error) {
	if region == "" {
		return "", errors.New("ads: region is required")
	}
	return adsPrefix + "/" + url.PathEscape(region) + "/workspaces", nil
}

func adsWorkspacePath(region, uuid string) (string, error) {
	if uuid == "" {
		return "", errors.New("ads: workspace uuid is required")
	}
	base, err := adsWorkspacesPath(region)
	if err != nil {
		return "", err
	}
	return base + "/" + url.PathEscape(uuid), nil
}

func adsTokensPath(region string) (string, error) {
	if region == "" {
		return "", errors.New("ads: region is required")
	}
	return adsPrefix + "/" + url.PathEscape(region) + "/tokens", nil
}

func adsWorkspaceSecretsPath(region, uuid string) (string, error) {
	base, err := adsWorkspacePath(region, uuid)
	if err != nil {
		return "", err
	}
	return base + "/secrets", nil
}

func adsWorkspaceSecretPath(region, uuid, secretKey string) (string, error) {
	if secretKey == "" {
		return "", errors.New("ads: secret key is required")
	}
	base, err := adsWorkspaceSecretsPath(region, uuid)
	if err != nil {
		return "", err
	}
	return base + "/" + url.PathEscape(secretKey), nil
}

func adsWorkspaceQueryCredentialsPath(region, uuid string) (string, error) {
	base, err := adsWorkspacePath(region, uuid)
	if err != nil {
		return "", err
	}
	return base + "/query/credentials", nil
}

// ListPlans returns the regions and query engine sizes a team can use when
// creating a workspace.
func (s *ADSServiceOp) ListPlans(ctx context.Context) (*ADSPlansResponse, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, adsPrefix+"/plans", nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSPlansResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// CreateWorkspace creates a workspace in region.
func (s *ADSServiceOp) CreateWorkspace(ctx context.Context, region string, create *ADSWorkspaceCreateRequest) (*ADSWorkspace, *Response, error) {
	if create == nil {
		return nil, nil, errors.New("ads: create request is required")
	}
	path, err := adsWorkspacesPath(region)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, create)
	if err != nil {
		return nil, nil, err
	}
	root := new(adsWorkspaceRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Workspace == nil {
		return nil, resp, errors.New("ads: create workspace returned no workspace")
	}
	return root.Workspace, resp, nil
}

// CreateToken creates a token. Region comes from create and is used as the path segment.
func (s *ADSServiceOp) CreateToken(ctx context.Context, create *ADSTokenCreateRequest) (*ADSTokenCreateResponse, *Response, error) {
	if create == nil {
		return nil, nil, errors.New("ads: create request is required")
	}
	path, err := adsTokensPath(create.Region)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, create)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSTokenCreateResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// ListTokens returns tokens in the region on list. Optional filters become query parameters.
// An empty Purpose omits the purpose query param so the server default applies.
func (s *ADSServiceOp) ListTokens(ctx context.Context, list *ADSTokenListRequest) (*ADSListTokensResponse, *Response, error) {
	if list == nil {
		return nil, nil, errors.New("ads: list request is required")
	}
	path, err := adsTokensPath(list.Region)
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
	root := new(ADSListTokensResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// PutWorkspaceSecret writes a workspace secret at
// /v2/data-cloud-platform/{region}/workspaces/{uuid}/secrets/{secret_key}.
func (s *ADSServiceOp) PutWorkspaceSecret(ctx context.Context, put *ADSPutWorkspaceSecretRequest) (*ADSPutWorkspaceSecretResponse, *Response, error) {
	if put == nil {
		return nil, nil, errors.New("ads: put request is required")
	}
	if len(put.Data) == 0 {
		return nil, nil, errors.New("ads: secret data is required")
	}
	path, err := adsWorkspaceSecretPath(put.Region, put.UUID, put.SecretKey)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPut, path, put)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSPutWorkspaceSecretResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetWorkspaceSecret returns metadata and property names for a workspace secret.
func (s *ADSServiceOp) GetWorkspaceSecret(ctx context.Context, region, uuid, secretKey string) (*ADSGetWorkspaceSecretResponse, *Response, error) {
	path, err := adsWorkspaceSecretPath(region, uuid, secretKey)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSGetWorkspaceSecretResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// ListWorkspaceSecrets returns secret key names for a workspace.
func (s *ADSServiceOp) ListWorkspaceSecrets(ctx context.Context, region, uuid string) (*ADSListWorkspaceSecretsResponse, *Response, error) {
	path, err := adsWorkspaceSecretsPath(region, uuid)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSListWorkspaceSecretsResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// DeleteWorkspaceSecret permanently deletes a workspace secret.
func (s *ADSServiceOp) DeleteWorkspaceSecret(ctx context.Context, region, uuid, secretKey string) (*Response, error) {
	path, err := adsWorkspaceSecretPath(region, uuid, secretKey)
	if err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// GetWorkspaceQueryCredentials returns query user and password for a workspace.
func (s *ADSServiceOp) GetWorkspaceQueryCredentials(ctx context.Context, region, uuid string) (*ADSQueryCredentials, *Response, error) {
	path, err := adsWorkspaceQueryCredentialsPath(region, uuid)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSQueryCredentials)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// ListWorkspaces returns the workspaces in region.
func (s *ADSServiceOp) ListWorkspaces(ctx context.Context, region string) (*ADSWorkspacesResponse, *Response, error) {
	path, err := adsWorkspacesPath(region)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(ADSWorkspacesResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetWorkspace returns one workspace by region and uuid.
func (s *ADSServiceOp) GetWorkspace(ctx context.Context, region, uuid string) (*ADSWorkspace, *Response, error) {
	path, err := adsWorkspacePath(region, uuid)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(adsWorkspaceRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Workspace == nil {
		return nil, resp, errors.New("ads: get workspace returned no workspace")
	}
	return root.Workspace, resp, nil
}

// DeleteWorkspace deletes one workspace.
func (s *ADSServiceOp) DeleteWorkspace(ctx context.Context, region, uuid string) (*Response, error) {
	path, err := adsWorkspacePath(region, uuid)
	if err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
