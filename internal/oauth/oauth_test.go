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

func TestServerMetadataFor(t *testing.T) {
	t.Run("derives the endpoints from the issuer", func(t *testing.T) {
		md := ServerMetadataFor("https://cloud.example.com/")

		assert.Equal(t, "https://cloud.example.com", md.Issuer)
		assert.Equal(t, "https://cloud.example.com"+authorizePath, md.AuthorizationEndpoint)
		assert.Equal(t, "https://cloud.example.com"+tokenPath, md.TokenEndpoint)
		assert.Equal(t, "https://cloud.example.com"+revocationPath, md.RevocationEndpoint)
	})

	t.Run("defaults to the DigitalOcean issuer", func(t *testing.T) {
		md := ServerMetadataFor("  ")

		assert.Equal(t, DefaultIssuer, md.Issuer)
		assert.Equal(t, DefaultIssuer+tokenPath, md.TokenEndpoint)
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
