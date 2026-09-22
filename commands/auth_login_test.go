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

package commands

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/oauth"
)

func TestRunAuthLogin(t *testing.T) {
	var registrations int
	server := newTestAuthorizationServer(t, func() int { registrations++; return registrations })
	defer server.Close()

	config, token := newOAuthTestCmdConfig(t, server.URL)

	var capturedOpts oauth.LoginOptions
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		capturedOpts = opts
		return &oauth.Token{
			AccessToken:  "doo_v1_access",
			RefreshToken: "dor_v1_refresh",
			Scope:        "api:read api:write",
			Expiry:       time.Now().Add(time.Hour),
			Info:         oauth.TokenInfo{Email: "sammy@example.com", TeamName: "My Team"},
		}, nil
	})

	require.NoError(t, RunAuthLogin(config))

	assert.Equal(t, 1, registrations)
	assert.Equal(t, "doo_v1_access", *token)

	assert.Equal(t, "client-1", capturedOpts.ClientID)
	assert.Empty(t, capturedOpts.Scopes, "without --scope the authorization screen decides the permissions")
	assert.Equal(t, server.URL+"/v1/oauth/token", capturedOpts.Metadata.TokenEndpoint)

	client := loadOAuthClientState()
	require.NotNil(t, client)
	assert.Equal(t, "client-1", client.ClientID)
	assert.Equal(t, server.URL, client.Issuer)
	assert.Equal(t, "registration-token-1", client.RegistrationAccessToken)
	assert.Equal(t, oauth.RegistrationRedirectURIs(), client.RedirectURIs)
	assert.True(t, testClientIssuedAt.Equal(client.RegisteredAt))

	state := loadOAuthTokenState(doctl.ArgDefaultContext)
	require.NotNil(t, state)
	assert.Equal(t, "dor_v1_refresh", state.RefreshToken)
	assert.Equal(t, "client-1", state.ClientID)
	assert.Equal(t, server.URL+"/v1/oauth/token", state.TokenEndpoint)
	assert.False(t, state.ExpiresAt.IsZero())
}

func TestRunAuthLoginRequestsTheGivenScopes(t *testing.T) {
	server := newTestAuthorizationServer(t, func() int { return 1 })
	defer server.Close()

	config, _ := newOAuthTestCmdConfig(t, server.URL)
	config.Doit.Set(config.NS, doctl.ArgOAuthScopes, "read write")

	var capturedOpts oauth.LoginOptions
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		capturedOpts = opts
		return &oauth.Token{AccessToken: "doo_v1_access"}, nil
	})

	require.NoError(t, RunAuthLogin(config))

	assert.Equal(t, []string{"read", "write"}, capturedOpts.Scopes)
}

func TestRunAuthLoginReusesTheRegisteredClient(t *testing.T) {
	var registrations int
	server := newTestAuthorizationServer(t, func() int { registrations++; return registrations })
	defer server.Close()

	config, _ := newOAuthTestCmdConfig(t, server.URL)
	storeOAuthClientState(&oauthClientState{Issuer: server.URL, ClientID: "already-registered"})

	var clientID string
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		clientID = opts.ClientID
		return &oauth.Token{AccessToken: "doo_v1_access"}, nil
	})

	require.NoError(t, RunAuthLogin(config))

	assert.Zero(t, registrations, "an existing registration should be reused")
	assert.Equal(t, "already-registered", clientID)
}

func TestRunAuthLoginRegistersAgainWhenTheClientIsRejected(t *testing.T) {
	var registrations int
	server := newTestAuthorizationServer(t, func() int { registrations++; return registrations })
	defer server.Close()

	config, token := newOAuthTestCmdConfig(t, server.URL)
	storeOAuthClientState(&oauthClientState{Issuer: server.URL, ClientID: "deleted-client"})

	var attempts []string
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		attempts = append(attempts, opts.ClientID)
		if opts.ClientID == "deleted-client" {
			return nil, &oauth.AuthorizationError{Code: "invalid_client"}
		}
		return &oauth.Token{AccessToken: "doo_v1_access"}, nil
	})

	require.NoError(t, RunAuthLogin(config))

	assert.Equal(t, []string{"deleted-client", "client-1"}, attempts)
	assert.Equal(t, 1, registrations)
	assert.Equal(t, "doo_v1_access", *token)
}

func TestRunAuthLoginFailure(t *testing.T) {
	server := newTestAuthorizationServer(t, func() int { return 1 })
	defer server.Close()

	config, token := newOAuthTestCmdConfig(t, server.URL)
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		return nil, &oauth.AuthorizationError{Code: "access_denied"}
	})

	err := RunAuthLogin(config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access_denied")
	assert.Empty(t, *token, "a failed login must not store a token")
	assert.Nil(t, loadOAuthTokenState(doctl.ArgDefaultContext))
}

