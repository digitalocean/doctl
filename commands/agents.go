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

// The `doctl open-harness-runtime` command (aliases: agent, agents, ohr) wraps the
// godo HostedAgents service for Managed Agents Runtime Services (M.A.R.S).
// Wire types and the SSE iterator live in godo; this file handles CLI plumbing,
// argument parsing, and human-readable rendering of streamed events.
package commands

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/agentproxy"
	"github.com/digitalocean/doctl/internal/agentproxy/codex"
	"github.com/digitalocean/godo"
	"github.com/muesli/termenv"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	yaml "gopkg.in/yaml.v2"
)

// stylingEnabled gates ANSI color and markdown rendering. It is flipped on by
// the interactive entrypoints (attach/logs) when stdout is a real terminal and
// NO_COLOR is unset; it stays false in unit tests and piped output so their
// results are plain and deterministic.
var stylingEnabled bool

// Agent chat palette, sourced from doctl's shared color scheme.
var (
	colSuccess   = charm.Colors.Success
	colError     = charm.Colors.Error
	colWarning   = charm.Colors.Warning
	colHighlight = charm.Colors.Highlight
	colMuted     = charm.Colors.Muted
)

// detectStyling reports whether ANSI styling should be emitted for the current
// process: stdout is a terminal and NO_COLOR is unset.
func detectStyling() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// colorize applies a foreground color when styling is enabled, else returns s.
func colorize(s string, c lipgloss.Color) string {
	if !stylingEnabled {
		return s
	}
	return lipgloss.NewStyle().Foreground(c).Render(s)
}

// boldColor applies a bold foreground color when styling is enabled.
func boldColor(s string, c lipgloss.Color) string {
	if !stylingEnabled {
		return s
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true).Render(s)
}

// renderMarkdown turns a markdown document into styled terminal text. With
// styling disabled (pipes, CI, unit tests) it returns the text unchanged so
// scripts keep clean, greppable output.
func renderMarkdown(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	if !stylingEnabled {
		return text
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithColorProfile(termenv.TrueColor),
		glamour.WithWordWrap(mdWrapWidth()),
		// Keep line breaks the agent emits. Without this, Markdown collapses
		// single newlines into spaces, so code the agent writes without a
		// well-formed fence (e.g. a ```lang tag mid-sentence) gets flattened
		// onto one line and becomes unreadable.
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return text
	}
	out, err := r.Render(normalizeCodeFences(text))
	if err != nil {
		return text
	}
	return out
}

// normalizeCodeFences rewrites the agent's Markdown so every ``` code-fence
// marker starts on its own line, and any opening fence with an info string
// (```python) is closed by a bare ``` before end-of-message.
//
// Streamed agent output routinely glues an opening fence to the end of a
// sentence ("...straightforward.```python") and/or never emits a closing
// fence. Either mistake stops the Markdown parser from recognizing the code
// block, so the code renders as flowed plain text with its indentation
// collapsed. Detaching the fences (and balancing an odd one) lets the renderer
// syntax-highlight the block and preserve indentation verbatim.
func normalizeCodeFences(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 16)
	atLineStart := true
	fences := 0
	i := 0
	for i < len(s) {
		if strings.HasPrefix(s[i:], "```") {
			if !atLineStart {
				b.WriteByte('\n')
			}
			b.WriteString("```")
			i += 3
			fences++
			// Copy the rest of the fence line (the info string, e.g. "python")
			// verbatim; the code body starts on the following line.
			for i < len(s) && s[i] != '\n' {
				b.WriteByte(s[i])
				i++
			}
			atLineStart = false
			continue
		}
		c := s[i]
		b.WriteByte(c)
		atLineStart = c == '\n'
		i++
	}
	// An odd fence count means a block was opened but never closed; add the
	// missing closing fence so the parser renders it as code rather than
	// swallowing the rest of the message.
	if fences%2 == 1 {
		if !atLineStart {
			b.WriteByte('\n')
		}
		b.WriteString("```\n")
	}
	return b.String()
}

// mdWrapWidth is the word-wrap column for markdown, clamped to a readable range.
func mdWrapWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w >= 40 {
		if w > 100 {
			return 100
		}
		return w
	}
	return 80
}

// msgAccumulator buffers an assistant turn's streamed token deltas so the whole
// message can be rendered as markdown once it's complete, rather than emitting
// raw tokens one at a time.
type msgAccumulator struct {
	buf strings.Builder
}

func (m *msgAccumulator) add(s string) { m.buf.WriteString(s) }

// flush renders the buffered message as markdown and writes it, then resets.
func (m *msgAccumulator) flush(out io.Writer) {
	if m.buf.Len() == 0 {
		return
	}
	text := m.buf.String()
	m.buf.Reset()
	rendered := renderMarkdown(text)
	if !strings.HasSuffix(rendered, "\n") {
		rendered += "\n"
	}
	fmt.Fprint(out, rendered)
}

// Agents creates the `doctl open-harness-runtime` command tree (aliases: agent, agents, ohr).
func Agents() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     agentCmdName,
			Aliases: []string{"agent", "agents", "ohr"},
			Short:   "Managed Agents Runtime Services (M.A.R.S)",
			Long:    agentsRootHelpMD,
			GroupID: hostedAgentsGroup,
		},
	}

	cmdStart := CmdBuilder(cmd, RunAgentsStart, "start",
		"Start a new session",
		agentsStartHelpMD,
		Writer, agentsNS(aliasOpt("deploy"),
			displayerType(&displayers.HostedAgentSession{}))...)
	AddStringFlag(cmdStart, doctl.ArgAgentHarness, "", "", "Coding-agent harness (opencode, claude-code, codex). Builds the manifest for you. Mutually exclusive with --spec and --config-id.")
	AddStringFlag(cmdStart, doctl.ArgAgentSpec, "", "", `Path to an agent manifest in YAML or JSON (flat format; minimal: "agent: opencode"). Set to "-" to read from stdin. ${VAR} references are resolved from the local environment. Mutually exclusive with --harness and --config-id.`)
	AddStringFlag(cmdStart, doctl.ArgAgentConfigID, "", "", "ID of an existing Agent Config to start the session from. Requires --name. Mutually exclusive with --harness and --spec.")
	AddStringFlag(cmdStart, doctl.ArgAgentRepo, "", "", "GitHub repository to clone into the workspace (https://github.com/org/repo or org/repo). Only with --harness or --spec.")
	AddStringFlag(cmdStart, doctl.ArgAgentTriggerPrompt, "", "", "Initial prompt to send once the session is ready")
	AddStringFlag(cmdStart, doctl.ArgAgentName, "", "", "Name for the new session (sets the manifest's name). If omitted, the server auto-generates a name. Must be unique among your team's active sessions. Required with --config-id.")
	AddIntFlag(cmdStart, doctl.ArgAgentWaitTimeout, "", 300, "Maximum seconds to wait for the session to become ready (0 uses the default). Ignored with -o json.")
	cmdStart.MarkFlagsMutuallyExclusive(doctl.ArgAgentHarness, doctl.ArgAgentSpec)
	cmdStart.MarkFlagsMutuallyExclusive(doctl.ArgAgentHarness, doctl.ArgAgentConfigID)
	cmdStart.MarkFlagsMutuallyExclusive(doctl.ArgAgentSpec, doctl.ArgAgentConfigID)
	cmdStart.Example = agentCLI + ` start --harness claude-code --gh-repo owner/repo --prompt "Review the README"; ` + agentCLI + ` start --spec agent-spec.yaml --name my-session; ` + agentCLI + ` start --config-id cfg_abc123 --name my-session`

	cmdValidate := CmdBuilder(cmd, RunAgentsValidate, "validate",
		"Validate an agent manifest",
		agentsValidateHelpMD,
		Writer, agentsNS()...)
	AddStringFlag(cmdValidate, doctl.ArgAgentSpec, "", "", `Path to an agent manifest in YAML or JSON (flat or legacy envelope). Set to "-" to read from stdin.`, requiredOpt())
	cmdValidate.Example = agentCLI + ` validate --spec agent.yaml`

	cmdRun := CmdBuilder(cmd, RunAgentsRun, "run",
		"Start one session and attach",
		agentsRunHelpMD,
		Writer, agentsNS(aliasOpt("up"))...)
	AddStringFlag(cmdRun, doctl.ArgAgentHarness, "", "", "Coding-agent harness (opencode, claude-code, codex). Mutually exclusive with --spec and --config-id.")
	AddStringFlag(cmdRun, doctl.ArgAgentSpec, "", "", `Optional manifest file instead of --harness or --config-id. Same format as start --spec.`)
	AddStringFlag(cmdRun, doctl.ArgAgentConfigID, "", "", "ID of an existing Agent Config to run from, instead of --harness/--spec. Requires --name.")
	AddStringFlag(cmdRun, doctl.ArgAgentRepo, "", "", "GitHub repository to clone into the workspace (https://github.com/org/repo or org/repo). Only with --harness or --spec.")
	AddStringFlag(cmdRun, doctl.ArgAgentTriggerPrompt, "", "", "Initial prompt to send once the session is ready")
	AddStringFlag(cmdRun, doctl.ArgAgentName, "", "", "Session name (required with --config-id; otherwise auto-generated when omitted). Must be unique among active sessions.")
	AddBoolFlag(cmdRun, doctl.ArgAgentNoAttach, "", false, "Wait for readiness but do not attach")
	AddIntFlag(cmdRun, doctl.ArgAgentWaitTimeout, "", 300, "Maximum seconds to wait for the session to become ready (0 uses the default)")
	cmdRun.MarkFlagsMutuallyExclusive(doctl.ArgAgentHarness, doctl.ArgAgentSpec)
	cmdRun.MarkFlagsMutuallyExclusive(doctl.ArgAgentHarness, doctl.ArgAgentConfigID)
	cmdRun.MarkFlagsMutuallyExclusive(doctl.ArgAgentSpec, doctl.ArgAgentConfigID)
	cmdRun.Example = agentCLI + ` run --harness opencode --gh-repo https://github.com/katanemo/plano --prompt "Review the README"; ` + agentCLI + ` run --config-id cfg_abc123 --name my-session --prompt "Review the README"`

	cmdStartProxy := CmdBuilder(cmd, RunAgentsStartProxy, "start-proxy",
		"Bridge the Codex CLI to a hosted session",
		agentsStartProxyHelpMD,
		Writer, agentsNS()...)
	AddStringFlag(cmdStartProxy, doctl.ArgAgentProxyType, "", "codex", "Coding-agent protocol to bridge (v1: codex)")
	AddStringFlag(cmdStartProxy, doctl.ArgAgentProxySession, "", "", "Session ID or name to bridge to", requiredOpt())
	AddIntFlag(cmdStartProxy, doctl.ArgAgentProxyPort, "", 1144, "Local port to listen on")
	AddBoolFlag(cmdStartProxy, doctl.ArgAgentProxyReplay, "", false, "Replay the session's event history into the first thread on connect")
	cmdStartProxy.Example = agentCLI + ` start-proxy --type codex --session my-session --port 1144`

	cmdAttach := CmdBuilder(cmd, RunAgentsAttach, "attach <session>",
		"Attach to a session",
		agentsAttachHelpMD,
		Writer, agentsNS(aliasOpt("chat"))...)
	cmdAttach.Example = agentCLI + ` attach sess_abc123; ` + agentCLI + ` attach my-session-name`

	cmdList := CmdBuilder(cmd, RunAgentsList, "list",
		"List your sessions",
		agentsListHelpMD,
		Writer, agentsNS(aliasOpt("ls"),
			displayerType(&displayers.HostedAgentSession{}))...)
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of sessions to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	AddStringFlag(cmdList, doctl.ArgAgentStatus, "", "", "Filter by session status (e.g. SESSION_STATUS_READY, SESSION_STATUS_DESTROYED)")
	AddStringFlag(cmdList, doctl.ArgAgentName, "", "", "Filter by session name")
	AddStringFlag(cmdList, doctl.ArgAgentParentSessionID, "", "", "Filter to forked children of this parent session ID or name")
	cmdList.Example = agentCLI + ` list --page-size 10 --status SESSION_STATUS_READY; ` + agentCLI + ` list --name demo-agent; ` + agentCLI + ` list --parent-session-id sess_abc123`

	CmdBuilder(cmd, RunAgentsShow, "show <session>",
		"Show one session",
		agentsShowHelpMD,
		Writer, agentsNS(aliasOpt("get"),
			displayerType(&displayers.HostedAgentSession{}))...)

	CmdBuilder(cmd, RunAgentsLogs, "logs <session>",
		"Replay the event history for a session",
		agentsLogsHelpMD,
		Writer, agentsNS()...)

	CmdBuilder(cmd, RunAgentsApprove, "approve <session> <request-id> <approve|reject|defer>",
		"Resolve a pending HITL request out of band",
		agentsApproveHelpMD,
		Writer, agentsNS()...)

	cmdRemove := CmdBuilder(cmd, RunAgentsDestroy, "remove <session>",
		"Remove a session",
		agentsRemoveHelpMD,
		Writer, agentsNS(aliasOpt("destroy", "rm"))...)
	cmdRemove.Example = `doctl open-harness-runtime remove sess_abc123; doctl open-harness-runtime remove my-session; doctl open-harness-runtime destroy my-session`

	cmdPause := CmdBuilder(cmd, RunAgentsPause, "pause <session>",
		"Pause a session",
		agentsPauseHelpMD,
		Writer, agentsNS()...)
	cmdPause.Example = `doctl open-harness-runtime pause sess_abc123`

	cmdResume := CmdBuilder(cmd, RunAgentsResume, "resume <session>",
		"Resume a paused session",
		agentsResumeHelpMD,
		Writer, agentsNS()...)
	cmdResume.Example = `doctl open-harness-runtime resume sess_abc123`

	cmdUpload := CmdBuilder(cmd, RunAgentsUpload, "upload <session>",
		"Upload a file into a session workspace",
		agentsUploadHelpMD,
		Writer, agentsNS(
			displayerType(&displayers.HostedAgentWorkspaceUpload{}))...)
	AddStringFlag(cmdUpload, doctl.ArgAgentWorkspacePath, "", "", "Destination path inside the workspace root (/workspace)", requiredOpt())
	AddStringFlag(cmdUpload, doctl.ArgAgentLocalFile, "", "", "Path to the local file to upload", requiredOpt())
	AddBoolFlag(cmdUpload, doctl.ArgAgentArchive, "", false, "Treat the local file as an uncompressed tar archive to extract at the destination (not .tgz / .tar.gz)")
	cmdUpload.Example = `doctl open-harness-runtime upload sess_abc123 --local-file ./main.go --workspace-path src/main.go`

	cmdDownload := CmdBuilder(cmd, RunAgentsDownload, "download <session>",
		"Download a file from a session workspace",
		agentsDownloadHelpMD,
		Writer, agentsNS()...)
	AddStringFlag(cmdDownload, doctl.ArgAgentWorkspacePath, "", "", "Source path inside the workspace root (/workspace)", requiredOpt())
	AddStringFlag(cmdDownload, doctl.ArgAgentSaveTo, "", "", "Local file path to write the download to", requiredOpt())
	AddBoolFlag(cmdDownload, doctl.ArgAgentArchive, "", false, "Tar-stream the directory at the source path")
	cmdDownload.Example = `doctl open-harness-runtime download sess_abc123 --workspace-path src/main.go --save-to ./main.go`

	cmdExec := CmdBuilder(cmd, RunAgentsExec, "exec <session> -- <command> [args...]",
		"Run a command in a session's sandbox",
		agentsExecHelpMD,
		Writer, agentsNS(
			displayerType(&displayers.HostedAgentSandboxExec{}))...)
	AddStringFlag(cmdExec, doctl.ArgAgentExecWorkdir, "", "", "Absolute guest directory to run in (defaults to the workspace root)")
	AddIntFlag(cmdExec, doctl.ArgAgentExecTimeout, "", 0, "Maximum seconds the command may run (0 uses the server default)")
	cmdExec.Example = `doctl open-harness-runtime exec sess_abc123 -- ls -la; doctl open-harness-runtime exec my-session --workdir /workspace/src -- go test ./...`

	cmdAuth := CmdBuilder(cmd, RunAgentsAuth, "auth <provider>",
		"Connect an external provider (e.g. github) for agent git operations",
		agentsAuthHelpMD,
		Writer, agentsNS()...)
	AddBoolFlag(cmdAuth, doctl.ArgAgentAuthNoBrowser, "", false, "Print the authorization URL instead of opening a browser")
	AddBoolFlag(cmdAuth, doctl.ArgAgentAuthNoWait, "", false, "Print the authorization URL and exit without waiting for authorization to complete")
	cmdAuth.Example = `doctl open-harness-runtime auth github`

	cmdFork := CmdBuilder(cmd, RunAgentsFork, "fork <session>",
		"Fork a session into independent child sessions",
		agentsForkHelpMD,
		Writer, agentsNS(
			displayerType(&displayers.HostedAgentSession{}))...)
	AddStringFlag(cmdFork, doctl.ArgAgentFromCheckpoint, "", "", "Checkpoint ID to fork from (omit to checkpoint now first)")
	AddIntFlag(cmdFork, doctl.ArgAgentForkCount, "", 1, "Number of child sessions to create (1–4)")
	cmdFork.Example = `doctl open-harness-runtime fork sess_abc123 --from-checkpoint cp_9f2c1a4b --count 2`

	CmdBuilder(cmd, RunAgentsRollback, "rollback <session> <checkpoint-id>",
		"Roll a session back to a checkpoint in place",
		agentsRollbackHelpMD,
		Writer, agentsNS(
			displayerType(&displayers.HostedAgentSession{}))...)

	cmd.AddCommand(AgentCheckpoints())
	cmd.AddCommand(AgentTriggers())
	cmd.AddCommand(AgentConfigs())

	cmd.Command.SetHelpFunc(agentsStyledHelpFunc)

	return cmd
}

// --- runners ----------------------------------------------------------------

// RunAgentsStart creates a hosted agent session from --harness / --spec /
// --config-id, waits until ready (text mode), optionally sends --prompt, and
// prints a ready summary. It does not attach; use `run` for create-and-attach.
//
// For adapter codex-agentapi it first creates an OpenAI Agents session, resolves
// ${ENV_ID}, and passes openai_session_id to harness-api as a query parameter.
func RunAgentsStart(c *CmdConfig) error {
	harness, err := c.Doit.GetString(c.NS, doctl.ArgAgentHarness)
	if err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
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

	harness = strings.TrimSpace(harness)
	configID = strings.TrimSpace(configID)
	repo = strings.TrimSpace(repo)
	prompt = strings.TrimSpace(prompt)

	// Never call GetString(--spec) when another source is selected: a stale
	// required.agents.start.spec viper mark (or LiveConfig required check)
	// would fail --harness / --config-id even though --spec is optional.
	specPath := ""
	switch {
	case configID != "":
		if c.Doit.IsSet(doctl.ArgAgentSpec) || c.Doit.IsSet(doctl.ArgAgentHarness) {
			return errors.New("--harness, --spec, and --config-id are mutually exclusive; provide only one")
		}
	case harness != "":
		if c.Doit.IsSet(doctl.ArgAgentSpec) {
			return fmt.Errorf("--%s, --%s, and --%s are mutually exclusive; provide only one", doctl.ArgAgentHarness, doctl.ArgAgentSpec, doctl.ArgAgentConfigID)
		}
	default:
		var err error
		specPath, err = c.Doit.GetString(c.NS, doctl.ArgAgentSpec)
		if err != nil {
			return err
		}
		specPath = strings.TrimSpace(specPath)
	}

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

	if Output != "json" {
		stylingEnabled = detectStyling()
	}
	if repo != "" {
		if err := maybeOfferGitHubAuth(c); err != nil {
			return err
		}
	}

	prog := (*creationProgress)(nil)
	if Output != "json" {
		prog = newCreationProgress(c.Out)
		prog.header("Launching agent session")
	}

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
	case harness != "":
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
	default:
		raw, err = readManifestBytes(os.Stdin, specPath)
		if err != nil {
			return err
		}
		if manifestUsesLegacyEnvelope(raw) {
			warn("this manifest uses the deprecated apiVersion/kind/metadata/spec envelope format; " +
				"switch to the flat format (top-level `agent:` key, no envelope — see `" + agentCLI + " start --help`). " +
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
	}

	repoRef, _ := normalizeHarnessRepoRef(repo)
	return finishAgentsStartSession(c, sess, prog, runReadySummary{
		Session: sess,
		Harness: harness,
		Repo:    repoRef,
		Prompt:  prompt,
	}, prompt, raw)
}

// createSessionFromConfig creates a session from an Agent Config ID. Shared by
// `start --config-id` and `run --config-id`.
func createSessionFromConfig(c *CmdConfig, configID, name string, prog *creationProgress) (*do.HostedAgentSession, error) {
	if name == "" {
		return nil, errors.New("--name is required when starting from --config-id")
	}
	if err := validateHostedAgentIdentifier(name); err != nil {
		return nil, err
	}
	if prog != nil {
		prog.wait("Creating hosted session from config…")
	}
	sess, err := c.HostedAgents().CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
		Name:     name,
		ConfigID: configID,
	})
	if err != nil {
		if sessionLimitErr(err) {
			msg, _, _ := agentAPIError(err)
			return nil, fmt.Errorf("%s. Free a slot by removing one: run `%s list` to find a session ID, then `%s remove SESSION_ID`", strings.TrimRight(msg, "."), agentCLI, agentCLI)
		}
		return nil, err
	}
	if prog != nil {
		prog.ok(fmt.Sprintf("Session created · %s", displaySessionRef(sess)))
	}
	return sess, nil
}

