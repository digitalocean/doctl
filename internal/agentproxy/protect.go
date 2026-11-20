package agentproxy

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

// OpenCodeProxyBasicUser is the Basic Auth user opencode attach sends
// (`opencode attach --password`). Same username as `opencode serve`.
const OpenCodeProxyBasicUser = "opencode"

// NewProxyPassword mints a one-time Basic Auth password for start-proxy.
// The alphabet is URL-safe base64 without padding, so the value is safe to
// paste unquoted on a shell command line.
func NewProxyPassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ProtectLoopbackHTTP gates the OpenCode HTTP facade: Basic Auth, the same
// Origin allowlist as the WebSocket proxy, and JSON-only bodies on writes
// (blocks browser form Content-Types). password must be non-empty.
func ProtectLoopbackHTTP(next http.Handler, password string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Origin before auth: browsers may attach cached Basic Auth on a
		// form POST to 127.0.0.1, so Origin/CT checks must still run.
		if !AllowedProxyOrigin(r.Header.Get("Origin")) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		if password == "" || !validBasicAuth(r, OpenCodeProxyBasicUser, password) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if isStateChanging(r.Method) && !allowedWriteContentType(r) {
			http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validBasicAuth(r *http.Request, wantUser, wantPass string) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	// Both compares always run (no && short-circuit around ConstantTimeCompare).
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(wantUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(wantPass)) == 1
	return userOK && passOK
}

func isStateChanging(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// allowedWriteContentType accepts application/json, or an omitted
// Content-Type only when there is no body. Form types and a body with no
// Content-Type are rejected (CSRF simple requests use text/plain / forms).
func allowedWriteContentType(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return r.ContentLength == 0
	}
	media, _, _ := strings.Cut(ct, ";")
	return strings.EqualFold(strings.TrimSpace(media), "application/json")
}
