package integration

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

// ansiEscape matches the styling the CLI applies to the authorization URL when
// it renders it, so the test can recover the bare URL.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// authorizationServer is a stand-in for DigitalOcean's authorization server.
// It records what doctl sent to the authorize endpoint so the token endpoint
// can verify the PKCE verifier against the challenge from the same login.
type authorizationServer struct {
	*httptest.Server

	mu            sync.Mutex
	codeChallenge string
	challengeKind string
	authClientID  string
	authScope     string
	redirectURI   string
	tokenForm     url.Values
}

func newAuthorizationServer(t *testing.T) *authorizationServer {
	as := &authorizationServer{}

	mux := http.NewServeMux()

	// The browser lands here. A real authorization server would render a
	// consent screen; this one approves immediately and redirects back to the
	// loopback address doctl is listening on.
	mux.HandleFunc("/v1/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		as.mu.Lock()
		as.codeChallenge = query.Get("code_challenge")
		as.challengeKind = query.Get("code_challenge_method")
		as.authClientID = query.Get("client_id")
		as.authScope = query.Get("scope")
		as.redirectURI = query.Get("redirect_uri")
		as.mu.Unlock()

		redirect, err := url.Parse(query.Get("redirect_uri"))
		if err != nil {
			http.Error(w, "bad redirect_uri", http.StatusBadRequest)
			return
		}

		params := redirect.Query()
		params.Set("code", "the-authorization-code")
		params.Set("state", query.Get("state"))
		redirect.RawQuery = params.Encode()

		http.Redirect(w, r, redirect.String(), http.StatusFound)
	})

	mux.HandleFunc("/v1/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}

		as.mu.Lock()
		challenge := as.codeChallenge
		as.tokenForm = r.PostForm
		as.mu.Unlock()

		// Proving the verifier against the challenge is the point of PKCE, so
		// reject the exchange rather than hand out a token when it fails.
		sum := sha256.Sum256([]byte(r.FormValue("code_verifier")))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error":             "invalid_grant",
				"error_description": "the code verifier did not match the code challenge",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "doo_v1_from_the_code_exchange",
			"refresh_token": "dor_v1_from_the_code_exchange",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "read write",
			"info": map[string]any{
				"email":     "sammy@example.com",
				"team_name": "My Team",
			},
		})
	})

	as.Server = httptest.NewServer(mux)
	t.Cleanup(as.Close)

	return as
}

func (a *authorizationServer) snapshot() (challenge, kind, clientID, scope, redirectURI string, tokenForm url.Values) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.codeChallenge, a.challengeKind, a.authClientID, a.authScope, a.redirectURI, a.tokenForm
}

var _ = suite("auth/login", func(t *testing.T, when spec.G, it spec.S) {
	var expect *require.Assertions

	it.Before(func() {
		expect = require.New(t)
	})

	when("the browser completes the authorization", func() {
		it("exchanges the code and saves the session", func() {
			as := newAuthorizationServer(t)

			tmpDir := t.TempDir()
			testConfig := filepath.Join(tmpDir, "test-config.yml")

			cmd := exec.Command(builtBinaryPath,
				"--config", testConfig,
				"auth", "login",
				"--oauth-server", as.URL,
				"--client-id", "doctl-e2e-client",
				"--scope", "read write",
				"--no-browser",
				"--timeout", "30s",
			)

			output, authURL := startLoginAndCaptureURL(t, expect, cmd)

			// Stand in for the browser: follow the authorization URL, which
			// redirects back into doctl's loopback callback server.
			visitAuthorizationURL(t, expect, authURL)

			expect.NoError(cmd.Wait(), output.String())

			combined := output.String()
			expect.Contains(combined, "Signed in as")
			expect.Contains(combined, "sammy@example.com")
			expect.Contains(combined, "My Team")

			challenge, kind, clientID, scope, redirectURI, tokenForm := as.snapshot()

			expect.Equal("doctl-e2e-client", clientID, "doctl must authorize as the configured application")
			expect.Equal("read write", scope)
			expect.Equal("S256", kind, "doctl must use the S256 PKCE challenge method")
			expect.NotEmpty(challenge)
			expect.Regexp(`^http://127\.0\.0\.1:\d+/callback$`, redirectURI)

			expect.Equal("authorization_code", tokenForm.Get("grant_type"))
			expect.Equal("the-authorization-code", tokenForm.Get("code"))
			expect.Equal("doctl-e2e-client", tokenForm.Get("client_id"))
			expect.Equal(redirectURI, tokenForm.Get("redirect_uri"))
			expect.Empty(tokenForm.Get("client_secret"), "doctl is a public client and must not send a secret")

			config := readConfigFile(t, expect, testConfig)
			expect.Contains(config, "access-token: doo_v1_from_the_code_exchange")
			expect.Contains(config, "refresh-token: dor_v1_from_the_code_exchange")
			expect.NotContains(config, "the-authorization-code", "the authorization code is single use and must not be stored")
		})
	})

	when("the authorization server reports an error", func() {
		it("exits non-zero without saving a token", func() {
			denying := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()

				redirect, err := url.Parse(query.Get("redirect_uri"))
				if err != nil {
					http.Error(w, "bad redirect_uri", http.StatusBadRequest)
					return
				}

				params := redirect.Query()
				params.Set("error", "access_denied")
				params.Set("error_description", "the user declined the request")
				params.Set("state", query.Get("state"))
				redirect.RawQuery = params.Encode()

				http.Redirect(w, r, redirect.String(), http.StatusFound)
			}))
			defer denying.Close()

			tmpDir := t.TempDir()
			testConfig := filepath.Join(tmpDir, "test-config.yml")

			cmd := exec.Command(builtBinaryPath,
				"--config", testConfig,
				"auth", "login",
				"--oauth-server", denying.URL,
				"--client-id", "doctl-e2e-client",
				"--no-browser",
				"--timeout", "30s",
			)

			output, authURL := startLoginAndCaptureURL(t, expect, cmd)
			visitAuthorizationURL(t, expect, authURL)

			err := cmd.Wait()
			expect.Error(err, "a declined authorization must fail the command")

			combined := output.String()
			expect.Contains(combined, "access_denied")
			expect.Contains(combined, "the user declined the request")

			if contents, err := os.ReadFile(testConfig); err == nil {
				expect.NotContains(string(contents), "access-token: doo_v1")
			}
		})
	})
})