// finishAgentsStartSession waits for readiness and prints a styled ready card
// in text mode; JSON mode returns the create response without waiting.
func finishAgentsStartSession(c *CmdConfig, sess *do.HostedAgentSession, prog *creationProgress, sum runReadySummary, prompt string, raw []byte) error {
	if sess == nil {
		return errors.New("session create returned no session")
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}, Single: true})
	}

	sessionID := sess.SessionID
	if sessionID == "" {
		return errors.New("session create returned no session id")
	}

	wait := runWaitTimeout
	if waitSec, err := c.Doit.GetInt(c.NS, doctl.ArgAgentWaitTimeout); err == nil && waitSec > 0 {
		wait = time.Duration(waitSec) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	if prog == nil {
		prog = newCreationProgress(c.Out)
	}
	sess, err := waitForSessionReady(ctx, c.HostedAgents(), sessionID, prog)
	if err != nil {
		return err
	}
	sum.Session = sess

	if prompt != "" && (raw == nil || !manifestIncludesPrompt(raw, prompt)) {
		if _, err := c.HostedAgents().SendInput(sessionID, &godo.HostedAgentSendInputRequest{Text: prompt}); err != nil {
			return fmt.Errorf("sending initial prompt: %w", err)
		}
	}

	printRunReadySummary(c.Out, sum)
	return nil
}

// RunAgentsStartProxy runs a local WebSocket facade that impersonates a
// coding-agent's own app-server protocol, bridging an unmodified agent CLI to
// a hosted session.
func RunAgentsStartProxy(c *CmdConfig) error {
	proxyType, err := c.Doit.GetString(c.NS, doctl.ArgAgentProxyType)
	if err != nil {
		return err
	}
	if proxyType != "codex" {
		return fmt.Errorf("unsupported --type %q; v1 supports only \"codex\"", proxyType)
	}

	sessionRef, err := c.Doit.GetString(c.NS, doctl.ArgAgentProxySession)
	if err != nil {
		return err
	}
	port, err := c.Doit.GetInt(c.NS, doctl.ArgAgentProxyPort)
	if err != nil {
		return err
	}
	replay, err := c.Doit.GetBool(c.NS, doctl.ArgAgentProxyReplay)
	if err != nil {
		return err
	}

	svc := c.HostedAgents()
	sessionID, err := resolveSessionRef(svc, sessionRef)
	if err != nil {
		return err
	}
	sess, err := svc.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("session %q not found: %w", sessionRef, err)
	}

	stylingEnabled = detectStyling()
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Proxy listening", colSuccess))
	body.WriteString(cardRow("Session", displaySessionRef(sess)))
	body.WriteString(cardRow("Listen", fmt.Sprintf("ws://127.0.0.1:%d", port)))
	body.WriteString(cardRow("Connect", fmt.Sprintf("codex --remote ws://127.0.0.1:%d", port)))
	renderAgentCard(c.Out, body.String())

	// SIGTERM alongside SIGINT: under a process manager or plain `kill` (not
	// `-9`), only handling os.Interrupt meant the graceful-shutdown path in
	// ServeListener never triggered — the process just died abruptly instead.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Fresh Facade per connection (see ServeListener): per-connection state
	// (notifier, in-flight turns, event loop, --replay gate) must not leak
	// across a disconnect/reconnect. A reconnecting client is a new TUI with
	// empty scrollback — --replay must run again for that connection.
	//
	// AgentKind gates raw passthrough: the facade forwards an event's native
	// source bytes only when the session's agent actually speaks this
	// facade's protocol — a codex TUI must never be handed raw OpenCode
	// frames just because they exist.
	return agentproxy.Serve(ctx, port, func() agentproxy.Facade {
		return &codex.Facade{
			SessionID: sessionID,
			Sessions:  svc,
			Replay:    replay,
			AgentKind: sess.AgentKind,
		}
	})
}

// readManifest returns the spec file as raw bytes with ${VAR} references
// resolved from the local environment (see expandManifestEnv). path "-" reads
// from stdin. Beyond env expansion, the only client-side validation is
// "non-empty after trim" so a stray `--spec /dev/null` fails fast instead of
// hitting the server.
func readManifest(stdin io.Reader, path string) ([]byte, error) {
	raw, err := readManifestBytes(stdin, path)
	if err != nil {
		return nil, err
	}
	return expandManifestEnv(raw)
}

// readManifestBytes loads the spec file without env expansion. Used by start
// so OpenAI orchestration can mint ENV_ID before ${...} resolution.
func readManifestBytes(stdin io.Reader, path string) ([]byte, error) {
	var src io.Reader
	if path == "-" && stdin != nil {
		src = stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("opening manifest: %s does not exist", path)
			}
			return nil, fmt.Errorf("opening manifest: %w", err)
		}
		defer f.Close()
		src = f
	}

	raw, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("manifest is empty")
	}
	return raw, nil
}

// manifestEnvRef matches ${VAR} env references and their $${VAR} escape form.
// Only the strict braced form expands: bare $VAR is left alone so shell
// snippets embedded in manifests (skills instructions, prompts) survive.
var manifestEnvRef = regexp.MustCompile(`\$?\$\{[A-Za-z_][A-Za-z0-9_]*\}`)

// expandManifestEnv resolves ${VAR} references in the manifest against the
// local environment before the manifest is sent to the server. $${VAR}
// escapes to a literal ${VAR}. Missing variables are collected interactively
// when stdin/stdout are a TTY; otherwise they error with a clear instruction.
func expandManifestEnv(manifest []byte) ([]byte, error) {
	return expandManifestEnvCollect(manifest, os.LookupEnv)
}

func envLookupWithOverlay(overlay map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		if overlay != nil {
			if v, ok := overlay[name]; ok {
				return v, true
			}
		}
		return os.LookupEnv(name)
	}
}

func expandManifestEnvLookup(manifest []byte, lookup func(string) (string, bool)) ([]byte, error) {
	var missing []string
	seen := map[string]bool{}
	out := manifestEnvRef.ReplaceAllFunc(manifest, func(m []byte) []byte {
		if bytes.HasPrefix(m, []byte("$$")) {
			return m[1:] // $${VAR} -> literal ${VAR}
		}
		name := string(m[2 : len(m)-1])
		val, ok := lookup(name)
		if !ok {
			if !seen[name] {
				seen[name] = true
				missing = append(missing, name)
			}
			return m
		}
		return []byte(val)
	})
	if len(missing) > 0 {
		return nil, fmt.Errorf("manifest references environment variable(s) not set locally: %s (escape a literal with $${...})", strings.Join(missing, ", "))
	}
	return out, nil
}

// manifestUsesLegacyEnvelope reports whether the manifest is written in the
// deprecated `agents.digitalocean.com/v1alpha1` envelope format. Detection
// matches the server's routing rule: a top-level `apiVersion` key selects the
// legacy parser; its absence selects the flat format. Unparsable YAML returns
// false — the server will produce the authoritative error.
func manifestUsesLegacyEnvelope(manifest []byte) bool {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		return false
	}
	_, ok := doc["apiVersion"]
	return ok
}

// injectManifestName sets the session name on the manifest: top-level `name`
// for flat manifests, `metadata.name` for legacy envelope ones. An empty name
// returns the manifest unchanged so the server can auto-generate one. The
// server still owns full manifest validation (including name syntax); this only
// wires the --name convenience flag into the YAML the server parses.
func injectManifestName(manifest []byte, name string) ([]byte, error) {
	if name == "" {
		return manifest, nil
	}

	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		return nil, fmt.Errorf("parsing manifest to apply --name: %w", err)
	}
	if doc == nil {
		doc = map[string]any{}
	}

	if _, legacy := doc["apiVersion"]; legacy {
		// yaml.v2 decodes nested mappings as map[any]any.
		meta, ok := doc["metadata"].(map[any]any)
		if !ok {
			meta = map[any]any{}
		}
		meta["name"] = name
		doc["metadata"] = meta
	} else {
		doc["name"] = name
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("applying --name to manifest: %w", err)
	}
	return out, nil
}

// RunAgentsList lists hosted agent sessions visible to the caller.
func RunAgentsList(c *CmdConfig) error {
	opt, err := agentsListOptions(c)
	if err != nil {
		return err
	}
	sessions, nextPageToken, err := c.HostedAgents().ListSessions(opt)
	if err != nil {
		return err
	}
	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentSession{Sessions: sessions}); err != nil {
			return err
		}
		if nextPageToken != "" {
			fmt.Fprintf(os.Stderr, "Next page token: %s\n", nextPageToken)
		}
		return nil
	}

	stylingEnabled = detectStyling()
	printSessionsList(c.Out, sessions)
	if nextPageToken != "" {
		fmt.Fprintf(c.Out, "\n%s %s\n", colorize("Next page token:", colMuted), nextPageToken)
	}
	return nil
}

func agentsListOptions(c *CmdConfig) (*godo.HostedAgentSessionListOptions, error) {
	pageSize, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPageSize)
	if err != nil {
		return nil, err
	}
	pageToken, err := c.Doit.GetString(c.NS, doctl.ArgAgentPageToken)
	if err != nil {
		return nil, err
	}
	status, err := c.Doit.GetString(c.NS, doctl.ArgAgentStatus)
	if err != nil {
		return nil, err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	if err != nil {
		return nil, err
	}
	parentRef, err := c.Doit.GetString(c.NS, doctl.ArgAgentParentSessionID)
	if err != nil {
		return nil, err
	}
	var parentSessionID string
	if parentRef != "" {
		parentSessionID, err = resolveSessionRef(c.HostedAgents(), parentRef)
		if err != nil {
			return nil, err
		}
	}
	if pageSize == 0 && pageToken == "" && status == "" && name == "" && parentSessionID == "" {
		return nil, nil
	}
	opt := &godo.HostedAgentSessionListOptions{}
	if pageSize > 0 {
		opt.PageSize = pageSize
	}
	if pageToken != "" {
		opt.PageToken = pageToken
	}
	if status != "" {
		opt.Status = godo.HostedAgentSessionStatus(status)
	}
	if name != "" {
		opt.Name = name
	}
	if parentSessionID != "" {
		opt.ParentSessionID = parentSessionID
	}
	return opt, nil
}

// sessionIDPrefix is a legacy/opaque session-ID prefix. Session IDs are
// canonically UUIDs, but we also accept this prefix defensively.
const sessionIDPrefix = "sess_"

// sessionUUIDRe matches the canonical hosted-agent session ID format (a UUID,
// e.g. 019f275e-96dc-7ea0-98bd-9ecf2a0834c3). It lets us tell an ID from a
// human-supplied session name without an API round-trip.
var sessionUUIDRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// looksLikeSessionID reports whether ref is already a session ID rather than a
// name, so it can be used directly without a name lookup.
func looksLikeSessionID(ref string) bool {
	return sessionUUIDRe.MatchString(ref) || strings.HasPrefix(ref, sessionIDPrefix)
}

// terminalSessionStatuses are the lifecycle states in which a session no longer
// owns its name (the name is freed for reuse). They're excluded from name
// resolution so a destroyed session can't shadow a live one that reused the name.
func isTerminalSessionStatus(s godo.HostedAgentSessionStatus) bool {
	switch s {
	case godo.HostedAgentSessionStatusDestroying,
		godo.HostedAgentSessionStatusDestroyed,
		godo.HostedAgentSessionStatusFailed:
		return true
	default:
		return false
	}
}

// humanSessionStatus renders a session status for display in plain-English
// messages, e.g. SESSION_STATUS_DESTROYED -> "destroyed". Table output
// (agents list/get) keeps the raw enum value; this is only for prose.
func humanSessionStatus(s godo.HostedAgentSessionStatus) string {
	return strings.ToLower(strings.TrimPrefix(string(s), "SESSION_STATUS_"))
}

// resolveSessionRef turns a user-supplied session reference into a session ID.
// References that already look like an ID (a UUID) are returned unchanged with
// no API call, so existing scripts keep working with no added latency.
func resolveSessionRef(svc do.HostedAgentsService, ref string) (string, error) {
	if ref == "" {
		return "", errors.New("a session ID or name is required")
	}
	if looksLikeSessionID(ref) {
		return ref, nil
	}

	sessions, _, err := svc.ListSessions(&godo.HostedAgentSessionListOptions{Name: ref})
	if err != nil {
		return "", fmt.Errorf("resolving session name %q: %w", ref, err)
	}

	// The name filter is case-insensitive server-side; mirror that here while
	// keeping only exact (not fuzzy/substring) matches, and drop terminal
	// sessions whose name has been freed for reuse.
	live := make([]do.HostedAgentSession, 0, len(sessions))
	for _, s := range sessions {
		if strings.EqualFold(s.Name, ref) && !isTerminalSessionStatus(s.Status) {
			live = append(live, s)
		}
	}

	switch len(live) {
	case 0:
		return "", fmt.Errorf("no agent session goes by the name %q; pass a session ID or run `doctl open-harness-runtime list` to see available sessions", ref)
	case 1:
		return live[0].SessionID, nil
	default:
		ids := make([]string, 0, len(live))
		for _, m := range live {
			ids = append(ids, m.SessionID)
		}
		return "", fmt.Errorf("many agent sessions go by the name %q, they have the following IDs: %s", ref, strings.Join(ids, ", "))
	}
}

// sessionIDArg validates that exactly one positional argument was supplied and
// resolves it (either a session ID or a session name) to a session ID.
func sessionIDArg(c *CmdConfig) (string, error) {
	if err := ensureOneArg(c); err != nil {
		return "", err
	}
	return resolveSessionRef(c.HostedAgents(), c.Args[0])
}

// RunAgentsShow prints one session.
func RunAgentsShow(c *CmdConfig) error {
	sessionID, err := sessionIDArg(c)
	if err != nil {
		return err
	}
	sess, err := c.HostedAgents().GetSession(sessionID)
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}, Single: true})
	}
	stylingEnabled = detectStyling()
	printSessionShowCard(c.Out, sess)
	return nil
}

// RunAgentsDestroy tears down a session (CLI: `doctl open-harness-runtime remove`,
// with aliases `destroy` and `rm`).
func RunAgentsDestroy(c *CmdConfig) error {
	sessionID, err := sessionIDArg(c)
	if err != nil {
		return err
	}
	if err := c.HostedAgents().DestroySession(sessionID); err != nil {
		return err
	}
	stylingEnabled = detectStyling()
	printAgentSuccess(c.Out, fmt.Sprintf("Session %s removed", sessionID))
	return nil
}

// RunAgentsPause pauses a session.
func RunAgentsPause(c *CmdConfig) error {
	sessionID, err := sessionIDArg(c)
	if err != nil {
		return err
	}
	if err := c.HostedAgents().PauseSession(sessionID); err != nil {
		return err
	}
	stylingEnabled = detectStyling()
	printAgentSuccess(c.Out, fmt.Sprintf("Session %s paused", sessionID))
	return nil
}

// RunAgentsResume resumes a paused session.
func RunAgentsResume(c *CmdConfig) error {
	sessionID, err := sessionIDArg(c)
	if err != nil {
		return err
	}
	if err := c.HostedAgents().ResumeSession(sessionID); err != nil {
		return err
	}
	stylingEnabled = detectStyling()
	printAgentSuccess(c.Out, fmt.Sprintf("Session %s resumed", sessionID))
	return nil
}

// agentProviderAuthStatusSuccess is the connect-flow status harness-api returns
// once the team's authorization handle is persisted (matches oauth_connect.go).
const agentProviderAuthStatusSuccess = "success"

// agentsAuthPollInterval is how often the connect poll endpoint is checked while
// waiting for the browser authorization to complete. A package var so tests can
// shorten it.
var agentsAuthPollInterval = 2 * time.Second

// RunAgentsAuth connects an external provider (e.g. github) for the caller's
// team via the harness-api connect flow. It starts (or resumes) the flow, opens
// the browser authorization URL, then polls until the handle is authorized.
func RunAgentsAuth(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	provider := strings.ToLower(strings.TrimSpace(c.Args[0]))
	if provider == "" {
		return errors.New("a provider is required, e.g. `doctl open-harness-runtime auth github`")
	}
	noBrowser, err := c.Doit.GetBool(c.NS, doctl.ArgAgentAuthNoBrowser)
	if err != nil {
		return err
	}
	noWait, err := c.Doit.GetBool(c.NS, doctl.ArgAgentAuthNoWait)
	if err != nil {
		return err
	}

	svc := c.HostedAgents()
	start, err := svc.StartProviderAuth(provider)
	if err != nil {
		return err
	}
	return completeProviderAuth(c, svc, provider, start, noBrowser, noWait)
}

// completeProviderAuth finishes a StartProviderAuth response: success, browser
// connect card, and optional poll until authorized.
func completeProviderAuth(c *CmdConfig, svc do.HostedAgentsService, provider string, start *godo.HostedAgentProviderAuthStart, noBrowser, noWait bool) error {
	if start == nil {
		return fmt.Errorf("empty provider auth response for %s", provider)
	}
	if strings.EqualFold(start.Status, agentProviderAuthStatusSuccess) {
		stylingEnabled = detectStyling()
		printAgentSuccess(c.Out, fmt.Sprintf("%s is already connected for your team", provider))
		return nil
	}
	if start.ConnectURL == "" {
		return fmt.Errorf("harness-api returned status %q with no authorization URL", start.Status)
	}

	stylingEnabled = detectStyling()
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Connect "+provider, colHighlight))
	body.WriteString(cardRow("URL", start.ConnectURL))
	if start.VerificationCode != "" {
		body.WriteString(cardRow("Code", boldColor(start.VerificationCode, colHighlight)))
	}
	renderAgentCard(c.Out, body.String())
	if !noBrowser {
		if berr := browser.OpenURL(start.ConnectURL); berr != nil {
			warn("could not open a browser automatically; open the URL above manually: %v", berr)
		}
	}

	if noWait || start.PollURL == "" {
		printAgentSuccess(c.Out, fmt.Sprintf("Re-run `doctl open-harness-runtime auth %s` after authorizing to confirm the connection", provider))
		return nil
	}

	// SIGTERM alongside SIGINT so the wait loop exits cleanly under a process
	// manager or plain `kill`, not only on Ctrl-C.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Fprintln(c.Out, "Waiting for authorization to complete... (Ctrl-C to stop)")
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("stopped waiting; re-run `doctl open-harness-runtime auth %s` to check the connection later", provider)
		case <-time.After(agentsAuthPollInterval):
		}

		poll, err := svc.PollProviderAuth(provider, start.PollURL)
		if err != nil {
			return err
		}
		if strings.EqualFold(poll.Status, agentProviderAuthStatusSuccess) {
			printAgentSuccess(c.Out, fmt.Sprintf("%s connected successfully", provider))
			return nil
		}
	}
}

