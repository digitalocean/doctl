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

// Package oauth implements the client side of DigitalOcean's OAuth 2.1
// authorization code flow with PKCE (RFC 7636) for native applications
// (RFC 8252), including dynamic client registration (RFC 7591).
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultIssuer is the DigitalOcean authorization server.
	DefaultIssuer = "https://cloud.digitalocean.com"

	// CodeChallengeMethodS256 is the only PKCE code challenge method the
	// authorization server supports.
	CodeChallengeMethodS256 = "S256"

	// TokenEndpointAuthMethodNone designates a public client that
	// authenticates at the token endpoint with PKCE instead of a secret.
	TokenEndpointAuthMethodNone = "none"

	grantTypeAuthorizationCode = "authorization_code"
	grantTypeRefreshToken      = "refresh_token"
	responseTypeCode           = "code"

	metadataPath     = "/.well-known/oauth-authorization-server"
	authorizePath    = "/v1/oauth/authorize"
	tokenPath        = "/v1/oauth/token"
	registrationPath = "/v1/oauth/register"
	revocationPath   = "/v1/oauth/revoke"

	formContentType = "application/x-www-form-urlencoded"
)

// ErrInvalidClient is returned when the authorization server rejects the
// client we are registered as. Callers can use it to discard a stale dynamic
// registration and register again.
var ErrInvalidClient = errors.New("the authorization server rejected this client registration")

// ServerMetadata is the subset of the RFC 8414 authorization server metadata
// document that doctl uses.
type ServerMetadata struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	RegistrationEndpoint          string   `json:"registration_endpoint"`
	RevocationEndpoint            string   `json:"revocation_endpoint"`
	GrantTypesSupported           []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

// ResolveServerMetadata fetches the authorization server metadata document for
// issuer. The document is advisory: any endpoint the server does not advertise
// falls back to its well-known DigitalOcean path, so a server that does not
// serve the metadata document is still usable.
func ResolveServerMetadata(ctx context.Context, client *http.Client, issuer string) *ServerMetadata {
	issuer = strings.TrimSuffix(strings.TrimSpace(issuer), "/")
	if issuer == "" {
		issuer = DefaultIssuer
	}

	md := &ServerMetadata{Issuer: issuer}
	if discovered, err := discoverServerMetadata(ctx, client, issuer); err == nil {
		md = discovered
	}

	if md.Issuer == "" {
		md.Issuer = issuer
	}
	if md.AuthorizationEndpoint == "" {
		md.AuthorizationEndpoint = issuer + authorizePath
	}
	if md.TokenEndpoint == "" {
		md.TokenEndpoint = issuer + tokenPath
	}
	if md.RegistrationEndpoint == "" {
		md.RegistrationEndpoint = issuer + registrationPath
	}
	if md.RevocationEndpoint == "" {
		md.RevocationEndpoint = issuer + revocationPath
	}

	return md
}

func discoverServerMetadata(ctx context.Context, client *http.Client, issuer string) (*ServerMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer+metadataPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := doRequest(client, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authorization server metadata request returned %s", resp.Status)
	}

	md := &ServerMetadata{}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(md); err != nil {
		return nil, err
	}

	return md, nil
}

