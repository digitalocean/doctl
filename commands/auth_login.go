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
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/internal/oauth"
)

const (
	// oauthTokensConfigKey holds the per-context OAuth session state used to
	// refresh access tokens.
	oauthTokensConfigKey = "oauth-tokens"
	// oauthDefaultScopesConfigKey holds the optional default scopes reused on
	// later logins when the user passed --save-scope.
	oauthDefaultScopesConfigKey = "oauth-default-scopes"

	// defaultOAuthScopes is empty on purpose: when doctl sends no scope, the
	// authorization screen decides what the token is granted, which lets the
	// user pick the permissions they want rather than the CLI assuming them.
	defaultOAuthScopes = ""

	// oauthRefreshSkew refreshes an access token slightly before it expires so
	// a command that takes a moment to reach the API is not rejected.
	oauthRefreshSkew = 2 * time.Minute

	// oauthHTTPTimeout bounds each individual call to the authorization
	// server. It does not bound the wait for the user's browser.
	oauthHTTPTimeout = 30 * time.Second
)

// oauthLogin runs the browser-based authorization code flow. It is a variable
// so tests can exercise the command without a browser.
var oauthLogin = oauth.Login

// oauthTokenState is the per-context OAuth session persisted in the config
// file. The access token itself lives with the rest of the context's
// credentials; only what is needed to renew it is kept here.
type oauthTokenState struct {
	Issuer        string
	ClientID      string
	TokenEndpoint string
	RefreshToken  string
	Scope         string
	ExpiresAt     time.Time
}

func (s *oauthTokenState) needsRefresh(now time.Time) bool {
	if s == nil || s.RefreshToken == "" {
		return false
	}
	// An unknown expiry means the token was issued by a server that does not
	// report one; leave it alone and let the API reject it if it is stale.
	if s.ExpiresAt.IsZero() {
		return false
	}

	return !now.Add(oauthRefreshSkew).Before(s.ExpiresAt)
}

// RunAuthLogin authenticates doctl using the OAuth 2.1 authorization code flow
// with PKCE, storing the resulting access token in the current context.
func RunAuthLogin(c *CmdConfig) error {
	authContext := currentAuthContext()

	issuer, err := oauthServer(c)
	if err != nil {
		return err
	}
	clientID, err := oauthClientID(c)
	if err != nil {
		return err
	}
	scopes, err := oauthScopes(c)
	if err != nil {
		return err
	}
	saveScope, err := oauthSaveScope(c)
	if err != nil {
		return err
	}
	if saveScope && !c.Doit.IsSet(doctl.ArgOAuthScopes) {
		return errors.New("--save-scope requires --scope")
	}
	port, err := oauthCallbackPort(c)
	if err != nil {
		return err
	}
	noBrowser, err := oauthNoBrowser(c)
	if err != nil {
		return err
	}
	timeout, err := oauthTimeout(c)
	if err != nil {
		return err
	}
	// writeConfig persists every viper setting, so login-only flags from this
	// invocation must not stick around for later runs.
	clearEphemeralOAuthLoginFlags(c)

	ctx := context.Background()
	httpClient := &http.Client{Timeout: oauthHTTPTimeout}
	metadata := oauth.ServerMetadataFor(issuer)

	token, err := oauthLogin(ctx, oauth.LoginOptions{
		Metadata:           metadata,
		ClientID:           clientID,
		Scopes:             strings.Fields(scopes),
		Port:               port,
		Timeout:            timeout,
		HTTPClient:         httpClient,
		NoBrowser:          noBrowser,
		OnAuthorizationURL: authorizationURLPrinter(c, noBrowser),
	})
	if errors.Is(err, oauth.ErrInvalidClient) {
		template.Render(c.Out, `{{error crossmark}}{{nl}}{{nl}}`, nil)
		return fmt.Errorf("%s rejected the doctl application (client ID %s): %s", metadata.Issuer, clientID, err)
	}
	if err != nil {
		template.Render(c.Out, `{{error crossmark}}{{nl}}{{nl}}`, nil)
		return fmt.Errorf("Unable to authenticate with DigitalOcean: %s", err)
	}

	template.Render(c.Out, `{{success checkmark}}{{nl}}{{nl}}`, nil)

	c.setContextAccessToken(token.AccessToken)
	if token.RefreshToken == "" {
		// Without a refresh token there is nothing to renew later, so make
		// sure an earlier session is not left behind to be refreshed over the
		// top of this one.
		removeOAuthTokenState(authContext)
	} else {
		storeOAuthTokenState(authContext, &oauthTokenState{
			Issuer:        metadata.Issuer,
			ClientID:      clientID,
			TokenEndpoint: metadata.TokenEndpoint,
			RefreshToken:  token.RefreshToken,
			Scope:         token.Scope,
			ExpiresAt:     token.Expiry,
		})
	}
	if saveScope {
		storeOAuthDefaultScopes(scopes)
	}

	if err := writeConfig(); err != nil {
		return err
	}

	displayOAuthLoginSummary(c, authContext, token)
	if saveScope {
		displayOAuthDefaultScopesSaved(c, scopes)
	}

	return nil
}