// maxWorkspaceTransferBytes is the hard cap for workspace transfers (OHS contract).
// Tests may lower this to avoid allocating a 50 GiB file on disk.
var maxWorkspaceTransferBytes int64 = 50 << 30 // 50 GiB

// workspaceObjectHTTPClient performs direct PUT/GET against presigned object
// URLs returned by the staged transfer APIs. No client timeout so large
// transfers are not cut off by an arbitrary deadline.
var workspaceObjectHTTPClient = &http.Client{}

// workspaceTransferPollInterval is how often GetTransfer is polled after commit
// (upload) or create (download). Tests may shorten this.
var workspaceTransferPollInterval = time.Second

// RunAgentsUpload sends a local file (or tar archive) into a session's
// workspace sandbox via the workspace transfer API. The SHA-256 of the payload
// is computed up front and forwarded so the guest can verify what it received.
func RunAgentsUpload(c *CmdConfig) error {
	sessionID, err := sessionIDArg(c)
	if err != nil {
		return err
	}

	workspacePath, err := c.Doit.GetString(c.NS, doctl.ArgAgentWorkspacePath)
	if err != nil {
		return err
	}
	localFile, err := c.Doit.GetString(c.NS, doctl.ArgAgentLocalFile)
	if err != nil {
		return err
	}
	isArchive, err := c.Doit.GetBool(c.NS, doctl.ArgAgentArchive)
	if err != nil {
		return err
	}

	info, err := os.Stat(localFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("opening upload file: %s does not exist", localFile)
		}
		return fmt.Errorf("opening upload file: %w", err)
	}
	if info.Size() > maxWorkspaceTransferBytes {
		return fmt.Errorf("upload file exceeds the workspace transfer limit of 50 GiB (%d bytes)", maxWorkspaceTransferBytes)
	}

	f, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("opening upload file: %w", err)
	}
	defer f.Close()

	if isArchive {
		if err := validateArchiveUpload(f); err != nil {
			return err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("rewinding upload file: %w", err)
		}
	}

	// Hash the payload before sending so integrity can be verified end-to-end;
	// rewind afterward so the same bytes stream as the body / parts.
	sum, err := hashFile(f)
	if err != nil {
		return fmt.Errorf("hashing upload file: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewinding upload file: %w", err)
	}

	return workspaceTransferUpload(c, sessionID, workspacePath, f, info.Size(), isArchive, sum)
}

// validateArchiveUpload checks that --archive payloads are an uncompressed tar
// the guest can extract. Gzip-wrapped archives (.tgz / .tar.gz) and other
// formats are rejected up front so users get a clear error instead of a late
// transfer failure.
func validateArchiveUpload(r io.Reader) error {
	br := bufio.NewReader(r)
	magic, err := br.Peek(2)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("reading archive: %w", err)
	}
	if len(magic) == 0 {
		return fmt.Errorf("--archive expects an uncompressed .tar file; file is empty")
	}
	if len(magic) >= 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		return fmt.Errorf("--archive expects an uncompressed .tar file; got a gzip-compressed archive (.tgz / .tar.gz). Re-pack with `tar -cf archive.tar ...` (no -z) and retry")
	}
	if len(magic) >= 2 && magic[0] == 'P' && magic[1] == 'K' {
		return fmt.Errorf("--archive expects an uncompressed .tar file; got a zip archive. Re-pack with `tar -cf archive.tar ...` and retry")
	}

	tr := tar.NewReader(br)
	if _, err := tr.Next(); err != nil {
		if errors.Is(err, io.EOF) {
			// Two 512-byte zero blocks is a valid empty tar.
			return nil
		}
		return fmt.Errorf("--archive expects an uncompressed .tar file: %w", err)
	}
	return nil
}

// hashFile returns the hex-encoded SHA-256 of r, reading it to EOF.
func hashFile(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// workspacePartUploadURLBatchSize is how many part numbers doctl requests per
// CreatePartUploadURLs call. OHS supports batching; requesting one-by-one adds
// unnecessary round-trips for multi-GiB uploads.
const workspacePartUploadURLBatchSize = 32

// workspaceTransferUpload implements upload via /workspace/transfers:
// CreateTransfer → batched CreatePartUploadURLs + PUT → CommitTransfer → poll GetTransfer.
func workspaceTransferUpload(c *CmdConfig, sessionID, workspacePath string, f *os.File, size int64, isArchive bool, sha256hex string) error {
	svc := c.HostedAgents()
	create, err := svc.CreateWorkspaceTransfer(sessionID, &godo.HostedAgentWorkspaceTransferCreateRequest{
		Direction: godo.HostedAgentWorkspaceTransferDirectionUpload,
		Path:      workspacePath,
		IsArchive: isArchive,
		SizeBytes: size,
		SHA256:    sha256hex,
	})
	if err != nil {
		return err
	}
	if create.PartSize <= 0 {
		return fmt.Errorf("workspace transfer returned invalid part_size %d", create.PartSize)
	}

	transferID := create.TransferID
	cancel := func(reason string) {
		_, _ = svc.CancelWorkspaceTransfer(sessionID, transferID, &godo.HostedAgentWorkspaceTransferCancelRequest{Reason: reason})
	}

	totalParts := int((size + create.PartSize - 1) / create.PartSize)
	for startPart := 1; startPart <= totalParts; startPart += workspacePartUploadURLBatchSize {
		endPart := startPart + workspacePartUploadURLBatchSize - 1
		if endPart > totalParts {
			endPart = totalParts
		}
		partNumbers := make([]int, 0, endPart-startPart+1)
		for n := startPart; n <= endPart; n++ {
			partNumbers = append(partNumbers, n)
		}

		urlsByPart, err := workspacePartUploadURLs(svc, sessionID, transferID, partNumbers)
		if err != nil {
			cancel("failed to obtain part upload URLs")
			return fmt.Errorf("obtaining upload URLs for parts %d-%d: %w", startPart, endPart, err)
		}

		for partNumber := startPart; partNumber <= endPart; partNumber++ {
			offset := int64(partNumber-1) * create.PartSize
			partLen := create.PartSize
			if remaining := size - offset; remaining < partLen {
				partLen = remaining
			}
			section := io.NewSectionReader(f, offset, partLen)

			uploadURL := urlsByPart[partNumber]
			if uploadURL == "" {
				cancel("failed to obtain part upload URL")
				return fmt.Errorf("part upload URLs response missing part %d", partNumber)
			}
			if err := putWorkspaceObject(uploadURL, section, partLen); err != nil {
				// URL may have expired; refresh once and retry the same part.
				uploadURL, retryErr := workspacePartUploadURL(svc, sessionID, transferID, partNumber)
				if retryErr != nil {
					cancel("part upload failed")
					return fmt.Errorf("uploading part %d: %w (refresh URL: %v)", partNumber, err, retryErr)
				}
				if _, seekErr := section.Seek(0, io.SeekStart); seekErr != nil {
					cancel("part upload failed")
					return fmt.Errorf("rewinding part %d: %w", partNumber, seekErr)
				}
				if retryPut := putWorkspaceObject(uploadURL, section, partLen); retryPut != nil {
					cancel("part upload failed")
					return fmt.Errorf("uploading part %d: %w", partNumber, retryPut)
				}
			}
		}
	}

	if _, err := svc.CommitWorkspaceTransfer(sessionID, transferID, &godo.HostedAgentWorkspaceTransferCommitRequest{
		SHA256: sha256hex,
	}); err != nil {
		cancel("commit failed")
		return fmt.Errorf("committing workspace upload: %w", err)
	}

	xfer, err := pollWorkspaceTransfer(svc, sessionID, transferID)
	if err != nil {
		return err
	}

	written := xfer.BytesWritten
	if written == 0 {
		written = size
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentWorkspaceUpload{
			Uploads: []*godo.HostedAgentWorkspaceUploadResponse{{
				Path:         workspacePath,
				BytesWritten: written,
			}},
			Single: true,
		})
	}
	stylingEnabled = detectStyling()
	printWorkspaceUploadCard(c.Out, workspacePath, written)
	return nil
}

// workspacePartUploadURLs mints presigned PUT URLs for one or more 1-based parts.
func workspacePartUploadURLs(svc do.HostedAgentsService, sessionID, transferID string, partNumbers []int) (map[int]string, error) {
	if len(partNumbers) == 0 {
		return nil, fmt.Errorf("part_numbers must not be empty")
	}
	out, err := svc.CreateWorkspaceTransferPartUploadURLs(sessionID, transferID, &godo.HostedAgentWorkspaceTransferPartUploadURLsRequest{
		PartNumbers: partNumbers,
	})
	if err != nil {
		return nil, err
	}
	urlsByPart := make(map[int]string, len(partNumbers))
	for _, part := range out.PartURLs {
		if part.PartNumber > 0 && part.UploadURL != "" {
			urlsByPart[part.PartNumber] = part.UploadURL
		}
	}
	for _, n := range partNumbers {
		if urlsByPart[n] == "" {
			return nil, fmt.Errorf("part upload URLs response missing part %d", n)
		}
	}
	return urlsByPart, nil
}

// workspacePartUploadURL mints a presigned PUT URL for a single 1-based part.
func workspacePartUploadURL(svc do.HostedAgentsService, sessionID, transferID string, partNumber int) (string, error) {
	urls, err := workspacePartUploadURLs(svc, sessionID, transferID, []int{partNumber})
	if err != nil {
		return "", err
	}
	return urls[partNumber], nil
}

// putWorkspaceObject PUTs part bytes to a presigned object URL.
func putWorkspaceObject(uploadURL string, body io.Reader, contentLength int64) error {
	req, err := http.NewRequest(http.MethodPut, uploadURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = contentLength

	resp, err := workspaceObjectHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("presigned upload returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// pollWorkspaceTransfer waits until a staged transfer reaches a terminal status.
func pollWorkspaceTransfer(svc do.HostedAgentsService, sessionID, transferID string) (*godo.HostedAgentWorkspaceTransfer, error) {
	for {
		xfer, err := svc.GetWorkspaceTransfer(sessionID, transferID)
		if err != nil {
			return nil, err
		}
		switch xfer.Status {
		case godo.HostedAgentWorkspaceTransferStatusCompleted:
			return xfer, nil
		case godo.HostedAgentWorkspaceTransferStatusFailed:
			msg := xfer.ErrorMessage
			if msg == "" {
				msg = "unknown error"
			}
			return nil, fmt.Errorf("workspace transfer failed: %s", msg)
		default:
			time.Sleep(workspaceTransferPollInterval)
		}
	}
}

// RunAgentsDownload fetches a file (or tar archive) from a session workspace via
// the workspace transfer API: CreateTransfer → poll GetTransfer for download_url
// + sha256 → GET the presigned URL → verify digest. Bytes are written to a
// temporary file first and only moved into place once verification succeeds; a
// failed transfer is discarded.
func RunAgentsDownload(c *CmdConfig) error {
	sessionID, err := sessionIDArg(c)
	if err != nil {
		return err
	}

	workspacePath, err := c.Doit.GetString(c.NS, doctl.ArgAgentWorkspacePath)
	if err != nil {
		return err
	}
	saveTo, err := c.Doit.GetString(c.NS, doctl.ArgAgentSaveTo)
	if err != nil {
		return err
	}
	asArchive, err := c.Doit.GetBool(c.NS, doctl.ArgAgentArchive)
	if err != nil {
		return err
	}

	written, err := workspaceTransferDownload(c.HostedAgents(), sessionID, workspacePath, saveTo, asArchive)
	if err != nil {
		return err
	}

	stylingEnabled = detectStyling()
	printAgentSuccess(c.Out, fmt.Sprintf("Downloaded %d bytes to %s", written, saveTo))
	return nil
}

// workspaceTransferDownload implements download via /workspace/transfers:
// CreateTransfer → poll GetTransfer → GET download_url → verify sha256.
func workspaceTransferDownload(svc do.HostedAgentsService, sessionID, workspacePath, saveTo string, asArchive bool) (int64, error) {
	create, err := svc.CreateWorkspaceTransfer(sessionID, &godo.HostedAgentWorkspaceTransferCreateRequest{
		Direction: godo.HostedAgentWorkspaceTransferDirectionDownload,
		Path:      workspacePath,
		AsArchive: asArchive,
	})
	if err != nil {
		return 0, err
	}

	xfer, err := pollWorkspaceTransfer(svc, sessionID, create.TransferID)
	if err != nil {
		return 0, err
	}
	if xfer.DownloadURL == "" {
		return 0, fmt.Errorf("workspace transfer completed without a download_url")
	}

	return downloadWorkspaceObject(xfer.DownloadURL, saveTo, xfer.SHA256)
}

// downloadWorkspaceObject GETs a presigned URL into saveTo and optionally
// verifies the SHA-256 digest from GetTransfer (no DOWSSHA1 body footer).
func downloadWorkspaceObject(downloadURL, saveTo, wantSHA256 string) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := workspaceObjectHTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, resp.Body)
		return 0, fmt.Errorf("presigned download returned HTTP %d", resp.StatusCode)
	}

	dir := filepath.Dir(saveTo)
	tmp, err := os.CreateTemp(dir, ".doctl-download-*")
	if err != nil {
		return 0, fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}

	h := sha256.New()
	written, copyErr := io.Copy(tmp, io.TeeReader(resp.Body, h))
	if copyErr != nil {
		cleanup()
		return 0, fmt.Errorf("downloading workspace file: %w", copyErr)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return 0, fmt.Errorf("flushing download: %w", err)
	}

	if wantSHA256 != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if !strings.EqualFold(got, wantSHA256) {
			os.Remove(tmpName)
			return 0, fmt.Errorf("workspace download checksum mismatch: got %s, want %s", got, wantSHA256)
		}
	}

	if err := os.Rename(tmpName, saveTo); err != nil {
		os.Remove(tmpName)
		return 0, fmt.Errorf("saving download to %s: %w", saveTo, err)
	}
	return written, nil
}

// RunAgentsApprove resolves a pending HITL request out of band.
func RunAgentsApprove(c *CmdConfig) error {
	if len(c.Args) < 3 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if len(c.Args) > 3 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	requestID := c.Args[1]
	outcome, err := hitlOutcomeFor(c.Args[2])
	if err != nil {
		return err
	}
	if err := c.HostedAgents().ResolveHITL(sessionID, requestID, &godo.HostedAgentResolveHITLRequest{
		Outcome: outcome,
		Source:  godo.HostedAgentResolutionSourceOutOfBand,
	}); err != nil {
		return err
	}
	stylingEnabled = detectStyling()
	printAgentSuccess(c.Out, fmt.Sprintf("HITL request %s resolved as %s", requestID, outcome))
	return nil
}

func hitlOutcomeFor(s string) (godo.HostedAgentHITLOutcome, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "approve":
		return godo.HostedAgentHITLOutcomeApprove, nil
	case "reject":
		return godo.HostedAgentHITLOutcomeReject, nil
	case "defer":
		return godo.HostedAgentHITLOutcomeDefer, nil
	default:
		return "", fmt.Errorf("unknown outcome %q; expected approve, reject, or defer", s)
	}
}

