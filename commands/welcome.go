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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// releasesURL is where a user goes when doctl cannot work out how the
	// binary was installed and therefore cannot name an upgrade command.
	releasesURL = "https://github.com/digitalocean/doctl/releases/latest"

	// updateCheckTTL bounds how often a bare `doctl` asks GitHub for the
	// newest release. The answer changes on the order of weeks, so a day of
	// staleness costs nothing and keeps the greeting instant.
	updateCheckTTL = 24 * time.Hour

	// updateCheckTimeout caps the delay the release lookup can add. The
	// greeting is worth showing without an upgrade notice; it is not worth
	// making a user wait for one.
	updateCheckTimeout = 2 * time.Second

	// tokenCheckTimeout caps the delay token validation can add. It is more
	// generous than updateCheckTimeout because the answer is the point of the
	// greeting rather than an aside.
	tokenCheckTimeout = 3 * time.Second

	updateCheckCacheName = "version-check.json"

	// envNoUpdateCheck disables the release lookup entirely.
	envNoUpdateCheck = "DOCTL_NO_UPDATE_CHECK"

	// envAccessToken supplies a token for the default context.
	envAccessToken = "DIGITALOCEAN_ACCESS_TOKEN"
)

// tokenSource is the layer that supplied the active token: the config file,
// DIGITALOCEAN_ACCESS_TOKEN, or --access-token, in ascending order of
// precedence. The outer two win silently, so naming an account without naming
// its source leaves a user no way to explain who doctl is acting as.
type tokenSource int

const (
	// tokenSourceConfigFile is a token that `auth init` saved.
	tokenSourceConfigFile tokenSource = iota

	// tokenSourceEnvVar is DIGITALOCEAN_ACCESS_TOKEN.
	tokenSourceEnvVar

	// tokenSourceFlag is --access-token.
	tokenSourceFlag
)

func (s tokenSource) String() string {
	switch s {
	case tokenSourceEnvVar:
		return "environment"
	case tokenSourceFlag:
		return "flag"
	default:
		return "config-file"
	}
}

// label names the source as the greeting shows it: the thing a user would go
// and unset, rather than the category String reports to machines.
func (s tokenSource) label() string {
	switch s {
	case tokenSourceEnvVar:
		return envAccessToken
	case tokenSourceFlag:
		return "--" + doctl.ArgAccessToken
	default:
		return "config file"
	}
}

// authState is what doctl was able to establish about the active context's
// token. "Configured" and "working" are different claims, and conflating them
// is how a user ends up debugging the wrong problem, so each outcome that a
// user would act on differently gets its own state.
type authState int

const (
	// authStateNoToken means no token is configured for the active context.
	authStateNoToken authState = iota

	// authStateValid means the API accepted the token.
	authStateValid

	// authStateInvalid means the API rejected the token outright. Only a 401
	// produces this: it is the one response that proves the token is the
	// problem.
	authStateInvalid

	// authStateUnverified means a token is configured but doctl could not
	// reach the API to say whether it works. Reporting this separately matters
	// because telling an offline user that their token is bad would send them
	// off to rotate a perfectly good credential.
	authStateUnverified
)

func (s authState) String() string {
	switch s {
	case authStateValid:
		return "valid"
	case authStateInvalid:
		return "invalid"
	case authStateUnverified:
		return "unverified"
	default:
		return "no-token"
	}
}

// welcome is what a bare `doctl` reports: which build is running, whether the
// active context can actually talk to the API, and whether a newer release is
// waiting.
type welcome struct {
	version string

	// context is the name of the active authentication context, which is
	// "default" unless the user selected another one.
	context string

	auth authState

	// account and team identify whoever the token belongs to, when the API
	// told us. A token scoped to a team reports both, and which team is doing
	// the work is as much of the answer as which login is.
	account string
	team    string

	// tokenSource qualifies account and team: a token from outside the config
	// file shadows the saved one, so the names above may not be the ones the
	// user configured.
	tokenSource tokenSource

	// latest is a published release newer than version, or "" when doctl is
	// current or the lookup was skipped or failed.
	latest string

	// upgradeCmd updates doctl for the way this binary was installed, or "" when
	// the install method is unknown.
	upgradeCmd string
}

// identity is who a token belongs to, as reported by the API. Both fields are
// empty when the token was not accepted.
type identity struct {
	account string
	team    string
}

// tokenVerifier reports what the API thinks of a token, and who it belongs to
// when it is accepted. It is a function so that tests can drive every branch
// without a network.
type tokenVerifier func(token string) (identity, authState)

