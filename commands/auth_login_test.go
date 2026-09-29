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
	config, token := newOAuthTestCmdConfig(t, "https://cloud.example.com")

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

	assert.Equal(t, "doo_v1_access", *token)

	assert.Equal(t, oauth.DefaultClientID, capturedOpts.ClientID, "doctl signs in as its own published application")
	assert.Empty(t, capturedOpts.Scopes, "without --scope the authorization screen decides the permissions")
	assert.Equal(t, "https://cloud.example.com/v1/oauth/token", capturedOpts.Metadata.TokenEndpoint)
	assert.False(t, capturedOpts.NoBrowser)

	state := loadOAuthTokenState(doctl.ArgDefaultContext)
	require.NotNil(t, state)
	assert.Equal(t, "dor_v1_refresh", state.RefreshToken)
	assert.Equal(t, oauth.DefaultClientID, state.ClientID)
	assert.Equal(t, "https://cloud.example.com/v1/oauth/token", state.TokenEndpoint)
	assert.False(t, state.ExpiresAt.IsZero())
}

func TestRunAuthLoginDoesNotCallTheAuthorizationServerBeforeTheBrowser(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()

	config, _ := newOAuthTestCmdConfig(t, server.URL)
	stubOAuthLogin(t, func(_ context.Context, _ oauth.LoginOptions) (*oauth.Token, error) {
		return &oauth.Token{AccessToken: "doo_v1_access"}, nil
	})

	require.NoError(t, RunAuthLogin(config))

	assert.Empty(t, requests, "endpoints are derived locally, so no metadata or registration request is made")
}

func TestRunAuthLoginHonorsClientIDFlag(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	config.Doit.Set(config.NS, doctl.ArgOAuthClientID, "staging-client")

	var capturedOpts oauth.LoginOptions
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		capturedOpts = opts
		return &oauth.Token{AccessToken: "doo_v1_access", RefreshToken: "dor_v1_refresh"}, nil
	})

	require.NoError(t, RunAuthLogin(config))

	assert.Equal(t, "staging-client", capturedOpts.ClientID)
	assert.Equal(t, "staging-client", loadOAuthTokenState(doctl.ArgDefaultContext).ClientID)
	assert.Equal(t, oauth.DefaultClientID, viper.Get(config.NS+"."+doctl.ArgOAuthClientID))
}

func TestRunAuthLoginClientIDAppliesOnlyToThisInvocation(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	// Left behind by an earlier `doctl auth login --client-id staging-client`,
	// which writeConfig saves with the rest of the settings.
	config.Doit.Set(config.NS, doctl.ArgOAuthClientID, "staging-client")
	config.Doit.(*doctl.TestConfig).IsSetMap[doctl.ArgOAuthClientID] = false

	var capturedOpts oauth.LoginOptions
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		capturedOpts = opts
		return &oauth.Token{AccessToken: "doo_v1_access"}, nil
	})

	require.NoError(t, RunAuthLogin(config))

	assert.Equal(t, oauth.DefaultClientID, capturedOpts.ClientID)
}

func TestRunAuthLoginRejectsAnEmptyClientID(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	config.Doit.Set(config.NS, doctl.ArgOAuthClientID, "")

	err := RunAuthLogin(config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--client-id cannot be empty")
}

func TestRunAuthLoginExplainsARejectedClient(t *testing.T) {
	config, token := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	stubOAuthLogin(t, func(_ context.Context, _ oauth.LoginOptions) (*oauth.Token, error) {
		return nil, &oauth.AuthorizationError{Code: "invalid_client"}
	})

	err := RunAuthLogin(config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), oauth.DefaultClientID)
	assert.Empty(t, *token)
}

func TestRunAuthLoginNoBrowserAppliesOnlyToThisInvocation(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	// Left behind by an earlier `doctl auth login --no-browser`, which
	// writeConfig saves with the rest of the settings.
	config.Doit.Set(config.NS, doctl.ArgOAuthNoBrowser, true)
	config.Doit.(*doctl.TestConfig).IsSetMap[doctl.ArgOAuthNoBrowser] = false

	var capturedOpts oauth.LoginOptions
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		capturedOpts = opts
		return &oauth.Token{AccessToken: "doo_v1_access", RefreshToken: "dor_v1_refresh"}, nil
	})

	require.NoError(t, RunAuthLogin(config))
	assert.False(t, capturedOpts.NoBrowser, "a saved --no-browser must not keep the browser closed")
	assert.Equal(t, false, viper.Get(config.NS+"."+doctl.ArgOAuthNoBrowser))
}

func TestRunAuthLoginHonorsNoBrowserFlag(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	config.Doit.Set(config.NS, doctl.ArgOAuthNoBrowser, true)

	var capturedOpts oauth.LoginOptions
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		capturedOpts = opts
		return &oauth.Token{AccessToken: "doo_v1_access", RefreshToken: "dor_v1_refresh"}, nil
	})

	require.NoError(t, RunAuthLogin(config))
	assert.True(t, capturedOpts.NoBrowser)
}

func TestRunAuthLoginRequestsTheGivenScopes(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	config.Doit.Set(config.NS, doctl.ArgOAuthScopes, "read write")

	var capturedOpts oauth.LoginOptions
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		capturedOpts = opts
		return &oauth.Token{AccessToken: "doo_v1_access"}, nil
	})

	require.NoError(t, RunAuthLogin(config))

	assert.Equal(t, []string{"read", "write"}, capturedOpts.Scopes)
	assert.Equal(t, defaultOAuthScopes, viper.Get(config.NS+"."+doctl.ArgOAuthScopes))
}