// RunAgentsLogs replays the session's stored event history, then exits. The
// stream is finite: the server ends it after the last stored event, which is
// what terminates the loop below.
func RunAgentsLogs(c *CmdConfig) error {
	sessionID, err := sessionIDArg(c)
	if err != nil {
		return err
	}
	stylingEnabled = detectStyling()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := c.HostedAgents().StreamSession(ctx, sessionID, &godo.HostedAgentSessionStreamOptions{
		ReplayOnly: true,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	// TOKEN_CHUNK events stream token-by-token; buffer them so the whole
	// assistant message renders as markdown at once. Other event kinds are
	// discrete: flush any buffered message first, then print with a header.
	acc := &msgAccumulator{}
	for stream.Next() {
		ev := stream.Current()
		// Connection health, not session activity — never part of the history.
		if ev.Kind == godo.HostedAgentEventKindStreamState {
			continue
		}
		if ev.Kind == godo.HostedAgentEventKindTokenChunk {
			var p tokenChunkPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil {
				acc.add(p.Text)
			}
			continue
		}
		acc.flush(c.Out)
		fmt.Fprintf(c.Out, "[%s] %s\n", ev.At.Time.UTC().Format("2006-01-02T15:04:05Z"), ev.Kind)
		// Show the full hitl_id so request ids are copyable for out-of-band approve.
		renderEvent(c.Out, ev)
	}
	acc.flush(c.Out)
	return stream.Err()
}

// RunAgentsAttach opens the interactive TUI: one goroutine drains the SSE
// stream (with auto-reconnect), the main goroutine reads stdin.
// For AGENT_KIND_OPENAI_CODEX sessions it bridges to the OpenAI Agents session
// instead of DO's event stream.
func RunAgentsAttach(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	return runAgentsAttachSession(c, sessionID)
}

func runAgentsAttachSession(c *CmdConfig, sessionID string) error {
	svc := c.HostedAgents()

	stylingEnabled = detectStyling()

	sess, err := svc.GetSession(sessionID)
	if err != nil {
		return beautifyAgentError(err)
	}
	if isTerminalSessionStatus(sess.Status) {
		return fmt.Errorf("session %s cannot be attached (status: %s)", sessionID, humanSessionStatus(sess.Status))
	}

	if isOpenAISandboxSession(sess) {
		return runOpenAIAgentsAttach(c, sess)
	}

	pending := &pendingHITL{}
	cursor := &eventCursor{}
	state := newAttachState(c.Out, pending)
	state.sessionRef = displaySessionRef(sess)

	printAttachBanner(c.Out, sess, "")

	// All writes flow through the display so events don't clobber the user's
	// in-progress input once raw mode is on. Pass-through until raw=true.
	originalOut := c.Out
	c.Out = state.display
	defer func() { c.Out = originalOut }()

	thinking := newThinkingState(c.Out)
	defer thinking.stop()

	warmup := newWarmupState(c.Out, sess.CreatedAt.Time)
	warmup.enableStatusPoll(func() (*do.HostedAgentSession, error) {
		return svc.GetSession(sessionID)
	})
	defer warmup.clear()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go streamWithReconnect(ctx, svc, sessionID, c.Out, pending, cursor, thinking, warmup)

	return runAttach(c, svc, sessionID, os.Stdin, state, warmup)
}

func isOpenAISandboxSession(sess *do.HostedAgentSession) bool {
	if sess == nil || sess.HostedAgentSession == nil {
		return false
	}
	return sess.AgentKind == godo.HostedAgentKindOpenAICodex || strings.TrimSpace(sess.OpenAISessionID) != ""
}

func runOpenAIAgentsAttach(c *CmdConfig, sess *do.HostedAgentSession) error {
	apiKey, err := ensureEnvVar(openAIAPIKeyEnv)
	if err != nil {
		return fmt.Errorf("%s is required to attach to an OpenAI Agents sandbox session: %w", openAIAPIKeyEnv, err)
	}
	openaiSessionID := strings.TrimSpace(sess.OpenAISessionID)
	if openaiSessionID == "" {
		return fmt.Errorf("session %s is missing openai_session_id; cannot bridge to OpenAI", sess.SessionID)
	}

	pending := &pendingHITL{}
	state := newAttachState(c.Out, pending)
	state.sessionRef = displaySessionRef(sess)

	printAttachBanner(c.Out, sess, fmt.Sprintf("OpenAI · %s", openaiSessionID))

	originalOut := c.Out
	c.Out = state.display
	defer func() { c.Out = originalOut }()

	thinking := newThinkingState(c.Out)
	defer thinking.stop()

	warmup := newWarmupState(c.Out, sess.CreatedAt.Time)
	defer warmup.clear()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	client := newOpenAIAgentsAttachClient()
	renderer := &openAIAttachRenderer{out: c.Out, thinking: thinking, warmup: warmup}

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Stream(ctx, apiKey, openaiSessionID, renderer.handle)
	}()

	var inputErr error
	if term.IsTerminal(int(os.Stdin.Fd())) {
		inputErr = openaiAttachLoopTTY(c, ctx, client, apiKey, openaiSessionID, os.Stdin, state, thinking, warmup)
	} else {
		warmup.start()
		inputErr = openaiAttachLoop(c, ctx, client, apiKey, openaiSessionID, os.Stdin, state, thinking)
	}
	cancel()

	streamErr := <-errCh
	if inputErr != nil {
		return inputErr
	}
	if streamErr != nil && !errors.Is(streamErr, context.Canceled) && ctx.Err() == nil {
		return streamErr
	}
	return nil
}

// openAIAttachRenderer maps OpenAI Agents SSE events onto the same attach UX
// primitives used by the DO stream (spinner, tool lines, markdown replies).
type openAIAttachRenderer struct {
	out      io.Writer
	thinking *thinkingState
	warmup   *warmupState
	acc      msgAccumulator
	// sawOutputDelta tracks whether we already buffered streamed assistant text
	// so output_text.done does not duplicate it.
	sawOutputDelta bool
	// activeToolCmd is the in-flight command_execution command line.
	activeToolCmd string
}

func (r *openAIAttachRenderer) clearWarmup() {
	if r.warmup != nil {
		r.warmup.clear()
	}
}

func (r *openAIAttachRenderer) handle(evt map[string]any) {
	t, _ := evt["type"].(string)
	switch t {
	case "session.in_progress", "session.turn.in_progress", "session.turn.created":
		r.clearWarmup()
		if r.thinking != nil {
			r.thinking.start()
		}
	case "session.environment.connected":
		r.clearWarmup()
		if r.thinking != nil {
			r.thinking.stop()
		}
		fmt.Fprintf(r.out, "\n%s\n", colorize("● environment connected", colSuccess))
	case "session.environment.failed":
		r.clearWarmup()
		if r.thinking != nil {
			r.thinking.stop()
		}
		msg := "environment failed — sandbox executor did not connect to OpenAI (check guest codex exec-server / egress)"
		if errObj, ok := evt["error"].(map[string]any); ok {
			if m, ok := errObj["message"].(string); ok && m != "" {
				msg = m
			}
		}
		if env, ok := evt["environment"].(map[string]any); ok {
			if m, ok := env["error"].(string); ok && m != "" {
				msg = m
			}
		}
		fmt.Fprintf(r.out, "\n%s %s\n", colorize("✗", colError), colorize(msg, colError))
		fmt.Fprintln(r.out, colorize("Tip: remove and start a fresh session; wait for ● environment connected before sending work.", colMuted))
	case "session.turn.output_text.delta":
		r.clearWarmup()
		if r.thinking != nil {
			r.thinking.stop()
		}
		if d, ok := evt["delta"].(string); ok && d != "" {
			r.acc.add(d)
			r.sawOutputDelta = true
		}
	case "session.turn.output_text.done":
		r.clearWarmup()
		if r.thinking != nil {
			r.thinking.stop()
		}
		if !r.sawOutputDelta {
			if text, ok := evt["text"].(string); ok && text != "" {
				r.acc.add(text)
			}
		}
	case "session.turn.item.added":
		r.clearWarmup()
		r.handleItemAdded(evt)
	case "session.turn.item.done":
		r.clearWarmup()
		r.handleItemDone(evt)
	case "session.turn.completed":
		r.clearWarmup()
		if r.thinking != nil {
			r.thinking.stop()
		}
		r.acc.flush(r.out)
		r.sawOutputDelta = false
		r.activeToolCmd = ""
		summary := "run complete"
		if usage, ok := evt["usage"].(map[string]any); ok {
			inTok, _ := usage["input_tokens"].(float64)
			outTok, _ := usage["output_tokens"].(float64)
			if inTok > 0 || outTok > 0 {
				summary = fmt.Sprintf("run complete · %d in / %d out tokens", int(inTok), int(outTok))
			}
		}
		fmt.Fprintf(r.out, "\n%s %s\n", colorize("✓", colSuccess), colorize(summary, colMuted))
		fmt.Fprintln(r.out, colorize(runSeparator, colMuted))
	case "session.turn.failed":
		r.clearWarmup()
		if r.thinking != nil {
			r.thinking.stop()
		}
		r.acc.flush(r.out)
		r.sawOutputDelta = false
		r.activeToolCmd = ""
		msg := "run failed"
		if errObj, ok := evt["error"].(map[string]any); ok {
			if m, ok := errObj["message"].(string); ok && m != "" {
				msg = m
			}
		}
		fmt.Fprintf(r.out, "\n%s %s\n", colorize("✗", colError), colorize(msg, colError))
		fmt.Fprintln(r.out, colorize(runSeparator, colMuted))
	case "session.idle":
		r.clearWarmup()
		if r.thinking != nil {
			r.thinking.stop()
		}
		r.acc.flush(r.out)
		r.sawOutputDelta = false
		r.activeToolCmd = ""
	case "session.failed":
		r.clearWarmup()
		if r.thinking != nil {
			r.thinking.stop()
		}
		r.acc.flush(r.out)
		r.activeToolCmd = ""
		fmt.Fprintf(r.out, "\n%s %s\n", colorize("✗", colError), colorize("session failed", colError))
	default:
		// Suppress noisy protocol events (reasoning dumps, item echoes, etc.).
	}
}

func (r *openAIAttachRenderer) handleItemAdded(evt map[string]any) {
	item, _ := evt["item"].(map[string]any)
	if item == nil {
		return
	}
	switch item["type"] {
	case "command_execution":
		if r.thinking != nil {
			r.thinking.stop()
		}
		r.acc.flush(r.out)
		r.sawOutputDelta = false
		cmd, _ := item["command"].(string)
		r.activeToolCmd = cmd
		renderToolStart(r.out, cmd)
		if r.thinking != nil {
			r.thinking.start()
		}
	}
}

func (r *openAIAttachRenderer) handleItemDone(evt map[string]any) {
	item, _ := evt["item"].(map[string]any)
	if item == nil {
		return
	}
	switch item["type"] {
	case "command_execution":
		if r.thinking != nil {
			r.thinking.stop()
		}
		status, _ := item["status"].(string)
		mark := colorize("✓", colSuccess)
		summary := "done"
		if status != "" && status != "completed" {
			mark = colorize("✗", colError)
			summary = status
		}
		if out, ok := item["output"].(string); ok {
			out = strings.TrimSpace(out)
			if out != "" {
				// Keep the tool completion line short; full output stays in the sandbox.
				if lines := strings.Split(out, "\n"); len(lines) > 0 && lines[0] != "" {
					summary = truncateRunes(lines[0], 80)
				}
			}
		}
		fmt.Fprintf(r.out, "  %s %s\n", mark, colorize(summary, colMuted))
		r.activeToolCmd = ""
		if r.thinking != nil {
			r.thinking.start()
		}
	case "message":
		// Final assistant message content is preferred via output_text deltas.
		if role, _ := item["role"].(string); role == "assistant" && !r.sawOutputDelta {
			if content, ok := item["content"].([]any); ok {
				for _, c := range content {
					part, _ := c.(map[string]any)
					if part == nil {
						continue
					}
					if typ, _ := part["type"].(string); typ == "output_text" {
						if text, ok := part["text"].(string); ok && text != "" {
							r.acc.add(text)
						}
					}
				}
			}
		}
	}
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func openaiAttachLoop(c *CmdConfig, ctx context.Context, client openAIAgentsClient, apiKey, openaiSessionID string, in io.Reader, state *attachState, thinking *thinkingState) error {
	reader := bufio.NewReader(in)
	lines := startAttachLineReader(reader)
	var pendingLine *attachLineRead
	interactiveLineMode := false
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		interactiveLineMode = true
	}
	for {
		fmt.Fprint(c.Out, "\n", attachPrompt(state.pending))
		read := nextAttachLine(lines, &pendingLine)
		line, err := read.line, read.err
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(c.Out)
			return nil
		}
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			fmt.Fprintln(c.Out, colorize("slash commands are not available on OpenAI sandbox attach yet", colMuted))
			continue
		}
		if batched := collectAttachLines(line, lines, &pendingLine, true); len(batched) > 1 {
			line = strings.Join(batched, "\n")
			fmt.Fprintln(c.Out, colorize("Detected rapid multiline input; sending it as one message.", colMuted))
		}
		if n, ok := needsLargePasteConfirmation(line); ok && interactiveLineMode {
			decision, err := confirmLargePasteLineMode(c.Out, n, lines, &pendingLine)
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(c.Out)
				return nil
			}
			if err != nil {
				return err
			}
			switch decision {
			case largePasteSendTogether:
			case largePasteSendSeparately:
				for _, part := range splitSubmittedLines(line) {
					if strings.HasPrefix(part, "/") {
						fmt.Fprintln(c.Out, colorize("slash commands are not available on OpenAI sandbox attach yet", colMuted))
						continue
					}
					if thinking != nil {
						thinking.start()
					}
					if err := client.SendInput(ctx, apiKey, openaiSessionID, part); err != nil {
						if thinking != nil {
							thinking.stop()
						}
						fmt.Fprintf(c.Out, "send failed: %v\n", err)
					}
				}
				continue
			case largePasteDiscard:
				fmt.Fprintln(c.Out, colorize("large paste discarded", colMuted))
				continue
			}
		}
		if thinking != nil {
			thinking.start()
		}
		if err := client.SendInput(ctx, apiKey, openaiSessionID, line); err != nil {
			if thinking != nil {
				thinking.stop()
			}
			fmt.Fprintf(c.Out, "send failed: %v\n", err)
			continue
		}
	}
}

func openaiAttachLoopTTY(c *CmdConfig, ctx context.Context, client openAIAgentsClient, apiKey, openaiSessionID string, f *os.File, state *attachState, thinking *thinkingState, warmup *warmupState) error {
	fd := int(f.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		if warmup != nil {
			warmup.start()
		}
		return openaiAttachLoop(c, ctx, client, apiKey, openaiSessionID, f, state, thinking)
	}
	defer term.Restore(fd, oldState)
	setBracketedPasteMode(f, true)
	defer setBracketedPasteMode(f, false)

	state.display.setRaw(true)
	defer state.display.setRaw(false)
	state.display.redraw()
	if warmup != nil {
		warmup.start()
	}

	bytesCh := make(chan byte, 64)
	readErrCh := make(chan error, 1)
	go func() {
		var buf [1]byte
		for {
			n, err := f.Read(buf[:])
			if err != nil {
				readErrCh <- err
				return
			}
			if n == 1 {
				bytesCh <- buf[0]
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			printDetachNotice(c.Out, state.sessionRef)
			return nil
		case b := <-bytesCh:
			stop, err := handleOpenAIAttachByte(c, ctx, client, apiKey, openaiSessionID, b, state, thinking, warmup)
			if err != nil {
				return err
			}
			if stop {
				printDetachNotice(c.Out, state.sessionRef)
				return nil
			}
		case err := <-readErrCh:
			if errors.Is(err, io.EOF) {
				printDetachNotice(c.Out, state.sessionRef)
				return nil
			}
			return err
		}
	}
}

func startThinkingIfReady(thinking *thinkingState, warmup *warmupState) {
	if thinking != nil && (warmup == nil || !warmup.isActive()) {
		thinking.start()
	}
}

func printAttachSendAck(out io.Writer, warmup *warmupState) {
	if warmup != nil && warmup.isActive() {
		// The warm-up banner already shows the queued notice in-place.
		if warmup.isBannerVisible() {
			return
		}
		fmt.Fprintln(out, colorize("Message queued — will send when agent is ready", colMuted))
		return
	}
	fmt.Fprintln(out, colorize("… waiting for the agent", colMuted))
}

func echoAttachSubmitNewline(display *promptDisplay, warmup *warmupState) {
	if warmup != nil && warmup.isBannerVisible() {
		return
	}
	display.echo([]byte("\r\n"))
}

func handleOpenAIAttachByte(c *CmdConfig, ctx context.Context, client openAIAgentsClient, apiKey, openaiSessionID string, b byte, state *attachState, thinking *thinkingState, warmup *warmupState) (stop bool, err error) {
	if confirm := state.largePasteConfirmation(); confirm != nil {
		switch b {
		case 'y', 'Y':
			state.display.echo([]byte{b, '\r', '\n'})
			state.takeLargePasteConfirmation()
			startThinkingIfReady(thinking, warmup)
			if err := client.SendInput(ctx, apiKey, openaiSessionID, confirm.text); err != nil {
				if thinking != nil {
					thinking.stop()
				}
				fmt.Fprintf(c.Out, "send failed: %v\n", err)
			}
			state.display.redraw()
			return false, nil
		case 'n', 'N', 0x0d, 0x0a:
			state.display.echo([]byte("\r\n"))
			state.takeLargePasteConfirmation()
			for _, part := range splitSubmittedLines(confirm.text) {
				if strings.HasPrefix(part, "/") {
					fmt.Fprintln(c.Out, colorize("slash commands are not available on OpenAI sandbox attach yet", colMuted))
					continue
				}
				startThinkingIfReady(thinking, warmup)
				if err := client.SendInput(ctx, apiKey, openaiSessionID, part); err != nil {
					if thinking != nil {
						thinking.stop()
					}
					fmt.Fprintf(c.Out, "send failed: %v\n", err)
				}
			}
			state.display.redraw()
			return false, nil
		case 0x03, 0x04:
			state.display.echo([]byte("\r\n"))
			state.takeLargePasteConfirmation()
			fmt.Fprintln(c.Out, colorize("large paste discarded", colMuted))
			state.display.redraw()
			return false, nil
		default:
			return false, nil
		}
	}
	if state.pasting {
		handlePastedByte(b, state)
		warmup.noteQueued()
		if warmup.isBannerVisible() {
			state.display.redraw()
		}
		return false, nil
	}
	if handleAttachEscapeSequence(b, state) {
		return false, nil
	}

	switch b {
	case 0x0d, 0x0a:
		line := readSubmittedInput(state)
		if line != "" && warmup.inputAlreadyQueued() {
			fmt.Fprintln(c.Out, colorize("Message already queued — waiting for agent to start", colMuted))
			state.display.redraw()
			return false, nil
		}
		echoAttachSubmitNewline(state.display, warmup)
		if line != "" {
			if n, ok := needsLargePasteConfirmation(line); ok {
				state.setLargePasteConfirmation(line, n)
				state.display.redraw()
				return false, nil
			}
			if strings.HasPrefix(line, "/") {
				fmt.Fprintln(c.Out, colorize("slash commands are not available on OpenAI sandbox attach yet", colMuted))
			} else {
				startThinkingIfReady(thinking, warmup)
				if err := client.SendInput(ctx, apiKey, openaiSessionID, line); err != nil {
					if thinking != nil {
						thinking.stop()
					}
					fmt.Fprintf(c.Out, "send failed: %v\n", err)
				} else if warmup.isActive() {
					warmup.markInputQueued()
					printAttachSendAck(c.Out, warmup)
				}
			}
		}
		state.display.redraw()
		return false, nil
	case 0x7f, 0x08:
		state.mu.Lock()
		atEnd := state.cursor == len(state.lineBuf)
		if state.cursor > 0 {
			i := state.cursor - 1
			state.lineBuf = append(state.lineBuf[:i], state.lineBuf[i+1:]...)
			state.cursor = i
			state.mu.Unlock()
			if warmup.isBannerVisible() {
				state.display.redraw()
			} else if atEnd {
				state.display.echo([]byte("\b \b"))
			} else {
				state.display.redraw()
			}
		} else {
			state.mu.Unlock()
		}
		return false, nil
	case 0x03:
		state.display.echo([]byte("\r\n"))
		return true, nil
	case 0x04:
		state.mu.Lock()
		empty := len(state.lineBuf) == 0
		state.mu.Unlock()
		if empty {
			state.display.echo([]byte("\r\n"))
			return true, nil
		}
		return false, nil
	default:
		if b >= 0x20 && b < 0x7f {
			state.mu.Lock()
			atEnd := state.cursor == len(state.lineBuf)
			if atEnd {
				state.lineBuf = append(state.lineBuf, b)
			} else {
				state.lineBuf = append(state.lineBuf[:state.cursor], append([]byte{b}, state.lineBuf[state.cursor:]...)...)
			}
			state.cursor++
			state.mu.Unlock()
			warmup.noteQueued()
			if warmup.isBannerVisible() {
				state.display.redraw()
			} else if atEnd {
				state.display.echo([]byte{b})
			} else {
				state.display.redraw()
			}
		}
		return false, nil
	}
}

// runAttach dispatches to the raw-mode TTY loop or the legacy bufio line-mode
// loop based on whether stdin is an interactive terminal.
func runAttach(c *CmdConfig, svc do.HostedAgentsService, sessionID string, in io.Reader, state *attachState, warmup *warmupState) error {
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return attachLoopTTY(c, svc, sessionID, f, state, warmup)
	}
	if warmup != nil {
		warmup.start()
	}
	return attachLoop(c, svc, sessionID, in, state)
}

// eventCursor holds the EventID of the latest event rendered. The stream
// goroutine writes; reconnect attempts read. Empty string means "no events
// rendered yet" and the server will start from the live tail.
type eventCursor struct {
	mu sync.Mutex
	id string
}

func (c *eventCursor) set(id string) {
	if id == "" {
		return
	}
	c.mu.Lock()
	c.id = id
	c.mu.Unlock()
}

func (c *eventCursor) get() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.id
}

// Backoff schedule for auto reconnects between attempts. maxAutoReconnectAttempts
// bounds CONSECUTIVE failed reconnects, not the lifetime total: a connection
// that stays healthy resets the budget (see healthyStreamDuration).
const (
	maxAutoReconnectAttempts = 5
	initialReconnectBackoff  = 1 * time.Second
	maxReconnectBackoff      = 30 * time.Second
	msgReconnecting          = "Reconnecting..."
	msgReconnectFailed       = "Failed to reconnect to agent activity stream."
	msgSuperseded            = "This session was attached from another window on this device. Stopping the stream here."
)

// healthyStreamDuration is how long a stream must stay connected before a
// mid-stream drop is treated as a normal idle timeout (which resets the
// reconnect budget) rather than a failing connection. This lets a long, quiet
// attach survive an unbounded number of server idle drops while still giving up
// on a session that keeps dropping immediately. Overridable in tests.
var healthyStreamDuration = 30 * time.Second

// streamClock returns the current time; overridable in tests.
var streamClock = time.Now

// Warm-up notice (MARSOHS-796 / MARSOHS-972): freshly provisioned sessions can
// take up to ~90s before the in-guest agent is actually listening. Show a
// friendly status for young sessions so the CLI doesn't look frozen/silent
// while the backend retries, and make it explicit that typed input will be
// queued once the user starts entering a prompt. Overridable in tests.
const (
	msgAgentWarmup       = "Agent is warming up… please wait"
	msgAgentWarmupQueued = "Input queued until agent is ready"
)

var (
	warmupDuration       = 60 * time.Second
	warmupEligibleAge    = 2 * time.Minute
	warmupClock          = time.Now
	warmupStatusInterval = 2 * time.Second
)