// welcomeDeps are the outside-world lookups the greeting needs, collected so
// tests can substitute them.
type welcomeDeps struct {
	latest    doctl.LatestVersioner
	verify    tokenVerifier
	cachePath string

	// checkUpdates reports whether the release lookup may run. The caller
	// resolves it rather than gatherWelcome reading the environment, so that
	// what the greeting does is a property of its arguments and not of
	// whichever CI provider happens to be running.
	checkUpdates bool
}

// runWelcome greets the user when doctl is invoked with no arguments. Cobra
// would otherwise print the full command listing, which buries the two things
// a newcomer needs: whether their credentials work, and how to start.
func runWelcome(cmd *cobra.Command, args []string) {
	out := cmd.OutOrStdout()
	env := resolveUIEnv(out)

	w := gatherWelcome(welcomeDeps{
		latest:       defaultLatestVersioner(),
		verify:       verifyTokenWithAPI,
		cachePath:    updateCheckCachePath(),
		checkUpdates: updateCheckEnabled(ui.IsCI()),
	})

	if env.Machine {
		fmt.Fprint(out, welcomeJSON(w))
		return
	}

	fmt.Fprint(out, renderWelcome(env, w))
}

func gatherWelcome(deps welcomeDeps) welcome {
	context := Context
	if context == "" {
		context = viper.GetString("context")
	}
	if context == "" {
		context = doctl.ArgDefaultContext
	}

	w := welcome{
		version: doctl.DoitVersion.String(),
		context: context,
		auth:    authStateNoToken,
	}

	token, source := accessTokenForContext(context)
	w.tokenSource = source

	// A freshly installed doctl reaches nothing. There is no account to
	// resolve, and a binary the user has not yet configured should not make
	// unsolicited outbound requests to say a newer one exists; the answer the
	// greeting owes them is `auth init`, which needs no network to give.
	if token == "" {
		return w
	}

	// Both lookups hit the network, so run them together: a user waits for the
	// slower of the two rather than for their sum.
	var (
		wg     sync.WaitGroup
		who    identity
		state  authState
		latest string
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		who, state = deps.verify(token)
	}()

	if deps.checkUpdates {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if newest := latestRelease(deps.latest, deps.cachePath, updateCheckTTL); newest != "" {
				latest = doctl.DoitVersion.NewerThan(newest)
			}
		}()
	}

	wg.Wait()

	w.auth = state
	w.account = who.account
	w.team = who.team
	w.latest = latest

	if w.latest != "" {
		w.upgradeCmd = upgradeCommand()
	}

	return w
}

// accessTokenForContext mirrors CmdConfig.getContextAccessToken and also
// reports which layer supplied the token. The welcome screen runs on the root
// command, which never builds a CmdConfig.
func accessTokenForContext(context string) (string, tokenSource) {
	// --access-token and DIGITALOCEAN_ACCESS_TOKEN only ever feed the default
	// context, so a named context is always whatever the file holds.
	if context != doctl.ArgDefaultContext {
		return viper.GetStringMapString("auth-contexts")[context], tokenSourceConfigFile
	}

	token := viper.GetString(doctl.ArgAccessToken)

	return token, tokenSourceFor(token)
}

// tokenSourceFor identifies the layer that produced token by comparing it
// against each one, in viper's order of precedence. Viper cannot answer this:
// it merges the layers and forgets where every value came from.
func tokenSourceFor(token string) tokenSource {
	switch {
	case token == "":
		return tokenSourceConfigFile
	case token == Token:
		return tokenSourceFlag
	case token == os.Getenv(envAccessToken):
		return tokenSourceEnvVar
	default:
		return tokenSourceConfigFile
	}
}

// verifyTokenWithAPI asks the API who the token belongs to, giving up after
// tokenCheckTimeout.
func verifyTokenWithAPI(token string) (identity, authState) {
	return withTimeout(lookupAccount, tokenCheckTimeout)(token)
}

// withTimeout bounds how long verify may take, reporting an unverified token
// when it runs long. The account lookup takes no context, so the only way to
// bound it is to stop waiting: the abandoned goroutine finishes on its own,
// and the process is on its way out regardless.
func withTimeout(verify tokenVerifier, timeout time.Duration) tokenVerifier {
	return func(token string) (identity, authState) {
		type result struct {
			who   identity
			state authState
		}

		done := make(chan result, 1)

		go func() {
			who, state := verify(token)
			done <- result{who, state}
		}()

		select {
		case r := <-done:
			return r.who, r.state
		case <-time.After(timeout):
			return identity{}, authStateUnverified
		}
	}
}

