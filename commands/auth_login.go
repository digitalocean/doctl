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
	// oauthClientConfigKey holds the dynamic client registration doctl shares
	// across every authentication context on this machine.
	oauthClientConfigKey = "oauth-client"
	// oauthTokensConfigKey holds the per-context OAuth session state used to
	// refresh access tokens.
	oauthTokensConfigKey = "oauth-tokens"

	// oauthClientName and oauthClientURI identify doctl on the authorization
	// screen the user is shown.
	oauthClientName = "doctl"
	oauthClientURI  = "https://github.com/digitalocean/doctl"

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

// oauthClientState is the dynamic client registration persisted in the config
// file. doctl registers itself once per authorization server and reuses the
// resulting public client for every login.
type oauthClientState struct {
	Issuer                  string
	ClientID                string
	RedirectURIs            []string
	RegisteredAt            time.Time
	RegistrationAccessToken string
	RegistrationClientURI   string
}

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

	issuer, err := c.Doit.GetString(c.NS, doctl.ArgOAuthServer)
	if err != nil {
		return err
	}
	scopes, err := c.Doit.GetString(c.NS, doctl.ArgOAuthScopes)
	if err != nil {
		return err
	}
	port, err := c.Doit.GetInt(c.NS, doctl.ArgOAuthCallbackPort)
	if err != nil {
		return err
	}
	noBrowser, err := oauthNoBrowser(c)
	if err != nil {
		return err
	}
	// writeConfig persists every viper setting, so a single --no-browser would
	// otherwise stay true and keep the browser closed on later logins.
	viper.Set(c.NS+"."+doctl.ArgOAuthNoBrowser, false)
	timeout, err := c.Doit.GetDuration(c.NS, doctl.ArgOAuthTimeout)
	if err != nil {
		return err
	}

	ctx := context.Background()
	httpClient := &http.Client{Timeout: oauthHTTPTimeout}
	metadata := oauth.ResolveServerMetadata(ctx, httpClient, issuer)

	client, registered, err := ensureOAuthClient(ctx, c, httpClient, metadata)
	if err != nil {
		return err
	}

	opts := oauth.LoginOptions{
		Metadata:           metadata,
		ClientID:           client.ClientID,
		Scopes:             strings.Fields(scopes),
		Port:               port,
		Timeout:            timeout,
		HTTPClient:         httpClient,
		NoBrowser:          noBrowser,
		OnAuthorizationURL: authorizationURLPrinter(c, noBrowser),
	}

	token, err := oauthLogin(ctx, opts)
	if errors.Is(err, oauth.ErrInvalidClient) && !registered {
		// The stored registration is no longer valid, which happens when it
		// was deleted server side. Register again and make one more attempt.
		template.Render(c.Out, `{{nl}}{{warning "The saved OAuth application is no longer valid."}}{{nl}}`, nil)

		client, err = registerOAuthClient(ctx, c, httpClient, metadata)
		if err != nil {
			return err
		}

		opts.ClientID = client.ClientID
		token, err = oauthLogin(ctx, opts)
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
			ClientID:      client.ClientID,
			TokenEndpoint: metadata.TokenEndpoint,
			RefreshToken:  token.RefreshToken,
			Scope:         token.Scope,
			ExpiresAt:     token.Expiry,
		})
	}

	if err := writeConfig(); err != nil {
		return err
	}

	displayOAuthLoginSummary(c, authContext, token)

	return nil
}

// ensureOAuthClient returns the dynamic client registration for the
// authorization server, registering doctl if this machine does not have one
// yet. The second return value reports whether a new client was registered.
func ensureOAuthClient(ctx context.Context, c *CmdConfig, httpClient *http.Client, metadata *oauth.ServerMetadata) (*oauthClientState, bool, error) {
	if existing := loadOAuthClientState(); existing != nil && existing.Issuer == metadata.Issuer {
		return existing, false, nil
	}

	client, err := registerOAuthClient(ctx, c, httpClient, metadata)
	if err != nil {
		return nil, false, err
	}

	return client, true, nil
}

func registerOAuthClient(ctx context.Context, c *CmdConfig, httpClient *http.Client, metadata *oauth.ServerMetadata) (*oauthClientState, error) {
	template.Render(c.Out, `Registering doctl with {{highlight .}}... `, metadata.Issuer)

	registration, err := oauth.RegisterClient(
		ctx, httpClient, metadata.RegistrationEndpoint,
		oauth.RegistrationRedirectURIs(), oauthClientName, oauthClientURI,
	)
	if err != nil {
		template.Render(c.Out, `{{error crossmark}}{{nl}}{{nl}}`, nil)
		return nil, err
	}

	template.Render(c.Out, `{{success checkmark}}{{nl}}`, nil)

	client := &oauthClientState{
		Issuer:                  metadata.Issuer,
		ClientID:                registration.ClientID,
		RedirectURIs:            registration.RedirectURIs,
		RegistrationAccessToken: registration.RegistrationAccessToken,
		RegistrationClientURI:   registration.RegistrationClientURI,
	}
	if registration.ClientIDIssuedAt > 0 {
		client.RegisteredAt = time.Unix(registration.ClientIDIssuedAt, 0).UTC()
	}
	storeOAuthClientState(client)

	// Persist the registration immediately so an interrupted login does not
	// leave an orphaned application behind on the next attempt.
	if err := writeConfig(); err != nil {
		return nil, err
	}

	return client, nil
}

// oauthNoBrowser reports whether this invocation passed --no-browser.
// A value left in the config file from an earlier login is ignored.
func oauthNoBrowser(c *CmdConfig) (bool, error) {
	if !c.Doit.IsSet(doctl.ArgOAuthNoBrowser) {
		return false, nil
	}
	return c.Doit.GetBool(c.NS, doctl.ArgOAuthNoBrowser)
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
		endpoint = oauth.ResolveServerMetadata(context.Background(), nil, state.Issuer).TokenEndpoint
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

func loadOAuthClientState() *oauthClientState {
	values := viper.GetStringMap(oauthClientConfigKey)
	if len(values) == 0 {
		return nil
	}

	client := &oauthClientState{
		Issuer:                  configMapString(values, "issuer"),
		ClientID:                configMapString(values, "client-id"),
		RedirectURIs:            configMapStringSlice(values, "redirect-uris"),
		RegistrationAccessToken: configMapString(values, "registration-access-token"),
		RegistrationClientURI:   configMapString(values, "registration-client-uri"),
	}
	if registeredAt := configMapString(values, "registered-at"); registeredAt != "" {
		if parsed, err := time.Parse(time.RFC3339, registeredAt); err == nil {
			client.RegisteredAt = parsed
		}
	}
	if client.ClientID == "" || client.Issuer == "" {
		return nil
	}

	return client
}

func storeOAuthClientState(client *oauthClientState) {
	var registeredAt string
	if !client.RegisteredAt.IsZero() {
		registeredAt = client.RegisteredAt.UTC().Format(time.RFC3339)
	}

	viper.Set(oauthClientConfigKey, map[string]any{
		"issuer":                    client.Issuer,
		"client-id":                 client.ClientID,
		"redirect-uris":             client.RedirectURIs,
		"registered-at":             registeredAt,
		"registration-access-token": client.RegistrationAccessToken,
		"registration-client-uri":   client.RegistrationClientURI,
	})
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

// configMapStringSlice reads a list of strings, which viper hands back as
// []any when it comes from the YAML file and as []string when it was set in
// this process.
func configMapStringSlice(values map[string]any, key string) []string {
	switch typed := values[key].(type) {
	case []string:
		return typed
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
		return items
	default:
		return nil
	}
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
