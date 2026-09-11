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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubVersioner struct {
	version string
	err     error
	calls   int
}

func (s *stubVersioner) LatestVersion() (string, error) {
	s.calls++
	return s.version, s.err
}

func TestRenderWelcomeGreetsAndReportsState(t *testing.T) {
	var buf bytes.Buffer
	env := ui.Plain(&buf, &buf)

	out := renderWelcome(env, welcome{
		version: "1.2.3",
		context: "default",
		auth:    authStateValid,
		account: "sammy@example.com",
		team:    "Sharks",
	})

	assert.Contains(t, out, "Welcome to DigitalOcean")
	assert.Contains(t, out, "1.2.3")
	assert.Contains(t, out, "sammy@example.com")
	assert.Regexp(t, `Team\s+Sharks`, out)
	assert.Regexp(t, `Context\s+default`, out)
	assert.NotContains(t, out, "Update available")
}

// The team is its own row rather than a suffix on the account, and it is
// omitted entirely when the token is not scoped to a team.
func TestRenderWelcomeShowsTeamOnlyWhenKnown(t *testing.T) {
	var buf bytes.Buffer
	env := ui.Plain(&buf, &buf)

	base := welcome{version: "1.2.3", context: "default", auth: authStateValid, account: "sammy@example.com"}

	withTeam := base
	withTeam.team = "Sharks"
	out := renderWelcome(env, withTeam)
	assert.Regexp(t, `Account\s+.*sammy@example\.com`, out)
	assert.Regexp(t, `Team\s+Sharks`, out)
	assert.NotContains(t, out, "sammy@example.com (Sharks)")

	out = renderWelcome(env, base)
	assert.NotContains(t, out, "Team")
}

// A token from outside the config file outranks the saved one, so the account
// on screen may not be the configured one. Name the source, and say nothing in
// the ordinary case.
func TestRenderWelcomeNamesAnAmbientTokenSource(t *testing.T) {
	base := welcome{version: "1.2.3", context: "default", auth: authStateValid, account: "ci-robot@example.com"}

	render := func(source tokenSource) string {
		var buf bytes.Buffer
		w := base
		w.tokenSource = source

		return renderWelcome(ui.Plain(&buf, &buf), w)
	}

	out := render(tokenSourceEnvVar)
	assert.Contains(t, out, "ci-robot@example.com (from DIGITALOCEAN_ACCESS_TOKEN)")

	out = render(tokenSourceFlag)
	assert.Contains(t, out, "ci-robot@example.com (from --access-token)")

	// The saved token is the unremarkable case and earns no annotation.
	out = render(tokenSourceConfigFile)
	assert.Contains(t, out, "ci-robot@example.com")
	assert.NotContains(t, out, "(from")
}

// Naming the source is meant to tell the user which token to fix, so it has
// to survive the states where something is wrong.
func TestRenderWelcomeNamesTheSourceOfABadToken(t *testing.T) {
	for _, state := range []authState{authStateInvalid, authStateUnverified} {
		t.Run(state.String(), func(t *testing.T) {
			var buf bytes.Buffer

			out := renderWelcome(ui.Plain(&buf, &buf), welcome{
				version:     "1.2.3",
				context:     "default",
				auth:        state,
				tokenSource: tokenSourceEnvVar,
			})

			assert.Contains(t, out, "(from DIGITALOCEAN_ACCESS_TOKEN)")
		})
	}

	// With no token at all there is no source to report.
	var buf bytes.Buffer
	out := renderWelcome(ui.Plain(&buf, &buf), welcome{version: "1.2.3", context: "default", auth: authStateNoToken})
	assert.NotContains(t, out, "(from")
}