// ClientRegistration is an RFC 7591 client information response for a
// dynamically registered public client.
type ClientRegistration struct {
	ClientID                string   `json:"client_id"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at,omitempty"`
	RedirectURIs            []string `json:"redirect_uris,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	RegistrationAccessToken string   `json:"registration_access_token,omitempty"`
	RegistrationClientURI   string   `json:"registration_client_uri,omitempty"`
}

type registerClientRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
}

// RegisterClient performs RFC 7591 dynamic client registration against
// endpoint, registering a public client that authenticates with PKCE. The
// endpoint is unauthenticated, so no token is required.
func RegisterClient(ctx context.Context, client *http.Client, endpoint string, redirectURIs []string, name, uri string) (*ClientRegistration, error) {
	if len(redirectURIs) == 0 {
		return nil, errors.New("at least one redirect URI is required to register a client")
	}

	body, err := json.Marshal(&registerClientRequest{
		RedirectURIs:            redirectURIs,
		TokenEndpointAuthMethod: TokenEndpointAuthMethodNone,
		GrantTypes:              []string{grantTypeAuthorizationCode, grantTypeRefreshToken},
		ResponseTypes:           []string{responseTypeCode},
		ClientName:              name,
		ClientURI:               uri,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := doRequest(client, req)
	if err != nil {
		return nil, fmt.Errorf("registering doctl as an OAuth application: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, responseError(resp, "registering doctl as an OAuth application")
	}

	registration := &ClientRegistration{}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(registration); err != nil {
		return nil, fmt.Errorf("decoding client registration response: %w", err)
	}
	if registration.ClientID == "" {
		return nil, errors.New("the authorization server did not return a client ID")
	}

	return registration, nil
}

// Token is an access token grant issued by the authorization server.
type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scope        string
	// Expiry is the absolute time the access token expires. It is the zero
	// value when the server does not report an expiration.
	Expiry time.Time
	Info   TokenInfo
}

// TokenInfo identifies the user and team a token was issued for.
type TokenInfo struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	UUID     string `json:"uuid"`
	TeamUUID string `json:"team_uuid"`
	TeamName string `json:"team_name"`
}

// tokenGrantResponse mirrors the authorization server's token response. Public
// clients receive the RFC 6749 token_type field; confidential clients receive
// the legacy bearer field instead.
type tokenGrantResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	LegacyBearer string     `json:"bearer"`
	ExpiresIn    *int64     `json:"expires_in"`
	Scope        string     `json:"scope"`
	Info         *TokenInfo `json:"info"`
}

func (r *tokenGrantResponse) token(now time.Time) *Token {
	t := &Token{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		TokenType:    r.TokenType,
		Scope:        r.Scope,
	}
	if t.TokenType == "" {
		t.TokenType = r.LegacyBearer
	}
	if r.ExpiresIn != nil && *r.ExpiresIn > 0 {
		t.Expiry = now.Add(time.Duration(*r.ExpiresIn) * time.Second)
	}
	if r.Info != nil {
		t.Info = *r.Info
	}

	return t
}

// ExchangeCode exchanges an authorization code for a token grant, proving
// possession of the PKCE code verifier in place of a client secret.
func ExchangeCode(ctx context.Context, client *http.Client, tokenEndpoint, clientID, code, codeVerifier, redirectURI string) (*Token, error) {
	return postTokenRequest(ctx, client, tokenEndpoint, url.Values{
		"grant_type":    {grantTypeAuthorizationCode},
		"code":          {code},
		"client_id":     {clientID},
		"code_verifier": {codeVerifier},
		"redirect_uri":  {redirectURI},
	}, "exchanging the authorization code")
}

// RefreshToken trades a refresh token for a new token grant. Refresh tokens
// are single use: the response carries the replacement refresh token.
func RefreshToken(ctx context.Context, client *http.Client, tokenEndpoint, clientID, refreshToken string) (*Token, error) {
	values := url.Values{
		"grant_type":    {grantTypeRefreshToken},
		"refresh_token": {refreshToken},
	}
	if clientID != "" {
		values.Set("client_id", clientID)
	}

	token, err := postTokenRequest(ctx, client, tokenEndpoint, values, "refreshing the access token")
	if err != nil {
		return nil, err
	}

	// The server rotates refresh tokens, but tolerate one that reuses the
	// presented token rather than losing the ability to refresh again.
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}

	return token, nil
}

func postTokenRequest(ctx context.Context, client *http.Client, endpoint string, values url.Values, action string) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", formContentType)
	req.Header.Set("Accept", "application/json")

	resp, err := doRequest(client, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, responseError(resp, action)
	}

	grant := &tokenGrantResponse{}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(grant); err != nil {
		return nil, fmt.Errorf("decoding the token response: %w", err)
	}
	if grant.AccessToken == "" {
		return nil, errors.New("the authorization server did not return an access token")
	}

	return grant.token(time.Now()), nil
}

// RevokeToken invalidates an access token. The authorization server requires
// the token both as a bearer credential and as a form value.
func RevokeToken(ctx context.Context, client *http.Client, revocationEndpoint, accessToken string) error {
	values := url.Values{"token": {accessToken}}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revocationEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", formContentType)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := doRequest(client, req)
	if err != nil {
		return fmt.Errorf("revoking the access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return responseError(resp, "revoking the access token")
	}

	return nil
}

func doRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

// responseError converts an OAuth error response body (RFC 6749 section 5.2,
// RFC 7591 section 3.2.2) into an error.
func responseError(resp *http.Response, action string) error {
	var body struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
		Message     string `json:"message"`
		ID          string `json:"id"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body)

	code := body.Error
	if code == "" {
		code = body.ID
	}
	detail := body.Description
	if detail == "" {
		detail = body.Message
	}

	var err error
	switch {
	case code != "" && detail != "":
		err = fmt.Errorf("%s: %s: %s", action, code, detail)
	case code != "":
		err = fmt.Errorf("%s: %s", action, code)
	case detail != "":
		err = fmt.Errorf("%s: %s", action, detail)
	default:
		err = fmt.Errorf("%s: the authorization server returned %s", action, resp.Status)
	}

	if isInvalidClientCode(code) {
		return fmt.Errorf("%w: %s", ErrInvalidClient, err)
	}

	return err
}

func isInvalidClientCode(code string) bool {
	switch code {
	case "invalid_client", "unauthorized_client", "invalid_redirect_uri":
		return true
	default:
		return false
	}
}
