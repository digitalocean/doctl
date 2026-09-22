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

package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveServerMetadata(t *testing.T) {
	t.Run("uses the advertised endpoints", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, metadataPath, r.URL.Path)
			writeJSON(t, w, http.StatusOK, map[string]any{
				"issuer":                           "https://cloud.digitalocean.com",
				"authorization_endpoint":           "https://cloud.digitalocean.com/v1/oauth/authorize",
				"token_endpoint":                   "https://cloud.digitalocean.com/v1/oauth/token",
				"registration_endpoint":            "https://cloud.digitalocean.com/v1/oauth/register",
				"revocation_endpoint":              "https://cloud.digitalocean.com/v1/oauth/revoke",
				"code_challenge_methods_supported": []string{"S256"},
			})
		}))
		defer server.Close()

		md := ResolveServerMetadata(context.Background(), server.Client(), server.URL)

		assert.Equal(t, "https://cloud.digitalocean.com", md.Issuer)
		assert.Equal(t, "https://cloud.digitalocean.com/v1/oauth/token", md.TokenEndpoint)
		assert.Equal(t, []string{"S256"}, md.CodeChallengeMethodsSupported)
	})

	t.Run("falls back to the well-known paths", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer server.Close()

		md := ResolveServerMetadata(context.Background(), server.Client(), server.URL+"/")

		assert.Equal(t, server.URL, md.Issuer)
		assert.Equal(t, server.URL+authorizePath, md.AuthorizationEndpoint)
		assert.Equal(t, server.URL+tokenPath, md.TokenEndpoint)
		assert.Equal(t, server.URL+registrationPath, md.RegistrationEndpoint)
		assert.Equal(t, server.URL+revocationPath, md.RevocationEndpoint)
	})

	t.Run("defaults to the DigitalOcean issuer", func(t *testing.T) {
		md := ResolveServerMetadata(context.Background(), failingClient(t), "")

		assert.Equal(t, DefaultIssuer, md.Issuer)
		assert.Equal(t, DefaultIssuer+tokenPath, md.TokenEndpoint)
	})
}

func TestRegisterClient(t *testing.T) {
	t.Run("registers a public client", func(t *testing.T) {
		var body registerClientRequest
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"client_id":                  "client-id",
				"redirect_uris":              body.RedirectURIs,
				"registration_access_token":  "registration-token",
				"registration_client_uri":    "https://cloud.digitalocean.com/v1/oauth/register/client-id",
				"token_endpoint_auth_method": TokenEndpointAuthMethodNone,
			})
		}))
		defer server.Close()

		registration, err := RegisterClient(
			context.Background(), server.Client(), server.URL,
			RegistrationRedirectURIs(), "doctl", "https://example.com",
		)
		require.NoError(t, err)

		assert.Equal(t, "client-id", registration.ClientID)
		assert.Equal(t, "registration-token", registration.RegistrationAccessToken)

		assert.Equal(t, TokenEndpointAuthMethodNone, body.TokenEndpointAuthMethod)
		assert.Equal(t, []string{grantTypeAuthorizationCode, grantTypeRefreshToken}, body.GrantTypes)
		assert.Equal(t, []string{responseTypeCode}, body.ResponseTypes)
		assert.Equal(t, "doctl", body.ClientName)
		assert.Equal(t, []string{"http://127.0.0.1/callback", "http://localhost/callback"}, body.RedirectURIs)
	})

	t.Run("surfaces registration errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusBadRequest, map[string]any{
				"error":             "invalid_client_metadata",
				"error_description": "grant_types must include \"authorization_code\"",
			})
		}))
		defer server.Close()

		_, err := RegisterClient(context.Background(), server.Client(), server.URL, RegistrationRedirectURIs(), "doctl", "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid_client_metadata")
		assert.Contains(t, err.Error(), "grant_types must include")
	})

	t.Run("requires a redirect URI", func(t *testing.T) {
		_, err := RegisterClient(context.Background(), failingClient(t), "https://example.com", nil, "doctl", "")
		assert.Error(t, err)
	})
}

