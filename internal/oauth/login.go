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
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/browser"
)

// callbackPath is the path the local redirect server listens on. It is part of
// the registered redirect URIs, so it must stay stable across releases.
const callbackPath = "/callback"

// DefaultLoginTimeout bounds how long the login flow waits for the user to
// finish authorizing in their browser.
const DefaultLoginTimeout = 5 * time.Minute

// RedirectURIs are the redirect URIs the doctl OAuth application registers
// with DigitalOcean. Neither carries a port: the authorization server matches
// loopback redirects on any port (RFC 8252 section 7.3), which lets doctl bind
// an ephemeral port at login time.
func RedirectURIs() []string {
	return []string{
		"http://127.0.0.1" + callbackPath,
		"http://localhost" + callbackPath,
	}
}

// AuthorizationError is an error the authorization server reported by
// redirecting back to doctl with an error parameter.
type AuthorizationError struct {
	Code        string
	Description string
}

func (e *AuthorizationError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Description)
	}
	return e.Code
}

// Unwrap reports a rejected client as ErrInvalidClient so callers can explain
// that the doctl application, rather than the user, was turned away.
func (e *AuthorizationError) Unwrap() error {
	if isInvalidClientCode(e.Code) {
		return ErrInvalidClient
	}
	return nil
}

// LoginOptions configures a single authorization code login.
type LoginOptions struct {
	// Metadata describes the authorization server endpoints to use.
	Metadata *ServerMetadata
	// ClientID identifies the dynamically registered public client.
	ClientID string
	// Scopes are the OAuth scopes to request. An empty list lets the
	// authorization server apply its default scope.
	Scopes []string
	// Port is the local port to listen on for the redirect. Zero picks an
	// unused port, which is the recommended behavior for native apps.
	Port int
	// Timeout bounds how long to wait for the browser redirect. Defaults to
	// DefaultLoginTimeout.
	Timeout time.Duration
	// HTTPClient is used for the token exchange.
	HTTPClient *http.Client
	// NoBrowser prints the authorization URL instead of opening a browser.
	NoBrowser bool
	// OnAuthorizationURL, when set, is called with the authorization URL once
	// the local redirect server is listening.
	OnAuthorizationURL func(authURL string)

	// openURL is a test hook for opening the browser.
	openURL func(authURL string) error
}

// Login runs the OAuth 2.1 authorization code flow with PKCE against the
// configured authorization server and returns the resulting token grant.
// It is a variable so tests can run the command without opening a browser.
var Login = login

func login(ctx context.Context, opts LoginOptions) (*Token, error) {
	if opts.Metadata == nil {
		return nil, errors.New("authorization server metadata is required")
	}
	if opts.ClientID == "" {
		return nil, errors.New("a client ID is required")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultLoginTimeout
	}
	if opts.openURL == nil {
		opts.openURL = browser.OpenURL
	}

	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, err
	}
	state, err := randomURLSafeString(32)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", opts.Port))
	if err != nil {
		return nil, fmt.Errorf("listening for the authorization redirect: %w", err)
	}
	defer listener.Close()

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d%s", listener.Addr().(*net.TCPAddr).Port, callbackPath)
	authURL := authorizationURL(opts, redirectURI, state, verifier)

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	results := make(chan callbackResult, 1)
	server := &http.Server{Handler: &callbackHandler{state: state, results: results}}
	defer server.Close()

	go func() { _ = server.Serve(listener) }()

	if opts.OnAuthorizationURL != nil {
		opts.OnAuthorizationURL(authURL)
	}
	if !opts.NoBrowser {
		if err := opts.openURL(authURL); err != nil {
			return nil, fmt.Errorf("opening the authorization URL in a browser: %w", err)
		}
	}

	var result callbackResult
	select {
	case result = <-results:
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("timed out after %s waiting for browser authorization", opts.Timeout)
		}
		return nil, ctx.Err()
	}

	if result.err != nil {
		return nil, result.err
	}

	return ExchangeCode(ctx, opts.HTTPClient, opts.Metadata.TokenEndpoint, opts.ClientID, result.code, verifier, redirectURI)
}

func authorizationURL(opts LoginOptions, redirectURI, state, verifier string) string {
	query := url.Values{
		"client_id":             {opts.ClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {responseTypeCode},
		"state":                 {state},
		"code_challenge":        {codeChallengeS256(verifier)},
		"code_challenge_method": {CodeChallengeMethodS256},
	}
	if scope := strings.TrimSpace(strings.Join(opts.Scopes, " ")); scope != "" {
		query.Set("scope", scope)
	}

	separator := "?"
	if strings.Contains(opts.Metadata.AuthorizationEndpoint, "?") {
		separator = "&"
	}

	return opts.Metadata.AuthorizationEndpoint + separator + query.Encode()
}

type callbackResult struct {
	code string
	err  error
}

type callbackHandler struct {
	state   string
	results chan<- callbackResult
}

func (h *callbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != callbackPath {
		http.NotFound(w, r)
		return
	}

	query := r.URL.Query()
	result, content, status := h.resultFor(query)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	// Execute the html/template onto the response so text from the
	// authorization server is escaped for the HTML context it appears in.
	// Rendering to a string and writing those bytes would hide that escaping
	// from the response path.
	if err := pageTemplate.Execute(w, content); err != nil {
		_, _ = w.Write([]byte("Return to your terminal to continue."))
	}

	select {
	case h.results <- result:
	default:
	}
}

func (h *callbackHandler) resultFor(query url.Values) (callbackResult, pageContent, int) {
	// The state check comes first: a mismatched state means the redirect did
	// not originate from the authorization request we started.
	if subtle.ConstantTimeCompare([]byte(query.Get("state")), []byte(h.state)) != 1 {
		err := errors.New("the authorization response state did not match the request; the login attempt may have been forged")
		return callbackResult{err: err}, errorContent(err.Error()), http.StatusBadRequest
	}

	if code := query.Get("error"); code != "" {
		err := &AuthorizationError{Code: code, Description: query.Get("error_description")}
		return callbackResult{err: err}, errorContent(err.Error()), http.StatusBadRequest
	}

	code := query.Get("code")
	if code == "" {
		err := errors.New("the authorization response did not include an authorization code")
		return callbackResult{err: err}, errorContent(err.Error()), http.StatusBadRequest
	}

	return callbackResult{code: code}, successContent, http.StatusOK
}

// generateCodeVerifier returns a PKCE code verifier: 32 random bytes encoded
// as 43 unreserved characters, within the 43-128 character range RFC 7636
// section 4.1 allows.
func generateCodeVerifier() (string, error) {
	return randomURLSafeString(32)
}

func codeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomURLSafeString(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating random data: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
