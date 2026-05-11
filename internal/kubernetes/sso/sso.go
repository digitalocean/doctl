package sso

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	"github.com/coreos/go-oidc"
	"github.com/google/uuid"
	"github.com/pkg/browser"
)

var (
	//go:embed auth_success.html
	authSuccessHTML []byte

	//go:embed auth_error.html
	authErrorHTML []byte

	now = time.Now
)

const (
	defaultLocalServerPort uint16 = 8080
)

// GetIDToken obtains an ID token from an OIDC provider, following the Authorization Code Flow with PKCE:
// https://auth0.com/docs/get-started/authentication-and-authorization-flow/authorization-code-flow-with-pkce
func GetIDToken(ctx context.Context, clientID, issuerURL string, opts ...LocalOIDCLoginOption) (string, time.Time, error) {
	ssoTool, err := newLocalOIDCLogin(clientID, issuerURL, opts...)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("setting up SSO login tool: %w", err)
	}
	return ssoTool.getIDToken(ctx)
}

type localOIDCLogin struct {
	port   uint16
	logger *log.Logger

	// set up on creation
	oauth2Config oauth2.Config
	state        string
	codeVerifier string
	nonce        string
	provider     *oidc.Provider

	ssoServer *ssoServer

	// only for stubbing in tests
	openURL func(url string) error
}

// LocalOIDCLoginOption is a function that can be used to configure a local OIDC login tool.
type LocalOIDCLoginOption func(*localOIDCLogin)

// WithLocalServerPort sets the port to use for the local server which handles SSO authentication flow.
func WithLocalServerPort(port uint16) func(*localOIDCLogin) {
	return func(l *localOIDCLogin) {
		l.port = port
	}
}

// WithLogger sets the logger to use for the local OIDC login tool.
func WithLogger(logger *log.Logger) func(*localOIDCLogin) {
	return func(l *localOIDCLogin) {
		l.logger = logger
	}
}

// NewLocalOIDCLogin creates a new local OIDC login tool.
func newLocalOIDCLogin(clientID, issuerURL string, opts ...LocalOIDCLoginOption) (*localOIDCLogin, error) {
	t := &localOIDCLogin{
		port:    defaultLocalServerPort,
		openURL: browser.OpenURL,
		logger:  log.Default(),
	}
	for _, opt := range opts {
		opt(t)
	}

	provider, err := oidc.NewProvider(context.Background(), issuerURL)
	if err != nil {
		return nil, fmt.Errorf("creating OIDC provider: %w", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:    clientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: t.redirectURL(),
		Scopes:      []string{oidc.ScopeOpenID, "email", "team_role"},
	}

	state := uuid.New().String()
	codeVerifier := oauth2.GenerateVerifier()
	codeChallenge := oauth2.S256ChallengeOption(codeVerifier)
	nonce := uuid.New().String()

	ssoServer := &ssoServer{
		authCodeURL: oauth2Config.AuthCodeURL(state, codeChallenge, oidc.Nonce(nonce)),
		logger:      t.logger,
		authCodeCh:  make(chan authCodeResponse),
		errorCh:     make(chan error),
	}

	t.oauth2Config = oauth2Config
	t.state = state
	t.codeVerifier = codeVerifier
	t.nonce = nonce
	t.ssoServer = ssoServer
	t.provider = provider

	return t, nil
}

func (t *localOIDCLogin) redirectURL() string {
	return fmt.Sprintf("http://localhost:%d/callback", t.port)
}

func (t *localOIDCLogin) loginURL() string {
	return fmt.Sprintf("http://localhost:%d/login", t.port)
}

func (t *localOIDCLogin) getAuthCode(ctx context.Context) (string, error) {
	var group errgroup.Group

	server := &http.Server{
		Handler: t.ssoServer,
		Addr:    fmt.Sprintf("127.0.0.1:%d", t.port),
	}
	defer server.Close()

	group.Go(func() error {
		t.logger.Printf("Starting local server for OIDC authentication on port %d\n", t.port)
		if err := server.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				return fmt.Errorf("web server shutdown unexpectedly: %w", err)
			}
		}

		return nil
	})

	var authCode authCodeResponse
	group.Go(func() error {
		defer func() {
			ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			defer cancel()
			t.logger.Println("Shutting down local OIDC authentication server")
			_ = server.Shutdown(ctx)
		}()

		select {
		case code := <-t.ssoServer.authCodeCh:
			authCode = code
			return nil
		case err := <-t.ssoServer.errorCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	t.logger.Printf("Opening login URL in the default browser. If it didn't open automatically, paste this URL into your preferred browser:\n\t%s\n", t.loginURL())
	if err := t.openURL(t.loginURL()); err != nil {
		return "", fmt.Errorf("opening OIDC login URL in browser: %v", err)
	}

	if err := group.Wait(); err != nil {
		return "", err
	}

	if authCode.code == "" {
		return "", errors.New("no authorization code received")
	}

	if authCode.state != t.state {
		return "", errors.New("authorzation flow state mismatch")
	}

	return authCode.code, nil
}

func (t *localOIDCLogin) getIDToken(ctx context.Context) (string, time.Time, error) {
	code, err := t.getAuthCode(ctx)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("getting authorization code: %w", err)
	}

	t.logger.Println("Received an authorization code, exchanging for ID token")
	token, err := t.oauth2Config.Exchange(ctx, code, oauth2.VerifierOption(t.codeVerifier))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("exchanging authorization code for ID token: %w", err)
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		return "", time.Time{}, errors.New("no ID token found")
	}

	verifier := t.provider.Verifier(&oidc.Config{ClientID: t.oauth2Config.ClientID, Now: now})
	verifiedIDToken, err := verifier.Verify(ctx, idToken)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("could not verify ID token: %w", err)
	}
	if t.nonce != verifiedIDToken.Nonce {
		return "", time.Time{}, fmt.Errorf("nonce did not match (wants %s but got %s)", t.nonce, verifiedIDToken.Nonce)
	}

	return idToken, token.Expiry, nil
}

type authCodeResponse struct {
	code  string
	state string
}

type ssoServer struct {
	authCodeURL string
	logger      *log.Logger
	authCodeCh  chan authCodeResponse
	errorCh     chan error
}

func (s *ssoServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	switch {
	case r.Method == "GET" && r.URL.Path == "/callback" && q.Get("error") != "":
		s.handleError(w, r)
	case r.Method == "GET" && r.URL.Path == "/callback" && q.Get("code") != "":
		s.handleAuthorizationCode(w, r)
	case r.Method == "GET" && r.URL.Path == "/login":
		s.handleLogin(w, r)
	case r.Method == "GET" && r.URL.Path == "/error":
		s.handleError(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *ssoServer) handleError(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	code := q.Get("error")
	if code == "" {
		code = "unknown_error"
	}
	desc := q.Get("error_description")
	if desc != "" {
		s.errorCh <- fmt.Errorf("%s: %s", code, desc)
	} else {
		s.errorCh <- errors.New(code)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write(authErrorHTML)
}

func (s *ssoServer) handleAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	s.authCodeCh <- authCodeResponse{
		code:  q.Get("code"),
		state: q.Get("state"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(authSuccessHTML)
}

func (s *ssoServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	s.logger.Println("Redirecting to IDP login URL")
	http.Redirect(w, r, s.authCodeURL, http.StatusFound)
}
