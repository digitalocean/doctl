package godo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	hostedAgentsConfigsBasePath   = "/v2/agents/configs"
	hostedAgentConfigByIDPath     = hostedAgentsConfigsBasePath + "/%s"
	hostedAgentConfigSessionsPath = hostedAgentConfigByIDPath + "/sessions"
)

// HostedAgentConfigCredentialSlot is redacted credential metadata on GET/create.
// It echoes only the manifest's declaration for the slot; secret values are
// never returned. Whether a slot is fulfilled is implied by Source: a
// tenantSecret is always stored at create, and an oauth slot is a declaration
// only until oauth brokering is implemented.
type HostedAgentConfigCredentialSlot struct {
	Name     string `json:"name"`
	Source   string `json:"source,omitempty"`
	Provider string `json:"provider,omitempty"`
}

// HostedAgentConfig is an immutable team-scoped Agent Config.
type HostedAgentConfig struct {
	ID                     string                            `json:"id"`
	Name                   string                            `json:"name"`
	AgentSpecSchemaVersion string                            `json:"agentspec_schema_version"`
	Manifest               json.RawMessage                   `json:"manifest"`
	ContentHash            string                            `json:"content_hash"`
	CreatedBy              string                            `json:"created_by"`
	CreatedAt              Timestamp                         `json:"created_at"`
	UpdatedAt              Timestamp                         `json:"updated_at"`
	Credentials            []HostedAgentConfigCredentialSlot `json:"credentials,omitempty"`
	// Warnings carries non-fatal create-time advisories computed from the
	// manifest. It is populated on the create response only.
	Warnings []string `json:"warnings,omitempty"`
}

// HostedAgentConfigSummary is the list view (no manifest / credentials).
type HostedAgentConfigSummary struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	AgentSpecSchemaVersion string    `json:"agentspec_schema_version"`
	ContentHash            string    `json:"content_hash"`
	CreatedBy              string    `json:"created_by"`
	CreatedAt              Timestamp `json:"created_at"`
	UpdatedAt              Timestamp `json:"updated_at"`
}

// HostedAgentConfigCreateRequest is the body for POST /v2/agents/configs.
// Credentials are not request fields: every secret is declared in the manifest
// under spec.secrets, and a tenantSecret slot carries its plaintext in a
// write-only spec.secrets[].value that is extracted server-side and never
// persisted or returned. Sending the retired secrets or oauth_assignments maps
// is rejected with a 400.
type HostedAgentConfigCreateRequest struct {
	Name         string `json:"name"`
	ManifestYAML string `json:"manifest_yaml"`
}

// HostedAgentConfigListOptions specifies optional list pagination.
type HostedAgentConfigListOptions struct {
	PageToken string `url:"page_token,omitempty"`
	PageSize  int    `url:"page_size,omitempty"`
}

// HostedAgentConfigsListResponse is returned by GET /v2/agents/configs.
type HostedAgentConfigsListResponse struct {
	Configs       []HostedAgentConfigSummary `json:"configs"`
	NextPageToken string                     `json:"next_page_token"`
}

type hostedAgentConfigRoot struct {
	Config *HostedAgentConfig `json:"config"`
}

// ListAgentConfigs lists active Agent Configs for the caller's team.
func (s *HostedAgentsServiceOp) ListAgentConfigs(ctx context.Context, opt *HostedAgentConfigListOptions) (*HostedAgentConfigsListResponse, *Response, error) {
	path, err := addOptions(hostedAgentsConfigsBasePath, opt)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentConfigsListResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetAgentConfig returns one Agent Config with redacted credential slots.
func (s *HostedAgentsServiceOp) GetAgentConfig(ctx context.Context, configID string) (*HostedAgentConfig, *Response, error) {
	if configID == "" {
		return nil, nil, errors.New("hosted agents: config id is required")
	}
	path := fmt.Sprintf(hostedAgentConfigByIDPath, configID)
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(hostedAgentConfigRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Config == nil {
		return nil, resp, errors.New("hosted agents: get config returned no config")
	}
	return root.Config, resp, nil
}

// CreateAgentConfig creates an immutable Agent Config from a name and
// agents.yaml. Secret values belong in spec.secrets[].value inside the
// manifest, not as separate request fields.
func (s *HostedAgentsServiceOp) CreateAgentConfig(ctx context.Context, create *HostedAgentConfigCreateRequest) (*HostedAgentConfig, *Response, error) {
	if create == nil {
		return nil, nil, errors.New("hosted agents: create request is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, hostedAgentsConfigsBasePath, create)
	if err != nil {
		return nil, nil, err
	}
	root := new(hostedAgentConfigRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Config == nil {
		return nil, resp, errors.New("hosted agents: create config returned no config")
	}
	return root.Config, resp, nil
}

// DeleteAgentConfig soft-deletes an Agent Config. The API returns HTTP 204.
func (s *HostedAgentsServiceOp) DeleteAgentConfig(ctx context.Context, configID string) (*Response, error) {
	if configID == "" {
		return nil, errors.New("hosted agents: config id is required")
	}
	path := fmt.Sprintf(hostedAgentConfigByIDPath, configID)
	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// ListAgentConfigSessions lists sessions created from a config.
func (s *HostedAgentsServiceOp) ListAgentConfigSessions(ctx context.Context, configID string, opt *HostedAgentSessionListOptions) (*HostedAgentSessionsListResponse, *Response, error) {
	if configID == "" {
		return nil, nil, errors.New("hosted agents: config id is required")
	}
	path := fmt.Sprintf(hostedAgentConfigSessionsPath, configID)
	path, err := addOptions(path, opt)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentSessionsListResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}
