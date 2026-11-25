package agentproxy

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProxyPassword(t *testing.T) {
	a, err := NewProxyPassword()
	require.NoError(t, err)
	b, err := NewProxyPassword()
	require.NoError(t, err)
	assert.NotEmpty(t, a)
	assert.NotEqual(t, a, b)
	_, err = base64.RawURLEncoding.DecodeString(a)
	assert.NoError(t, err)
	assert.NotContains(t, a, "+", "raw URL encoding avoids + / = for shell safety")
	assert.NotContains(t, a, "/")
	assert.NotContains(t, a, "=")
}

func TestProtectLoopbackHTTPRequiresBasicAuth(t *testing.T) {
	srv := httptest.NewServer(ProtectLoopbackHTTP(okHandler(), "s3cret"))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	require.NoError(t, err)
	req.SetBasicAuth("wrong", "s3cret")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	req, err = http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	require.NoError(t, err)
	req.SetBasicAuth(OpenCodeProxyBasicUser, "wrong-pass")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	req, err = http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	require.NoError(t, err)
	req.SetBasicAuth(OpenCodeProxyBasicUser, "s3cret")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "ok", string(body))
}

func TestProtectLoopbackHTTPRejectsForeignOrigin(t *testing.T) {
	srv := httptest.NewServer(ProtectLoopbackHTTP(okHandler(), "s3cret"))
	t.Cleanup(srv.Close)

	for _, origin := range []string{"https://attacker.example", "null", "https://example.com"} {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
		require.NoError(t, err)
		req.SetBasicAuth(OpenCodeProxyBasicUser, "s3cret")
		req.Header.Set("Origin", origin)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, "origin %q", origin)
	}

	for _, origin := range []string{"", "http://127.0.0.1:3000", "http://localhost", "http://[::1]:8080"} {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
		require.NoError(t, err)
		req.SetBasicAuth(OpenCodeProxyBasicUser, "s3cret")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode, "origin %q", origin)
	}
}

func TestProtectLoopbackHTTPRejectsFormContentTypes(t *testing.T) {
	srv := httptest.NewServer(ProtectLoopbackHTTP(okHandler(), "s3cret"))
	t.Cleanup(srv.Close)

	// Foreign origin is rejected before Content-Type is considered.
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/prompt", strings.NewReader(`{"parts":[]}`))
	require.NoError(t, err)
	req.SetBasicAuth(OpenCodeProxyBasicUser, "s3cret")
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", "https://attacker.example")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	for _, ct := range []string{
		"text/plain",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"text/plain;charset=UTF-8",
	} {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/prompt", strings.NewReader(`{"parts":[]}`))
		require.NoError(t, err)
		req.SetBasicAuth(OpenCodeProxyBasicUser, "s3cret")
		req.Header.Set("Content-Type", ct)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode, ct)
	}

	// Body with no Content-Type must not slip through (CSRF-adjacent).
	req, err = http.NewRequest(http.MethodPost, srv.URL+"/prompt", strings.NewReader(`{"parts":[]}`))
	require.NoError(t, err)
	req.SetBasicAuth(OpenCodeProxyBasicUser, "s3cret")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)

	// JSON (with charset) and empty-body POST without CT are allowed.
	req, err = http.NewRequest(http.MethodPost, srv.URL+"/prompt", strings.NewReader(`{"parts":[]}`))
	require.NoError(t, err)
	req.SetBasicAuth(OpenCodeProxyBasicUser, "s3cret")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	req, err = http.NewRequest(http.MethodPost, srv.URL+"/prompt", http.NoBody)
	require.NoError(t, err)
	req.SetBasicAuth(OpenCodeProxyBasicUser, "s3cret")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProtectLoopbackHTTPGETIgnoresContentType(t *testing.T) {
	srv := httptest.NewServer(ProtectLoopbackHTTP(okHandler(), "s3cret"))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	require.NoError(t, err)
	req.SetBasicAuth(OpenCodeProxyBasicUser, "s3cret")
	req.Header.Set("Content-Type", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProtectLoopbackHTTPEmptyPasswordRejects(t *testing.T) {
	srv := httptest.NewServer(ProtectLoopbackHTTP(okHandler(), ""))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/", nil)
	require.NoError(t, err)
	req.SetBasicAuth(OpenCodeProxyBasicUser, "")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})
}