// thinkingState shows a spinner between RunStarted and the first real
// output. Animates above the prompt when out is a *promptDisplay; falls back
// to a one-shot "(thinking...)" print otherwise (pipes, line-mode).
type thinkingState struct {
	mu     sync.Mutex
	out    io.Writer
	active bool
	cancel context.CancelFunc
	done   chan struct{}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func newThinkingState(out io.Writer) *thinkingState {
	return &thinkingState{out: out}
}

func (s *thinkingState) start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true

	display, ok := s.out.(*promptDisplay)
	if !ok {
		fmt.Fprintln(s.out, "(thinking...)")
		s.mu.Unlock()
		return
	}
	// Reserve the spinner line and draw its first frame atomically so no
	// redraw or token can slip in between and shift the line the animator
	// (and stop) expect one row above the prompt.
	display.spinnerInit(spinnerFrames[0], "thinking...")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.mu.Unlock()
	go s.animate(ctx, display, done)
}

func (s *thinkingState) stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	// Wait for the animator to exit before touching the line, so no stray
	// frame lands after we replace it.
	if done != nil {
		<-done
	}
	if display, ok := s.out.(*promptDisplay); ok {
		display.spinnerStop()
	}
}

func (s *thinkingState) animate(ctx context.Context, d *promptDisplay, done chan struct{}) {
	defer close(done)
	t := time.NewTicker(80 * time.Millisecond)
	defer t.Stop()
	ix := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ix = (ix + 1) % len(spinnerFrames)
			d.spinnerFrame(spinnerFrames[ix], "thinking...")
		}
	}
}

// warmupState shows "Agent is warming up…" on attach for sessions still in
// their initial boot window. Clears on the first meaningful agent event or
// after warmupDuration, whichever comes first. No-ops for older sessions.
// While active it can surface backend progress (session status / boot events)
// in the banner label so a long wait doesn't look frozen.
type warmupState struct {
	mu            sync.Mutex
	out           io.Writer
	eligible      bool
	active        bool
	dismissed     bool
	queued        bool
	inputQueued   bool
	phase         string
	getSession    func() (*do.HostedAgentSession, error)
	timeoutCancel context.CancelFunc
	animCancel    context.CancelFunc
	animDone      chan struct{}
	pollCancel    context.CancelFunc
}

func newWarmupState(out io.Writer, createdAt time.Time) *warmupState {
	w := &warmupState{out: out}
	if !createdAt.IsZero() && warmupClock().Sub(createdAt) <= warmupEligibleAge {
		w.eligible = true
	}
	return w
}

// enableStatusPoll periodically refreshes the warm-up banner from GetSession
// so developers see sandbox/agent state while waiting.
func (w *warmupState) enableStatusPoll(getSession func() (*do.HostedAgentSession, error)) {
	if w == nil {
		return
	}
	w.mu.Lock()
	w.getSession = getSession
	w.mu.Unlock()
}

func (w *warmupState) isActive() bool {
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.active && !w.dismissed
}

func (w *warmupState) inputAlreadyQueued() bool {
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.inputQueued
}

func (w *warmupState) markInputQueued() {
	if w == nil {
		return
	}
	w.mu.Lock()
	w.inputQueued = true
	w.mu.Unlock()
}

func (w *warmupState) isBannerVisible() bool {
	if w == nil || !w.isActive() {
		return false
	}
	display, ok := w.out.(*promptDisplay)
	if !ok {
		return false
	}
	return display.warmupBannerActive()
}

// start shows the warm-up notice immediately. Prefer the erasable prompt
// spinner when raw mode is on so clear() can remove it; fall back to a plain
// line for pipes/tests. Safe to call more than once; no-ops when the session
// is outside the warm-up window or already cleared.
func (w *warmupState) start() {
	if w == nil {
		return
	}
	w.mu.Lock()
	if !w.eligible || w.active || w.dismissed {
		w.mu.Unlock()
		return
	}
	w.active = true
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), warmupDuration)
	w.timeoutCancel = timeoutCancel
	getSession := w.getSession

	display, ok := w.out.(*promptDisplay)
	if ok {
		display.warmupInit(spinnerFrames[0], msgAgentWarmup)
		animCtx, animCancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		w.animCancel = animCancel
		w.animDone = done
		w.mu.Unlock()
		go w.animate(animCtx, display, done)
	} else {
		w.mu.Unlock()
		fmt.Fprintf(w.out, "%s\n", colorize("⟳ "+msgAgentWarmup, colMuted))
	}
	go w.waitTimeout(timeoutCtx)
	if getSession != nil {
		pollCtx, pollCancel := context.WithCancel(context.Background())
		w.mu.Lock()
		w.pollCancel = pollCancel
		w.mu.Unlock()
		go w.pollStatus(pollCtx, getSession)
	}
}

// setPhase updates the warm-up banner with a short backend progress hint
// on its own grey line under the spinner.
func (w *warmupState) setPhase(phase string) {
	if w == nil {
		return
	}
	phase = strings.TrimSpace(phase)
	if phase == "" {
		return
	}
	w.mu.Lock()
	if !w.active || w.dismissed || w.phase == phase {
		w.mu.Unlock()
		return
	}
	w.phase = phase
	display, ok := w.out.(*promptDisplay)
	w.mu.Unlock()
	if ok {
		display.warmupSetPhase(phase)
		return
	}
	fmt.Fprintf(w.out, "%s\n", colorize(phase, colMuted))
}

// noteBackendEvent surfaces boot/lifecycle stream events on the warm-up banner
// instead of printing competing lines. Returns true when the event was
// consumed as warm-up progress (caller should skip normal render).
func (w *warmupState) noteBackendEvent(ev godo.HostedAgentEvent) bool {
	if w == nil || !w.isActive() {
		return false
	}
	phase := backendPhaseFromEvent(ev)
	if phase == "" {
		return false
	}
	w.setPhase(phase)
	return true
}

func backendPhaseFromEvent(ev godo.HostedAgentEvent) string {
	switch ev.Kind {
	case godo.HostedAgentEventKindSessionUpdated:
		if phase := sessionUpdatedPhase(ev.Payload); phase != "" {
			return phase
		}
		return "syncing session"
	case godo.HostedAgentEventKindRunSandboxAllocated:
		return "sandbox allocated"
	case godo.HostedAgentEventKindRunSandboxReleased:
		return "releasing sandbox"
	case godo.HostedAgentEventKindRunLog:
		if msg := runLogPhase(ev.Payload); msg != "" {
			return msg
		}
		return "runtime log"
	case godo.HostedAgentEventKindStreamState:
		var st godo.HostedAgentStreamState
		if err := json.Unmarshal(ev.Payload, &st); err == nil && st.State == godo.HostedAgentStreamStateCatchingUp {
			return "syncing event stream"
		}
	}
	return ""
}

func sessionUpdatedPhase(payload json.RawMessage) string {
	var p struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Phase   string `json:"phase"`
		State   string `json:"state"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return ""
	}
	if msg := strings.TrimSpace(p.Message); msg != "" {
		return truncateWarmupPhase(msg)
	}
	if phase := strings.TrimSpace(p.Phase); phase != "" {
		return truncateWarmupPhase(phase)
	}
	if state := strings.TrimSpace(p.State); state != "" {
		return truncateWarmupPhase(state)
	}
	if status := strings.TrimSpace(p.Status); status != "" {
		return backendPhaseFromStatus(godo.HostedAgentSessionStatus(status))
	}
	return ""
}

func runLogPhase(payload json.RawMessage) string {
	var p struct {
		Message string `json:"message"`
		Text    string `json:"text"`
		Line    string `json:"line"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return ""
	}
	for _, s := range []string{p.Message, p.Text, p.Line} {
		if msg := strings.TrimSpace(s); msg != "" {
			return truncateWarmupPhase(msg)
		}
	}
	return ""
}

func backendPhaseFromStatus(status godo.HostedAgentSessionStatus) string {
	switch status {
	case godo.HostedAgentSessionStatusProvisioning:
		return "provisioning sandbox"
	case godo.HostedAgentSessionStatusReady, godo.HostedAgentSessionStatusDetached:
		return "sandbox ready · starting agent"
	case godo.HostedAgentSessionStatusPaused:
		return "session paused"
	case godo.HostedAgentSessionStatusFailed:
		return "session failed"
	case godo.HostedAgentSessionStatusDestroying, godo.HostedAgentSessionStatusDestroyed:
		return "session tearing down"
	default:
		if s := humanSessionStatus(status); s != "" && s != "unspecified" {
			return s
		}
		return ""
	}
}

func truncateWarmupPhase(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 56 {
		return s[:53] + "…"
	}
	return s
}

func (w *warmupState) pollStatus(ctx context.Context, getSession func() (*do.HostedAgentSession, error)) {
	ticker := time.NewTicker(warmupStatusInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !w.isActive() {
				return
			}
			sess, err := getSession()
			if err != nil || sess == nil {
				continue
			}
			if phase := backendPhaseFromStatus(sess.Status); phase != "" {
				w.setPhase(phase)
			}
		}
	}
}

func (w *warmupState) animate(ctx context.Context, d *promptDisplay, done chan struct{}) {
	defer close(done)
	t := time.NewTicker(80 * time.Millisecond)
	defer t.Stop()
	ix := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ix = (ix + 1) % len(spinnerFrames)
			d.warmupSetFrame(spinnerFrames[ix])
		}
	}
}

// noteQueued reveals the queued-input notice once the user starts typing so
// attach makes it obvious their prompt is buffered until the agent is ready.
func (w *warmupState) noteQueued() {
	if w == nil {
		return
	}
	w.mu.Lock()
	if !w.active || w.dismissed {
		w.mu.Unlock()
		return
	}
	already := w.queued
	w.queued = true
	display, ok := w.out.(*promptDisplay)
	w.mu.Unlock()
	if ok {
		display.warmupSetQueued(msgAgentWarmupQueued)
		return
	}
	if !already {
		fmt.Fprintf(w.out, "%s\n", colorize(msgAgentWarmupQueued, colMuted))
	}
}

func (w *warmupState) waitTimeout(ctx context.Context) {
	<-ctx.Done()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		w.clear()
	}
}

// clear dismisses the warm-up notice. Idempotent. When shown via the prompt
// banner, the spinner and queued rows are erased so they truly go away once
// the session is ready.
func (w *warmupState) clear() {
	if w == nil {
		return
	}
	w.mu.Lock()
	w.dismissed = true
	if !w.active {
		w.mu.Unlock()
		return
	}
	w.active = false
	w.inputQueued = false
	w.queued = false
	timeoutCancel := w.timeoutCancel
	animCancel := w.animCancel
	animDone := w.animDone
	pollCancel := w.pollCancel
	w.timeoutCancel = nil
	w.animCancel = nil
	w.animDone = nil
	w.pollCancel = nil
	w.mu.Unlock()

	if timeoutCancel != nil {
		timeoutCancel()
	}
	if pollCancel != nil {
		pollCancel()
	}
	if animCancel != nil {
		animCancel()
	}
	if animDone != nil {
		<-animDone
	}
	if display, ok := w.out.(*promptDisplay); ok {
		display.warmupStop()
	}
}

// streamWithReconnect drains the SSE iterator and reconnects on transient
// errors. It shows Reconnecting... before each retry and gives up (printing
// msgReconnectFailed) only after maxAutoReconnectAttempts CONSECUTIVE failures.
// A connection that stays up for at least healthyStreamDuration before dropping
// is treated as a normal server idle timeout and resets the failure budget, so
// a long, quiet attach can recover from an unbounded number of idle drops.
func streamWithReconnect(
	ctx context.Context,
	svc do.HostedAgentsService,
	sessionID string,
	out io.Writer,
	pending *pendingHITL,
	cursor *eventCursor,
	thinking *thinkingState,
	warmup *warmupState,
) {
	dedup := &tokenDeduper{}
	backoff := initialReconnectBackoff
	failures := 0
	reconnecting := false

	for {
		if ctx.Err() != nil {
			return
		}

		// Show the reconnect notice on every attempt after the first, whether
		// this is a retry after a failed connect or a fresh reconnect after a
		// healthy idle drop. The failure budget below governs when we give up;
		// it must not gate this message, since a healthy drop resets the budget
		// to zero and would otherwise silently suppress the notice.
		if reconnecting {
			fmt.Fprintf(out, "\n%s\n", msgReconnecting)
		}
		reconnecting = true

		opt := &godo.HostedAgentSessionStreamOptions{ReplayFrom: cursor.get()}
		stream, err := svc.StreamSession(ctx, sessionID, opt)
		if err != nil {
			thinking.stop()
			if msg, terminal := classifyStreamError(err); terminal {
				fmt.Fprintln(out, msg)
				return
			}
			failures++
			if failures >= maxAutoReconnectAttempts {
				fmt.Fprintf(out, "\n%s\n", msgReconnectFailed)
				return
			}
			if !reconnectSleepFn(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		connectedAt := streamClock()
		superseded := drainStream(stream, out, pending, cursor, thinking, warmup, dedup)
		streamErr := stream.Err()
		stream.Close()

		if ctx.Err() != nil {
			return
		}

		thinking.stop()

		// Another connection from this device owns the session now. Reconnecting
		// would take it back and start an eviction loop between the two.
		if superseded {
			return
		}

		// An interactive attach ends only when the user detaches (ctx cancel,
		// handled above) or the session is gone (a terminal error). Any other
		// stream end is an unexpected drop we reconnect from — including a clean
		// EOF, which is how a server idle-timeout close looks (err == nil). A
		// genuinely finished session surfaces as a terminal error (404) on the
		// next connect, which stops the loop below.
		if streamErr != nil {
			if msg, terminal := classifyStreamError(streamErr); terminal {
				fmt.Fprintln(out, msg)
				return
			}
		}

		// A drop after a healthy, long-lived connection is a normal idle
		// timeout, not a failing session: reset the budget and backoff so the
		// attach keeps recovering. Only rapid, back-to-back drops accumulate
		// toward the give-up limit.
		if streamClock().Sub(connectedAt) >= healthyStreamDuration {
			failures = 0
			backoff = initialReconnectBackoff
		} else {
			failures++
		}
		if failures >= maxAutoReconnectAttempts {
			fmt.Fprintf(out, "\n%s\n", msgReconnectFailed)
			return
		}
		if !reconnectSleepFn(ctx, backoff) {
			return
		}
		backoff = nextBackoff(backoff)
	}
}

// reconnectSleepFn is the backoff wait between reconnect attempts. Tests may
// replace it to avoid real-time delays.
var reconnectSleepFn = sleepCtx

// classifyStreamError returns (user-facing message, terminal). Terminal
// errors stop the reconnect loop (auth, missing session, V0 single-connection
// rejection); status codes follow harness-api's apierr convention.
func classifyStreamError(err error) (string, bool) {
	msg, status, ok := agentAPIError(err)
	if !ok {
		return "", false
	}
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict:
		pretty := beautifyAgentError(err)
		var ape *agentPrettyError
		if errors.As(pretty, &ape) {
			if status == http.StatusConflict && ape.title == "Conflict" {
				// V0 single-connection rejection; prefer a clearer title.
				ape.title = "Session already attached elsewhere"
				ape.tips = []string{"Detach on the other device, then re-run doctl open-harness-runtime attach"}
			}
			if status == http.StatusNotFound {
				ape.title = "Session not found"
			}
			if reason := strings.TrimSpace(msg); reason != "" {
				ape.reason = reason
			}
			return "\n" + ape.DisplayError(), true
		}
		return "\n" + strings.TrimSpace(err.Error()), true
	}
	return "", false
}

// isRunTerminalErr reports the harness-api 409 returned when input is sent to
// a session whose run is already terminal; recovery is a fresh `agents start`.
func isRunTerminalErr(err error) bool {
	msg, status, ok := agentAPIError(err)
	return ok && status == http.StatusConflict &&
		strings.Contains(strings.ToLower(msg), "run is terminal")
}

// agentAPIError unwraps err to a *godo.ErrorResponse and returns its effective
// message and HTTP status. The reason lives in NestedError.Message when the
// top-level Message is empty (harness-api's {"error":{...}} envelope).
func agentAPIError(err error) (message string, status int, ok bool) {
	var er *godo.ErrorResponse
	if !errors.As(err, &er) || er.Response == nil {
		return "", 0, false
	}
	msg := er.Message
	if msg == "" && er.NestedError != nil {
		msg = er.NestedError.Message
	}
	return msg, er.Response.StatusCode, true
}

// sessionLimitErr reports the 409 from `agents start` when the team has hit its
// active-session cap (e.g. "team is at the limit of 4 active sessions").
func sessionLimitErr(err error) bool {
	msg, status, ok := agentAPIError(err)
	if !ok || status != http.StatusConflict {
		return false
	}
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "limit of") && strings.Contains(lower, "active sessions")
}

// drainStream consumes events until the iterator stops, rendering each one
// and updating the HITL pending map, the reconnect cursor, and the spinner.
//
// It returns superseded when the server reported that another connection from
// this device took over the session. That is the one stream end the caller must
// not reconnect from: re-attaching would supersede the connection that just
// superseded us, and the two would evict each other indefinitely.
func drainStream(stream *godo.HostedAgentSessionStream, out io.Writer, pending *pendingHITL, cursor *eventCursor, thinking *thinkingState, warmup *warmupState, dedup *tokenDeduper) (superseded bool) {
	// acc buffers the current assistant turn's tokens; the thinking spinner
	// stays up for the whole turn and the buffered text is rendered as markdown
	// when the run finishes or a discrete event interrupts it.
	acc := &msgAccumulator{}
	// awaiting is a FIFO queue of HITL approvals whose lines haven't printed
	// yet: we hold each until a paired tool-call reveals the command being
	// approved, so the approval shows "● Approval required · <command>" on one
	// line. A queue (not a single slot) is required because the adapter can
	// enqueue several approvals before the matching tool calls arrive; pairing
	// them oldest-first keeps each line labelled instead of blanking the first.
	// The summary captured from the HITL payload is the fallback label when no
	// paired tool call arrives to name the command.
	var awaiting []awaitingApproval
	for stream.Next() {
		ev := stream.Current()

		// stream.state reports the health of the connection, not session
		// activity, so it never renders and never moves the cursor.
		if ev.Kind == godo.HostedAgentEventKindStreamState {
			var st godo.HostedAgentStreamState
			if err := json.Unmarshal(ev.Payload, &st); err == nil && st.State == godo.HostedAgentStreamStateSuperseded {
				thinking.stop()
				acc.flush(out)
				flushAwaitingApproval(out, &awaiting)
				fmt.Fprintf(out, "\n%s\n", msgSuperseded)
				return true
			}
			_ = warmup.noteBackendEvent(ev)
			continue
		}

		switch ev.Kind {
		case godo.HostedAgentEventKindHITLRequested:
			var p hitlRequestedPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil {
				pending.set(p.id(), p.actionLabel())
			}
		case godo.HostedAgentEventKindHITLResolved:
			var p hitlResolvedPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil {
				pending.clearIf(p.HitlID)
			}
		}

		switch ev.Kind {
		case godo.HostedAgentEventKindRunStarted:
			warmup.clear()
			thinking.stop()
			acc.flush(out)
			flushAwaitingApproval(out, &awaiting)
			dedup.reset()
			renderEvent(out, ev)
			thinking.start()
		case godo.HostedAgentEventKindTokenChunk:
			warmup.clear()
			var p tokenChunkPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil && dedup.allow(p.Text) {
				acc.add(p.Text)
			}
		case godo.HostedAgentEventKindHITLRequested:
			warmup.clear()
			thinking.stop()
			acc.flush(out)
			dedup.reset()
			var p hitlRequestedPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil {
				// Queue until a paired tool_call_started names the command; on
				// reattach the server re-injects only this frame, so summary
				// must come from the HITL payload itself (details.command).
				awaiting = append(awaiting, awaitingApproval{id: p.id(), summary: p.commandSummary()})
			}
		case godo.HostedAgentEventKindToolCallStarted:
			warmup.clear()
			thinking.stop()
			acc.flush(out)
			dedup.reset()
			var p toolCallStartedPayload
			_ = json.Unmarshal(ev.Payload, &p)
			cmd := p.commandLine()
			if len(awaiting) > 0 {
				// Pair with the oldest waiting approval. Prefer the tool call's
				// command; fall back to the label the HITL payload carried so
				// the line is never blank.
				a := awaiting[0]
				awaiting = awaiting[1:]
				if cmd == "" {
					cmd = a.summary
				}
				renderApprovalLine(out, a.id, cmd)
			} else {
				renderToolStart(out, cmd)
			}
		case godo.HostedAgentEventKindSessionUpdated,
			godo.HostedAgentEventKindRunSandboxAllocated,
			godo.HostedAgentEventKindRunSandboxReleased,
			godo.HostedAgentEventKindRunLog:
			// Boot/lifecycle noise during warm-up — fold into the banner instead
			// of printing competing lines or dismissing the notice.
			if warmup.noteBackendEvent(ev) {
				continue
			}
			thinking.stop()
			acc.flush(out)
			flushAwaitingApproval(out, &awaiting)
			dedup.reset()
			renderEvent(out, ev)
		default:
			warmup.clear()
			thinking.stop()
			acc.flush(out)
			flushAwaitingApproval(out, &awaiting)
			dedup.reset()
			renderEvent(out, ev)
		}

		// A finished run means the server has cancelled any still-pending tool
		// calls, so flush the local queue rather than leave stale entries.
		if ev.Kind == godo.HostedAgentEventKindRunCompleted || ev.Kind == godo.HostedAgentEventKindRunFailed {
			if n := pending.reset(); n > 0 {
				fmt.Fprintf(out, "(%d pending approval(s) cancelled — run ended)\n", n)
			}
		}

		cursor.set(ev.EventID)
	}
	// Stream ended without a terminal run event (e.g. transient disconnect):
	// render whatever assistant text we have so it isn't lost.
	thinking.stop()
	acc.flush(out)
	flushAwaitingApproval(out, &awaiting)
	return false
}

