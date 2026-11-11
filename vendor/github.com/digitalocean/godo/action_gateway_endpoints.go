package godo

import (
	"context"
	"encoding/json"
	"net/http"
)

// ActionGatewayEndpointsService manages named endpoints and their immutable versions.
type ActionGatewayEndpointsService struct{ client *Client }

// ActionGatewayEndpointNetwork binds a version to a team VPC at creation.
type ActionGatewayEndpointNetwork struct {
	VPCUUID string `json:"vpcUuid"`
}

// ActionGatewayEndpointNetworkResponse is a version's recorded VPC binding.
type ActionGatewayEndpointNetworkResponse struct {
	VPCUUID string `json:"vpcUuid"`
	Region  string `json:"region"`
}

// ActionGatewayEndpointVersionSpec is the immutable configuration of a new
// endpoint version. Omission of Tools enables every tool; an explicitly empty
// slice enables none. The insights field is deprecated and ignored: consent
// lives on the endpoint.
type ActionGatewayEndpointVersionSpec struct {
	ActorID  string                        `json:"actor_id,omitempty"`
	Policy   *ActionGatewaySessionPolicy   `json:"policy,omitempty"`
	Tools    []string                      `json:"tools,omitempty"`
	Config   json.RawMessage               `json:"config,omitempty"`
	Network  *ActionGatewayEndpointNetwork `json:"network,omitempty"`
	Insights *ActionGatewaySessionInsights `json:"insights,omitempty"`
}

