/*
Copyright 2026 The Doctl Authors All rights reserved.
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
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm/confirm"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"golang.org/x/term"
	yaml "gopkg.in/yaml.v2"
)

const agentGitHubProvider = "github"

// askConnectGitHub asks whether to start the GitHub OAuth connect flow when
// --gh-repo is set and the team is not connected. Tests replace this.
var askConnectGitHub = defaultAskConnectGitHub

func defaultAskConnectGitHub() (bool, error) {
	if !canPromptForEnv() {
		return false, nil
	}
	choice, err := confirm.New(
		"GitHub is not connected for your team. Connect now? (optional — needed for private repos)",
		confirm.WithDefaultChoice(confirm.No),
	).Prompt()
	if err != nil {
		return false, err
	}
	return choice == confirm.Yes, nil
}

// maybeOfferGitHubAuth checks team GitHub connect status when --gh-repo is used.
// Already connected → continue. Otherwise optionally run `agent auth github`.
// Declining (or non-interactive) continues; public repos do not require auth.
func maybeOfferGitHubAuth(c *CmdConfig) error {
	svc := c.HostedAgents()
	start, err := svc.StartProviderAuth(agentGitHubProvider)
	if err != nil {
		warn("could not check GitHub connection status: %v (continuing; use `doctl harness-runtime auth github` for private repos)", err)
		return nil
	}
	if start != nil && strings.EqualFold(start.Status, agentProviderAuthStatusSuccess) {
		fmt.Fprintf(c.Out, "%s %s\n", colorize("✓", colSuccess), colorize("GitHub already connected", colMuted))
		return nil
	}

	connect, err := askConnectGitHub()
	if err != nil {
		return err
	}
	if !connect {
		fmt.Fprintf(c.Out, "%s %s\n",
			colorize("•", colMuted),
			colorize("Skipping GitHub connect — fine for public repos. Use `doctl harness-runtime auth github` later for private ones.", colMuted))
		return nil
	}

	return completeProviderAuth(c, svc, agentGitHubProvider, start, false, false)
}

const (
	defaultCodexRunModel = "gpt-5.6-sol"
	defaultRunWait       = 5 * time.Minute

	claudeCodeAgentName = "claude-code"
	anthropicAPIKeyEnv  = "ANTHROPIC_API_KEY"
)

var (
	sessionReadyPollInterval = time.Second
	runWaitTimeout           = defaultRunWait
	creationHintInterval     = 5 * time.Second
	creationClock            = time.Now
)

// harnessAgentNames maps friendly --harness values to flat-manifest agent keys.
var harnessAgentNames = map[string]string{
	"opencode":       "opencode",
	"open-code":      "opencode",
	"claude-code":    "claude-code",
	"claude":         "claude-code",
	"codex":          openAIAgentsAdapter,
	"codex-agentapi": openAIAgentsAdapter,
	"openai-codex":   openAIAgentsAdapter,
}

// harnessDisplayNames maps canonical flat-manifest agent keys (the values of
// harnessAgentNames) to their proper display name, matching the casing used
// by prettyAgentKind (e.g. "opencode" -> "OpenCode").
var harnessDisplayNames = map[string]string{
	"opencode":          "OpenCode",
	"claude-code":       "Claude Code",
	openAIAgentsAdapter: "Codex",
}

// prettyHarnessName returns the display name for a raw --harness value or
// alias (case-insensitive), matching agentKindDisplayNames' casing. Falls
// back to the trimmed input when the harness isn't recognized.
func prettyHarnessName(h string) string {
	h = strings.TrimSpace(h)
	if h == "" {
		return ""
	}
	canon, ok := harnessAgentNames[strings.ToLower(h)]
	if !ok {
		canon = strings.ToLower(h)
	}
	if name, ok := harnessDisplayNames[canon]; ok {
		return name
	}
	return h
}

// RunAgentsLaunch gets the caller looking at a session, whichever way that
// takes. Given creation flags or a manifest it creates one and attaches; given
// the name or ID of a session that already exists it attaches to that,
// resuming it first if it is paused.
//
// The two modes are one command because they are one intent — "put me in a
// session" — and splitting them is what made `run` and `attach` read as
// near-synonyms. `create` is the command for the other intent, "make a session
// and leave it alone".
func RunAgentsLaunch(c *CmdConfig) error {
	if err := rejectCreateOnlyLaunchFlags(c); err != nil {
		return err
	}

	stylingEnabled = detectStyling()

	ref := ""
	if len(c.Args) == 1 {
		ref = strings.TrimSpace(c.Args[0])
	}

	// Disambiguation, documented in agentsLaunchHelpMD: a positional argument
	// naming a readable file is a manifest; anything else is a session ref.
	// Creation flags settle it before we ever look at the filesystem.
	if !agentCreationFlagSet(c) && ref != "" && !isReadableFile(ref) {
		return launchExistingSession(c, ref)
	}
	return launchNewSession(c)
}

// launchNewSession is `launch`'s create-then-attach mode: byte-for-byte the
// same creation path as `create`, ending in the chat instead of a ready card.
func launchNewSession(c *CmdConfig) error {
	if !isInteractiveTerminal() {
		return fmt.Errorf("`%s launch` ends in an interactive chat, which needs a terminal; use `%s create` to create a session without attaching", agentCLI, agentCLI)
	}

	src, err := resolveAgentCreationSource(c)
	if err != nil {
		return err
	}

	prog := newCreationProgress(c.Out)
	defer prog.stop()
	prog.header("Launching agent session")

	sess, err := createAgentSession(c, src, prog)
	if err != nil {
		return err
	}
	if sess == nil {
		return errors.New("session create returned no session")
	}
	sess, err = waitForAgentSessionReady(c, sess, prog)
	if err != nil {
		return err
	}
	if err := sendInitialPrompt(c, sess.SessionID, src); err != nil {
		return err
	}

	printCodexProxyTip(c, sess, src.harness)
	return runAgentsAttachSession(c, sess.SessionID)
}

// launchExistingSession is `launch`'s attach-only mode. A paused session is
// resumed on the way in, since "show me this session" and "it happens to be
// asleep" should not be two commands the user has to sequence themselves.
func launchExistingSession(c *CmdConfig, ref string) error {
	if err := rejectCreationFlagsForExistingSession(c); err != nil {
		return err
	}

	svc := c.HostedAgents()
	sessionID, err := resolveSessionRef(svc, ref)
	if err != nil {
		return launchUnresolvableRefErr(ref, err)
	}

	sess, err := svc.GetSession(sessionID)
	if err != nil {
		return beautifyAgentError(err)
	}
	// A destroyed or failed session is not asleep, it is gone; attachToSession
	// reports that rather than this trying to resume it.
	if sess.Status == godo.HostedAgentSessionStatusPaused {
		fmt.Fprintf(c.Out, "%s\n", colorize(fmt.Sprintf("Session %s is paused — resuming…", displaySessionRef(sess)), colMuted))
		if err := svc.ResumeSession(sessionID); err != nil {
			return beautifyAgentError(err)
		}
		// Re-read: the session we hold says "paused", which would banner and
		// stream as though the resume had not happened.
		return runAgentsAttachSession(c, sessionID)
	}

	return attachToSession(c, sess)
}

// launchUnresolvableRefErr explains both readings of a ref that matched
// neither a session nor a file, because from the user's side the failure is
// ambiguous: they may have fat-fingered a session name or a path.
func launchUnresolvableRefErr(ref string, cause error) error {
	if !agentRefNotFound(cause) {
		return cause
	}
	return fmt.Errorf(`%q isn't an existing session, and no readable file exists at that path either.
  %s launch %s    (if this is a session name)
  %s create --harness ...    (to start a new one)`, ref, agentCLI, ref, agentCLI)
}

// agentRefNotFound reports whether resolveSessionRef failed because nothing
// goes by that name, as opposed to an API or ambiguity error worth surfacing.
func agentRefNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no agent session goes by the name")
}

// agentCreationFlagSet reports whether the user named a creation source
// explicitly, which decides Mode A without consulting the filesystem.
func agentCreationFlagSet(c *CmdConfig) bool {
	return c.Doit.IsSet(doctl.ArgAgentHarness) ||
		c.Doit.IsSet(doctl.ArgAgentSpec) ||
		c.Doit.IsSet(doctl.ArgAgentFromConfig)
}

// isReadableFile reports whether path names a regular file doctl can open.
// This is the whole disambiguation rule for `launch`'s positional argument.
func isReadableFile(path string) bool {
	if path == "-" {
		return true
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// rejectCreateOnlyLaunchFlags turns `launch --dry-run` / `--on-hitl` into a
// pointer at `create`, which is where those flags mean something.
func rejectCreateOnlyLaunchFlags(c *CmdConfig) error {
	for _, flag := range []string{doctl.ArgAgentDryRun, doctl.ArgAgentOnHITL} {
		if c.Doit.IsSet(flag) {
			return fmt.Errorf("--%s only applies when creating a session without attaching; use `%s create --%s`", flag, agentCLI, flag)
		}
	}
	if Output == "json" {
		return fmt.Errorf("`%s launch` opens an interactive chat and has no JSON form; use `%s create -o json`", agentCLI, agentCLI)
	}
	return nil
}

// rejectCreationFlagsForExistingSession guards Mode B: the session already
// exists, so a flag describing how to build one was either a typo or a
// misunderstanding, and silently ignoring it would hide that.
func rejectCreationFlagsForExistingSession(c *CmdConfig) error {
	for _, flag := range []string{
		doctl.ArgAgentSecret,
		doctl.ArgAgentName,
		doctl.ArgAgentRepo,
		doctl.ArgAgentTriggerPrompt,
		doctl.ArgAgentWaitTimeout,
	} {
		if c.Doit.IsSet(flag) {
			return fmt.Errorf("--%s only applies when creating a new session; did you mean `%s create --%s`?", flag, agentCLI, flag)
		}
	}
	return nil
}

func displaySessionRef(sess *do.HostedAgentSession) string {
	if sess != nil && strings.TrimSpace(sess.Name) != "" {
		return sess.Name
	}
	if sess != nil && strings.TrimSpace(sess.SessionID) != "" {
		return sess.SessionID
	}
	return "<session>"
}

func resolveHarnessAgent(harness string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(harness))
	if key == "" {
		return "", fmt.Errorf("--%s is required when --%s is not set", doctl.ArgAgentHarness, doctl.ArgAgentSpec)
	}
	agent, ok := harnessAgentNames[key]
	if !ok {
		return "", fmt.Errorf("unsupported --%s %q; supported values: opencode, claude-code, codex",
			doctl.ArgAgentHarness, harness)
	}
	return agent, nil
}

type harnessManifest struct {
	Name   string            `yaml:"name,omitempty"`
	Agent  string            `yaml:"agent"`
	Repos  []string          `yaml:"repos,omitempty"`
	Config map[string]any    `yaml:"config,omitempty"`
	Env    map[string]string `yaml:"env,omitempty"`
}

func buildHarnessManifest(harness, repo, prompt, name string) ([]byte, error) {
	agent, err := resolveHarnessAgent(harness)
	if err != nil {
		return nil, err
	}

	doc := harnessManifest{Agent: agent}
	if ref, err := normalizeHarnessRepoRef(repo); err != nil {
		return nil, err
	} else if ref != "" {
		doc.Repos = []string{ref}
	}
	if name != "" {
		doc.Name = name
	}

	switch {
	case isOpenAISandboxAdapter(agent):
		doc.Env = map[string]string{
			"CODEX_ENVIRONMENT_ID": "${ENV_ID}",
			"CODEX_API_KEY":        "${OPENAI_API_KEY}",
		}
		doc.Config = defaultCodexRunConfig(prompt)
	case agent == claudeCodeAgentName:
		// Claude Code needs its own inference key just like Codex needs
		// OPENAI_API_KEY. Referencing it here (rather than baking in a
		// literal) still routes through expandManifestEnvCollect/ensureEnvVar
		// so a missing key is prompted for on a TTY, but a present-but-bogus
		// key is caught too: prepareClaudeCodeStart validates it against
		// Anthropic's API before doctl ever calls CreateSessionFromManifest,
		// instead of failing deep into a hosted session that was never going
		// to work.
		doc.Env = map[string]string{
			"ANTHROPIC_API_KEY": "${" + anthropicAPIKeyEnv + "}",
		}
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("building harness manifest: %w", err)
	}
	return out, nil
}

// normalizeHarnessRepoRef maps --gh-repo input onto the flat agentspec repos[]
// entry as a GitHub OWNER/REPO reference (e.g. katanemo/plano).
func normalizeHarnessRepoRef(repo string) (string, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "", nil
	}

	repo = strings.TrimSuffix(repo, "/")
	repo = strings.TrimSuffix(repo, ".git")

	if strings.HasPrefix(repo, "git@github.com:") {
		ref := strings.TrimPrefix(repo, "git@github.com:")
		return validateGitHubOwnerRepo(ref)
	}

	if strings.Contains(repo, "://") {
		u, err := url.Parse(repo)
		if err != nil {
			return "", fmt.Errorf("invalid --%s %q: %w", doctl.ArgAgentRepo, repo, err)
		}
		switch strings.ToLower(u.Host) {
		case "github.com", "www.github.com":
			return validateGitHubOwnerRepo(strings.Trim(u.Path, "/"))
		default:
			return "", fmt.Errorf("invalid --%s %q: only GitHub repositories are supported (use https://github.com/owner/repo or owner/repo)", doctl.ArgAgentRepo, repo)
		}
	}

	return validateGitHubOwnerRepo(repo)
}

func validateGitHubOwnerRepo(ref string) (string, error) {
	ref = strings.Trim(ref, "/")
	if ref == "" {
		return "", fmt.Errorf("invalid --%s: missing GitHub owner/repo", doctl.ArgAgentRepo)
	}
	if strings.Count(ref, "/") != 1 || strings.ContainsAny(ref, " \t") {
		return "", fmt.Errorf("invalid --%s %q: use https://github.com/owner/repo or owner/repo", doctl.ArgAgentRepo, ref)
	}
	return ref, nil
}

func defaultCodexRunConfig(prompt string) map[string]any {
	cfg := map[string]any{
		"agent": map[string]any{
			"model": defaultCodexRunModel,
		},
		"environment": map[string]any{
			"type":                "self_hosted",
			"workspace_directory": "/workspace",
		},
	}
	prompt = strings.TrimSpace(prompt)
	if prompt != "" {
		cfg["input"] = []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "input_text",
						"text": prompt,
					},
				},
			},
		}
	}
	return cfg
}

func manifestIncludesPrompt(manifest []byte, prompt string) bool {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return false
	}
	return strings.Contains(string(manifest), prompt)
}

// startSessionFromRawManifest uploads a manifest and creates a hosted session.
// When prog is non-nil it prints Plano-style lifecycle steps around each phase.
func startSessionFromRawManifest(c *CmdConfig, raw []byte, prog *creationProgress) (*do.HostedAgentSession, error) {
	if prog != nil {
		prog.step("Validating configuration…")
	}

	needsOpenAI, err := manifestNeedsOpenAIPrepare(raw)
	if err != nil {
		return nil, err
	}
	if needsOpenAI && prog != nil {
		prog.wait("Creating OpenAI Agents environment…")
	}

	openaiSessionID, envOverlay, err := prepareOpenAISandboxStart(context.Background(), raw)
	if err != nil {
		return nil, err
	}
	if openaiSessionID != "" {
		raw, err = stripSpecOpenAI(raw)
		if err != nil {
			return nil, err
		}
		if prog != nil {
			prog.ok("OpenAI Agents environment ready")
		}
	}

	if err := prepareClaudeCodeStart(context.Background(), raw, prog); err != nil {
		return nil, err
	}

	lookup := envLookupWithOverlay(envOverlay)
	manifest, err := expandManifestEnvCollect(raw, lookup)
	if err != nil {
		return nil, err
	}
	if err := reportAgentManifestValidation(validateAgentManifest(manifest)); err != nil {
		return nil, err
	}

	var createOpt *godo.HostedAgentManifestCreateOptions
	if openaiSessionID != "" {
		createOpt = &godo.HostedAgentManifestCreateOptions{OpenAISessionID: openaiSessionID}
	}

	if prog != nil {
		prog.wait("Creating hosted session…")
	}
	sess, err := c.HostedAgents().CreateSessionFromManifest(manifest, createOpt)
	if err != nil {
		if sessionLimitErr(err) {
			msg, _, _ := agentAPIError(err)
			return nil, fmt.Errorf("%s. Free a slot by removing one: run `doctl harness-runtime list` to find a session ID, then `doctl harness-runtime remove SESSION_ID`", strings.TrimRight(msg, "."))
		}
		return nil, err
	}
	if prog != nil {
		prog.ok(fmt.Sprintf("Session created · %s", displaySessionRef(sess)))
	}
	return sess, nil
}

// prepareClaudeCodeStart mirrors prepareOpenAISandboxStart for claude-code:
// it resolves ANTHROPIC_API_KEY (prompting on a TTY if missing, exactly like
// ensureEnvVar already did) and then confirms Anthropic actually accepts it
// with a real, free API call — closing the gap where a bogus (but non-empty)
// key used to sail through to CreateSessionFromManifest and only fail deep
// inside the hosted sandbox. No-ops for every other harness.
func prepareClaudeCodeStart(ctx context.Context, raw []byte, prog *creationProgress) error {
	doc, err := parseAgentManifest(raw)
	if err != nil {
		return err
	}
	if doc.adapter() != claudeCodeAgentName {
		return nil
	}

	apiKey, err := ensureEnvVar(anthropicAPIKeyEnv)
	if err != nil {
		return fmt.Errorf("%s is required to start a %s session: %w", anthropicAPIKeyEnv, claudeCodeAgentName, err)
	}

	if prog != nil {
		prog.wait(anthropicKeyCheckLabel)
	}
	if err := validateAnthropicAPIKey(ctx, apiKey); err != nil {
		return err
	}
	if prog != nil {
		prog.ok("Anthropic API key OK")
	}
	return nil
}

func manifestNeedsOpenAIPrepare(raw []byte) (bool, error) {
	doc, err := parseAgentManifest(raw)
	if err != nil {
		// Let prepareOpenAISandboxStart / expand own the parse error later.
		return false, nil
	}
	return isOpenAISandboxAdapter(doc.adapter()) || hasOpenAICreateBody(doc), nil
}

// creationSpinnerInterval controls how often the live spinner frame advances
// while creationProgress.wait animates a line on a real terminal.
var creationSpinnerInterval = 120 * time.Millisecond

// creationProgress prints Plano-style lifecycle lines during session create/wait.
// On a real terminal, wait() animates a single overwritten line (spinner +
// ticking elapsed time) instead of a static "…" line, so a slow blocking call
// (e.g. the initial create-session round trip) still looks alive. Piped or
// non-TTY output (including tests) keeps the old one-line-per-call behavior.
type creationProgress struct {
	out   io.Writer
	start time.Time
	spin  *lineSpinner
}

func newCreationProgress(out io.Writer) *creationProgress {
	return &creationProgress{out: out, start: creationClock()}
}

func (p *creationProgress) stopSpin() {
	if p == nil || p.spin == nil {
		return
	}
	p.spin.stopAndClear()
	p.spin = nil
}

// stop ends any in-flight spinner animation, leaving the cursor on a clean
// line. Callers should defer this right after newCreationProgress so an
// early return (error path) never leaves a half-drawn spinner line behind.
func (p *creationProgress) stop() {
	p.stopSpin()
}

func (p *creationProgress) header(msg string) {
	if p == nil {
		return
	}
	p.stopSpin()
	fmt.Fprintln(p.out, boldColor(msg, colHighlight))
}

func (p *creationProgress) step(msg string) {
	if p == nil {
		return
	}
	p.stopSpin()
	fmt.Fprintf(p.out, "%s %s\n", colorize("•", colHighlight), colorize(msg, colHighlight))
}

// wait announces a step that involves waiting on a network call. On a TTY it
// animates in place; repeated calls while already animating just relabel the
// spinner (e.g. periodic provisioning hints) instead of printing new lines.
func (p *creationProgress) wait(msg string) {
	if p == nil {
		return
	}
	if p.spin != nil {
		p.spin.setLabel(msg)
		return
	}
	if isTerminalWriter(p.out) {
		p.spin = newLineSpinner(p.out, msg, p.start)
		return
	}
	fmt.Fprintf(p.out, "%s %s %s\n", colorize("…", colWarning), colorize(msg, colWarning), colorize(fmt.Sprintf("(%s)", p.elapsed()), colMuted))
}

func (p *creationProgress) ok(msg string) {
	if p == nil {
		return
	}
	p.stopSpin()
	fmt.Fprintf(p.out, "%s %s\n", colorize("✓", colSuccess), boldColor(msg, colSuccess))
}

func (p *creationProgress) elapsed() time.Duration {
	if p == nil {
		return 0
	}
	d := creationClock().Sub(p.start)
	if d < 0 {
		return 0
	}
	return d.Truncate(time.Second)
}

// isTerminalWriter reports whether w is a real terminal doctl can safely
// animate a spinner on (as opposed to a pipe, file, or in-memory buffer).
func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// lineSpinner animates a single overwritten status line
// ("\r<frame> <label> (<elapsed>)") while a blocking call is in flight.
// Only meaningful when writing to a real terminal — isTerminalWriter gates
// construction so piped/non-TTY output (including tests) never sees ANSI
// cursor codes.
type lineSpinner struct {
	out   io.Writer
	start time.Time

	mu    sync.Mutex
	label string

	stop chan struct{}
	done chan struct{}
}

func newLineSpinner(out io.Writer, label string, start time.Time) *lineSpinner {
	s := &lineSpinner{
		out:   out,
		start: start,
		label: label,
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *lineSpinner) setLabel(label string) {
	s.mu.Lock()
	s.label = label
	s.mu.Unlock()
}

func (s *lineSpinner) currentLabel() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.label
}

func (s *lineSpinner) run() {
	defer close(s.done)
	ticker := time.NewTicker(creationSpinnerInterval)
	defer ticker.Stop()

	frame := 0
	s.paint(frame)
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			frame = (frame + 1) % len(spinnerFrames)
			s.paint(frame)
		}
	}
}

func (s *lineSpinner) paint(frame int) {
	elapsed := time.Since(s.start).Truncate(time.Second)
	fmt.Fprintf(s.out, "\r\x1b[K%s %s %s",
		colorize(spinnerFrames[frame], colWarning),
		colorize(s.currentLabel(), colWarning),
		colorize(fmt.Sprintf("(%s)", elapsed), colMuted))
}

// stopAndClear halts the animation goroutine and clears the spinner line so
// the next print starts fresh at column 0.
func (s *lineSpinner) stopAndClear() {
	close(s.stop)
	<-s.done
	fmt.Fprint(s.out, "\r\x1b[K")
}

// provisioningHints keep a long PROVISIONING wait feeling alive. Later lines
// call out that workspace bits may exist while the agent is still starting.
var provisioningHints = []string{
	"Allocating sandbox…",
	"Starting workspace…",
	"Starting agent runtime…",
	"Workspace may be up; waiting for agent…",
}

func waitForSessionReady(ctx context.Context, svc do.HostedAgentsService, sessionID string, prog *creationProgress) (*do.HostedAgentSession, error) {
	ticker := time.NewTicker(sessionReadyPollInterval)
	defer ticker.Stop()

	var (
		lastStatus   godo.HostedAgentSessionStatus
		hintIdx      int
		nextHint     time.Time
		sawBitsReady bool
		announced    bool
		out          io.Writer
	)
	if prog != nil {
		out = prog.out
		nextHint = creationClock().Add(creationHintInterval)
	}

	for {
		sess, err := svc.GetSession(sessionID)
		if err != nil {
			return nil, err
		}

		if prog != nil && !sawBitsReady {
			if note := bitsReadyNote(sess); note != "" {
				sawBitsReady = true
				prog.ok(note)
			}
		}

		if sess.Status != lastStatus {
			switch sess.Status {
			case godo.HostedAgentSessionStatusProvisioning:
				if prog != nil && !announced {
					prog.wait("Waiting for agent…")
					announced = true
					hintIdx = 0
					nextHint = creationClock().Add(creationHintInterval)
				} else if out != nil && prog == nil {
					fmt.Fprintf(out, "  %s %s\n", runStatusGlyph(sess.Status), colorize(runStatusLabel(sess.Status), colMuted))
				}
			case godo.HostedAgentSessionStatusReady, godo.HostedAgentSessionStatusDetached:
				if prog != nil {
					elapsed := prog.elapsed()
					if elapsed > 0 {
						prog.ok(fmt.Sprintf("Agent is ready (%s)", elapsed))
					} else {
						prog.ok("Agent is ready")
					}
				} else if out != nil {
					fmt.Fprintf(out, "  %s %s\n", runStatusGlyph(sess.Status), colorize(runStatusLabel(sess.Status), colMuted))
				}
			default:
				if prog != nil {
					prog.wait(runStatusLabel(sess.Status))
				} else if out != nil {
					fmt.Fprintf(out, "  %s %s\n", runStatusGlyph(sess.Status), colorize(runStatusLabel(sess.Status), colMuted))
				}
			}
			lastStatus = sess.Status
		} else if prog != nil &&
			sess.Status == godo.HostedAgentSessionStatusProvisioning &&
			hintIdx < len(provisioningHints) &&
			!creationClock().Before(nextHint) {
			hint := provisioningHints[hintIdx]
			if sawBitsReady {
				hint = "Waiting for agent…"
			}
			prog.wait(hint)
			hintIdx++
			nextHint = creationClock().Add(creationHintInterval)
		}

		switch sess.Status {
		case godo.HostedAgentSessionStatusReady, godo.HostedAgentSessionStatusDetached:
			return sess, nil
		case godo.HostedAgentSessionStatusFailed,
			godo.HostedAgentSessionStatusDestroyed,
			godo.HostedAgentSessionStatusDestroying:
			return nil, fmt.Errorf("session entered %s before becoming ready", humanSessionStatus(sess.Status))
		}

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("timed out waiting for session %s to become ready (last status: %s)", sessionID, humanSessionStatus(lastStatus))
			}
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// bitsReadyNote returns a one-shot message when the session payload shows
// environment identifiers while status is still not READY.
func bitsReadyNote(sess *do.HostedAgentSession) string {
	if sess == nil || sess.HostedAgentSession == nil {
		return ""
	}
	switch sess.Status {
	case godo.HostedAgentSessionStatusReady, godo.HostedAgentSessionStatusDetached:
		return ""
	}
	switch {
	case strings.TrimSpace(sess.OpenAIEnvironmentID) != "":
		return "Environment ready · waiting for agent"
	case strings.TrimSpace(sess.OpenAISessionID) != "":
		return "Environment ready · waiting for agent"
	default:
		return ""
	}
}

func runStatusGlyph(status godo.HostedAgentSessionStatus) string {
	switch status {
	case godo.HostedAgentSessionStatusReady, godo.HostedAgentSessionStatusDetached:
		return colorize("✓", colSuccess)
	case godo.HostedAgentSessionStatusFailed,
		godo.HostedAgentSessionStatusDestroyed,
		godo.HostedAgentSessionStatusDestroying:
		return colorize("✗", colError)
	default:
		return colorize("⟳", colMuted)
	}
}

func runStatusLabel(status godo.HostedAgentSessionStatus) string {
	switch status {
	case godo.HostedAgentSessionStatusProvisioning:
		return "Waiting for agent…"
	case godo.HostedAgentSessionStatusReady:
		return "Agent ready"
	case godo.HostedAgentSessionStatusDetached:
		return "Agent ready (detached)"
	case godo.HostedAgentSessionStatusPaused:
		return "Agent paused"
	default:
		return humanSessionStatus(status)
	}
}

type runReadySummary struct {
	Session *do.HostedAgentSession
	Harness string
	Repo    string
	Prompt  string
}

// printRunReadySummary renders a compact session card after create/wait
// succeeds. The lifecycle line already printed "✓ Agent is ready", so
// the card focuses on identity and next steps without repeating that headline.
func printRunReadySummary(w io.Writer, sum runReadySummary) {
	ref := displaySessionRef(sum.Session)
	agent := prettyAgentKind(sum.Session.AgentKind)
	if agent == "agent" && strings.TrimSpace(sum.Harness) != "" {
		agent = prettyHarnessName(sum.Harness)
	}

	var body strings.Builder
	body.WriteString(cardRow("Session", ref))
	if Verbose && sum.Session != nil && sum.Session.SessionID != "" && sum.Session.SessionID != ref {
		body.WriteString(cardRow("ID", colorize(sum.Session.SessionID, colMuted)))
	}
	body.WriteString(cardRow("Agent", agent))
	if Verbose && sum.Repo != "" {
		body.WriteString(cardRow("Repo", sum.Repo))
	}
	if prompt := strings.TrimSpace(sum.Prompt); prompt != "" {
		if len(prompt) > 72 {
			prompt = prompt[:69] + "…"
		}
		body.WriteString(cardRow("Prompt", colorize("\""+prompt+"\"", colMuted)))
	}

	fmt.Fprintln(&body)
	fmt.Fprintln(&body, colorize("Next step", colMuted))
	body.WriteString(cardRow("launch", agentCLI+" launch "+ref))
	if isCodexReadyAgent(sum) {
		body.WriteString(cardRow("proxy", agentCLI+" start-proxy --type codex --session "+ref+" --port 1144"))
	}

	renderAgentCard(w, body.String())
}

func isCodexReadyAgent(sum runReadySummary) bool {
	if sum.Session != nil && sum.Session.HostedAgentSession != nil {
		if sum.Session.AgentKind == godo.HostedAgentKindOpenAICodex {
			return true
		}
	}
	h := strings.ToLower(strings.TrimSpace(sum.Harness))
	return h == "codex" || h == "codex-agentapi" || h == "openai-codex"
}

// printSessionsList renders a styled text list of sessions (not used for -o json).
func printSessionsList(w io.Writer, sessions []do.HostedAgentSession) {
	if len(sessions) == 0 {
		fmt.Fprintln(w, colorize("No sessions", colMuted))
		return
	}

	noun := "sessions"
	if len(sessions) == 1 {
		noun = "session"
	}
	fmt.Fprintln(w, boldColor(fmt.Sprintf("%d %s", len(sessions), noun), colHighlight))
	fmt.Fprintln(w)

	for i, sess := range sessions {
		if i > 0 {
			fmt.Fprintln(w)
		}
		printSessionListItem(w, &sess)
	}
}

func printSessionListItem(w io.Writer, sess *do.HostedAgentSession) {
	if sess == nil || sess.HostedAgentSession == nil {
		return
	}
	ref := displaySessionRef(sess)
	agent := prettyAgentKind(sess.AgentKind)

	fmt.Fprintf(w, "%s %s\n", sessionStatusGlyph(sess.Status), boldColor(ref, colHighlight))
	meta := []string{agent, colorizeSessionStatus(sess.Status)}
	if !sess.CreatedAt.Time.IsZero() {
		meta = append(meta, colorize(createdAgo(sess.CreatedAt.Time), colMuted))
	}
	fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
	if Verbose {
		if id := strings.TrimSpace(sess.SessionID); id != "" && id != ref {
			fmt.Fprintf(w, "  %s\n", colorize(id, colMuted))
		}
		if repo := strings.TrimSpace(sess.RepoHint); repo != "" {
			fmt.Fprintf(w, "  %s %s\n", colorize("repo", colMuted), repo)
		}
	}
}

// printSessionShowCard renders a single-session detail card for `agent show`.
func printSessionShowCard(w io.Writer, sess *do.HostedAgentSession) {
	if sess == nil || sess.HostedAgentSession == nil {
		fmt.Fprintln(w, colorize("No session", colMuted))
		return
	}
	ref := displaySessionRef(sess)
	agent := prettyAgentKind(sess.AgentKind)

	var body strings.Builder
	body.WriteString(cardRow("Session", ref))
	if Verbose {
		if id := strings.TrimSpace(sess.SessionID); id != "" && id != ref {
			body.WriteString(cardRow("ID", colorize(id, colMuted)))
		}
	}
	body.WriteString(cardRow("Agent", agent))
	body.WriteString(cardRow("Status", sessionStatusGlyph(sess.Status)+" "+colorizeSessionStatus(sess.Status)))
	if Verbose {
		if repo := strings.TrimSpace(sess.RepoHint); repo != "" {
			body.WriteString(cardRow("Repo", repo))
		}
	}
	if parent := strings.TrimSpace(sess.ParentSessionID); parent != "" {
		body.WriteString(cardRow("Parent", colorize(parent, colMuted)))
	}
	if !sess.CreatedAt.Time.IsZero() {
		body.WriteString(cardRow("Created", colorize(formatCreatedAt(sess.CreatedAt.Time), colMuted)))
	}

	switch sess.Status {
	case godo.HostedAgentSessionStatusReady, godo.HostedAgentSessionStatusDetached, godo.HostedAgentSessionStatusPaused:
		fmt.Fprintln(&body)
		fmt.Fprintln(&body, colorize("Next step", colMuted))
		// launch resumes a paused session on the way in, so the same line is
		// the right next step whether this one is awake or not.
		body.WriteString(cardRow("launch", agentCLI+" launch "+ref))
	}

	renderAgentCard(w, body.String())
}

func sessionStatusGlyph(status godo.HostedAgentSessionStatus) string {
	switch status {
	case godo.HostedAgentSessionStatusReady, godo.HostedAgentSessionStatusDetached:
		return colorize("●", colSuccess)
	case godo.HostedAgentSessionStatusProvisioning:
		return colorize("…", colWarning)
	case godo.HostedAgentSessionStatusPaused:
		return colorize("○", colWarning)
	case godo.HostedAgentSessionStatusFailed:
		return colorize("✗", colError)
	case godo.HostedAgentSessionStatusDestroying, godo.HostedAgentSessionStatusDestroyed:
		return colorize("✗", colMuted)
	default:
		return colorize("·", colMuted)
	}
}

func colorizeSessionStatus(status godo.HostedAgentSessionStatus) string {
	label := humanSessionStatus(status)
	switch status {
	case godo.HostedAgentSessionStatusReady, godo.HostedAgentSessionStatusDetached:
		return colorize(label, colSuccess)
	case godo.HostedAgentSessionStatusProvisioning, godo.HostedAgentSessionStatusPaused:
		return colorize(label, colWarning)
	case godo.HostedAgentSessionStatusFailed:
		return colorize(label, colError)
	default:
		return colorize(label, colMuted)
	}
}