func lookupAccount(token string) (identity, authState) {
	// Retries are disabled: a greeting must not sit through a backoff sequence
	// on a rate-limited or flaky API.
	client, err := (&doctl.LiveConfig{}).GetGodoClient(Trace, false, token)
	if err != nil {
		return identity{}, authStateUnverified
	}

	account, err := do.NewAccountService(client).Get()
	if err != nil {
		return identity{}, authStateForError(err)
	}

	return accountIdentity(account), authStateValid
}

// authStateForError decides what an account-lookup failure proves about the
// token. Only a 401 proves the token itself was rejected; everything else —
// no network, a 5xx, or a scoped token that cannot read the account — leaves
// its validity genuinely unknown, and must not be reported as a bad token.
func authStateForError(err error) authState {
	var errResp *godo.ErrorResponse

	if errors.As(err, &errResp) && errResp.Response != nil &&
		errResp.Response.StatusCode == http.StatusUnauthorized {
		return authStateInvalid
	}

	return authStateUnverified
}

// accountIdentity names the login and the team a token belongs to. Both
// matter: the same login belongs to several teams, so the email alone does not
// say which one a command is about to act on.
func accountIdentity(account *do.Account) identity {
	if account == nil || account.Account == nil {
		return identity{}
	}

	who := identity{account: account.Email}

	if account.Team != nil {
		who.team = account.Team.Name
	}

	return who
}

func renderWelcome(env ui.Env, w welcome) string {
	bold := func(s string) string { return env.Sprint(env.NewStyle().Bold(true), s) }
	dim := func(s string) string { return env.Sprint(env.NewStyle().Foreground(ui.ColorMuted), s) }
	paint := func(s string, c lipgloss.TerminalColor) string {
		return env.Sprint(env.NewStyle().Foreground(c), s)
	}

	var b strings.Builder

	fmt.Fprintf(&b, "\n%s\n", bold("Welcome to DigitalOcean"))
	fmt.Fprintf(&b, "%s\n\n", dim("doctl is the command line interface for the DigitalOcean API."))

	fmt.Fprintf(&b, "  %s %s\n", dim(pad("Version", 9)), w.version)

	glyph, glyphColor, summary := authSummary(env, w)
	fmt.Fprintf(&b, "  %s %s %s", dim(pad("Account", 9)), paint(glyph, glyphColor), summary)

	// Only a token that outranks the saved one is worth annotating, and it
	// qualifies the account named on this row rather than standing alone,
	// including when that "account" is a rejected token the user has to find.
	if w.tokenSource != tokenSourceConfigFile {
		fmt.Fprintf(&b, " %s", dim("(from "+w.tokenSource.label()+")"))
	}

	fmt.Fprintln(&b)

	// Only shown when the API named a team, since a token that is not scoped
	// to one has no team to report and a blank row would just raise questions.
	if w.team != "" {
		fmt.Fprintf(&b, "  %s %s\n", dim(pad("Team", 9)), w.team)
	}

	fmt.Fprintf(&b, "  %s %s\n", dim(pad("Context", 9)), w.context)

	fmt.Fprintf(&b, "\n%s\n", bold("Get started"))
	for _, step := range welcomeSteps(w.auth) {
		fmt.Fprintf(&b, "  %s %s\n", bold(pad(step.command, 34)), dim(step.summary))
	}

	if w.latest != "" {
		fmt.Fprintf(&b, "\n%s %s\n",
			paint(env.Glyphs().Warning, ui.ColorWarning),
			fmt.Sprintf("Update available: %s %s", bold(w.latest), dim("(you have "+w.version+")")),
		)

		if w.upgradeCmd != "" {
			fmt.Fprintf(&b, "  %s %s\n", dim("run"), bold(w.upgradeCmd))
		} else {
			fmt.Fprintf(&b, "  %s %s\n", dim("download"), releasesURL)
		}
	}

	fmt.Fprintln(&b)

	return b.String()
}

// authSummary maps an auth state onto the glyph, colour, and wording used for
// the Account line.
func authSummary(env ui.Env, w welcome) (string, lipgloss.TerminalColor, string) {
	g := env.Glyphs()

	switch w.auth {
	case authStateValid:
		if w.account != "" {
			return g.Success, ui.ColorSuccess, w.account
		}
		return g.Success, ui.ColorSuccess, "authenticated"
	case authStateInvalid:
		return g.Failure, ui.ColorError, "token rejected by the API"
	case authStateUnverified:
		return g.Pending, ui.ColorWarning, "token could not be verified"
	default:
		return g.Warning, ui.ColorWarning, "not authenticated"
	}
}

type welcomeStep struct {
	command string
	summary string
}

