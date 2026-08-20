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
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	yaml "gopkg.in/yaml.v2"
)

const (
	defaultCodexRunModel = "gpt-5.6-sol"
	defaultRunWait       = 5 * time.Minute
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

// RunAgentsRun creates a hosted agent session from --harness/--gh-repo/--prompt
// (or --spec / --config-id), waits until the session is ready, optionally sends
// an initial prompt, and attaches interactively.
func RunAgentsRun(c *CmdConfig) error {
	harness, err := c.Doit.GetString(c.NS, doctl.ArgAgentHarness)
	if err != nil {
		return err
	}
	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAgentSpec)
	if err != nil {
		return err
	}
	configID, err := c.Doit.GetString(c.NS, doctl.ArgAgentConfigID)
	if err != nil {
		return err
	}
	repo, err := c.Doit.GetString(c.NS, doctl.ArgAgentRepo)
	if err != nil {
		return err
	}
	prompt, err := c.Doit.GetString(c.NS, doctl.ArgAgentTriggerPrompt)
	if err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	if err != nil {
		return err
	}
	noAttach, err := c.Doit.GetBool(c.NS, doctl.ArgAgentNoAttach)
	if err != nil {
		return err
	}
	waitSec, err := c.Doit.GetInt(c.NS, doctl.ArgAgentWaitTimeout)
	if err != nil {
		return err
	}

	harness = strings.TrimSpace(harness)
	specPath = strings.TrimSpace(specPath)
	configID = strings.TrimSpace(configID)
	repo = strings.TrimSpace(repo)
	prompt = strings.TrimSpace(prompt)

	sources := 0
	for _, s := range []string{harness, specPath, configID} {
		if s != "" {
			sources++
		}
	}
	if sources > 1 {
		return fmt.Errorf("--%s, --%s, and --%s are mutually exclusive; provide only one", doctl.ArgAgentHarness, doctl.ArgAgentSpec, doctl.ArgAgentConfigID)
	}
	if sources == 0 {
		return fmt.Errorf("one of --%s, --%s, or --%s is required", doctl.ArgAgentHarness, doctl.ArgAgentSpec, doctl.ArgAgentConfigID)
	}
	if configID != "" && repo != "" {
		return fmt.Errorf("--%s cannot be used with --%s; put the repo in the Agent Config instead", doctl.ArgAgentRepo, doctl.ArgAgentConfigID)
	}
	if name != "" {
		if err := validateHostedAgentIdentifier(name); err != nil {
			return err
		}
	}

	stylingEnabled = detectStyling()
	prog := newCreationProgress(c.Out)
	prog.header("Launching agent session")

	var (
		sess *do.HostedAgentSession
		raw  []byte
	)
	switch {
	case configID != "":
		sess, err = createSessionFromConfig(c, configID, name, prog)
		if err != nil {
			return err
		}
	case specPath != "":
		raw, err = readManifestBytes(os.Stdin, specPath)
		if err != nil {
			return err
		}
		if manifestUsesLegacyEnvelope(raw) {
			warn("this manifest uses the deprecated apiVersion/kind/metadata/spec envelope format; " +
				"switch to the flat format (top-level `agent:` key, no envelope — see `doctl agent start --help`). " +
				"The envelope is still accepted for now but will be rejected after the transition window")
		}
		raw, err = injectManifestName(raw, name)
		if err != nil {
			return err
		}
		sess, err = startSessionFromRawManifest(c, raw, prog)
		if err != nil {
			return err
		}
	default:
		raw, err = buildHarnessManifest(harness, repo, prompt, name)
		if err != nil {
			return err
		}
		raw, err = injectManifestName(raw, name)
		if err != nil {
			return err
		}
		sess, err = startSessionFromRawManifest(c, raw, prog)
		if err != nil {
			return err
		}
	}

	sessionID := sess.SessionID
	if sessionID == "" {
		return errors.New("session create returned no session id")
	}

	wait := runWaitTimeout
	if waitSec > 0 {
		wait = time.Duration(waitSec) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	sess, err = waitForSessionReady(ctx, c.HostedAgents(), sessionID, prog)
	if err != nil {
		return err
	}

	if noAttach {
		repoRef, _ := normalizeHarnessRepoRef(repo)
		printRunReadySummary(c.Out, runReadySummary{
			Session: sess,
			Harness: harness,
			Repo:    repoRef,
			Prompt:  prompt,
		})
		return nil
	}

	if prompt != "" && (raw == nil || !manifestIncludesPrompt(raw, prompt)) {
		if _, err := c.HostedAgents().SendInput(sessionID, &godo.HostedAgentSendInputRequest{Text: prompt}); err != nil {
			return fmt.Errorf("sending initial prompt: %w", err)
		}
	}

	return runAgentsAttachSession(c, sessionID)
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

	if isOpenAISandboxAdapter(agent) {
		doc.Env = map[string]string{
			"CODEX_ENVIRONMENT_ID": "${ENV_ID}",
			"CODEX_API_KEY":        "${OPENAI_API_KEY}",
		}
		doc.Config = defaultCodexRunConfig(prompt)
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
	manifest, err := expandManifestEnvLookup(raw, envLookupWithOverlay(envOverlay))
	if err != nil {
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
			return nil, fmt.Errorf("%s. Free a slot by removing one: run `doctl agent list` to find a session ID, then `doctl agent remove SESSION_ID`", strings.TrimRight(msg, "."))
		}
		return nil, err
	}
	if prog != nil {
		prog.ok(fmt.Sprintf("Session created · %s", displaySessionRef(sess)))
	}
	return sess, nil
}

func manifestNeedsOpenAIPrepare(raw []byte) (bool, error) {
	doc, err := parseAgentManifest(raw)
	if err != nil {
		// Let prepareOpenAISandboxStart / expand own the parse error later.
		return false, nil
	}
	return isOpenAISandboxAdapter(doc.adapter()) || hasOpenAICreateBody(doc), nil
}

// creationProgress prints Plano-style lifecycle lines during session create/wait.
type creationProgress struct {
	out   io.Writer
	start time.Time
}

func newCreationProgress(out io.Writer) *creationProgress {
	return &creationProgress{out: out, start: creationClock()}
}

func (p *creationProgress) header(msg string) {
	if p == nil {
		return
	}
	fmt.Fprintln(p.out, boldColor(msg, colHighlight))
}

func (p *creationProgress) step(msg string) {
	if p == nil {
		return
	}
	fmt.Fprintf(p.out, "%s %s\n", colorize("•", colHighlight), colorize(msg, colHighlight))
}

func (p *creationProgress) wait(msg string) {
	if p == nil {
		return
	}
	fmt.Fprintf(p.out, "%s %s\n", colorize("…", colWarning), colorize(msg, colWarning))
}

func (p *creationProgress) ok(msg string) {
	if p == nil {
		return
	}
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
			prog.wait(fmt.Sprintf("%s (%s)", hint, prog.elapsed()))
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
		agent = strings.TrimSpace(sum.Harness)
	}

	var body strings.Builder
	body.WriteString(cardRow("Session", ref))
	if sum.Session != nil && sum.Session.SessionID != "" && sum.Session.SessionID != ref {
		body.WriteString(cardRow("ID", colorize(sum.Session.SessionID, colMuted)))
	}
	body.WriteString(cardRow("Agent", agent))
	if sum.Repo != "" {
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
	body.WriteString(cardRow("attach", "doctl agent attach "+ref))

	renderAgentCard(w, body.String())
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
		meta = append(meta, colorize(sess.CreatedAt.Time.UTC().Format("2006-01-02 15:04"), colMuted))
	}
	fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
	if id := strings.TrimSpace(sess.SessionID); id != "" && id != ref {
		fmt.Fprintf(w, "  %s\n", colorize(id, colMuted))
	}
	if repo := strings.TrimSpace(sess.RepoHint); repo != "" {
		fmt.Fprintf(w, "  %s %s\n", colorize("repo", colMuted), repo)
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
	if id := strings.TrimSpace(sess.SessionID); id != "" && id != ref {
		body.WriteString(cardRow("ID", colorize(id, colMuted)))
	}
	body.WriteString(cardRow("Agent", agent))
	body.WriteString(cardRow("Status", sessionStatusGlyph(sess.Status)+" "+colorizeSessionStatus(sess.Status)))
	if repo := strings.TrimSpace(sess.RepoHint); repo != "" {
		body.WriteString(cardRow("Repo", repo))
	}
	if parent := strings.TrimSpace(sess.ParentSessionID); parent != "" {
		body.WriteString(cardRow("Parent", colorize(parent, colMuted)))
	}
	if !sess.CreatedAt.Time.IsZero() {
		body.WriteString(cardRow("Created", colorize(sess.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC"), colMuted)))
	}

	switch sess.Status {
	case godo.HostedAgentSessionStatusReady, godo.HostedAgentSessionStatusDetached, godo.HostedAgentSessionStatusPaused:
		fmt.Fprintln(&body)
		fmt.Fprintln(&body, colorize("Next step", colMuted))
		body.WriteString(cardRow("attach", "doctl agent attach "+ref))
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