func TestRefreshExpiredOAuthToken(t *testing.T) {
	tokenEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
		assert.Equal(t, "dor_v1_old", r.FormValue("refresh_token"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "doo_v1_new",
			"refresh_token": "dor_v1_new",
			"expires_in":    3600,
		}))
	}))
	defer tokenEndpoint.Close()

	t.Run("renews an expired token", func(t *testing.T) {
		config, token := newOAuthTestCmdConfig(t, tokenEndpoint.URL)
		storeOAuthTokenState(doctl.ArgDefaultContext, &oauthTokenState{
			Issuer:        tokenEndpoint.URL,
			ClientID:      "client-1",
			TokenEndpoint: tokenEndpoint.URL,
			RefreshToken:  "dor_v1_old",
			ExpiresAt:     time.Now().Add(-time.Minute),
		})

		require.NoError(t, refreshExpiredOAuthToken(config))

		assert.Equal(t, "doo_v1_new", *token)

		state := loadOAuthTokenState(doctl.ArgDefaultContext)
		require.NotNil(t, state)
		assert.Equal(t, "dor_v1_new", state.RefreshToken)
		assert.True(t, state.ExpiresAt.After(time.Now()))
	})

	t.Run("leaves a valid token alone", func(t *testing.T) {
		config, token := newOAuthTestCmdConfig(t, tokenEndpoint.URL)
		storeOAuthTokenState(doctl.ArgDefaultContext, &oauthTokenState{
			TokenEndpoint: tokenEndpoint.URL,
			RefreshToken:  "dor_v1_old",
			ExpiresAt:     time.Now().Add(time.Hour),
		})

		require.NoError(t, refreshExpiredOAuthToken(config))

		assert.Empty(t, *token)
	})

	t.Run("ignores contexts without an OAuth session", func(t *testing.T) {
		config, token := newOAuthTestCmdConfig(t, tokenEndpoint.URL)

		require.NoError(t, refreshExpiredOAuthToken(config))

		assert.Empty(t, *token)
	})

	t.Run("ignores a token supplied for this command", func(t *testing.T) {
		config, token := newOAuthTestCmdConfig(t, tokenEndpoint.URL)
		storeOAuthTokenState(doctl.ArgDefaultContext, &oauthTokenState{
			TokenEndpoint: tokenEndpoint.URL,
			RefreshToken:  "dor_v1_old",
			ExpiresAt:     time.Now().Add(-time.Minute),
		})

		Token = "dop_v1_explicit"
		t.Cleanup(func() { Token = "" })

		require.NoError(t, refreshExpiredOAuthToken(config))

		assert.Empty(t, *token)
	})

	t.Run("explains how to recover when the refresh token is rejected", func(t *testing.T) {
		rejecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant"}))
		}))
		defer rejecting.Close()

		config, _ := newOAuthTestCmdConfig(t, rejecting.URL)
		storeOAuthTokenState(doctl.ArgDefaultContext, &oauthTokenState{
			TokenEndpoint: rejecting.URL,
			RefreshToken:  "dor_v1_old",
			ExpiresAt:     time.Now().Add(-time.Minute),
		})

		err := refreshExpiredOAuthToken(config)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "doctl auth login")
	})
}

func TestAuthInitDiscardsTheOAuthSession(t *testing.T) {
	resetOAuthConfig(t)
	storeOAuthTokenState(doctl.ArgDefaultContext, &oauthTokenState{
		RefreshToken: "dor_v1_refresh",
		ExpiresAt:    time.Now().Add(-time.Minute),
	})

	previousToken := viper.Get(doctl.ArgAccessToken)
	viper.Set(doctl.ArgAccessToken, "dop_v1_pat")
	t.Cleanup(func() { viper.Set(doctl.ArgAccessToken, previousToken) })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.oauth.EXPECT().TokenInfo(gomock.Any()).Return(&do.OAuthTokenInfo{}, nil)

		err := RunAuthInit(func() (string, error) { return "", nil })(config)
		require.NoError(t, err)
	})

	assert.Nil(t, loadOAuthTokenState(doctl.ArgDefaultContext))
}

func TestOAuthTokenStateIsScopedToAContext(t *testing.T) {
	resetOAuthConfig(t)

	storeOAuthTokenState("default", &oauthTokenState{RefreshToken: "default-refresh", Scope: "read"})
	storeOAuthTokenState("Your-Team", &oauthTokenState{RefreshToken: "team-refresh"})

	assert.Equal(t, "default-refresh", loadOAuthTokenState("default").RefreshToken)
	assert.Equal(t, "team-refresh", loadOAuthTokenState("your-team").RefreshToken, "context names are case insensitive")
	assert.Nil(t, loadOAuthTokenState("unknown"))

	removeOAuthTokenState("default")

	assert.Nil(t, loadOAuthTokenState("default"))
	assert.NotNil(t, loadOAuthTokenState("your-team"))
}