// Each auth state has to read differently, and in particular an unreachable
// API must never be presented as a bad token.
func TestRenderWelcomeDistinguishesAuthStates(t *testing.T) {
	tests := []struct {
		name      string
		w         welcome
		expect    string
		expectCmd string
		reject    string
	}{
		{
			name:      "no token asks the user to authenticate",
			w:         welcome{auth: authStateNoToken},
			expect:    "not authenticated",
			expectCmd: "doctl auth init",
		},
		{
			name:      "valid token names the account",
			w:         welcome{auth: authStateValid, account: "sammy@example.com"},
			expect:    "sammy@example.com",
			expectCmd: "doctl compute droplet list",
			reject:    "doctl auth init",
		},
		{
			name:      "valid token without an email still reads as authenticated",
			w:         welcome{auth: authStateValid},
			expect:    "authenticated",
			expectCmd: "doctl compute droplet list",
		},
		{
			name:      "rejected token points at replacing it",
			w:         welcome{auth: authStateInvalid},
			expect:    "token rejected",
			expectCmd: "doctl auth init",
		},
		{
			name:      "unverified token does not claim the token is bad",
			w:         welcome{auth: authStateUnverified},
			expect:    "could not be verified",
			expectCmd: "doctl compute droplet list",
			reject:    "rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			env := ui.Plain(&buf, &buf)

			tt.w.version = "1.2.3"
			tt.w.context = "default"

			out := renderWelcome(env, tt.w)

			assert.Contains(t, out, tt.expect)
			assert.Contains(t, out, tt.expectCmd)
			if tt.reject != "" {
				assert.NotContains(t, out, tt.reject)
			}
		})
	}
}

func TestRenderWelcomeShowsContextName(t *testing.T) {
	var buf bytes.Buffer
	env := ui.Plain(&buf, &buf)

	out := renderWelcome(env, welcome{version: "1.2.3", context: "work"})

	assert.Regexp(t, `Context\s+work`, out)
}

func TestRenderWelcomeShowsUpdateCommand(t *testing.T) {
	tests := []struct {
		name       string
		upgradeCmd string
		expect     string
	}{
		{
			name:       "known install method names the command",
			upgradeCmd: "brew upgrade doctl",
			expect:     "brew upgrade doctl",
		},
		{
			name:   "unknown install method points at the release page",
			expect: releasesURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			env := ui.Plain(&buf, &buf)

			out := renderWelcome(env, welcome{
				version:    "1.2.3",
				context:    "default",
				auth:       authStateValid,
				latest:     "1.3.0",
				upgradeCmd: tt.upgradeCmd,
			})

			assert.Contains(t, out, "Update available: 1.3.0")
			assert.Contains(t, out, "you have 1.2.3")
			assert.Contains(t, out, tt.expect)
		})
	}
}

// A plain Env must emit no escape sequences so that piping `doctl` somewhere
// yields readable text.
func TestRenderWelcomePlainEnvEmitsNoANSI(t *testing.T) {
	var buf bytes.Buffer
	env := ui.Plain(&buf, &buf)

	out := renderWelcome(env, welcome{
		version:    "1.2.3",
		context:    "default",
		auth:       authStateValid,
		account:    "sammy@example.com",
		latest:     "1.3.0",
		upgradeCmd: "brew upgrade doctl",
	})

	assert.NotContains(t, out, "\x1b[")
}

func TestRenderWelcomeUsesASCIIGlyphsWhenRequired(t *testing.T) {
	var buf bytes.Buffer
	env := ui.Detect(&buf, &buf, ui.WithASCII(true))

	out := renderWelcome(env, welcome{version: "1.2.3", context: "default", auth: authStateValid})

	assert.Contains(t, out, "+")
	assert.NotContains(t, out, "✔")
}

func TestWelcomeJSON(t *testing.T) {
	got := welcomeJSON(welcome{
		version:    "1.2.3",
		context:    "default",
		auth:       authStateValid,
		account:    "sammy@example.com",
		team:       "Sharks",
		latest:     "1.3.0",
		upgradeCmd: "brew upgrade doctl",
	})

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(got), &payload))

	assert.Equal(t, map[string]any{
		"version":       "1.2.3",
		"context":       "default",
		"authenticated": true,
		"authStatus":    "valid",
		"tokenSource":   "config-file",
		"account":       "sammy@example.com",
		"team":          "Sharks",
		"latestRelease": "1.3.0",
		"updateCommand": "brew upgrade doctl",
	}, payload)
}