func TestRunAuthLoginScopeAppliesOnlyToThisInvocation(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	// Left behind by an earlier `doctl auth login --scope "read write"`, which
	// writeConfig saves with the rest of the settings.
	config.Doit.Set(config.NS, doctl.ArgOAuthScopes, "read write")
	config.Doit.(*doctl.TestConfig).IsSetMap[doctl.ArgOAuthScopes] = false

	var capturedOpts oauth.LoginOptions
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		capturedOpts = opts
		return &oauth.Token{AccessToken: "doo_v1_access", RefreshToken: "dor_v1_refresh"}, nil
	})

	require.NoError(t, RunAuthLogin(config))
	assert.Empty(t, capturedOpts.Scopes, "a saved --scope must not be reused on later logins")
	assert.Equal(t, defaultOAuthScopes, viper.Get(config.NS+"."+doctl.ArgOAuthScopes))
}

func TestRunAuthLoginSaveScopePersistsDefault(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	config.Doit.Set(config.NS, doctl.ArgOAuthScopes, "read write")
	config.Doit.Set(config.NS, doctl.ArgOAuthSaveScope, true)

	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		assert.Equal(t, []string{"read", "write"}, opts.Scopes)
		return &oauth.Token{AccessToken: "doo_v1_access", RefreshToken: "dor_v1_refresh"}, nil
	})

	require.NoError(t, RunAuthLogin(config))
	assert.Equal(t, "read write", loadOAuthDefaultScopes())
	assert.Equal(t, false, viper.Get(config.NS+"."+doctl.ArgOAuthSaveScope))
}

func TestRunAuthLoginUsesSavedDefaultScopes(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	storeOAuthDefaultScopes("droplet:read account:read")

	var capturedOpts oauth.LoginOptions
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		capturedOpts = opts
		return &oauth.Token{AccessToken: "doo_v1_access", RefreshToken: "dor_v1_refresh"}, nil
	})

	require.NoError(t, RunAuthLogin(config))
	assert.Equal(t, []string{"droplet:read", "account:read"}, capturedOpts.Scopes)
}

func TestRunAuthLoginExplicitScopeOverridesSavedDefault(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	storeOAuthDefaultScopes("read write")
	config.Doit.Set(config.NS, doctl.ArgOAuthScopes, "droplet:read")

	var capturedOpts oauth.LoginOptions
	stubOAuthLogin(t, func(_ context.Context, opts oauth.LoginOptions) (*oauth.Token, error) {
		capturedOpts = opts
		return &oauth.Token{AccessToken: "doo_v1_access"}, nil
	})

	require.NoError(t, RunAuthLogin(config))
	assert.Equal(t, []string{"droplet:read"}, capturedOpts.Scopes)
	assert.Equal(t, "read write", loadOAuthDefaultScopes(), "override without --save-scope must leave the default alone")
}

func TestRunAuthLoginSaveScopeClearsDefault(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	storeOAuthDefaultScopes("read write")
	config.Doit.Set(config.NS, doctl.ArgOAuthScopes, "")
	config.Doit.Set(config.NS, doctl.ArgOAuthSaveScope, true)

	stubOAuthLogin(t, func(_ context.Context, _ oauth.LoginOptions) (*oauth.Token, error) {
		return &oauth.Token{AccessToken: "doo_v1_access"}, nil
	})

	require.NoError(t, RunAuthLogin(config))
	assert.Empty(t, loadOAuthDefaultScopes())
}

func TestRunAuthLoginSaveScopeRequiresScopeFlag(t *testing.T) {
	config, _ := newOAuthTestCmdConfig(t, "https://cloud.example.com")
	config.Doit.Set(config.NS, doctl.ArgOAuthSaveScope, true)

	err := RunAuthLogin(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--save-scope requires --scope")
}

func TestRunAuthLoginFailure(t *testing.T) {
	config, token := newOAuthTestCmdConfig(t, "https://cloud.example.com")
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
	previousTokens := viper.Get(oauthTokensConfigKey)
	previousDefaultScopes := viper.Get(oauthDefaultScopesConfigKey)
	previousViperContext := viper.Get(doctl.ArgContext)
	ephemeralKeys := []string{
		"test." + doctl.ArgOAuthServer,
		"test." + doctl.ArgOAuthClientID,
		"test." + doctl.ArgOAuthScopes,
		"test." + doctl.ArgOAuthSaveScope,
		"test." + doctl.ArgOAuthCallbackPort,
		"test." + doctl.ArgOAuthNoBrowser,
		"test." + doctl.ArgOAuthTimeout,
	}
	previousEphemeral := make(map[string]any, len(ephemeralKeys))
	for _, key := range ephemeralKeys {
		previousEphemeral[key] = viper.Get(key)
	}

	cfgFileWriter = func() (io.WriteCloser, error) { return &nopWriteCloser{Writer: io.Discard}, nil }
	Context = ""
	viper.Set(doctl.ArgContext, doctl.ArgDefaultContext)
	viper.Set(oauthTokensConfigKey, nil)
	viper.Set(oauthDefaultScopesConfigKey, nil)

	t.Cleanup(func() {
		cfgFileWriter = previousWriter
		Context = previousContext
		viper.Set(oauthTokensConfigKey, previousTokens)
		viper.Set(oauthDefaultScopesConfigKey, previousDefaultScopes)
		viper.Set(doctl.ArgContext, previousViperContext)
		for key, value := range previousEphemeral {
			viper.Set(key, value)
		}
	})
}

func stubOAuthLogin(t *testing.T, login func(context.Context, oauth.LoginOptions) (*oauth.Token, error)) {
	t.Helper()

	previous := oauthLogin
	oauthLogin = login
	t.Cleanup(func() { oauthLogin = previous })
}
