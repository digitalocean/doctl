package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	jose "gopkg.in/go-jose/go-jose.v2"

	"github.com/stretchr/testify/require"
)

type fakeIDP struct {
	issuerURL string
	clientID  string
	authCode  string

	// RSA key used to sign id_token and publish JWKS.
	privKey *rsa.PrivateKey

	mu        sync.Mutex
	lastNonce string

	// inject custom handlers to simulate errors
	authorizeHandler func(w http.ResponseWriter, r *http.Request)
}

func (p *fakeIDP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/.well-known/openid-configuration" && r.Method == http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                p.issuerURL,
			"authorization_endpoint":                p.issuerURL + "/oauth/authorize",
			"token_endpoint":                        p.issuerURL + "/oauth/token",
			"jwks_uri":                              p.issuerURL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	case r.URL.Path == "/jwks" && r.Method == http.MethodGet:
		pub := jose.JSONWebKey{Key: &p.privKey.PublicKey, KeyID: "test-kid", Algorithm: string(jose.RS256), Use: "sig"}
		set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{pub}}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(set); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case r.URL.Path == "/oauth/authorize" && r.Method == http.MethodGet:
		if p.authorizeHandler != nil {
			p.authorizeHandler(w, r)
			return
		}
		p.mu.Lock()
		p.lastNonce = r.URL.Query().Get("nonce")
		p.mu.Unlock()
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
		p.mu.Lock()
		nonce := p.lastNonce
		p.mu.Unlock()
		claims, err := json.Marshal(map[string]any{
			"iss":   p.issuerURL,
			"sub":   "test-user",
			"aud":   p.clientID,
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
			"nonce": nonce,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		signer, err := jose.NewSigner(
			jose.SigningKey{Algorithm: jose.RS256, Key: p.privKey},
			(&jose.SignerOptions{}).WithHeader(jose.HeaderKey("kid"), "test-kid"),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		object, err := signer.Sign(claims)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		idTokenStr, err := object.CompactSerialize()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
				"access_token": "fake-access-token",
				"token_type": "Bearer",
				"expires_in": 3600,
				"id_token": %q
			}`, idTokenStr)
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
	)

	tests := []struct {
		name             string
		authorizeHandler func(http.ResponseWriter, *http.Request)
		wantErr          bool
		errSubstring     string
	}{
		{
			name:             "success",
			authorizeHandler: nil,
			wantErr:          false,
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
			privKey, err := rsa.GenerateKey(rand.Reader, 2048)
			require.NoError(t, err)
			idp := &fakeIDP{
				clientID:         clientID,
				authCode:         authCode,
				privKey:          privKey,
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
				require.NotEmpty(t, gotToken)
				parts := strings.Split(gotToken, ".")
				require.Len(t, parts, 3, "compact JWS should have three segments")
				rawPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
				require.NoError(t, err)
				var claims struct {
					Iss   string `json:"iss"`
					Aud   string `json:"aud"`
					Nonce string `json:"nonce"`
				}
				require.NoError(t, json.Unmarshal(rawPayload, &claims))
				require.Equal(t, idp.issuerURL, claims.Iss)
				require.Equal(t, clientID, claims.Aud)
				require.Equal(t, login.nonce, claims.Nonce)
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