// Automation checking whether it is running on an ambient credential needs
// this without parsing the greeting, so it is always present.
func TestWelcomeJSONReportsTokenSource(t *testing.T) {
	tests := []struct {
		source tokenSource
		want   string
	}{
		{tokenSourceConfigFile, "config-file"},
		{tokenSourceEnvVar, "environment"},
		{tokenSourceFlag, "flag"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := welcomeJSON(welcome{version: "1.2.3", context: "default", tokenSource: tt.source})

			var payload map[string]any
			require.NoError(t, json.Unmarshal([]byte(got), &payload))

			assert.Equal(t, tt.want, payload["tokenSource"])
		})
	}
}

// An unverified token is not an authenticated one, so automation reading this
// must not see authenticated: true.
func TestWelcomeJSONReportsAuthStatus(t *testing.T) {
	tests := []struct {
		state      authState
		wantStatus string
		wantAuthed bool
	}{
		{authStateNoToken, "no-token", false},
		{authStateValid, "valid", true},
		{authStateInvalid, "invalid", false},
		{authStateUnverified, "unverified", false},
	}

	for _, tt := range tests {
		t.Run(tt.wantStatus, func(t *testing.T) {
			got := welcomeJSON(welcome{version: "1.2.3", context: "default", auth: tt.state})

			var payload map[string]any
			require.NoError(t, json.Unmarshal([]byte(got), &payload))

			assert.Equal(t, tt.wantStatus, payload["authStatus"])
			assert.Equal(t, tt.wantAuthed, payload["authenticated"])
		})
	}
}

func TestWelcomeJSONOmitsUpdateFieldsWhenCurrent(t *testing.T) {
	got := welcomeJSON(welcome{version: "1.2.3", context: "default"})

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(got), &payload))

	assert.NotContains(t, payload, "latestRelease")
	assert.NotContains(t, payload, "updateCommand")
	assert.NotContains(t, payload, "account")
	assert.NotContains(t, payload, "team")
}

// stubVerifier records the token it was handed so tests can assert which
// context's credential was checked.
type stubVerifier struct {
	who    identity
	state  authState
	tokens []string
}

func (s *stubVerifier) verify(token string) (identity, authState) {
	s.tokens = append(s.tokens, token)
	return s.who, s.state
}

func TestGatherWelcomeResolvesContextAndToken(t *testing.T) {
	tests := []struct {
		name        string
		context     string
		config      map[string]any
		wantContext string
		wantToken   string
	}{
		{
			name:        "default context with a token",
			config:      map[string]any{"context": "default", "access-token": "default-token"},
			wantContext: "default",
			wantToken:   "default-token",
		},
		{
			name:        "default context without a token",
			config:      map[string]any{"context": "default"},
			wantContext: "default",
		},
		{
			name:        "named context with a token",
			config:      map[string]any{"context": "work", "auth-contexts": map[string]string{"work": "work-token"}},
			wantContext: "work",
			wantToken:   "work-token",
		},
		{
			name:        "named context without a token",
			config:      map[string]any{"context": "work", "auth-contexts": map[string]string{"other": "other-token"}},
			wantContext: "work",
		},
		{
			name:        "--context flag wins over config",
			context:     "work",
			config:      map[string]any{"context": "default", "auth-contexts": map[string]string{"work": "work-token"}},
			wantContext: "work",
			wantToken:   "work-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer withStubConfig(t, tt.config)()
			defer withContext(t, tt.context)()

			verifier := &stubVerifier{
				who:   identity{account: "sammy@example.com", team: "Sharks"},
				state: authStateValid,
			}
			lv := &stubVersioner{err: errors.New("offline")}

			w := gatherWelcome(welcomeDeps{
				latest:       lv,
				verify:       verifier.verify,
				checkUpdates: true,
				// An empty cache path keeps the release lookup off disk.
				cachePath: "",
			})

			assert.Equal(t, tt.wantContext, w.context)
			assert.Equal(t, doctl.DoitVersion.String(), w.version)

			if tt.wantToken == "" {
				assert.Empty(t, verifier.tokens, "no token configured, so none should be verified")
				assert.Equal(t, authStateNoToken, w.auth)
				assert.Empty(t, w.account)
				assert.Empty(t, w.team)
				return
			}

			assert.Equal(t, []string{tt.wantToken}, verifier.tokens)
			assert.Equal(t, authStateValid, w.auth)
			assert.Equal(t, "sammy@example.com", w.account)
			assert.Equal(t, "Sharks", w.team)
		})
	}
}