// oauthServer returns the authorization server for this invocation.
// A value left in the config file from an earlier login is ignored.
func oauthServer(c *CmdConfig) (string, error) {
	if !c.Doit.IsSet(doctl.ArgOAuthServer) {
		return oauth.DefaultIssuer, nil
	}
	return c.Doit.GetString(c.NS, doctl.ArgOAuthServer)
}

// oauthClientID returns the OAuth application doctl signs in as. It is the
// application DigitalOcean registered for doctl unless --client-id names
// another one, which is what a non-production authorization server needs.
func oauthClientID(c *CmdConfig) (string, error) {
	if !c.Doit.IsSet(doctl.ArgOAuthClientID) {
		return oauth.DefaultClientID, nil
	}

	clientID, err := c.Doit.GetString(c.NS, doctl.ArgOAuthClientID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(clientID) == "" {
		return "", errors.New("--client-id cannot be empty")
	}

	return clientID, nil
}

// oauthScopes returns the scopes requested for this invocation. An explicit
// --scope wins; otherwise any default saved with --save-scope is used. A bare
// --scope value left in the config file from an earlier login is ignored.
func oauthScopes(c *CmdConfig) (string, error) {
	if c.Doit.IsSet(doctl.ArgOAuthScopes) {
		return c.Doit.GetString(c.NS, doctl.ArgOAuthScopes)
	}
	if saved := loadOAuthDefaultScopes(); saved != "" {
		return saved, nil
	}
	return defaultOAuthScopes, nil
}

// oauthSaveScope reports whether this invocation passed --save-scope.
// A value left in the config file from an earlier login is ignored.
func oauthSaveScope(c *CmdConfig) (bool, error) {
	if !c.Doit.IsSet(doctl.ArgOAuthSaveScope) {
		return false, nil
	}
	return c.Doit.GetBool(c.NS, doctl.ArgOAuthSaveScope)
}

// oauthCallbackPort returns the local callback port for this invocation.
// A value left in the config file from an earlier login is ignored.
func oauthCallbackPort(c *CmdConfig) (int, error) {
	if !c.Doit.IsSet(doctl.ArgOAuthCallbackPort) {
		return 0, nil
	}
	return c.Doit.GetInt(c.NS, doctl.ArgOAuthCallbackPort)
}

// oauthNoBrowser reports whether this invocation passed --no-browser.
// A value left in the config file from an earlier login is ignored.
func oauthNoBrowser(c *CmdConfig) (bool, error) {
	if !c.Doit.IsSet(doctl.ArgOAuthNoBrowser) {
		return false, nil
	}
	return c.Doit.GetBool(c.NS, doctl.ArgOAuthNoBrowser)
}

// oauthTimeout returns how long to wait for browser authorization.
// A value left in the config file from an earlier login is ignored.
func oauthTimeout(c *CmdConfig) (time.Duration, error) {
	if !c.Doit.IsSet(doctl.ArgOAuthTimeout) {
		return oauth.DefaultLoginTimeout, nil
	}
	return c.Doit.GetDuration(c.NS, doctl.ArgOAuthTimeout)
}

// clearEphemeralOAuthLoginFlags resets login-only flags so writeConfig does
// not persist them into the doctl configuration file.
func clearEphemeralOAuthLoginFlags(c *CmdConfig) {
	viper.Set(c.NS+"."+doctl.ArgOAuthServer, oauth.DefaultIssuer)
	viper.Set(c.NS+"."+doctl.ArgOAuthClientID, oauth.DefaultClientID)
	viper.Set(c.NS+"."+doctl.ArgOAuthScopes, defaultOAuthScopes)
	viper.Set(c.NS+"."+doctl.ArgOAuthSaveScope, false)
	viper.Set(c.NS+"."+doctl.ArgOAuthCallbackPort, 0)
	viper.Set(c.NS+"."+doctl.ArgOAuthNoBrowser, false)
	viper.Set(c.NS+"."+doctl.ArgOAuthTimeout, oauth.DefaultLoginTimeout)
}

// loadOAuthDefaultScopes returns the scopes saved with --save-scope, if any.
func loadOAuthDefaultScopes() string {
	return viper.GetString(oauthDefaultScopesConfigKey)
}

// storeOAuthDefaultScopes persists the default scopes for later logins. An
// empty value clears any previously saved default.
func storeOAuthDefaultScopes(scopes string) {
	if scopes == "" {
		viper.Set(oauthDefaultScopesConfigKey, nil)
		return
	}
	viper.Set(oauthDefaultScopesConfigKey, scopes)
}

func authorizationURLPrinter(c *CmdConfig, noBrowser bool) func(string) {
	return func(authURL string) {
		if noBrowser {
			template.Render(c.Out, `{{nl}}Visit the following URL to authorize doctl:{{nl}}{{nl}}  {{underline .}}{{nl}}{{nl}}Waiting for authorization... `, authURL)
			return
		}

		template.Render(c.Out,
			`{{nl}}Opening your browser to authorize doctl. If it does not open, visit:{{nl}}{{nl}}  {{underline .}}{{nl}}{{nl}}Waiting for authorization... `,
			authURL,
		)
	}
}

func displayOAuthLoginSummary(c *CmdConfig, authContext string, token *oauth.Token) {
	who := token.Info.Email
	if who == "" {
		who = token.Info.Name
	}
	if who != "" {
		template.Render(c.Out, `Signed in as {{highlight .}}{{nl}}`, who)
	}
	if token.Info.TeamName != "" {
		template.Render(c.Out, `Team: {{highlight .}}{{nl}}`, token.Info.TeamName)
	}

	template.Render(c.Out, `Saved to the {{highlight .}} authentication context.{{nl}}`, authContext)

	if token.RefreshToken != "" {
		template.Render(c.Out, `{{nl}}{{muted "doctl renews this token automatically; run doctl auth login again if the session is revoked."}}{{nl}}`, nil)
	}
}

func displayOAuthDefaultScopesSaved(c *CmdConfig, scopes string) {
	if scopes == "" {
		template.Render(c.Out, `{{muted "Cleared the saved default scopes."}}{{nl}}`, nil)
		return
	}
	template.Render(c.Out, `{{muted "Saved default scopes for later logins:"}} {{highlight .}}{{nl}}`, scopes)
}

// refreshExpiredOAuthToken renews the current context's access token when it
// was obtained with doctl auth login and has expired. It is a no-op for
// contexts authenticated with a personal access token.
func refreshExpiredOAuthToken(c *CmdConfig) error {
	if accessTokenOverridden() {
		return nil
	}

	authContext := currentAuthContext()
	state := loadOAuthTokenState(authContext)
	if !state.needsRefresh(time.Now()) {
		return nil
	}

	endpoint := state.TokenEndpoint
	if endpoint == "" {
		endpoint = oauth.ServerMetadataFor(state.Issuer).TokenEndpoint
	}

	httpClient := &http.Client{Timeout: oauthHTTPTimeout}
	token, err := oauth.RefreshToken(context.Background(), httpClient, endpoint, state.ClientID, state.RefreshToken)
	if err != nil {
		return fmt.Errorf("Your DigitalOcean session has expired and could not be renewed: %s\n\nRun `doctl auth login` to sign in again.", err)
	}

	c.setContextAccessToken(token.AccessToken)
	state.RefreshToken = token.RefreshToken
	state.ExpiresAt = token.Expiry
	if token.Scope != "" {
		state.Scope = token.Scope
	}
	storeOAuthTokenState(authContext, state)

	return writeConfig()
}

// accessTokenOverridden reports whether the user supplied a token for this
// invocation, in which case it takes precedence over any stored OAuth session.
func accessTokenOverridden() bool {
	return Token != "" || os.Getenv("DIGITALOCEAN_ACCESS_TOKEN") != ""
}

// currentAuthContext returns the name of the authentication context the
// command is running against.
func currentAuthContext() string {
	authContext := strings.ToLower(Context)
	if authContext == "" {
		authContext = strings.ToLower(viper.GetString(doctl.ArgContext))
	}
	if authContext == "" {
		authContext = doctl.ArgDefaultContext
	}

	return authContext
}

func loadOAuthTokenState(authContext string) *oauthTokenState {
	values, ok := oauthTokenStates()[strings.ToLower(authContext)].(map[string]any)
	if !ok || len(values) == 0 {
		return nil
	}

	state := &oauthTokenState{
		Issuer:        configMapString(values, "issuer"),
		ClientID:      configMapString(values, "client-id"),
		TokenEndpoint: configMapString(values, "token-endpoint"),
		RefreshToken:  configMapString(values, "refresh-token"),
		Scope:         configMapString(values, "scope"),
	}
	if expiresAt := configMapString(values, "expires-at"); expiresAt != "" {
		if parsed, err := time.Parse(time.RFC3339, expiresAt); err == nil {
			state.ExpiresAt = parsed
		}
	}
	if state.RefreshToken == "" {
		return nil
	}

	return state
}

func storeOAuthTokenState(authContext string, state *oauthTokenState) {
	var expiresAt string
	if !state.ExpiresAt.IsZero() {
		expiresAt = state.ExpiresAt.UTC().Format(time.RFC3339)
	}

	states := oauthTokenStates()
	states[strings.ToLower(authContext)] = map[string]any{
		"issuer":         state.Issuer,
		"client-id":      state.ClientID,
		"token-endpoint": state.TokenEndpoint,
		"refresh-token":  state.RefreshToken,
		"scope":          state.Scope,
		"expires-at":     expiresAt,
	}

	viper.Set(oauthTokensConfigKey, states)
}

// removeOAuthTokenState drops the stored OAuth session for a context. It is
// called whenever the context's credentials are replaced or removed so a stale
// refresh token cannot overwrite them later.
func removeOAuthTokenState(authContext string) {
	states := oauthTokenStates()
	if len(states) == 0 {
		return
	}

	delete(states, strings.ToLower(authContext))
	viper.Set(oauthTokensConfigKey, states)
}

// oauthTokenStates returns a mutable copy of the stored per-context sessions,
// normalizing the values viper hands back from YAML.
func oauthTokenStates() map[string]any {
	states := map[string]any{}
	for name, value := range viper.GetStringMap(oauthTokensConfigKey) {
		switch typed := value.(type) {
		case map[string]any:
			states[name] = typed
		case map[string]string:
			converted := make(map[string]any, len(typed))
			for k, v := range typed {
				converted[k] = v
			}
			states[name] = converted
		case map[any]any:
			converted := make(map[string]any, len(typed))
			for k, v := range typed {
				converted[fmt.Sprintf("%v", k)] = v
			}
			states[name] = converted
		}
	}

	return states
}

func configMapString(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}

	return ""
}