func welcomeSteps(state authState) []welcomeStep {
	switch state {
	case authStateNoToken:
		return []welcomeStep{
			{"doctl auth init", "Connect doctl to your DigitalOcean account"},
			{"doctl auth list", "Show the authentication contexts you have"},
			{"doctl --help", "Browse every command"},
		}
	case authStateInvalid:
		return []welcomeStep{
			{"doctl auth init", "Replace the token for this context"},
			{"doctl auth switch", "Use a different authentication context"},
			{"doctl --help", "Browse every command"},
		}
	default:
		return []welcomeStep{
			{"doctl compute droplet list", "List your Droplets"},
			{"doctl apps list", "List your Apps"},
			{"doctl auth list", "Show the authentication contexts you have"},
			{"doctl --help", "Browse every command"},
		}
	}
}

// pad right-pads s to width so the two columns line up. Padding is applied
// before styling so that ANSI escapes never count toward the width.
func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}

	return s + strings.Repeat(" ", width-len(s))
}

func welcomeJSON(w welcome) string {
	payload := struct {
		Version       string `json:"version"`
		Context       string `json:"context"`
		Authenticated bool   `json:"authenticated"`
		AuthStatus    string `json:"authStatus"`
		TokenSource   string `json:"tokenSource"`
		Account       string `json:"account,omitempty"`
		Team          string `json:"team,omitempty"`
		LatestRelease string `json:"latestRelease,omitempty"`
		UpdateCommand string `json:"updateCommand,omitempty"`
	}{
		Version:       w.version,
		Context:       w.context,
		Authenticated: w.auth == authStateValid,
		AuthStatus:    w.auth.String(),
		TokenSource:   w.tokenSource.String(),
		Account:       w.account,
		Team:          w.team,
		LatestRelease: w.latest,
		UpdateCommand: w.upgradeCmd,
	}

	// The payload is strings and a bool, so marshalling cannot fail.
	b, _ := json.MarshalIndent(payload, "", "  ")

	return string(b) + "\n"
}

// upgradeCommand names the command that updates doctl, inferred from where the
// running binary lives. An empty result means doctl should point at the
// release page instead of guessing.
func upgradeCommand() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}

	// Package managers install into a versioned directory and link a stable
	// path to it, so the install method is only visible after resolving.
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	exe = filepath.ToSlash(exe)

	switch {
	case strings.HasPrefix(exe, "/snap/"), strings.Contains(exe, "/snap/doctl/"):
		return "sudo snap refresh doctl"
	case strings.Contains(exe, "/Cellar/doctl/"), strings.Contains(exe, "/homebrew/"), strings.Contains(exe, "/linuxbrew/"):
		return "brew upgrade doctl"
	default:
		return ""
	}
}

// updateCheckEnabled reports whether the release lookup may run. isCI is a
// parameter rather than a call to ui.IsCI so that the rule can be tested
// without the ambient environment deciding the outcome: doctl's own tests run
// under a CI provider, and one that sets a variable this code did not clear
// would otherwise silently invert the expected answer.
func updateCheckEnabled(isCI bool) bool {
	if v := os.Getenv(envNoUpdateCheck); v != "" && v != "0" && !strings.EqualFold(v, "false") {
		return false
	}

	// A bare `doctl` in CI is not a person deciding whether to upgrade, so
	// spend neither the time nor the API request.
	return !isCI
}

func defaultLatestVersioner() doctl.LatestVersioner {
	return &doctl.GithubLatestVersioner{
		Client: &http.Client{Timeout: updateCheckTimeout},
	}
}

type updateCheckCache struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

func updateCheckCachePath() string {
	return filepath.Join(defaultConfigHome(), updateCheckCacheName)
}

// latestRelease returns the newest published release, preferring a cache entry
// younger than ttl. Every failure is silent: an upgrade notice is a courtesy,
// and losing it must never turn a greeting into an error.
func latestRelease(lv doctl.LatestVersioner, cachePath string, ttl time.Duration) string {
	if cached, err := readUpdateCheckCache(cachePath); err == nil && time.Since(cached.CheckedAt) < ttl {
		return cached.Latest
	}

	latest, err := lv.LatestVersion()
	if err != nil {
		return ""
	}

	writeUpdateCheckCache(cachePath, updateCheckCache{CheckedAt: time.Now(), Latest: latest})

	return latest
}

func readUpdateCheckCache(path string) (updateCheckCache, error) {
	var cache updateCheckCache

	if path == "" {
		return cache, os.ErrNotExist
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return cache, err
	}

	if err := json.Unmarshal(b, &cache); err != nil {
		return cache, err
	}

	return cache, nil
}

func writeUpdateCheckCache(path string, cache updateCheckCache) {
	if path == "" {
		return
	}

	b, err := json.Marshal(cache)
	if err != nil {
		return
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}

	os.WriteFile(path, b, 0600)
}