// Provenance is established by comparing the resolved token against each
// layer, because viper merges them and forgets where every value came from.
// The config values below are what viper resolves once the named layer wins.
func TestAccessTokenForContextReportsProvenance(t *testing.T) {
	tests := []struct {
		name       string
		context    string
		config     map[string]any
		env        string
		flag       string
		wantToken  string
		wantSource tokenSource
	}{
		{
			name:       "a token from the config file",
			config:     map[string]any{"context": "default", doctl.ArgAccessToken: "saved-token"},
			wantToken:  "saved-token",
			wantSource: tokenSourceConfigFile,
		},
		{
			name:       "a token from the environment",
			config:     map[string]any{"context": "default", doctl.ArgAccessToken: "env-token"},
			env:        "env-token",
			wantToken:  "env-token",
			wantSource: tokenSourceEnvVar,
		},
		{
			name:       "a token from the flag",
			config:     map[string]any{"context": "default", doctl.ArgAccessToken: "flag-token"},
			flag:       "flag-token",
			wantToken:  "flag-token",
			wantSource: tokenSourceFlag,
		},
		{
			name:       "the flag outranks the environment",
			config:     map[string]any{"context": "default", doctl.ArgAccessToken: "flag-token"},
			env:        "env-token",
			flag:       "flag-token",
			wantToken:  "flag-token",
			wantSource: tokenSourceFlag,
		},
		{
			// Nothing is shadowed, but the environment did supply it.
			name:       "the same token in the file and the environment",
			config:     map[string]any{"context": "default", doctl.ArgAccessToken: "same-token"},
			env:        "same-token",
			wantToken:  "same-token",
			wantSource: tokenSourceEnvVar,
		},
		{
			// The flag and the variable only ever feed the default context.
			name:       "a named context ignores the environment and the flag",
			context:    "work",
			config:     map[string]any{"context": "work", "auth-contexts": map[string]string{"work": "work-token"}},
			env:        "env-token",
			flag:       "flag-token",
			wantToken:  "work-token",
			wantSource: tokenSourceConfigFile,
		},
		{
			name:       "no token at all has no source to report",
			config:     map[string]any{"context": "default"},
			wantSource: tokenSourceConfigFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer withStubConfig(t, tt.config)()
			defer withTokenFlag(t, tt.flag)()
			t.Setenv(envAccessToken, tt.env)

			context := tt.context
			if context == "" {
				context = doctl.ArgDefaultContext
			}

			token, source := accessTokenForContext(context)

			assert.Equal(t, tt.wantToken, token)
			assert.Equal(t, tt.wantSource, source)
		})
	}
}

func TestGatherWelcomeCarriesVerifierVerdict(t *testing.T) {
	for _, state := range []authState{authStateValid, authStateInvalid, authStateUnverified} {
		t.Run(state.String(), func(t *testing.T) {
			defer withStubConfig(t, map[string]any{"context": "default", "access-token": "token"})()
			defer withContext(t, "")()

			verifier := &stubVerifier{state: state}

			w := gatherWelcome(welcomeDeps{
				latest: &stubVersioner{err: errors.New("offline")},
				verify: verifier.verify,
			})

			assert.Equal(t, state, w.auth)
		})
	}
}

// A doctl that has not been given a token yet must reach nothing at all: not
// the API, whose answer it already knows, and not GitHub, since a binary the
// user has not configured has no business making outbound requests.
func TestGatherWelcomeContactsNothingWithoutAToken(t *testing.T) {
	defer withStubConfig(t, map[string]any{"context": "default"})()
	defer withContext(t, "")()

	lv := &stubVersioner{version: "999.0.0"}
	verifier := &stubVerifier{}

	w := gatherWelcome(welcomeDeps{
		latest:       lv,
		verify:       verifier.verify,
		checkUpdates: true,
		cachePath:    filepath.Join(t.TempDir(), "cache.json"),
	})

	assert.Empty(t, verifier.tokens, "the API must not be asked about a token that does not exist")
	assert.Zero(t, lv.calls, "an unconfigured doctl must not contact GitHub")

	assert.Equal(t, authStateNoToken, w.auth)
	assert.Empty(t, w.latest)
	assert.Empty(t, w.upgradeCmd)
	assert.NotEmpty(t, w.version)
}