// awaitingApproval is a HITL approval whose line is deferred until a paired
// tool call names the command. summary is the fallback label extracted from the
// HITL payload itself, used when no paired tool call arrives.
type awaitingApproval struct {
	id      string
	summary string
}

// flushAwaitingApproval prints any still-unpaired approval lines (oldest
// first), using the label captured from each HITL payload when no tool call
// named the command, then clears the queue.
func flushAwaitingApproval(out io.Writer, awaiting *[]awaitingApproval) {
	for _, a := range *awaiting {
		renderApprovalLine(out, a.id, a.summary)
	}
	*awaiting = nil
}

// minDedupeLen keeps short, legitimately-repeated output (e.g. "ok\n") from
// being swallowed; double-emitted reasoning blocks are always longer.
const minDedupeLen = 8

// tokenDeduper drops a run.token_delta that repeats the whole segment already
// printed in the current text run — some adapters stream the model's reasoning
// live and then re-send it as one consolidated delta.
type tokenDeduper struct {
	seg strings.Builder
}

func (d *tokenDeduper) allow(text string) bool {
	if text == "" {
		return true
	}
	if d.seg.Len() >= minDedupeLen && text == d.seg.String() {
		return false
	}
	d.seg.WriteString(text)
	return true
}

func (d *tokenDeduper) reset() {
	d.seg.Reset()
}

func nextBackoff(cur time.Duration) time.Duration {
	next := cur * 2
	if next > maxReconnectBackoff {
		return maxReconnectBackoff
	}
	return next
}

// sleepCtx waits for d or returns false immediately if ctx is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// pendingEntry is one HITL approval waiting on the user.
type pendingEntry struct {
	id     string
	action string // tool / kind hint shown by /pending; "" if unknown
}

// pendingHITL is a FIFO queue of HITL approvals. The head (entries[0]) is
// what single-keystroke y/n/d resolves; the rest are surfaced by /pending and
// resolvable by explicit id via /a /r /d.
type pendingHITL struct {
	mu      sync.Mutex
	entries []pendingEntry
}

// set enqueues id at the tail if not already queued. action is optional and
// used only for display; pass "" if unknown. Re-enqueueing the same id is a
// no-op so duplicate HITLRequested events (e.g. SSE replay) don't double-up.
func (p *pendingHITL) set(id string, action ...string) {
	if id == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.entries {
		if e.id == id {
			return
		}
	}
	a := ""
	if len(action) > 0 {
		a = action[0]
	}
	p.entries = append(p.entries, pendingEntry{id: id, action: a})
}

// get returns the head (oldest) id, or "" if the queue is empty.
func (p *pendingHITL) get() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.entries) == 0 {
		return ""
	}
	return p.entries[0].id
}

// clearIf removes id from anywhere in the queue. Used by both the optimistic
// post-resolve clear and the server's HITLResolved event, which may target
// a non-head entry (resolved by another device or by /a <id>).
func (p *pendingHITL) clearIf(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, e := range p.entries {
		if e.id == id {
			p.entries = append(p.entries[:i], p.entries[i+1:]...)
			return
		}
	}
}

// len returns the queue depth (used to flip the prompt to show "(N pending)").
func (p *pendingHITL) len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.entries)
}

// list returns a snapshot of all pending entries in arrival order.
func (p *pendingHITL) list() []pendingEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]pendingEntry, len(p.entries))
	copy(out, p.entries)
	return out
}

// reset drops every queued entry and returns how many were cleared.
func (p *pendingHITL) reset() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(p.entries)
	p.entries = nil
	return n
}

func attachLoop(c *CmdConfig, svc do.HostedAgentsService, sessionID string, in io.Reader, state *attachState) error {
	pending := state.pending
	reader := bufio.NewReader(in)
	lines := startAttachLineReader(reader)
	var pendingLine *attachLineRead
	interactiveLineMode := false
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		interactiveLineMode = true
	}
	for {
		fmt.Fprint(c.Out, "\n", attachPrompt(pending))

		// HITL pending + TTY: single keystroke decides. Non-TTY falls back to
		// the bufio line-mode branch below.
		if id := pending.get(); id != "" {
			outcome, key, action := readHITLKeystroke(in)
			switch action {
			case hitlKeyResolve:
				fmt.Fprintf(c.Out, "%c\n", key)
				if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
					Outcome: outcome,
					Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
				}); err != nil {
					fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
				} else {
					// Clear client-side now; the server's HITLResolved event
					// will arrive over SSE and call clearIf again (idempotent).
					// Without this, the next loop iteration re-enters raw mode
					// and blocks on stdin until the SSE round-trip completes,
					// which the user sees as "had to press Enter after y".
					pending.clearIf(id)
				}
				continue
			case hitlKeyDetach:
				fmt.Fprintln(c.Out)
				printDetachNotice(c.Out, state.sessionRef)
				return nil
			case hitlKeyIgnore:
				fmt.Fprintln(c.Out, "(press y, n, or d to resolve the pending approval, or Ctrl-D to detach — session keeps running)")
				continue
			case hitlKeyFallback:
				// Non-TTY — fall through to line mode.
			}
		}

		read := nextAttachLine(lines, &pendingLine)
		line, err := read.line, read.err
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(c.Out)
			printDetachNotice(c.Out, state.sessionRef)
			return nil
		}
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Line-mode HITL shortcut (piped input only).
		if outcome, ok := hitlLetterShortcut(line); ok {
			if id := pending.get(); id != "" {
				if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
					Outcome: outcome,
					Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
				}); err != nil {
					fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
				} else {
					pending.clearIf(id)
				}
				continue
			}
		}
		if strings.HasPrefix(line, "/") {
			if err := handleAttachCommand(c, svc, sessionID, line, pending); err != nil {
				fmt.Fprintf(c.Out, "error: %v\n", err)
			}
			continue
		}
		if batched := collectAttachLines(line, lines, &pendingLine, false); len(batched) > 1 {
			line = strings.Join(batched, "\n")
			fmt.Fprintln(c.Out, colorize("Detected rapid multiline input; sending it as one message.", colMuted))
		}
		if n, ok := needsLargePasteConfirmation(line); ok && interactiveLineMode {
			decision, err := confirmLargePasteLineMode(c.Out, n, lines, &pendingLine)
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(c.Out)
				printDetachNotice(c.Out, state.sessionRef)
				return nil
			}
			if err != nil {
				return err
			}
			switch decision {
			case largePasteSendTogether:
			case largePasteSendSeparately:
				for _, part := range splitSubmittedLines(line) {
					if detach := processAttachLine(c, svc, sessionID, part, state, nil); detach {
						return nil
					}
				}
				continue
			case largePasteDiscard:
				fmt.Fprintln(c.Out, colorize("large paste discarded", colMuted))
				continue
			}
		}
		// Ack immediately; the first agent token can be tens of seconds away
		// and without this users re-submit, spawning a duplicate run.
		if _, err := svc.SendInput(sessionID, &godo.HostedAgentSendInputRequest{Text: line}); err != nil {
			if isRunTerminalErr(err) {
				printSessionEndedNotice(c.Out, state.sessionRef)
				return nil
			}
			fmt.Fprintf(c.Out, "send failed: %v\n", err)
			continue
		}
		fmt.Fprintln(c.Out, colorize("… waiting for the agent", colMuted))
	}
}

type attachLineRead struct {
	line string
	err  error
}

var attachLineBatchWindow = 40 * time.Millisecond

const largePasteConfirmMinLines = 6

func startAttachLineReader(reader *bufio.Reader) <-chan attachLineRead {
	ch := make(chan attachLineRead, 1)
	go func() {
		defer close(ch)
		for {
			line, err := reader.ReadString('\n')
			ch <- attachLineRead{line: line, err: err}
			if err != nil {
				return
			}
		}
	}()
	return ch
}

func nextAttachLine(lines <-chan attachLineRead, pending **attachLineRead) attachLineRead {
	if *pending != nil {
		read := **pending
		*pending = nil
		return read
	}
	read, ok := <-lines
	if !ok {
		return attachLineRead{err: io.EOF}
	}
	return read
}

func collectAttachLines(first string, lines <-chan attachLineRead, pending **attachLineRead, openAI bool) []string {
	if strings.HasPrefix(first, "/") {
		return []string{first}
	}
	collected := []string{first}
	timer := time.NewTimer(attachLineBatchWindow)
	defer timer.Stop()
	for {
		select {
		case read, ok := <-lines:
			if !ok {
				return collected
			}
			if read.err != nil {
				*pending = &read
				return collected
			}
			line := strings.TrimSpace(read.line)
			if line == "" {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(attachLineBatchWindow)
				continue
			}
			if strings.HasPrefix(line, "/") || (!openAI && hitlShortcutOnly(line)) {
				*pending = &attachLineRead{line: line}
				return collected
			}
			collected = append(collected, line)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(attachLineBatchWindow)
		case <-timer.C:
			return collected
		}
	}
}

func hitlShortcutOnly(line string) bool {
	_, ok := hitlLetterShortcut(line)
	return ok
}

func confirmLargePasteLineMode(out io.Writer, linesCount int, lines <-chan attachLineRead, pending **attachLineRead) (largePasteDecision, error) {
	fmt.Fprintf(out, "You pasted %d lines. Send them together as one message? [y/N] ", linesCount)
	read := nextAttachLine(lines, pending)
	if read.err != nil {
		return largePasteDiscard, read.err
	}
	answer := strings.ToLower(strings.TrimSpace(read.line))
	switch answer {
	case "y", "yes":
		return largePasteSendTogether, nil
	case "discard", "cancel":
		return largePasteDiscard, nil
	default:
		return largePasteSendSeparately, nil
	}
}

func largePasteLineCount(text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

func needsLargePasteConfirmation(text string) (int, bool) {
	n := largePasteLineCount(text)
	return n, n >= largePasteConfirmMinLines
}

func splitSubmittedLines(text string) []string {
	parts := strings.Split(text, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// attachPrompt reflects the HITL queue depth so the user knows when more than
// one approval is waiting. The single-keystroke shortcut always targets the
// head; the count nudges the user to use /pending if they want to see the rest.
func attachPrompt(pending *pendingHITL) string {
	n := pending.len()
	switch {
	case n == 0:
		return "> "
	case n == 1:
		return "[y/n/d] > "
	default:
		return fmt.Sprintf("[y/n/d] (%d pending) > ", n)
	}
}

// attachState bundles the line buffer, pending HITL id, and the synchronized
// display that the SSE goroutine writes through.
type attachState struct {
	pending    *pendingHITL
	display    *promptDisplay
	sessionRef string     // name or id for detach messaging
	mu         sync.Mutex // guards lineBuf, cursor, and hitlSel
	lineBuf    []byte
	cursor     int // byte index into lineBuf (ASCII-only input today)
	hitlSel    int // highlighted HITL menu option: 0 approve, 1 reject, 2 defer
	escSeq     []byte
	pasting    bool
	confirm    *largePasteConfirmation
}

type largePasteConfirmation struct {
	text  string
	lines int
}

type largePasteDecision int

const (
	largePasteSendTogether largePasteDecision = iota
	largePasteSendSeparately
	largePasteDiscard
)

func newAttachState(out io.Writer, pending *pendingHITL) *attachState {
	s := &attachState{pending: pending}
	s.display = &promptDisplay{
		out:    out,
		prompt: s.promptString,
		lineBuf: func() string {
			s.mu.Lock()
			defer s.mu.Unlock()
			return displayInputBuffer(s.lineBuf)
		},
		cursorPos: func() int {
			s.mu.Lock()
			defer s.mu.Unlock()
			return s.cursor
		},
	}
	return s
}

// promptString is the raw-mode prompt. With no pending approval it's the plain
// input prompt; while an approval is pending it becomes the arrow-navigable
// approve/reject/defer menu.
func (s *attachState) promptString() string {
	s.mu.Lock()
	confirm := s.confirm
	sel := s.hitlSel
	s.mu.Unlock()
	if confirm != nil {
		return fmt.Sprintf("You pasted %d lines. Send them together as one message? [y/N] ", confirm.lines)
	}
	n := s.pending.len()
	if n == 0 {
		return "> "
	}
	return hitlMenuPrompt(sel, n)
}

// moveHITLSelection shifts the highlighted menu option, wrapping around.
func (s *attachState) moveHITLSelection(delta int) {
	s.mu.Lock()
	s.hitlSel = ((s.hitlSel+delta)%3 + 3) % 3
	s.mu.Unlock()
}

func (s *attachState) hitlSelection() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hitlSel
}

func (s *attachState) resetHITLSelection() {
	s.mu.Lock()
	s.hitlSel = 0
	s.mu.Unlock()
}

func (s *attachState) setLargePasteConfirmation(text string, lines int) {
	s.mu.Lock()
	s.confirm = &largePasteConfirmation{text: text, lines: lines}
	s.mu.Unlock()
}

func (s *attachState) takeLargePasteConfirmation() *largePasteConfirmation {
	s.mu.Lock()
	defer s.mu.Unlock()
	confirm := s.confirm
	s.confirm = nil
	return confirm
}

func (s *attachState) largePasteConfirmation() *largePasteConfirmation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.confirm
}

// moveLineCursor shifts the text-input caret by delta bytes, clamped to the
// line buffer. Returns whether the caret actually moved.
func (s *attachState) moveLineCursor(delta int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.cursor + delta
	if next < 0 {
		next = 0
	}
	if next > len(s.lineBuf) {
		next = len(s.lineBuf)
	}
	if next == s.cursor {
		return false
	}
	s.cursor = next
	return true
}

func displayInputBuffer(buf []byte) string {
	if len(buf) == 0 {
		return ""
	}
	line := string(buf)
	line = strings.ReplaceAll(line, "\r\n", " ")
	line = strings.ReplaceAll(line, "\n", " ")
	line = strings.ReplaceAll(line, "\r", " ")
	return line
}

func submittedInput(buf []byte) string {
	line := strings.ReplaceAll(string(buf), "\r\n", "\n")
	line = strings.ReplaceAll(line, "\r", "\n")
	return strings.TrimSpace(line)
}

var (
	bracketedPasteStart = []byte{0x1b, '[', '2', '0', '0', '~'}
	bracketedPasteEnd   = []byte{0x1b, '[', '2', '0', '1', '~'}
)

const (
	enableBracketedPaste  = "\x1b[?2004h"
	disableBracketedPaste = "\x1b[?2004l"
)

func setBracketedPasteMode(w io.Writer, enabled bool) {
	if enabled {
		_, _ = io.WriteString(w, enableBracketedPaste)
		return
	}
	_, _ = io.WriteString(w, disableBracketedPaste)
}

func isPrefix(seq, candidate []byte) bool {
	if len(seq) > len(candidate) {
		return false
	}
	for i := range seq {
		if seq[i] != candidate[i] {
			return false
		}
	}
	return true
}

func isKnownEscPrefix(seq []byte) bool {
	for _, candidate := range [][]byte{
		{0x1b},
		{0x1b, '['},
		{0x1b, 'O'},
		{0x1b, '[', 'A'},
		{0x1b, '[', 'B'},
		{0x1b, '[', 'C'},
		{0x1b, '[', 'D'},
		{0x1b, 'O', 'A'},
		{0x1b, 'O', 'B'},
		{0x1b, 'O', 'C'},
		{0x1b, 'O', 'D'},
		bracketedPasteStart,
	} {
		if isPrefix(seq, candidate) {
			return true
		}
	}
	return false
}

func appendBufferedInput(state *attachState, chunk []byte) bool {
	if len(chunk) == 0 {
		return false
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.cursor != len(state.lineBuf) {
		return false
	}
	for _, b := range chunk {
		switch b {
		case 0x0d, 0x0a:
			if n := len(state.lineBuf); n > 0 && state.lineBuf[n-1] == '\n' {
				continue
			}
			state.lineBuf = append(state.lineBuf, '\n')
			state.cursor++
		default:
			if b >= 0x20 && b < 0x7f {
				state.lineBuf = append(state.lineBuf, b)
				state.cursor++
			}
		}
	}
	return true
}

func readSubmittedInput(state *attachState) string {
	state.mu.Lock()
	line := submittedInput(state.lineBuf)
	state.lineBuf = state.lineBuf[:0]
	state.cursor = 0
	state.mu.Unlock()
	return line
}

func handleAttachEscapeSequence(b byte, state *attachState) bool {
	if len(state.escSeq) == 0 {
		if b != 0x1b {
			return false
		}
		state.escSeq = []byte{b}
		return true
	}

	state.escSeq = append(state.escSeq, b)
	seq := state.escSeq
	switch {
	case bytes.Equal(seq, bracketedPasteStart):
		state.escSeq = nil
		state.pasting = true
		return true
	case bytes.Equal(seq, []byte{0x1b, '[', 'A'}), bytes.Equal(seq, []byte{0x1b, 'O', 'A'}):
		state.escSeq = nil
		if state.pending.get() != "" {
			state.moveHITLSelection(-1)
			state.display.redraw()
		}
		return true
	case bytes.Equal(seq, []byte{0x1b, '[', 'D'}), bytes.Equal(seq, []byte{0x1b, 'O', 'D'}):
		state.escSeq = nil
		if state.pending.get() != "" {
			state.moveHITLSelection(-1)
			state.display.redraw()
		} else if state.moveLineCursor(-1) {
			state.display.redraw()
		}
		return true
	case bytes.Equal(seq, []byte{0x1b, '[', 'B'}), bytes.Equal(seq, []byte{0x1b, 'O', 'B'}):
		state.escSeq = nil
		if state.pending.get() != "" {
			state.moveHITLSelection(1)
			state.display.redraw()
		}
		return true
	case bytes.Equal(seq, []byte{0x1b, '[', 'C'}), bytes.Equal(seq, []byte{0x1b, 'O', 'C'}):
		state.escSeq = nil
		if state.pending.get() != "" {
			state.moveHITLSelection(1)
			state.display.redraw()
		} else if state.moveLineCursor(1) {
			state.display.redraw()
		}
		return true
	case isKnownEscPrefix(seq):
		return true
	default:
		state.escSeq = nil
		return false
	}
}

func handlePastedByte(b byte, state *attachState) {
	if len(state.escSeq) > 0 || b == 0x1b {
		if len(state.escSeq) == 0 {
			state.escSeq = []byte{b}
			return
		}
		state.escSeq = append(state.escSeq, b)
		seq := state.escSeq
		if bytes.Equal(seq, bracketedPasteEnd) {
			state.escSeq = nil
			state.pasting = false
			state.display.redraw()
			return
		}
		if isPrefix(seq, bracketedPasteEnd) {
			return
		}
		if appendBufferedInput(state, seq) {
			state.display.redraw()
		}
		state.escSeq = nil
		return
	}
	if appendBufferedInput(state, []byte{b}) {
		state.display.redraw()
	}
}

// promptDisplay serializes terminal writes between the input loop and the
// SSE goroutine. In raw mode it tracks whether the cursor sits on the prompt
// line or mid-stream so that streaming tokens don't get wiped, events drop
// to a fresh line, and HITL prompt flips render instantly. Pass-through
// otherwise.
type promptDisplay struct {
	mu        sync.Mutex
	out       io.Writer
	prompt    func() string
	lineBuf   func() string
	cursorPos func() int // caret within lineBuf; nil / at-end leaves cursor at EOL
	raw       bool
	// midLine: cursor is at the end of a previous tokenless write. Next
	// Write must not clear-line, next echo must not paint to that line.
	midLine bool
	// warmupStatusLines reserves status rows above the prompt while the warm-up
	// banner is active (spinner + optional grey phase + optional queued notice).
	warmupStatusLines  int
	warmupSpinnerFrame string
	warmupSpinnerLabel string
	warmupPhaseLabel   string
	warmupQueuedLabel  string
}

func (p *promptDisplay) setRaw(on bool) {
	p.mu.Lock()
	p.raw = on
	p.mu.Unlock()
}

func (p *promptDisplay) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw {
		return p.out.Write(b)
	}
	if len(b) == 0 {
		return 0, nil
	}

	startsWithNL := b[0] == '\n'
	endsWithNL := b[len(b)-1] == '\n'

	if p.midLine {
		// Inject a separator only for discrete events that don't already
		// start with \n — so "(thinking...)" doesn't glue onto a token stream.
		if !startsWithNL && endsWithNL {
			fmt.Fprint(p.out, "\r\n")
		}
	} else {
		fmt.Fprint(p.out, "\r\x1b[K")
	}

	if _, err := io.WriteString(p.out, strings.ReplaceAll(string(b), "\n", "\r\n")); err != nil {
		return 0, err
	}

	if endsWithNL {
		p.paintPromptLocked(false)
		p.midLine = false
	} else {
		p.midLine = true
	}
	return len(b), nil
}

// paintPromptLocked draws prompt + lineBuf and restores the caret. When clear
// is true it first erases the current line (redraw / replace-in-place); when
// false it paints on the current (fresh) line after a newline-terminated write.
func (p *promptDisplay) paintPromptLocked(clear bool) {
	line := ""
	if p.lineBuf != nil {
		line = p.lineBuf()
	}
	if clear {
		fmt.Fprintf(p.out, "\r\x1b[K%s%s", p.prompt(), line)
	} else {
		fmt.Fprintf(p.out, "%s%s", p.prompt(), line)
	}
	if p.cursorPos == nil {
		return
	}
	cur := p.cursorPos()
	if cur < 0 {
		cur = 0
	}
	if cur > len(line) {
		cur = len(line)
	}
	if back := len(line) - cur; back > 0 {
		fmt.Fprintf(p.out, "\x1b[%dD", back)
	}
}

// echo writes a single keystroke for user feedback. Silent mid-stream so
// chars don't land on the agent's line; lineBuf still captures them and they
// reappear on the next prompt redraw.
func (p *promptDisplay) echo(b []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.raw && p.midLine {
		return
	}
	p.out.Write(b)
}

// redraw re-renders prompt + lineBuf with the caret restored. Flips "> " <->
// HITL menu the moment HITL state changes, no Enter needed.
func (p *promptDisplay) redraw() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw {
		return
	}
	if p.midLine {
		fmt.Fprint(p.out, "\r\n")
		p.midLine = false
	}
	if p.warmupStatusLines > 0 {
		p.warmupPaintLocked()
		return
	}
	p.paintPromptLocked(true)
}