func TestOAuthTokenStateNeedsRefresh(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		state    *oauthTokenState
		expected bool
	}{
		{name: "no session"},
		{name: "no refresh token", state: &oauthTokenState{ExpiresAt: now.Add(-time.Hour)}},
		{name: "unknown expiry", state: &oauthTokenState{RefreshToken: "refresh"}},
		{name: "valid", state: &oauthTokenState{RefreshToken: "refresh", ExpiresAt: now.Add(time.Hour)}, expected: false},
		{name: "expired", state: &oauthTokenState{RefreshToken: "refresh", ExpiresAt: now.Add(-time.Second)}, expected: true},
		{name: "expiring within the skew", state: &oauthTokenState{RefreshToken: "refresh", ExpiresAt: now.Add(time.Minute)}, expected: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.state.needsRefresh(now))
		})
	}
}

// testClientIssuedAt is the registration timestamp the test authorization
// server reports.
var testClientIssuedAt = time.Date(2026, time.September, 22, 10, 30, 0, 0, time.UTC)

// newTestAuthorizationServer serves the metadata and dynamic client
// registration endpoints doctl calls before starting the browser flow.
func newTestAuthorizationServer(t *testing.T, nextClient func() int) *httptest.Server {
	t.Helper()

	server := httptest.NewUnstartedServer(nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		issuer := "http://" + r.Host
		writeTestJSON(t, w, http.StatusOK, map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/v1/oauth/authorize",
			"token_endpoint":         issuer + "/v1/oauth/token",
			"registration_endpoint":  issuer + "/v1/oauth/register",
			"revocation_endpoint":    issuer + "/v1/oauth/revoke",
		})
	})
	mux.HandleFunc("/v1/oauth/register", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			RedirectURIs []string `json:"redirect_uris"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))

		id := strconv.Itoa(nextClient())
		writeTestJSON(t, w, http.StatusCreated, map[string]any{
			"client_id":                 "client-" + id,
			"client_id_issued_at":       testClientIssuedAt.Unix(),
			"redirect_uris":             request.RedirectURIs,
			"registration_access_token": "registration-token-" + id,
		})
	})
	server.Config.Handler = mux
	server.Start()

	return server
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(body))
}

// newOAuthTestCmdConfig returns a command config wired to an isolated view of
// the doctl configuration, along with a pointer to the access token the
// command stores for the current context.
func newOAuthTestCmdConfig(t *testing.T, issuer string) (*CmdConfig, *string) {
	t.Helper()

	resetOAuthConfig(t)

	doitConfig := doctl.NewTestConfig()
	var storedToken string

	config := &CmdConfig{
		NS:                    "test",
		Doit:                  doitConfig,
		Out:                   io.Discard,
		setContextAccessToken: func(token string) { storedToken = token },
		getContextAccessToken: func() string { return storedToken },
		initServices:          func(*CmdConfig) error { return nil },
	}

	doitConfig.Set(config.NS, doctl.ArgOAuthServer, issuer)
	doitConfig.Set(config.NS, doctl.ArgOAuthTimeout, time.Minute)

	return config, &storedToken
}

// resetOAuthConfig isolates a test from the global viper configuration the
// commands package writes to.
func resetOAuthConfig(t *testing.T) {
	t.Helper()

	previousWriter := cfgFileWriter
	previousContext := Context
	previousClient := viper.Get(oauthClientConfigKey)
	previousTokens := viper.Get(oauthTokensConfigKey)
	previousViperContext := viper.Get(doctl.ArgContext)

	cfgFileWriter = func() (io.WriteCloser, error) { return &nopWriteCloser{Writer: io.Discard}, nil }
	Context = ""
	viper.Set(doctl.ArgContext, doctl.ArgDefaultContext)
	viper.Set(oauthClientConfigKey, nil)
	viper.Set(oauthTokensConfigKey, nil)

	t.Cleanup(func() {
		cfgFileWriter = previousWriter
		Context = previousContext
		viper.Set(oauthClientConfigKey, previousClient)
		viper.Set(oauthTokensConfigKey, previousTokens)
		viper.Set(doctl.ArgContext, previousViperContext)
	})
}

func stubOAuthLogin(t *testing.T, login func(context.Context, oauth.LoginOptions) (*oauth.Token, error)) {
	t.Helper()

	previous := oauthLogin
	oauthLogin = login
	t.Cleanup(func() { oauthLogin = previous })
}