// The greeting must survive a failed release lookup, since it is useful
// offline.
func TestGatherWelcomeToleratesFailedUpdateCheck(t *testing.T) {
	defer withStubConfig(t, map[string]any{"context": "default", "access-token": "token"})()
	defer withContext(t, "")()

	w := gatherWelcome(welcomeDeps{
		latest:       &stubVersioner{err: errors.New("offline")},
		verify:       (&stubVerifier{state: authStateValid}).verify,
		checkUpdates: true,
		cachePath:    filepath.Join(t.TempDir(), "cache.json"),
	})

	assert.Empty(t, w.latest)
	assert.Empty(t, w.upgradeCmd)
	assert.NotEmpty(t, w.version)
	assert.Equal(t, authStateValid, w.auth, "a failed release lookup must not affect the auth verdict")
}

func TestGatherWelcomeSkipsUpdateCheckWhenDisabled(t *testing.T) {
	defer withStubConfig(t, map[string]any{"context": "default", "access-token": "token"})()
	defer withContext(t, "")()

	lv := &stubVersioner{version: "999.0.0"}
	w := gatherWelcome(welcomeDeps{
		latest:       lv,
		verify:       (&stubVerifier{}).verify,
		checkUpdates: false,
		cachePath:    filepath.Join(t.TempDir(), "cache.json"),
	})

	assert.Zero(t, lv.calls)
	assert.Empty(t, w.latest)
}

func TestGatherWelcomeReportsNewerRelease(t *testing.T) {
	defer withStubConfig(t, map[string]any{"context": "default", "access-token": "token"})()
	defer withContext(t, "")()

	lv := &stubVersioner{version: "999.0.0"}
	w := gatherWelcome(welcomeDeps{
		latest:       lv,
		verify:       (&stubVerifier{}).verify,
		checkUpdates: true,
		cachePath:    filepath.Join(t.TempDir(), "cache.json"),
	})

	assert.Equal(t, "999.0.0", w.latest)
}

func TestLatestReleaseCachesAcrossInvocations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	lv := &stubVersioner{version: "1.3.0"}

	assert.Equal(t, "1.3.0", latestRelease(lv, path, updateCheckTTL))
	assert.Equal(t, "1.3.0", latestRelease(lv, path, updateCheckTTL))

	assert.Equal(t, 1, lv.calls, "second call should be served from the cache")
}

func TestLatestReleaseRefetchesWhenCacheIsStale(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")

	stale, err := json.Marshal(updateCheckCache{
		CheckedAt: time.Now().Add(-48 * time.Hour),
		Latest:    "1.0.0",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, stale, 0600))

	lv := &stubVersioner{version: "1.3.0"}

	assert.Equal(t, "1.3.0", latestRelease(lv, path, updateCheckTTL))
	assert.Equal(t, 1, lv.calls)
}

func TestLatestReleaseIgnoresUnreadableCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0600))

	lv := &stubVersioner{version: "1.3.0"}

	assert.Equal(t, "1.3.0", latestRelease(lv, path, updateCheckTTL))
}

// CI is a parameter rather than something read from the environment, so this
// holds wherever the test itself happens to be running.
func TestUpdateCheckEnabled(t *testing.T) {
	tests := []struct {
		name   string
		optOut string
		isCI   bool
		want   bool
	}{
		{name: "enabled by default", want: true},
		{name: "opt out", optOut: "1", want: false},
		{name: "opt out with true", optOut: "true", want: false},
		{name: "explicit 0 keeps it on", optOut: "0", want: true},
		{name: "explicit false keeps it on", optOut: "false", want: true},
		{name: "skipped in CI", isCI: true, want: false},
		{name: "opt out still wins outside CI", optOut: "1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The opt-out is a genuine environment variable, so clear it
			// before applying what the case asks for.
			t.Setenv(envNoUpdateCheck, tt.optOut)

			assert.Equal(t, tt.want, updateCheckEnabled(tt.isCI))
		})
	}
}