// spinnerInit reserves the spinner's own line above the prompt and draws the
// first frame, then redraws the prompt below it — all in one locked write so
// the spinner row and the prompt row stay adjacent for spinnerFrame/spinnerStop.
func (p *promptDisplay) spinnerInit(frame, label string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw {
		return
	}
	if p.midLine {
		fmt.Fprint(p.out, "\r\n")
		p.midLine = false
	} else {
		fmt.Fprint(p.out, "\r\x1b[K")
	}
	fmt.Fprintf(p.out, "%s %s\r\n", frame, label)
	p.paintPromptLocked(false)
}

// spinnerFrame redraws the spinner line one row above the prompt.
// DECSC/DECRC (\x1b7 / \x1b8) save+restore the cursor so the prompt row
// below is preserved. No-op in non-raw or mid-stream state.
func (p *promptDisplay) spinnerFrame(frame, label string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw || p.midLine {
		return
	}
	fmt.Fprintf(p.out, "\x1b7\x1b[A\r\x1b[K%s %s\x1b8", frame, label)
}

// spinnerStop erases the spinner line entirely so no status text or frozen
// braille glyph is left behind in scrollback once the run produces output.
func (p *promptDisplay) spinnerStop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw || p.midLine {
		return
	}
	fmt.Fprint(p.out, "\x1b7\x1b[A\r\x1b[K\x1b8")
}

func (p *promptDisplay) warmupBannerActive() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.warmupStatusLines > 0
}

// warmupInit reserves a spinner row above the prompt. The queued-input row is
// added later via warmupSetQueued once the user starts typing.
func (p *promptDisplay) warmupInit(frame, label string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw {
		return
	}
	if p.midLine {
		fmt.Fprint(p.out, "\r\n")
		p.midLine = false
	} else {
		fmt.Fprint(p.out, "\r\x1b[K")
	}
	p.warmupStatusLines = 1
	p.warmupSpinnerFrame = frame
	p.warmupSpinnerLabel = label
	p.warmupPhaseLabel = ""
	p.warmupQueuedLabel = ""
	fmt.Fprintf(p.out, "%s %s\r\n", frame, label)
	p.paintPromptLocked(false)
}

// warmupSetPhase shows a grey backend-progress line under the spinner.
func (p *promptDisplay) warmupSetPhase(phase string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw || p.warmupStatusLines == 0 {
		return
	}
	p.warmupPhaseLabel = phase
	p.warmupEnsureRowsLocked()
	p.warmupPaintLocked()
}

// warmupSetQueued shows the grey queued-input notice and redraws the whole block.
func (p *promptDisplay) warmupSetQueued(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw || p.warmupStatusLines == 0 {
		return
	}
	p.warmupQueuedLabel = text
	p.warmupEnsureRowsLocked()
	p.warmupPaintLocked()
}

// warmupEnsureRowsLocked grows reserved status rows to fit spinner + phase + queued.
func (p *promptDisplay) warmupEnsureRowsLocked() {
	want := 1
	if p.warmupPhaseLabel != "" {
		want++
	}
	if p.warmupQueuedLabel != "" {
		want++
	}
	for p.warmupStatusLines < want {
		if p.midLine {
			fmt.Fprint(p.out, "\r\n")
			p.midLine = false
		}
		fmt.Fprint(p.out, "\r\n")
		p.paintPromptLocked(false)
		p.warmupStatusLines++
	}
}

// warmupSetFrame updates the spinner frame and redraws the whole block.
func (p *promptDisplay) warmupSetFrame(frame string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw || p.midLine || p.warmupStatusLines == 0 {
		return
	}
	p.warmupSpinnerFrame = frame
	p.warmupPaintLocked()
}

// warmupPaintLocked redraws the warm-up spinner, optional grey phase / queued
// lines, and prompt+input atomically from the prompt row. Caller must hold p.mu.
func (p *promptDisplay) warmupPaintLocked() {
	frame := p.warmupSpinnerFrame
	if frame == "" {
		frame = spinnerFrames[0]
	}
	label := p.warmupSpinnerLabel
	if label == "" {
		label = msgAgentWarmup
	}
	line := ""
	if p.lineBuf != nil {
		line = p.lineBuf()
	}
	cur := len(line)
	if p.cursorPos != nil {
		cur = p.cursorPos()
		if cur < 0 {
			cur = 0
		}
		if cur > len(line) {
			cur = len(line)
		}
	}

	n := p.warmupStatusLines
	var b strings.Builder
	fmt.Fprintf(&b, "\x1b7\x1b[%dA\r\x1b[K", n)
	fmt.Fprintf(&b, "%s %s", frame, label)
	if p.warmupPhaseLabel != "" {
		b.WriteString("\r\n\r\x1b[K")
		b.WriteString(colorize(p.warmupPhaseLabel, colMuted))
	}
	if p.warmupQueuedLabel != "" {
		b.WriteString("\r\n\r\x1b[K")
		b.WriteString(colorize(p.warmupQueuedLabel, colMuted))
	}
	b.WriteString("\r\n\r\x1b[K")
	b.WriteString(p.prompt())
	b.WriteString(line)
	if back := len(line) - cur; back > 0 {
		fmt.Fprintf(&b, "\x1b[%dD", back)
	}
	b.WriteString("\x1b8")
	io.WriteString(p.out, b.String())
}

// warmupStopLocked erases the warm-up banner rows entirely. Caller must hold p.mu.
func (p *promptDisplay) warmupStopLocked() {
	if !p.raw || p.warmupStatusLines == 0 {
		return
	}
	if p.midLine {
		fmt.Fprint(p.out, "\r\n")
		p.midLine = false
	}
	n := p.warmupStatusLines
	var b strings.Builder
	b.WriteString("\x1b7")
	for i := 0; i < n; i++ {
		b.WriteString("\x1b[A\r\x1b[K")
	}
	b.WriteString("\x1b8")
	io.WriteString(p.out, b.String())
	p.warmupStatusLines = 0
	p.warmupSpinnerFrame = ""
	p.warmupSpinnerLabel = ""
	p.warmupPhaseLabel = ""
	p.warmupQueuedLabel = ""
	p.paintPromptLocked(true)
}

// warmupStop erases the warm-up banner rows entirely.
func (p *promptDisplay) warmupStop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.warmupStopLocked()
}

// attachLoopTTY runs the raw-mode byte-by-byte input state machine. A 50ms
// ticker polls pending-HITL so a HITL event flips the prompt instantly,
// without needing the user to press Enter to "wake up" the loop.
func attachLoopTTY(c *CmdConfig, svc do.HostedAgentsService, sessionID string, f *os.File, state *attachState, warmup *warmupState) error {
	fd := int(f.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Raw mode unavailable; fall back to bufio line mode.
		if warmup != nil {
			warmup.start()
		}
		return attachLoop(c, svc, sessionID, f, state)
	}
	defer term.Restore(fd, oldState)
	setBracketedPasteMode(f, true)
	defer setBracketedPasteMode(f, false)

	state.display.setRaw(true)
	defer state.display.setRaw(false)

	state.display.redraw()
	if warmup != nil {
		warmup.start()
	}

	bytesCh := make(chan byte, 64)
	readErrCh := make(chan error, 1)
	go func() {
		var buf [1]byte
		for {
			n, err := f.Read(buf[:])
			if err != nil {
				readErrCh <- err
				return
			}
			if n == 1 {
				bytesCh <- buf[0]
			}
		}
	}()

	// 50ms ticker so HITL arrival reflows the prompt even when the user idles.
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	lastPending := state.pending.get()
	for {
		if cur := state.pending.get(); cur != lastPending {
			state.display.redraw()
			lastPending = cur
		}
		select {
		case b := <-bytesCh:
			stop, err := handleAttachByte(c, svc, sessionID, b, state, warmup)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		case <-ticker.C:
		case err := <-readErrCh:
			if errors.Is(err, io.EOF) {
				printDetachNotice(c.Out, state.sessionRef)
				return nil
			}
			return err
		}
	}
}

// handleAttachByte is the per-byte state machine. stop=true exits the loop
// (Ctrl-C, Ctrl-D on empty line).
func handleAttachByte(c *CmdConfig, svc do.HostedAgentsService, sessionID string, b byte, state *attachState, warmup *warmupState) (stop bool, err error) {
	if confirm := state.largePasteConfirmation(); confirm != nil {
		switch b {
		case 'y', 'Y':
			state.display.echo([]byte{b, '\r', '\n'})
			state.takeLargePasteConfirmation()
			if detach := processAttachLine(c, svc, sessionID, confirm.text, state, nil); detach {
				return true, nil
			}
			state.display.redraw()
			return false, nil
		case 'n', 'N', 0x0d, 0x0a:
			state.display.echo([]byte("\r\n"))
			state.takeLargePasteConfirmation()
			for _, part := range splitSubmittedLines(confirm.text) {
				if detach := processAttachLine(c, svc, sessionID, part, state, nil); detach {
					return true, nil
				}
			}
			state.display.redraw()
			return false, nil
		case 0x03, 0x04:
			state.display.echo([]byte("\r\n"))
			state.takeLargePasteConfirmation()
			fmt.Fprintln(c.Out, colorize("large paste discarded", colMuted))
			state.display.redraw()
			return false, nil
		default:
			return false, nil
		}
	}
	if state.pasting {
		handlePastedByte(b, state)
		warmup.noteQueued()
		if warmup.isBannerVisible() {
			state.display.redraw()
		}
		return false, nil
	}
	if handleAttachEscapeSequence(b, state) {
		return false, nil
	}

	if id := state.pending.get(); id != "" {
		var outcome godo.HostedAgentHITLOutcome
		var matched bool
		switch b {
		case 'y', 'Y', 'a', 'A':
			outcome, matched = godo.HostedAgentHITLOutcomeApprove, true
		case 'n', 'N', 'r', 'R':
			outcome, matched = godo.HostedAgentHITLOutcomeReject, true
		case 'd', 'D':
			outcome, matched = godo.HostedAgentHITLOutcomeDefer, true
		case 0x0d, 0x0a: // Enter confirms the highlighted menu option
			outcome, matched = hitlOutcomeForSelection(state.hitlSelection()), true
		case 0x03, 0x04: // Ctrl-C / Ctrl-D
			state.display.echo([]byte("\r\n"))
			printDetachNotice(c.Out, state.sessionRef)
			return true, nil
		}
		if !matched {
			// Ignore other bytes while HITL pending.
			return false, nil
		}
		if b == 0x0d || b == 0x0a {
			state.display.echo([]byte("\r\n"))
		} else {
			state.display.echo([]byte{b, '\r', '\n'})
		}
		if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
			Outcome: outcome,
			Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
		}); err != nil {
			fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
		} else {
			state.pending.clearIf(id)
		}
		state.resetHITLSelection()
		state.display.redraw()
		return false, nil
	}

	switch b {
	case 0x0d, 0x0a: // Enter
		line := readSubmittedInput(state)
		if line != "" && warmup.inputAlreadyQueued() {
			fmt.Fprintln(c.Out, colorize("Message already queued — waiting for agent to start", colMuted))
			state.display.redraw()
			return false, nil
		}
		echoAttachSubmitNewline(state.display, warmup)
		if line != "" {
			if n, ok := needsLargePasteConfirmation(line); ok {
				state.setLargePasteConfirmation(line, n)
				state.display.redraw()
				return false, nil
			}
			if detach := processAttachLine(c, svc, sessionID, line, state, warmup); detach {
				return true, nil
			}
		}
		state.display.redraw()
		return false, nil
	case 0x7f, 0x08: // Backspace / DEL
		state.mu.Lock()
		atEnd := state.cursor == len(state.lineBuf)
		if state.cursor > 0 {
			i := state.cursor - 1
			state.lineBuf = append(state.lineBuf[:i], state.lineBuf[i+1:]...)
			state.cursor = i
			state.mu.Unlock()
			if warmup.isBannerVisible() {
				state.display.redraw()
			} else if atEnd {
				state.display.echo([]byte("\b \b"))
			} else {
				state.display.redraw()
			}
		} else {
			state.mu.Unlock()
		}
		return false, nil
	case 0x03: // Ctrl-C
		state.display.echo([]byte("\r\n"))
		printDetachNotice(c.Out, state.sessionRef)
		return true, nil
	case 0x04: // Ctrl-D
		state.mu.Lock()
		empty := len(state.lineBuf) == 0
		state.mu.Unlock()
		if empty {
			state.display.echo([]byte("\r\n"))
			printDetachNotice(c.Out, state.sessionRef)
			return true, nil
		}
		return false, nil
	default:
		// Printable ASCII only; UTF-8 multibyte is still dropped in V0.
		if b >= 0x20 && b < 0x7f {
			state.mu.Lock()
			atEnd := state.cursor == len(state.lineBuf)
			if atEnd {
				state.lineBuf = append(state.lineBuf, b)
			} else {
				state.lineBuf = append(state.lineBuf[:state.cursor], append([]byte{b}, state.lineBuf[state.cursor:]...)...)
			}
			state.cursor++
			state.mu.Unlock()
			warmup.noteQueued()
			if warmup.isBannerVisible() {
				state.display.redraw()
			} else if atEnd {
				state.display.echo([]byte{b})
			} else {
				state.display.redraw()
			}
		}
		return false, nil
	}
}

// processAttachLine dispatches an Enter-submitted line: HITL word shortcut,
// slash command, or SendInput. Returns detach=true when the session can no
// longer accept input (terminal run) and the loop should exit.
func processAttachLine(c *CmdConfig, svc do.HostedAgentsService, sessionID, line string, state *attachState, warmup *warmupState) (detach bool) {
	if outcome, ok := hitlLetterShortcut(line); ok {
		if id := state.pending.get(); id != "" {
			if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
				Outcome: outcome,
				Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
			}); err != nil {
				fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
			} else {
				state.pending.clearIf(id)
			}
			return false
		}
	}
	if strings.HasPrefix(line, "/") {
		if err := handleAttachCommand(c, svc, sessionID, line, state.pending); err != nil {
			fmt.Fprintf(c.Out, "error: %v\n", err)
		}
		return false
	}
	if _, err := svc.SendInput(sessionID, &godo.HostedAgentSendInputRequest{Text: line}); err != nil {
		if isRunTerminalErr(err) {
			printSessionEndedNotice(c.Out, state.sessionRef)
			return true
		}
		fmt.Fprintf(c.Out, "send failed: %v\n", err)
		return false
	}
	if warmup != nil && warmup.isActive() {
		warmup.markInputQueued()
	}
	printAttachSendAck(c.Out, warmup)
	return false
}

// hitlLetterShortcut is the line-mode (piped / non-TTY) HITL path. Interactive
// terminals go through readHITLKeystroke instead. Returns (_, false) for any
// input that should fall through to slash-command / SendInput handling.
func hitlLetterShortcut(line string) (godo.HostedAgentHITLOutcome, bool) {
	switch strings.ToLower(line) {
	case "y", "yes", "a":
		return godo.HostedAgentHITLOutcomeApprove, true
	case "n", "no", "r":
		return godo.HostedAgentHITLOutcomeReject, true
	case "d", "defer":
		return godo.HostedAgentHITLOutcomeDefer, true
	}
	return "", false
}

type hitlKeyAction int

const (
	hitlKeyFallback hitlKeyAction = iota // not a TTY; caller should use bufio line mode
	hitlKeyResolve                       // y/n/d pressed; outcome is set
	hitlKeyDetach                        // Ctrl-C / Ctrl-D
	hitlKeyIgnore                        // any other key; byte was consumed
)

