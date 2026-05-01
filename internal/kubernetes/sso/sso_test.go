package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeIDP struct {
	issuerURL string
	clientID  string
	authCode  string
	idToken   string

	// inject custom handlers to simulate errors
	authorizeHandler func(w http.ResponseWriter, r *http.Request)
}

func (p *fakeIDP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/.well-known/openid-configuration" && r.Method == http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 p.issuerURL,
			"authorization_endpoint": p.issuerURL + "/oauth/authorize",
			"token_endpoint":         p.issuerURL + "/oauth/token",
		})
	case r.URL.Path == "/oauth/authorize" && r.Method == http.MethodGet:
		if p.authorizeHandler != nil {
			p.authorizeHandler(w, r)
			return
		}
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")
		u, err := url.Parse(redirectURI)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		q := u.Query()
		q.Set("code", p.authCode)
		q.Set("state", state)
		u.RawQuery = q.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	case r.URL.Path == "/oauth/token" && r.Method == http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "authorization_code" {
			http.Error(w, "unexpected grant_type", http.StatusBadRequest)
			return
		}
		if r.FormValue("code") != p.authCode {
			http.Error(w, "unexpected code", http.StatusBadRequest)
			return
		}
		if r.FormValue("client_id") != p.clientID {
			http.Error(w, "unexpected client_id", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
				"access_token": "fake-access-token",
				"token_type": "Bearer",
				"expires_in": 3600,
				"id_token": %q
			}`, p.idToken)
	default:
		http.NotFound(w, r)
	}
}

func authorizeHandlerAccessDenied(w http.ResponseWriter, r *http.Request) {
	redirectURI := r.URL.Query().Get("redirect_uri")
	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	q := u.Query()
	q.Set("error", "access_denied")
	q.Set("error_description", "user declined")
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func authorizeHandlerWrongState(authCode string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		redirectURI := r.URL.Query().Get("redirect_uri")
		u, err := url.Parse(redirectURI)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		q := u.Query()
		q.Set("code", authCode)
		q.Set("state", "wrong-state")
		u.RawQuery = q.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	}
}

func pickFreePort(t *testing.T) uint16 {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	return uint16(ln.Addr().(*net.TCPAddr).Port)
}

func waitForLocalServer(ctx context.Context, port uint16) error {
	const (
		pollInterval = 25 * time.Millisecond
		dialTimeout  = 2 * time.Second
	)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout : %w", ctx.Err())
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", addr, dialTimeout)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}

func TestGetIDToken(t *testing.T) {
	const (
		clientID = "test-client-id"
		authCode = "test-auth-code-from-idp"
		idToken  = "test-id-token-jwt"
	)

	tests := []struct {
		name             string
		authorizeHandler func(http.ResponseWriter, *http.Request)
		wantErr          bool
		errSubstring     string
		wantToken        string
	}{
		{
			name:             "success",
			authorizeHandler: nil,
			wantErr:          false,
			wantToken:        idToken,
		},
		{
			name:             "callback oauth error",
			authorizeHandler: authorizeHandlerAccessDenied,
			wantErr:          true,
			errSubstring:     "access_denied",
		},
		{
			name:             "state mismatch",
			authorizeHandler: authorizeHandlerWrongState(authCode),
			wantErr:          true,
			errSubstring:     "state mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idp := &fakeIDP{
				clientID:         clientID,
				authCode:         authCode,
				idToken:          idToken,
				authorizeHandler: tt.authorizeHandler,
			}
			srv := httptest.NewServer(idp)
			defer srv.Close()
			idp.issuerURL = srv.URL

			port := pickFreePort(t)
			browserDone := make(chan error, 1)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			login, err := newLocalOIDCLogin(clientID, idp.issuerURL,
				WithLocalServerPort(port),
				WithLogger(log.New(io.Discard, "", 0)),
			)
			require.NoError(t, err)

			login.openURL = func(loginURL string) error {
				go func() {
					if err := waitForLocalServer(ctx, port); err != nil {
						browserDone <- err
						return
					}
					client := &http.Client{Timeout: 20 * time.Second}
					resp, err := client.Get(loginURL)
					if err != nil {
						browserDone <- err
						return
					}
					resp.Body.Close()
					browserDone <- nil
				}()
				return nil
			}

			gotToken, exp, err := login.getIDToken(ctx)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errSubstring)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantToken, gotToken)
				require.True(t, exp.After(time.Now()), "expected non-zero token expiry from IdP response")
			}

			select {
			case berr := <-browserDone:
				require.NoError(t, berr)
			case <-time.After(5 * time.Second):
				t.Fatal("browser simulation did not complete")
			}
		})
	}
}