// MarshalJSON preserves an explicitly empty tools selection.
func (spec ActionGatewayEndpointVersionSpec) MarshalJSON() ([]byte, error) {
	type wire ActionGatewayEndpointVersionSpec
	encoded, err := json.Marshal(wire(spec))
	if err != nil || spec.Tools == nil {
		return encoded, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	fields["tools"], err = json.Marshal(spec.Tools)
	if err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}

// ActionGatewayEndpointVersion is an immutable, live endpoint version.
type ActionGatewayEndpointVersion struct {
	ID                        string                                `json:"id"`
	EndpointID                string                                `json:"endpoint_id"`
	VersionNumber             int64                                 `json:"version_number"`
	PromotedFromVersionNumber int64                                 `json:"promoted_from_version_number,omitempty"`
	Name                      string                                `json:"name"`
	ActorID                   string                                `json:"actor_id"`
	Policy                    *ActionGatewaySessionPolicy           `json:"policy"`
	Tools                     *ActionGatewayEndpointVersionToolSet  `json:"tools"`
	Config                    json.RawMessage                       `json:"config"`
	Network                   *ActionGatewayEndpointNetworkResponse `json:"network"`
	MCPURL                    string                                `json:"mcp_url"`
	CreatedAt                 *Timestamp                            `json:"created_at"`
	UpdatedAt                 *Timestamp                            `json:"updated_at"`
}

// ActionGatewayEndpointVersionToolSet is a version's canonical pinned tool
// selection. A nil set means every tool; an empty references list means none.
type ActionGatewayEndpointVersionToolSet struct {
	References []ActionGatewaySessionToolReference `json:"references"`
}

// ActionGatewayEndpoint is a named lineage of immutable versions.
type ActionGatewayEndpoint struct {
	ID          string                        `json:"id"`
	Name        string                        `json:"name"`
	ActorID     string                        `json:"actor_id"`
	IsDefault   bool                          `json:"is_default"`
	HeadVersion *ActionGatewayEndpointVersion `json:"head_version"`
	Insights    *ActionGatewaySessionInsights `json:"insights"`
	MCPURL      string                        `json:"mcp_url"`
	CreatedAt   *Timestamp                    `json:"created_at"`
	UpdatedAt   *Timestamp                    `json:"updated_at"`
}

// ActionGatewayEndpointCreateRequest creates an endpoint and its first version atomically.
type ActionGatewayEndpointCreateRequest struct {
	Name     string                            `json:"name"`
	Version  *ActionGatewayEndpointVersionSpec `json:"version"`
	Insights *ActionGatewaySessionInsights     `json:"insights,omitempty"`
}

// ActionGatewayEndpointUpdateRequest renames an endpoint, changes its Insights
// consent, or both. Send a name, at least one insights signal, or both.
type ActionGatewayEndpointUpdateRequest struct {
	Name     string                        `json:"name,omitempty"`
	Insights *ActionGatewaySessionInsights `json:"insights,omitempty"`
}

// ActionGatewayEndpointPromotionRequest copies a live version's configuration into a new head.
type ActionGatewayEndpointPromotionRequest struct {
	VersionID string `json:"version_id"`
}

// ActionGatewayEndpointResponse returns one endpoint with its live head.
type ActionGatewayEndpointResponse struct {
	Endpoint *ActionGatewayEndpoint `json:"endpoint"`
}

// ActionGatewayEndpointVersionResponse returns one immutable live version.
type ActionGatewayEndpointVersionResponse struct {
	Version *ActionGatewayEndpointVersion `json:"version"`
}

// ActionGatewayEndpointPublishResponse is returned when a new head version is created.
type ActionGatewayEndpointPublishResponse struct {
	Endpoint      *ActionGatewayEndpoint        `json:"endpoint"`
	Version       *ActionGatewayEndpointVersion `json:"version"`
	MCPURL        string                        `json:"mcp_url"`
	VersionMCPURL string                        `json:"version_mcp_url"`
}

// ActionGatewayEndpointsResponse contains a keyset page of endpoints.
type ActionGatewayEndpointsResponse struct {
	Endpoints     []ActionGatewayEndpoint `json:"endpoints"`
	NextPageToken string                  `json:"next_page_token"`
}

// ActionGatewayEndpointVersionsResponse contains a keyset page of live versions.
type ActionGatewayEndpointVersionsResponse struct {
	Versions      []ActionGatewayEndpointVersion `json:"versions"`
	NextPageToken string                         `json:"next_page_token"`
}

// ActionGatewayEndpointListOptions filters and paginates listed endpoints.
type ActionGatewayEndpointListOptions struct {
	PageSize  int    `url:"page_size,omitempty"`
	PageToken string `url:"page_token,omitempty"`
	Name      string `url:"name,omitempty"`
	ActorID   string `url:"actor_id,omitempty"`
}

// ActionGatewayEndpointSearchOptions searches endpoints by text and actor.
type ActionGatewayEndpointSearchOptions struct {
	Query     string `url:"query,omitempty"`
	ActorID   string `url:"actor_id,omitempty"`
	PageSize  int    `url:"page_size,omitempty"`
	PageToken string `url:"page_token,omitempty"`
}

func actionGatewayEndpointPath(endpointID string) (string, error) {
	return actionGatewayItem(actionGatewayPath+"/endpoints", actionGatewayPathPart{"endpointID", endpointID})
}

// Create creates an endpoint and its first version atomically.
func (s *ActionGatewayEndpointsService) Create(ctx context.Context, body *ActionGatewayEndpointCreateRequest) (*ActionGatewayEndpointPublishResponse, *Response, error) {
	if body == nil {
		return nil, nil, NewArgError("body", "cannot be nil")
	}
	return actionGatewayRequest[ActionGatewayEndpointPublishResponse](ctx, s.client, http.MethodPost, actionGatewayPath+"/endpoints", nil, body)
}

// List lists endpoints with keyset pagination, newest creation time first.
func (s *ActionGatewayEndpointsService) List(ctx context.Context, opt *ActionGatewayEndpointListOptions) (*ActionGatewayEndpointsResponse, *Response, error) {
	return actionGatewayRequest[ActionGatewayEndpointsResponse](ctx, s.client, http.MethodGet, actionGatewayPath+"/endpoints", opt, nil)
}

// Search searches endpoints by name, endpoint ID, actor ID, or agent name.
func (s *ActionGatewayEndpointsService) Search(ctx context.Context, opt *ActionGatewayEndpointSearchOptions) (*ActionGatewayEndpointsResponse, *Response, error) {
	return actionGatewayRequest[ActionGatewayEndpointsResponse](ctx, s.client, http.MethodGet, actionGatewayPath+"/endpoints/search", opt, nil)
}

// Get retrieves an endpoint, including its live head version when one exists.
func (s *ActionGatewayEndpointsService) Get(ctx context.Context, endpointID string) (*ActionGatewayEndpointResponse, *Response, error) {
	path, err := actionGatewayEndpointPath(endpointID)
	if err != nil {
		return nil, nil, err
	}
	return actionGatewayRequest[ActionGatewayEndpointResponse](ctx, s.client, http.MethodGet, path, nil, nil)
}

// Update renames an endpoint, changes its Insights consent, or both.
func (s *ActionGatewayEndpointsService) Update(ctx context.Context, endpointID string, body *ActionGatewayEndpointUpdateRequest) (*ActionGatewayEndpointResponse, *Response, error) {
	if body == nil {
		return nil, nil, NewArgError("body", "cannot be nil")
	}
	path, err := actionGatewayEndpointPath(endpointID)
	if err != nil {
		return nil, nil, err
	}
	return actionGatewayRequest[ActionGatewayEndpointResponse](ctx, s.client, http.MethodPatch, path, nil, body)
}

// Delete deletes an endpoint and all its versions.
func (s *ActionGatewayEndpointsService) Delete(ctx context.Context, endpointID string) (*Response, error) {
	path, err := actionGatewayEndpointPath(endpointID)
	if err != nil {
		return nil, err
	}
	_, response, err := actionGatewayRequest[struct{}](ctx, s.client, http.MethodDelete, path, nil, nil)
	return response, err
}

// ListVersions lists an endpoint's live versions by version number descending.
func (s *ActionGatewayEndpointsService) ListVersions(ctx context.Context, endpointID string, opt *ActionGatewayCursorOptions) (*ActionGatewayEndpointVersionsResponse, *Response, error) {
	path, err := actionGatewayEndpointPath(endpointID)
	if err != nil {
		return nil, nil, err
	}
	return actionGatewayRequest[ActionGatewayEndpointVersionsResponse](ctx, s.client, http.MethodGet, path+"/versions", opt, nil)
}

// CreateVersion publishes a new immutable version as the endpoint's head.
func (s *ActionGatewayEndpointsService) CreateVersion(ctx context.Context, endpointID string, body *ActionGatewayEndpointVersionSpec) (*ActionGatewayEndpointPublishResponse, *Response, error) {
	if body == nil {
		return nil, nil, NewArgError("body", "cannot be nil")
	}
	path, err := actionGatewayEndpointPath(endpointID)
	if err != nil {
		return nil, nil, err
	}
	return actionGatewayRequest[ActionGatewayEndpointPublishResponse](ctx, s.client, http.MethodPost, path+"/versions", nil, body)
}

// PromoteVersion copies a live version's configuration into a new, higher-numbered head version.
func (s *ActionGatewayEndpointsService) PromoteVersion(ctx context.Context, endpointID string, body *ActionGatewayEndpointPromotionRequest) (*ActionGatewayEndpointPublishResponse, *Response, error) {
	if body == nil {
		return nil, nil, NewArgError("body", "cannot be nil")
	}
	path, err := actionGatewayEndpointPath(endpointID)
	if err != nil {
		return nil, nil, err
	}
	return actionGatewayRequest[ActionGatewayEndpointPublishResponse](ctx, s.client, http.MethodPost, path+"/versions:promote", nil, body)
}

// GetVersion retrieves a live version by its composite public ID.
func (s *ActionGatewayEndpointsService) GetVersion(ctx context.Context, versionID string) (*ActionGatewayEndpointVersionResponse, *Response, error) {
	path, err := actionGatewayItem(actionGatewayPath+"/endpoint-versions", actionGatewayPathPart{"versionID", versionID})
	if err != nil {
		return nil, nil, err
	}
	return actionGatewayRequest[ActionGatewayEndpointVersionResponse](ctx, s.client, http.MethodGet, path, nil, nil)
}

// DeleteVersion deletes a live version by its composite public ID.
func (s *ActionGatewayEndpointsService) DeleteVersion(ctx context.Context, versionID string) (*Response, error) {
	path, err := actionGatewayItem(actionGatewayPath+"/endpoint-versions", actionGatewayPathPart{"versionID", versionID})
	if err != nil {
		return nil, err
	}
	_, response, err := actionGatewayRequest[struct{}](ctx, s.client, http.MethodDelete, path, nil, nil)
	return response, err
}