// Only a 401 proves the token is bad. Misclassifying anything else would send
// a user off to rotate a working credential.
func TestAuthStateForError(t *testing.T) {
	apiErr := func(status int) error {
		return &godo.ErrorResponse{
			Response: &http.Response{StatusCode: status},
			Message:  http.StatusText(status),
		}
	}

	tests := []struct {
		name string
		err  error
		want authState
	}{
		{"401 means the token was rejected", apiErr(http.StatusUnauthorized), authStateInvalid},
		{"403 may be a scoped token, not a bad one", apiErr(http.StatusForbidden), authStateUnverified},
		{"429 says nothing about the token", apiErr(http.StatusTooManyRequests), authStateUnverified},
		{"500 says nothing about the token", apiErr(http.StatusInternalServerError), authStateUnverified},
		{"a transport error says nothing about the token", errors.New("dial tcp: no such host"), authStateUnverified},
		{"an API error with no response is inconclusive", &godo.ErrorResponse{Message: "boom"}, authStateUnverified},
		{"a wrapped 401 is still a rejection", fmt.Errorf("get account: %w", apiErr(http.StatusUnauthorized)), authStateInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, authStateForError(tt.err))
		})
	}
}

func TestAccountIdentity(t *testing.T) {
	tests := []struct {
		name    string
		account *do.Account
		want    identity
	}{
		{
			name:    "email only",
			account: &do.Account{Account: &godo.Account{Email: "sammy@example.com"}},
			want:    identity{account: "sammy@example.com"},
		},
		{
			name: "email with a team",
			account: &do.Account{Account: &godo.Account{
				Email: "sammy@example.com",
				Team:  &godo.TeamInfo{Name: "Sharks"},
			}},
			want: identity{account: "sammy@example.com", team: "Sharks"},
		},
		{
			name: "an unnamed team leaves the team empty",
			account: &do.Account{Account: &godo.Account{
				Email: "sammy@example.com",
				Team:  &godo.TeamInfo{},
			}},
			want: identity{account: "sammy@example.com"},
		},
		{name: "nil account", account: nil, want: identity{}},
		{name: "nil inner account", account: &do.Account{}, want: identity{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, accountIdentity(tt.account))
		})
	}
}

// A slow API must not hold the greeting hostage, and must not be reported as a
// rejected token.
func TestWithTimeoutGivesUpOnSlowVerification(t *testing.T) {
	blocked := make(chan struct{})
	t.Cleanup(func() { close(blocked) })

	slow := func(string) (identity, authState) {
		<-blocked
		return identity{account: "sammy@example.com"}, authStateValid
	}

	who, state := withTimeout(slow, 10*time.Millisecond)("token")

	assert.Equal(t, authStateUnverified, state)
	assert.Empty(t, who.account)
	assert.Empty(t, who.team)
}

func TestWithTimeoutPassesThroughPromptResults(t *testing.T) {
	fast := func(token string) (identity, authState) {
		assert.Equal(t, "token", token)
		return identity{account: "sammy@example.com", team: "Sharks"}, authStateValid
	}

	who, state := withTimeout(fast, time.Minute)("token")

	assert.Equal(t, authStateValid, state)
	assert.Equal(t, identity{account: "sammy@example.com", team: "Sharks"}, who)
}

func TestPadIgnoresWidthsBelowContent(t *testing.T) {
	assert.Equal(t, "ab   ", pad("ab", 5))
	assert.Equal(t, "abcdef", pad("abcdef", 3))
}

// withStubConfig overrides just the viper keys the welcome screen reads and
// restores their previous values. It deliberately avoids viper.Reset, which
// would discard the pflag bindings registered in init() and so corrupt every
// later test in the package.
func withStubConfig(t *testing.T, cfg map[string]any) func() {
	t.Helper()

	keys := []string{"context", doctl.ArgAccessToken, "auth-contexts"}

	previous := make(map[string]any, len(keys))
	for _, k := range keys {
		previous[k] = viper.Get(k)
	}

	for _, k := range keys {
		viper.Set(k, cfg[k])
	}

	return func() {
		for _, k := range keys {
			viper.Set(k, previous[k])
		}
	}
}

// withContext sets the global --context value and restores it.
func withContext(t *testing.T, context string) func() {
	t.Helper()

	old := Context
	Context = context

	return func() { Context = old }
}

// withTokenFlag sets the global --access-token value and restores it.
func withTokenFlag(t *testing.T, token string) func() {
	t.Helper()

	old := Token
	Token = token

	return func() { Token = old }
}
