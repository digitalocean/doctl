package godo

import (
	"context"
	"net/http"
)

const actionGatewayProviderCredentialsPath = actionGatewayPath + "/provider-credentials"

// Kinds of provider credential that can be registered.
const (
	ActionGatewayProviderCredentialKindOAuthClient = "PROVIDER_CREDENTIAL_KIND_OAUTH_CLIENT"
	ActionGatewayProviderCredentialKindAPIKey      = "PROVIDER_CREDENTIAL_KIND_API_KEY"
	ActionGatewayProviderCredentialKindNone        = "PROVIDER_CREDENTIAL_KIND_NONE"
)

// Lifecycle states of a provider credential.
const (
	ActionGatewayProviderCredentialStatusActive   = "PROVIDER_CREDENTIAL_STATUS_ACTIVE"
	ActionGatewayProviderCredentialStatusDisabled = "PROVIDER_CREDENTIAL_STATUS_DISABLED"
	ActionGatewayProviderCredentialStatusDeleted  = "PROVIDER_CREDENTIAL_STATUS_DELETED"
)

// ActionGatewayProviderCredentialsService manages the OAuth clients and API
// keys a team registers for tool providers.
type ActionGatewayProviderCredentialsService struct{ client *Client }

// ActionGatewayProviderCredential describes a registered provider credential.
// The API returns metadata only: never an API key, a client ID, or a client
// secret.
type ActionGatewayProviderCredential struct {
	ID             string                                    `json:"id"`
	Provider       string                                    `json:"provider"`
	DisplayName    string                                    `json:"display_name"`
	Kind           string                                    `json:"kind"`
	BaseURL        string                                    `json:"base_url"`
	Status         string                                    `json:"status"`
	CreatedAt      *Timestamp                                `json:"created_at"`
	UpdatedAt      *Timestamp                                `json:"updated_at"`
	OwningUserID   string                                    `json:"owning_user_id"`
	ReusableByTeam bool                                      `json:"reusable_by_team"`
	RedirectURL    string                                    `json:"redirect_url"`
	Scope          string                                    `json:"scope"`
	Discovery      *ActionGatewayProviderCredentialDiscovery `json:"discovery"`
}

// ActionGatewayProviderCredentialDiscovery describes the authorization to
// discover a team-scoped MCP server's tools.
type ActionGatewayProviderCredentialDiscovery struct {
	Status            string     `json:"status"`
	ConsentedByUserID string     `json:"consented_by_user_id"`
	ConsentedAt       *Timestamp `json:"consented_at"`
	AuthorizationURL  string     `json:"authorization_url"`
}

// ActionGatewayProviderCredentialCreateRequest registers a provider
// credential. An API key requires APIKey; an OAuth client requires ClientID
// and ClientSecret.
type ActionGatewayProviderCredentialCreateRequest struct {
	Provider             string                 `json:"provider"`
	Kind                 string                 `json:"kind"`
	DisplayName          string                 `json:"display_name"`
	APIKey               string                 `json:"api_key,omitempty"`
	ClientID             string                 `json:"client_id,omitempty"`
	ClientSecret         string                 `json:"client_secret,omitempty"`
	BaseURL              string                 `json:"base_url,omitempty"`
	CredentialParameters map[string]interface{} `json:"credential_parameters,omitempty"`
	ReusableByTeam       bool                   `json:"reusable_by_team,omitempty"`
	RedirectURL          string                 `json:"redirect_url,omitempty"`
}

// ActionGatewayProviderCredentialUpdateRequest changes only the fields it
// sets. Changing an OAuth client's secret or base URL also requires its
// registered ClientID. A BaseURL pointing to an empty string clears the base
// URL. ReusableByTeam can only be turned on.
type ActionGatewayProviderCredentialUpdateRequest struct {
	DisplayName          string                 `json:"display_name,omitempty"`
	APIKey               string                 `json:"api_key,omitempty"`
	ClientID             string                 `json:"client_id,omitempty"`
	ClientSecret         string                 `json:"client_secret,omitempty"`
	BaseURL              *string                `json:"base_url,omitempty"`
	CredentialParameters map[string]interface{} `json:"credential_parameters,omitempty"`
	ReusableByTeam       bool                   `json:"reusable_by_team,omitempty"`
}