var _ = suite("auth/login/refresh", func(t *testing.T, when spec.G, it spec.S) {
	var expect *require.Assertions

	it.Before(func() {
		expect = require.New(t)
	})

	when("the stored access token has expired", func() {
		it("renews it before running the command and saves the rotated tokens", func() {
			var (
				mu          sync.Mutex
				refreshForm url.Values
				accountAuth string
			)

			mux := http.NewServeMux()

			mux.HandleFunc("/v1/oauth/token", func(w http.ResponseWriter, r *http.Request) {
				expect.NoError(r.ParseForm())

				mu.Lock()
				refreshForm = r.PostForm
				mu.Unlock()

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"access_token":  "doo_v1_renewed",
					"refresh_token": "dor_v1_rotated",
					"token_type":    "Bearer",
					"expires_in":    3600,
					"scope":         "read write",
				})
			})

			mux.HandleFunc("/v2/account", func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				accountAuth = r.Header.Get("Authorization")
				mu.Unlock()

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"account": map[string]any{
						"email":          "sammy@example.com",
						"uuid":           "a-uuid",
						"status":         "active",
						"droplet_limit":  25,
						"email_verified": true,
					},
				})
			})

			server := httptest.NewServer(mux)
			defer server.Close()

			tmpDir := t.TempDir()
			testConfig := filepath.Join(tmpDir, "test-config.yml")

			expired := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
			expect.NoError(os.WriteFile(testConfig, []byte(strings.Join([]string{
				"access-token: doo_v1_expired",
				"context: default",
				"oauth-tokens:",
				"  default:",
				"    issuer: " + server.URL,
				"    client-id: doctl-e2e-client",
				"    token-endpoint: " + server.URL + "/v1/oauth/token",
				"    refresh-token: dor_v1_stored",
				"    scope: read write",
				"    expires-at: " + expired,
				"",
			}, "\n")), 0600))

			cmd := exec.Command(builtBinaryPath,
				"-u", server.URL,
				"--config", testConfig,
				"account", "get",
			)
			cmd.Env = environmentWithoutDigitalOceanAuth()

			output, err := cmd.CombinedOutput()
			expect.NoError(err, string(output))
			expect.Contains(string(output), "sammy@example.com")

			mu.Lock()
			form, auth := refreshForm, accountAuth
			mu.Unlock()

			expect.NotNil(form, "doctl must renew the token before calling the API")
			expect.Equal("refresh_token", form.Get("grant_type"))
			expect.Equal("dor_v1_stored", form.Get("refresh_token"))
			expect.Equal("doctl-e2e-client", form.Get("client_id"))

			expect.Equal("Bearer doo_v1_renewed", auth, "the API call must use the renewed token")

			config := readConfigFile(t, expect, testConfig)
			expect.Contains(config, "access-token: doo_v1_renewed")
			expect.Contains(config, "refresh-token: dor_v1_rotated", "a rotated refresh token must replace the stored one")
			expect.NotContains(config, "dor_v1_stored", "the consumed refresh token must not be left behind")
		})
	})

	when("the stored access token is still valid", func() {
		it("uses it without contacting the token endpoint", func() {
			var (
				mu        sync.Mutex
				refreshed bool
				auth      string
			)

			mux := http.NewServeMux()

			mux.HandleFunc("/v1/oauth/token", func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				refreshed = true
				mu.Unlock()

				w.WriteHeader(http.StatusInternalServerError)
			})

			mux.HandleFunc("/v2/account", func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				auth = r.Header.Get("Authorization")
				mu.Unlock()

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{
					"account": map[string]any{
						"email":  "sammy@example.com",
						"uuid":   "a-uuid",
						"status": "active",
					},
				})
			})

			server := httptest.NewServer(mux)
			defer server.Close()

			tmpDir := t.TempDir()
			testConfig := filepath.Join(tmpDir, "test-config.yml")

			valid := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
			expect.NoError(os.WriteFile(testConfig, []byte(strings.Join([]string{
				"access-token: doo_v1_still_good",
				"context: default",
				"oauth-tokens:",
				"  default:",
				"    issuer: " + server.URL,
				"    client-id: doctl-e2e-client",
				"    token-endpoint: " + server.URL + "/v1/oauth/token",
				"    refresh-token: dor_v1_stored",
				"    scope: read write",
				"    expires-at: " + valid,
				"",
			}, "\n")), 0600))

			cmd := exec.Command(builtBinaryPath,
				"-u", server.URL,
				"--config", testConfig,
				"account", "get",
			)
			cmd.Env = environmentWithoutDigitalOceanAuth()

			output, err := cmd.CombinedOutput()
			expect.NoError(err, string(output))

			mu.Lock()
			didRefresh, usedAuth := refreshed, auth
			mu.Unlock()

			expect.False(didRefresh, "an unexpired token must not be renewed")
			expect.Equal("Bearer doo_v1_still_good", usedAuth)
		})
	})
})