// readHITLKeystroke captures one y/n/d keystroke from an interactive terminal
// with no Enter. Returns hitlKeyFallback on non-TTY input so tests and pipes
// can use the bufio line-mode path.
func readHITLKeystroke(in io.Reader) (godo.HostedAgentHITLOutcome, byte, hitlKeyAction) {
	f, ok := in.(*os.File)
	if !ok {
		return "", 0, hitlKeyFallback
	}
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return "", 0, hitlKeyFallback
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", 0, hitlKeyFallback
	}
	defer term.Restore(fd, oldState)

	var buf [1]byte
	n, err := f.Read(buf[:])
	if err != nil || n != 1 {
		return "", 0, hitlKeyFallback
	}
	switch buf[0] {
	case 'y', 'Y', 'a', 'A':
		return godo.HostedAgentHITLOutcomeApprove, buf[0], hitlKeyResolve
	case 'n', 'N', 'r', 'R':
		return godo.HostedAgentHITLOutcomeReject, buf[0], hitlKeyResolve
	case 'd', 'D':
		return godo.HostedAgentHITLOutcomeDefer, buf[0], hitlKeyResolve
	case 0x03, 0x04: // Ctrl-C, Ctrl-D
		return "", buf[0], hitlKeyDetach
	}
	return "", buf[0], hitlKeyIgnore
}

// handleAttachCommand parses a slash command. `/a`, `/r`, `/d` accept either a
// bare request id or "implicit" (use the most recent pending one).
func handleAttachCommand(c *CmdConfig, svc do.HostedAgentsService, sessionID, line string, pending *pendingHITL) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}
	verb := parts[0]
	switch verb {
	case "/help":
		fmt.Fprintln(c.Out, "Detach (does NOT remove the session):")
		fmt.Fprintln(c.Out, "  Ctrl-D           close the connection; reattach later with `doctl open-harness-runtime attach`")
		fmt.Fprintln(c.Out, "  Ctrl-C           same as Ctrl-D")
		fmt.Fprintln(c.Out, "Remove the session: `doctl open-harness-runtime remove <session>`")
		fmt.Fprintln(c.Out, "When a HITL approval is pending (no Enter needed in a TTY):")
		fmt.Fprintln(c.Out, "  ↑ / ↓ then Enter  move the highlight and confirm the selected outcome")
		fmt.Fprintln(c.Out, "  y | a             approve the oldest pending request")
		fmt.Fprintln(c.Out, "  n | r             reject  the oldest pending request")
		fmt.Fprintln(c.Out, "  d                 defer   the oldest pending request")
		fmt.Fprintln(c.Out, "  (piped input: send `yes` / `no` / `defer` followed by a newline)")
		fmt.Fprintln(c.Out, "With an explicit request id (works on any queued request):")
		fmt.Fprintln(c.Out, "  /a [request-id]   approve (defaults to the oldest pending)")
		fmt.Fprintln(c.Out, "  /r [request-id]   reject")
		fmt.Fprintln(c.Out, "  /d [request-id]   defer")
		fmt.Fprintln(c.Out, "  /pending          list all HITL approvals waiting on you")
		return nil
	case "/pending":
		return listPendingHITLs(c, pending)
	case "/a", "/approve":
		return resolveFromAttach(svc, sessionID, parts, pending, godo.HostedAgentHITLOutcomeApprove)
	case "/r", "/reject":
		return resolveFromAttach(svc, sessionID, parts, pending, godo.HostedAgentHITLOutcomeReject)
	case "/d", "/defer":
		return resolveFromAttach(svc, sessionID, parts, pending, godo.HostedAgentHITLOutcomeDefer)
	default:
		return fmt.Errorf("unknown command %q (try /help)", verb)
	}
}

// listPendingHITLs renders the current HITL queue. The head is marked with
// "*" because that's what a single keystroke will resolve next.
func listPendingHITLs(c *CmdConfig, pending *pendingHITL) error {
	list := pending.list()
	if len(list) == 0 {
		fmt.Fprintln(c.Out, "(no HITL approvals pending)")
		return nil
	}
	fmt.Fprintf(c.Out, "%d HITL approval(s) pending (oldest first; * is next for single-keystroke y/n/d):\n", len(list))
	for i, e := range list {
		marker := " "
		if i == 0 {
			marker = "*"
		}
		if e.action != "" {
			fmt.Fprintf(c.Out, "  %s %s  (%s)\n", marker, e.id, e.action)
		} else {
			fmt.Fprintf(c.Out, "  %s %s\n", marker, e.id)
		}
	}
	return nil
}

func resolveFromAttach(svc do.HostedAgentsService, sessionID string, parts []string, pending *pendingHITL, outcome godo.HostedAgentHITLOutcome) error {
	id := ""
	if len(parts) >= 2 {
		id = parts[1]
	} else {
		id = pending.get()
	}
	if id == "" {
		return errors.New("no pending HITL request; provide a request id explicitly")
	}
	if err := svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
		Outcome: outcome,
		Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
	}); err != nil {
		return err
	}
	// Optimistic local clear so the queue / prompt reflects the resolution
	// immediately, before the server's HITLResolved event comes back over SSE.
	pending.clearIf(id)
	return nil
}

// runSeparator visually divides agent output from the next user prompt.
const runSeparator = "────────────────────────────────────────"

// Payload shapes mirror the SPI data structs (spi/events.go) for the kinds
// doctl renders. godo leaves HostedAgentEvent.Payload as json.RawMessage.

type tokenChunkPayload struct {
	Text string `json:"text"`
}

type runStartedPayload struct {
	Agent string `json:"agent"`
}

type toolCallStartedPayload struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Input     json.RawMessage `json:"input"`
}

// commandLine returns the most descriptive one-liner for a tool call: the
// actual command pulled from arguments/input when present, otherwise the tool
// name (e.g. "bash").
func (p toolCallStartedPayload) commandLine() string {
	for _, raw := range []json.RawMessage{p.Arguments, p.Input} {
		if len(raw) == 0 {
			continue
		}
		var m map[string]any
		if json.Unmarshal(raw, &m) == nil {
			if cmd := searchCommand(m, 3); cmd != "" {
				return cmd
			}
		}
	}
	return p.Name
}

type toolCallCompletedPayload struct {
	OK         bool   `json:"ok"`
	DurationMS int64  `json:"duration_ms"`
	Summary    string `json:"summary,omitempty"`
}

// hitlRequestedPayload is the data body of a run.human_input_requested event.
// Harness-api forwards the HITLRequest shape (action + details) at the top
// level; older / nested adapters also wrap fields under "payload". Both are
// accepted so reattach (which re-injects only this frame, not tool_call_started)
// can still surface the command being approved.
type hitlRequestedPayload struct {
	HitlID    string         `json:"hitl_id"`
	RequestID string         `json:"request_id"`
	Action    string         `json:"action"`
	Details   map[string]any `json:"details"`
	Payload   map[string]any `json:"payload"`
}

func (p hitlRequestedPayload) id() string {
	if p.HitlID != "" {
		return p.HitlID
	}
	return p.RequestID
}

// fields returns the map hitlCommandSummary / hitlActionLabel should search.
// Merges nested "payload" (legacy adapters) with top-level action+details
// (HITLRequest as forwarded by harness-api) so reattach can read details.command
// even when no tool_call_started event is replayed.
func (p hitlRequestedPayload) fields() map[string]any {
	m := map[string]any{}
	for k, v := range p.Payload {
		m[k] = v
	}
	if p.Action != "" {
		if _, exists := m["action"]; !exists {
			m["action"] = p.Action
		}
	}
	if len(p.Details) > 0 {
		if _, exists := m["details"]; !exists {
			m["details"] = p.Details
		}
		// Flatten details so command/cmd/argv are found without relying on
		// recursion depth (matches harness-trigger runrender).
		for k, v := range p.Details {
			if _, exists := m[k]; !exists {
				m[k] = v
			}
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

func (p hitlRequestedPayload) commandSummary() string {
	return hitlCommandSummary(p.fields())
}

func (p hitlRequestedPayload) actionLabel() string {
	return hitlActionLabel(p.fields())
}

type hitlResolvedPayload struct {
	HitlID  string `json:"hitl_id"`
	Outcome int32  `json:"outcome"`
	Actor   string `json:"actor,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type runCompletedPayload struct {
	TotalTokensIn  int64 `json:"total_tokens_in"`
	TotalTokensOut int64 `json:"total_tokens_out"`
	RunCostMicros  int64 `json:"run_cost_micros"`
}

type runFailedPayload struct {
	Code    int32  `json:"code"`
	Message string `json:"message,omitempty"`
}

func renderEvent(w io.Writer, ev godo.HostedAgentEvent) {
	switch ev.Kind {
	case godo.HostedAgentEventKindTokenChunk:
		var p tokenChunkPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprint(w, p.Text)
		}
	case godo.HostedAgentEventKindRunStarted:
		// The server sometimes echoes the prompt in `agent`, so don't render it;
		// keep the marker clean and let the spinner convey activity.
		fmt.Fprintf(w, "\n%s\n", colorize("▶ run started", colMuted))
	case godo.HostedAgentEventKindToolCallStarted:
		var p toolCallStartedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			renderToolStart(w, p.commandLine())
		}
	case godo.HostedAgentEventKindToolCallCompleted:
		var p toolCallCompletedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			mark := colorize("✓", colSuccess)
			if !p.OK {
				mark = colorize("✗", colError)
			}
			summary := p.Summary
			if summary == "" {
				summary = "done"
			}
			fmt.Fprintf(w, "  %s %s %s\n", mark, summary, colorize(fmt.Sprintf("(%dms)", p.DurationMS), colMuted))
		}
	case godo.HostedAgentEventKindHITLRequested:
		var p hitlRequestedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			renderApprovalLine(w, p.id(), p.commandSummary())
		}
	case godo.HostedAgentEventKindHITLResolved:
		var p hitlResolvedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n%s %s\n",
				colorize(p.HitlID, colMuted), hitlOutcomeStyled(p.Outcome))
		}
	case godo.HostedAgentEventKindRunCompleted:
		var p runCompletedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			// Only show the usage/cost segment when the adapter actually
			// reported it; some adapters (e.g. opencode) send all zeros,
			// which looks broken rather than informative.
			summary := "run complete"
			if p.TotalTokensIn > 0 || p.TotalTokensOut > 0 || p.RunCostMicros > 0 {
				summary = fmt.Sprintf("run complete · %d in / %d out tokens · $%.4f",
					p.TotalTokensIn, p.TotalTokensOut, float64(p.RunCostMicros)/1_000_000)
			}
			fmt.Fprintf(w, "\n%s %s\n", colorize("✓", colSuccess), colorize(summary, colMuted))
			fmt.Fprintln(w, colorize(runSeparator, colMuted))
		}
	case godo.HostedAgentEventKindRunFailed:
		var p runFailedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			msg := fmt.Sprintf("run failed: code %d", p.Code)
			if p.Message != "" {
				msg = fmt.Sprintf("run failed: %s (code %d)", p.Message, p.Code)
			}
			fmt.Fprintf(w, "\n%s %s\n", colorize("✗", colError), colorize(msg, colError))
			fmt.Fprintln(w, colorize(runSeparator, colMuted))
		}
	case godo.HostedAgentEventKindSessionUpdated:
		fmt.Fprintf(w, "\n%s\n", colorize("• session updated", colMuted))
	case godo.HostedAgentEventKindRunSandboxAllocated:
		fmt.Fprintf(w, "\n%s\n", colorize("• sandbox allocated", colMuted))
	case godo.HostedAgentEventKindRunSandboxReleased:
		fmt.Fprintf(w, "\n%s\n", colorize("• sandbox released", colMuted))
	}
}

// hitlOutcomeStyled renders a HITL outcome verb with a matching color:
// approve green, reject red, defer yellow.
func hitlOutcomeStyled(code int32) string {
	label := hitlOutcomeLabel(code)
	switch code {
	case 1:
		return colorize(label, colSuccess)
	case 2:
		return colorize(label, colError)
	case 3:
		return colorize(label, colWarning)
	default:
		return colorize(label, colMuted)
	}
}

// hitlOutcomeLabel maps the proto HITLOutcome int (on the SPI
// human_input_received event) to a human-readable verb.
func hitlOutcomeLabel(code int32) string {
	switch code {
	case 1:
		return "approve"
	case 2:
		return "reject"
	case 3:
		return "defer"
	default:
		return fmt.Sprintf("outcome %d", code)
	}
}

// cardRow formats a label/value line for agent summary cards. Labels are padded
// before color codes are applied so columns stay aligned in the terminal.
func cardRow(label, value string) string {
	padded := label
	if len(padded) < 8 {
		padded += strings.Repeat(" ", 8-len(padded))
	}
	return fmt.Sprintf("  %s %s\n", boldColor(padded, colHighlight), value)
}

// renderAgentCard wraps body text in a rounded success border when styling is on.
func renderAgentCard(w io.Writer, body string) {
	out := body
	if stylingEnabled {
		out = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colSuccess).
			Padding(0, 2).
			MarginTop(1).
			Render(out)
	} else {
		out = "\n" + out
	}
	fmt.Fprintln(w, out)
}

// printAttachBanner renders the styled connection header and a compact help
// key shown when an interactive session is attached.
func printAttachBanner(w io.Writer, sess *do.HostedAgentSession, bridgeNote string) {
	ref := displaySessionRef(sess)
	agent := prettyAgentKind(sess.AgentKind)

	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Connected", colSuccess))

	body.WriteString(cardRow("Session", ref))
	if sess != nil && strings.TrimSpace(sess.SessionID) != "" && sess.SessionID != ref {
		body.WriteString(cardRow("ID", colorize(sess.SessionID, colMuted)))
	}
	body.WriteString(cardRow("Agent", agent))
	if note := strings.TrimSpace(bridgeNote); note != "" {
		body.WriteString(cardRow("Bridge", colorize(note, colMuted)))
	}

	fmt.Fprintln(&body)
	fmt.Fprintln(&body, colorize("Quick help", colMuted))

	yn := colorize("y", colSuccess) + colorize("/a", colSuccess) + " approve · " +
		colorize("n", colError) + colorize("/r", colError) + " reject · " +
		colorize("d", colWarning) + " defer"
	body.WriteString(cardRow("send", "type a message and press Enter"))
	body.WriteString(cardRow("approve", "↑/↓ then Enter, or "+yn))
	body.WriteString(cardRow("detach", colorize("Ctrl-D", colMuted)+" closes connection only — session keeps running"))
	body.WriteString(cardRow("remove", colorize("doctl open-harness-runtime remove "+ref, colMuted)))
	body.WriteString(cardRow("help", "type "+boldColor("/help", colHighlight)+" for the full command list"))

	renderAgentCard(w, body.String())
}

// prettyAgentKind turns AGENT_KIND_OPENCODE into a friendly "opencode" label.
func prettyAgentKind(k godo.HostedAgentKind) string {
	s := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(string(k), "AGENT_KIND_"), "_", " "))
	if s == "" || s == "unspecified" {
		return "agent"
	}
	return s
}

// renderApprovalLine prints the one-line approval prompt with an optional
// command and the full HITL request id. The outcomes are shown by the
// interactive menu.
func renderApprovalLine(w io.Writer, hitlID, cmd string) {
	parts := []string{boldColor("● Approval required", colWarning)}
	if cmd != "" {
		parts = append(parts, boldColor(cmd, colHighlight))
	} else {
		// The adapter didn't tell us what this approval is for (no command in
		// the HITL payload and no paired tool call). Label it rather than
		// leaving a bare id that reads as a glitch.
		parts = append(parts, colorize("action pending", colMuted))
	}
	parts = append(parts, colorize(hitlID, colMuted))
	fmt.Fprintf(w, "\n%s\n", strings.Join(parts, colorize("  ·  ", colMuted)))
}

// renderToolStart prints the "running a tool" line.
func renderToolStart(w io.Writer, cmd string) {
	fmt.Fprintf(w, "\n%s %s\n", colorize("▸", colHighlight), boldColor(cmd, colHighlight))
}

// hitlCommandSummary extracts the best one-line command/action label from a
// HITLRequested payload. The wire shape is a generic JSON object, so search the
// common command keys (recursing one level into nested objects) and argv, then
// fall back to a friendly name for the action kind.
func hitlCommandSummary(payload map[string]any) string {
	if s := searchCommand(payload, 3); s != "" {
		return s
	}
	if lbl := hitlActionLabel(payload); lbl != "" {
		return prettyHITLAction(lbl)
	}
	return ""
}

// searchCommand looks for a command string (or argv array) at the top level of
// m, then recurses into nested maps up to depth levels deep.
func searchCommand(m map[string]any, depth int) string {
	if m == nil || depth < 0 {
		return ""
	}
	for _, key := range []string{"command", "cmd", "command_line", "commandline", "script"} {
		if v, ok := m[key].(string); ok && v != "" {
			return v
		}
	}
	if argv, ok := m["argv"].([]any); ok && len(argv) > 0 {
		parts := make([]string, 0, len(argv))
		for _, a := range argv {
			if s, ok := a.(string); ok {
				parts = append(parts, s)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}
	for _, v := range m {
		if sub, ok := v.(map[string]any); ok {
			if s := searchCommand(sub, depth-1); s != "" {
				return s
			}
		}
	}
	return ""
}

// prettyHITLAction maps a HITL action-kind constant to a human-readable verb;
// unknown values are returned as-is.
func prettyHITLAction(s string) string {
	switch godo.HostedAgentHITLActionKind(s) {
	case godo.HostedAgentHITLActionBash:
		return "run a shell command"
	case godo.HostedAgentHITLActionFileWriteOutsideWorkspace:
		return "write a file outside the workspace"
	case godo.HostedAgentHITLActionGitHubCommitPush:
		return "commit and push to GitHub"
	case godo.HostedAgentHITLActionGitHubCreatePR:
		return "create a pull request"
	case godo.HostedAgentHITLActionGitHubBranchDelete:
		return "delete a branch"
	case godo.HostedAgentHITLActionGitHubForcePush:
		return "force-push to GitHub"
	}
	return s
}

// hitlOutcomeForSelection maps the arrow-menu index (0/1/2) to an outcome.
func hitlOutcomeForSelection(sel int) godo.HostedAgentHITLOutcome {
	switch sel {
	case 1:
		return godo.HostedAgentHITLOutcomeReject
	case 2:
		return godo.HostedAgentHITLOutcomeDefer
	default:
		return godo.HostedAgentHITLOutcomeApprove
	}
}

// hitlMenuPrompt renders the single-line, arrow-navigable approve/reject/defer
// menu shown as the prompt while a HITL request is pending. sel is the
// highlighted option; n is the queue depth.
func hitlMenuPrompt(sel, n int) string {
	labels := [3]string{"Approve", "Reject", "Defer"}
	keys := [3]string{"y", "n", "d"}
	colors := [3]lipgloss.Color{colSuccess, colError, colWarning}
	parts := make([]string, 3)
	for i := 0; i < 3; i++ {
		// Pair each label with its shortcut key so the two input methods
		// (arrow-navigate vs. press-a-key) read as one thing, not two
		// unrelated groups.
		label := fmt.Sprintf("%s (%s)", labels[i], keys[i])
		if i == sel {
			parts[i] = hitlOptionSelected(label, colors[i])
		} else {
			parts[i] = colorize(label, colMuted)
		}
	}
	menu := strings.Join(parts, "   ")
	hint := colorize("↑/↓ + Enter, or press a key", colMuted)
	pendingNote := ""
	if n > 1 {
		pendingNote = colorize(fmt.Sprintf(" · %d pending", n), colMuted)
	}
	return fmt.Sprintf("%s   %s%s > ", menu, hint, pendingNote)
}

// hitlOptionSelected styles the highlighted menu option. Without styling it
// falls back to bracketing so the selection is still visible.
func hitlOptionSelected(label string, c lipgloss.Color) string {
	if !stylingEnabled {
		return "[" + label + "]"
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true).Reverse(true).Render(" " + label + " ")
}

// hitlActionLabel pulls the best human-readable label out of a HITLRequested
// payload map. The harness-api wire shape isn't strongly typed in our event
// payload (it's a generic JSON object), so we try a couple of plausible keys.
// Returns "" if no label is present so the caller can decide how to render.
func hitlActionLabel(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	for _, key := range []string{"action", "kind", "tool", "name"} {
		if v, ok := payload[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}