// ActionGatewayProviderCredentialResponse contains one provider credential.
type ActionGatewayProviderCredentialResponse struct {
	Credential *ActionGatewayProviderCredential `json:"credential"`
	// RevokedConnections counts the connections deleted with the credential.
	// It is zero for every operation but Delete.
	RevokedConnections int `json:"revoked_connections"`
}

// ActionGatewayProviderCredentialsResponse contains the provider credentials
// visible to the caller. The list is not paginated.
type ActionGatewayProviderCredentialsResponse struct {
	Credentials []ActionGatewayProviderCredential `json:"credentials"`
}

// ActionGatewayProviderCredentialListOptions filters provider credentials.
type ActionGatewayProviderCredentialListOptions struct {
	Provider string `url:"provider,omitempty"`
}

// Create registers a provider credential.
func (s *ActionGatewayProviderCredentialsService) Create(ctx context.Context, body *ActionGatewayProviderCredentialCreateRequest) (*ActionGatewayProviderCredentialResponse, *Response, error) {
	if body == nil {
		return nil, nil, NewArgError("body", "cannot be nil")
	}
	return actionGatewayRequest[ActionGatewayProviderCredentialResponse](ctx, s.client, http.MethodPost, actionGatewayProviderCredentialsPath, nil, body)
}

// List lists the provider credentials visible to the caller.
func (s *ActionGatewayProviderCredentialsService) List(ctx context.Context, opt *ActionGatewayProviderCredentialListOptions) (*ActionGatewayProviderCredentialsResponse, *Response, error) {
	return actionGatewayRequest[ActionGatewayProviderCredentialsResponse](ctx, s.client, http.MethodGet, actionGatewayProviderCredentialsPath, opt, nil)
}

// Get retrieves one provider credential.
func (s *ActionGatewayProviderCredentialsService) Get(ctx context.Context, id string) (*ActionGatewayProviderCredentialResponse, *Response, error) {
	path, err := actionGatewayItem(actionGatewayProviderCredentialsPath, actionGatewayPathPart{"id", id})
	if err != nil {
		return nil, nil, err
	}
	return actionGatewayRequest[ActionGatewayProviderCredentialResponse](ctx, s.client, http.MethodGet, path, nil, nil)
}

// Update changes a provider credential in place. Its connections keep using
// the same credential ID.
func (s *ActionGatewayProviderCredentialsService) Update(ctx context.Context, id string, body *ActionGatewayProviderCredentialUpdateRequest) (*ActionGatewayProviderCredentialResponse, *Response, error) {
	if body == nil {
		return nil, nil, NewArgError("body", "cannot be nil")
	}
	path, err := actionGatewayItem(actionGatewayProviderCredentialsPath, actionGatewayPathPart{"id", id})
	if err != nil {
		return nil, nil, err
	}
	return actionGatewayRequest[ActionGatewayProviderCredentialResponse](ctx, s.client, http.MethodPatch, path, nil, body)
}

// Delete deletes a provider credential and every connection that uses it.
func (s *ActionGatewayProviderCredentialsService) Delete(ctx context.Context, id string) (*ActionGatewayProviderCredentialResponse, *Response, error) {
	path, err := actionGatewayItem(actionGatewayProviderCredentialsPath, actionGatewayPathPart{"id", id})
	if err != nil {
		return nil, nil, err
	}
	return actionGatewayRequest[ActionGatewayProviderCredentialResponse](ctx, s.client, http.MethodDelete, path, nil, nil)
}