// environmentWithoutDigitalOceanAuth drops the ambient credentials a developer
// may have exported, which would otherwise take precedence over the config
// file the test wrote.
func environmentWithoutDigitalOceanAuth() []string {
	var env []string
	for _, entry := range os.Environ() {
		switch {
		case strings.HasPrefix(entry, "DIGITALOCEAN_ACCESS_TOKEN="),
			strings.HasPrefix(entry, "DIGITALOCEAN_CONTEXT="):
			continue
		}
		env = append(env, entry)
	}

	return env
}

// startLoginAndCaptureURL runs the login command and returns the captured
// output along with the authorization URL doctl printed for the browser. The
// command is still running when this returns: it is waiting on its callback.
func startLoginAndCaptureURL(t *testing.T, expect *require.Assertions, cmd *exec.Cmd) (*safeBuffer, string) {
	t.Helper()

	stdout, err := cmd.StdoutPipe()
	expect.NoError(err)
	cmd.Stderr = cmd.Stdout

	output := &safeBuffer{}
	urls := make(chan string, 1)

	expect.NoError(cmd.Start())

	go func() {
		defer close(urls)

		scanner := bufio.NewScanner(io.TeeReader(stdout, output))
		for scanner.Scan() {
			line := strings.TrimSpace(ansiEscape.ReplaceAllString(scanner.Text(), ""))
			if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
				select {
				case urls <- line:
				default:
				}
			}
		}

		// The pipe closes when the command exits, which is the normal end of
		// this loop rather than a failure worth reporting.
		_ = scanner.Err()
	}()

	select {
	case authURL, ok := <-urls:
		expect.True(ok, "doctl did not print an authorization URL: %s", output.String())
		return output, authURL
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("timed out waiting for the authorization URL: %s", output.String())
		return output, ""
	}
}

// visitAuthorizationURL acts as the user's browser. The authorization server
// redirects to doctl's loopback callback, which the client follows.
func visitAuthorizationURL(t *testing.T, expect *require.Assertions, authURL string) {
	t.Helper()

	resp, err := http.Get(authURL)
	expect.NoError(err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	expect.NoError(err)

	// Whatever the outcome, the callback page is what the user is left looking
	// at, so it must not be a bare error from the Go http package.
	expect.Contains(string(body), "<!DOCTYPE html>")
}

func readConfigFile(t *testing.T, expect *require.Assertions, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	expect.NoError(err)

	return string(contents)
}

// safeBuffer collects command output that is written from the scanning
// goroutine and read from the test goroutine.
type safeBuffer struct {
	mu      sync.Mutex
	builder strings.Builder
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.builder.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.builder.String()
}