func TestExchangeCode(t *testing.T) {
	expiresIn := int64(3600)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, grantTypeAuthorizationCode, r.FormValue("grant_type"))
		assert.Equal(t, "auth-code", r.FormValue("code"))
		assert.Equal(t, "client-id", r.FormValue("client_id"))
		assert.Equal(t, "verifier", r.FormValue("code_verifier"))
		assert.Equal(t, "http://127.0.0.1:1234/callback", r.FormValue("redirect_uri"))
		assert.Empty(t, r.FormValue("client_secret"), "public clients must not send a secret")
		assert.Empty(t, r.FormValue("resource"))

		writeJSON(t, w, http.StatusOK, map[string]any{
			"access_token":  "doo_v1_access",
			"refresh_token": "dor_v1_refresh",
			"token_type":    "Bearer",
			"expires_in":    expiresIn,
			"scope":         "api:read api:write",
			"info": map[string]any{
				"name":      "Sammy",
				"email":     "sammy@example.com",
				"team_name": "My Team",
			},
		})
	}))
	defer server.Close()

	before := time.Now()
	token, err := ExchangeCode(
		context.Background(), server.Client(), server.URL,
		"client-id", "auth-code", "verifier", "http://127.0.0.1:1234/callback",
	)
	require.NoError(t, err)

	assert.Equal(t, "doo_v1_access", token.AccessToken)
	assert.Equal(t, "dor_v1_refresh", token.RefreshToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.Equal(t, "api:read api:write", token.Scope)
	assert.Equal(t, "sammy@example.com", token.Info.Email)
	assert.Equal(t, "My Team", token.Info.TeamName)
	assert.WithinDuration(t, before.Add(time.Duration(expiresIn)*time.Second), token.Expiry, time.Minute)
}

func TestExchangeCodeReportsInvalidClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusUnauthorized, map[string]any{"error": "invalid_client"})
	}))
	defer server.Close()

	_, err := ExchangeCode(context.Background(), server.Client(), server.URL, "client-id", "code", "verifier", "http://127.0.0.1/callback")

	assert.ErrorIs(t, err, ErrInvalidClient)
}

func TestRefreshToken(t *testing.T) {
	t.Run("returns the rotated refresh token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, r.ParseForm())
			assert.Equal(t, grantTypeRefreshToken, r.FormValue("grant_type"))
			assert.Equal(t, "old-refresh", r.FormValue("refresh_token"))

			writeJSON(t, w, http.StatusOK, map[string]any{
				"access_token":  "new-access",
				"refresh_token": "new-refresh",
				"expires_in":    3600,
			})
		}))
		defer server.Close()

		token, err := RefreshToken(context.Background(), server.Client(), server.URL, "client-id", "old-refresh")
		require.NoError(t, err)

		assert.Equal(t, "new-access", token.AccessToken)
		assert.Equal(t, "new-refresh", token.RefreshToken)
	})

	t.Run("keeps the presented refresh token when none is returned", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{"access_token": "new-access"})
		}))
		defer server.Close()

		token, err := RefreshToken(context.Background(), server.Client(), server.URL, "", "old-refresh")
		require.NoError(t, err)

		assert.Equal(t, "old-refresh", token.RefreshToken)
		assert.True(t, token.Expiry.IsZero())
	})

	t.Run("errors when the grant is rejected", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusUnauthorized, map[string]any{
				"error":             "invalid_grant",
				"error_description": "refresh token is expired",
			})
		}))
		defer server.Close()

		_, err := RefreshToken(context.Background(), server.Client(), server.URL, "client-id", "old-refresh")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "refresh token is expired")
		assert.NotErrorIs(t, err, ErrInvalidClient)
	})
}

func TestRevokeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "Bearer doo_v1_access", r.Header.Get("Authorization"))
		assert.Equal(t, "doo_v1_access", r.FormValue("token"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	assert.NoError(t, RevokeToken(context.Background(), server.Client(), server.URL, "doo_v1_access"))
}

func TestTokenGrantResponseUsesLegacyBearerField(t *testing.T) {
	grant := &tokenGrantResponse{AccessToken: "token", LegacyBearer: "bearer"}

	assert.Equal(t, "bearer", grant.token(time.Now()).TokenType)
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(body))
}

// failingClient fails any request, asserting that a code path does not reach
// the network.
func failingClient(t *testing.T) *http.Client {
	t.Helper()

	return &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, assert.AnError
		}),
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
