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

// The `doctl agents` subcommand wraps the godo HostedAgents service, which
// in turn talks to the hosted-agents Harness API. All wire types and the SSE
// iterator live in godo (hosted_agents.go); this file handles CLI plumbing,
// argument parsing, and human-readable rendering of streamed events.
package commands

import (
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

// Stream-state transport frames (SSE kind "stream.state"). Defined locally so
// doctl builds against godo pins that have not yet exported HostedAgentEventKindStreamState
// / HostedAgentStreamState*. Wire values match the published godo API.
const (
	hostedAgentEventKindStreamState  godo.HostedAgentEventKind = "stream.state"
	hostedAgentStreamStateSuperseded                           = "superseded"
)

type hostedAgentStreamState struct {
	State  string `json:"state"`
	Cursor string `json:"cursor,omitempty"`
}

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

// Agents creates the `doctl agents` command tree.
func Agents() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "agents",
			Aliases: []string{"agent"},
			Short:   "Launch and manage hosted DigitalOcean agent sessions",
			Long: `The ` + "`" + `doctl agents` + "`" + ` commands manage hosted coding-agent sessions running in DigitalOcean sandboxes.

A session is one long-lived agent process (Claude Code, OpenCode, ...) running inside a workspace sandbox. doctl drives it: starting it from an agent spec, attaching an interactive TUI, listing existing sessions, resolving HITL approvals out of band, and tearing it down.

Commands that act on a single session accept either the session ID or its name. A name must match exactly one session; if it is ambiguous, pass the session ID instead.`,
			GroupID: hostedAgentsGroup,
		},
	}

	cmdStart := CmdBuilder(cmd, RunAgentsStart, "start",
		"Start a new agent session",
		`Creates a new agent session from an agent manifest file and prints its session id and status.

The `+"`"+`--spec`+"`"+` flag is required and accepts a YAML manifest matching the `+"`"+`agents.digitalocean.com/v1alpha1`+"`"+` schema. `+"`"+`${VAR}`+"`"+` references in the manifest are resolved from your local environment before upload; referencing an unset variable is an error, and `+"`"+`$${VAR}`+"`"+` escapes to a literal `+"`"+`${VAR}`+"`"+`. The expanded manifest is then sent to the server, which owns parsing and validation.

Use `+"`"+`--name`+"`"+` to name the session (this sets the manifest's `+"`"+`metadata.name`+"`"+`). If omitted, the server auto-generates a name. The name must be unique among your team's active sessions, and once set you can reference the session by name in other commands (e.g. `+"`"+`doctl agents attach <name>`+"`"+`).`,
		Writer, aliasOpt("deploy"),
		displayerType(&displayers.HostedAgentSession{}))
	AddStringFlag(cmdStart, doctl.ArgAgentSpec, "", "", `Path to an agent manifest in YAML or JSON. Set to "-" to read from stdin. ${VAR} references are resolved from the local environment.`, requiredOpt())
	AddStringFlag(cmdStart, doctl.ArgAgentName, "", "", "Name for the new session (sets the manifest's metadata.name). If omitted, the server auto-generates a name. Must be unique among your team's active sessions.")
	cmdStart.Example = `doctl agents start --spec agent-spec.yaml --name my-session`

	cmdStartProxy := CmdBuilder(cmd, RunAgentsStartProxy, "start-proxy",
		"Run a local facade that lets a coding-agent CLI drive a hosted session",
		`Starts a local WebSocket server that impersonates a coding-agent's own app-server protocol, so the unmodified CLI can attach to a hosted session as if it were a local backend.

`+"`"+`--type`+"`"+` selects which protocol to impersonate (v1: `+"`"+`codex`+"`"+` only; future agents get their own facade behind this same command, not a new command). Once `+"`"+`start-proxy`+"`"+` is listening, connect the real CLI, e.g. `+"`"+`codex --remote ws://127.0.0.1:1144`+"`"+`.

Tested against `+"`"+`codex-cli `+codex.TestedVersion+"`"+`. Codex's WS/app-server transport is officially experimental and can change without notice — re-verify the protocol capture on every codex upgrade before trusting this against a newer CLI.

Run only one of the proxy and `+"`"+`doctl agents attach`+"`"+` per session from the same machine: both stream as this device, so the newer one takes the session over and the older stops. Close one before opening the other.`,
		Writer)
	AddStringFlag(cmdStartProxy, doctl.ArgAgentProxyType, "", "codex", "Coding-agent protocol to impersonate (v1: codex)")
	AddStringFlag(cmdStartProxy, doctl.ArgAgentProxySession, "", "", "Session ID or name to bridge to", requiredOpt())
	AddIntFlag(cmdStartProxy, doctl.ArgAgentProxyPort, "", 1144, "Local port to listen on")
	AddBoolFlag(cmdStartProxy, doctl.ArgAgentProxyReplay, "", false, "Replay the session's event history into the first thread on connect")
	cmdStartProxy.Example = `doctl agents start-proxy --type codex --session my-session --port 1144`

	cmdAttach := CmdBuilder(cmd, RunAgentsAttach, "attach <session>",
		"Attach to an agent session",
		`Opens an interactive line-mode TUI on an existing session. Streams events from the server and accepts typed input. If the SSE connection drops, doctl shows Reconnecting... and retries automatically (5 attempts with backoff). If reconnection fails, it prints an error and stops the stream.

When a HITL approval is pending, the prompt switches to a compact approve/reject/defer menu showing the command awaiting approval. In an interactive terminal you can move the highlight with the arrow keys and press Enter, or resolve directly with a single keystroke -- no Enter required: `+"`"+`y`+"`"+`/`+"`"+`a`+"`"+` approves, `+"`"+`n`+"`"+`/`+"`"+`r`+"`"+` rejects, `+"`"+`d`+"`"+` defers. Piped input (CI / scripts) must send the letter word (`+"`"+`yes`+"`"+`/`+"`"+`no`+"`"+`/`+"`"+`defer`+"`"+`) followed by a newline. The explicit `+"`"+`/a <request-id>`+"`"+`, `+"`"+`/r <request-id>`+"`"+`, `+"`"+`/d <request-id>`+"`"+` slash commands still work; type `+"`"+`/help`+"`"+` to see them. Ctrl-D detaches without destroying the session.`,
		Writer, aliasOpt("chat"))
	cmdAttach.Example = `doctl agents attach sess_abc123; doctl agents attach my-session-name`

	cmdList := CmdBuilder(cmd, RunAgentsList, "list",
		"List agent sessions",
		`Lists agent sessions visible to the caller. Supports pagination and filtering via `+"`"+`--page-size`+"`"+`, `+"`"+`--page-token`+"`"+`, `+"`"+`--status`+"`"+`, and `+"`"+`--name`+"`"+`. When more pages exist, the next page token is printed after the table.`,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HostedAgentSession{}))
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of sessions to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	AddStringFlag(cmdList, doctl.ArgAgentStatus, "", "", "Filter by session status (e.g. SESSION_STATUS_READY, SESSION_STATUS_DESTROYED)")
	AddStringFlag(cmdList, doctl.ArgAgentName, "", "", "Filter by session name")
	cmdList.Example = `doctl agents list --page-size 10 --status SESSION_STATUS_READY; doctl agents list --name demo-agent`

	CmdBuilder(cmd, RunAgentsShow, "show <session>",
		"Show a single agent session",
		"Prints details of one agent session.",
		Writer, aliasOpt("get"),
		displayerType(&displayers.HostedAgentSession{}))

	cmdLogs := CmdBuilder(cmd, RunAgentsLogs, "logs <session>",
		"Replay the event history for a session",
		`Replays the server-side event history for a session, then exits.

The server serves only the most recent stretch of a long transcript per request, so doctl reads the rest a page at a time, walking backwards until the session's history is exhausted. That paging is automatic; use `+"`"+`--tail`+"`"+` to stop early and print only the newest events, or `+"`"+`--page-size`+"`"+` to change how many events each request asks for.

History is also retained for a bounded window, which paging cannot recover: events that have aged out of a long-idle session are gone from the server, not merely unread.

`+"`"+`--before`+"`"+` reads a single page of the events older than one you already have, instead of walking the whole history, and reports the id to pass as the next `+"`"+`--before`+"`"+`.`,
		Writer)
	AddIntFlag(cmdLogs, doctl.ArgAgentLogsTail, "", 0, "Print only the newest N events (0 replays all retained history)")
	AddIntFlag(cmdLogs, doctl.ArgAgentPageSize, "", do.DefaultHistoryPageSize, "Maximum number of events to request per page while walking history backwards")
	AddStringFlag(cmdLogs, doctl.ArgAgentLogsBefore, "", "", "Print one page of the events older than this event ID, then exit")
	cmdLogs.Example = `doctl agents logs sess_abc123 --tail 100; doctl agents logs my-session --before evt_01hz... --page-size 50`

	CmdBuilder(cmd, RunAgentsApprove, "approve <session> <request-id> <approve|reject|defer>",
		"Resolve a pending HITL request out of band",
		"Approves, rejects, or defers a pending HITL request without attaching the interactive TUI. The resolution source is recorded as `RESOLUTION_SOURCE_OUT_OF_BAND`. Inside an attached session, the same outcomes are available as `/a`, `/r`, `/d` slash commands.",
		Writer)

	CmdBuilder(cmd, RunAgentsDestroy, "destroy <session>",
		"Destroy an agent session",
		"Tears down the workspace sandbox and removes the session.",
		Writer, aliasOpt("rm"))

	cmdPause := CmdBuilder(cmd, RunAgentsPause, "pause <session>",
		"Pause an agent session",
		"Pauses a running agent session. The sandbox is preserved and the session can be resumed later with `doctl agents resume`.",
		Writer)
	cmdPause.Example = `doctl agents pause sess_abc123`

	cmdResume := CmdBuilder(cmd, RunAgentsResume, "resume <session>",
		"Resume a paused agent session",
		"Resumes a previously paused agent session.",
		Writer)
	cmdResume.Example = `doctl agents resume sess_abc123`

	cmdUpload := CmdBuilder(cmd, RunAgentsUpload, "upload <session>",
		"Upload a file into a session workspace",
		`Uploads a local file (or tar archive) into the session's sandbox workspace.

`+"`"+`--workspace-path`+"`"+` is resolved inside the workspace root (`+"`"+`/workspace`+"`"+`); a path that escapes the root is rejected by the server. Pass `+"`"+`--archive`+"`"+` when the local file is a tar that the server should extract at the destination. doctl computes the SHA-256 of the payload and forwards it so the guest can verify the upload. All file sizes use the workspace transfer API (multipart upload for large payloads). Maximum size is 50 GiB.`,
		Writer,
		displayerType(&displayers.HostedAgentWorkspaceUpload{}))
	AddStringFlag(cmdUpload, doctl.ArgAgentWorkspacePath, "", "", "Destination path inside the workspace root (/workspace)", requiredOpt())
	AddStringFlag(cmdUpload, doctl.ArgAgentLocalFile, "", "", "Path to the local file to upload", requiredOpt())
	AddBoolFlag(cmdUpload, doctl.ArgAgentArchive, "", false, "Treat the local file as a tar archive to extract at the destination")
	cmdUpload.Example = `doctl agents upload sess_abc123 --local-file ./main.go --workspace-path src/main.go`

	cmdDownload := CmdBuilder(cmd, RunAgentsDownload, "download <session>",
		"Download a file from a session workspace",
		`Downloads a file (or tar archive) from the session's sandbox workspace and writes it to a local destination.

`+"`"+`--workspace-path`+"`"+` is resolved inside the workspace root (`+"`"+`/workspace`+"`"+`). Pass `+"`"+`--archive`+"`"+` to download a directory as a tar archive. All file sizes use the workspace transfer API: doctl polls for a presigned download URL, fetches the object directly, and verifies SHA-256 from the transfer status. Maximum size is 50 GiB.`,
		Writer)
	AddStringFlag(cmdDownload, doctl.ArgAgentWorkspacePath, "", "", "Source path inside the workspace root (/workspace)", requiredOpt())
	AddStringFlag(cmdDownload, doctl.ArgAgentSaveTo, "", "", "Local file path to write the download to", requiredOpt())
	AddBoolFlag(cmdDownload, doctl.ArgAgentArchive, "", false, "Tar-stream the directory at the source path")
	cmdDownload.Example = `doctl agents download sess_abc123 --workspace-path src/main.go --save-to ./main.go`

	cmd.AddCommand(AgentTriggers())

	return cmd
}

// --- runners ----------------------------------------------------------------

// RunAgentsStart creates a new hosted agent session by uploading an agent
// manifest verbatim.
func RunAgentsStart(c *CmdConfig) error {
	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAgentSpec)
	if err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	if err != nil {
		return err
	}

	manifest, err := readManifest(os.Stdin, specPath)
	if err != nil {
		return err
	}
	// --name is a convenience that sets the manifest's metadata.name. When it's
	// omitted the manifest is sent verbatim and the server auto-generates a name.
	manifest, err = injectManifestName(manifest, name)
	if err != nil {
		return err
	}

	sess, err := c.HostedAgents().CreateSessionFromManifest(manifest)
	if err != nil {
		if sessionLimitErr(err) {
			msg, _, _ := agentAPIError(err)
			return fmt.Errorf("%s. Free a slot by destroying one: run `doctl agents list` to find a session ID, then `doctl agents destroy SESSION_ID`", strings.TrimRight(msg, "."))
		}
		return err
	}
	return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}})
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
	if _, err := svc.GetSession(sessionID); err != nil {
		return fmt.Errorf("session %q not found: %w", sessionRef, err)
	}

	fmt.Fprintf(c.Out, "Proxying session %s as a codex app-server on ws://127.0.0.1:%d\n", sessionID, port)
	fmt.Fprintf(c.Out, "Connect with: codex --remote ws://127.0.0.1:%d\n", port)

	// SIGTERM alongside SIGINT: under a process manager or plain `kill` (not
	// `-9`), only handling os.Interrupt meant the graceful-shutdown path in
	// ServeListener never triggered — the process just died abruptly instead.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Fresh Facade per connection (see ServeListener): per-connection state
	// (notifier, in-flight turns, event loop, --replay gate) must not leak
	// across a disconnect/reconnect. A reconnecting client is a new TUI with
	// empty scrollback — --replay must run again for that connection.
	return agentproxy.Serve(ctx, port, func() agentproxy.Facade {
		return &codex.Facade{
			SessionID: sessionID,
			Sessions:  svc,
			Replay:    replay,
		}
	})
}

// readManifest returns the spec file as raw bytes with ${VAR} references
// resolved from the local environment (see expandManifestEnv). path "-" reads
// from stdin. Beyond env expansion, the only client-side validation is
// "non-empty after trim" so a stray `--spec /dev/null` fails fast instead of
// hitting the server.
func readManifest(stdin io.Reader, path string) ([]byte, error) {
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
	return expandManifestEnv(raw)
}

// manifestEnvRef matches ${VAR} env references and their $${VAR} escape form.
// Only the strict braced form expands: bare $VAR is left alone so shell
// snippets embedded in manifests (skills instructions, prompts) survive.
var manifestEnvRef = regexp.MustCompile(`\$?\$\{[A-Za-z_][A-Za-z0-9_]*\}`)

// expandManifestEnv resolves ${VAR} references in the manifest against the
// local environment before the manifest is sent to the server. $${VAR}
// escapes to a literal ${VAR}. Referencing a variable that is not set locally
// is an error rather than a silent empty substitution, so a missing key fails
// here instead of inside the sandbox.
func expandManifestEnv(manifest []byte) ([]byte, error) {
	var missing []string
	seen := map[string]bool{}
	out := manifestEnvRef.ReplaceAllFunc(manifest, func(m []byte) []byte {
		if bytes.HasPrefix(m, []byte("$$")) {
			return m[1:] // $${VAR} -> literal ${VAR}
		}
		name := string(m[2 : len(m)-1])
		val, ok := os.LookupEnv(name)
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

// injectManifestName sets metadata.name on the manifest to name. An empty name
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

	// yaml.v2 decodes nested mappings as map[any]any.
	meta, ok := doc["metadata"].(map[any]any)
	if !ok {
		meta = map[any]any{}
	}
	meta["name"] = name
	doc["metadata"] = meta

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
	if err := c.Display(&displayers.HostedAgentSession{Sessions: sessions}); err != nil {
		return err
	}
	if nextPageToken != "" {
		if Output == "json" {
			fmt.Fprintf(os.Stderr, "Next page token: %s\n", nextPageToken)
		} else {
			fmt.Fprintf(c.Out, "Next page token: %s\n", nextPageToken)
		}
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
	if pageSize == 0 && pageToken == "" && status == "" && name == "" {
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
		return "", fmt.Errorf("no agent session goes by the name %q; pass a session ID or run `doctl agents list` to see available sessions", ref)
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
	return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}})
}

// RunAgentsDestroy tears down a session.
func RunAgentsDestroy(c *CmdConfig) error {
	sessionID, err := sessionIDArg(c)
	if err != nil {
		return err
	}
	if err := c.HostedAgents().DestroySession(sessionID); err != nil {
		return err
	}
	notice("Session %s destroyed", sessionID)
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
	notice("Session %s paused", sessionID)
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
	notice("Session %s resumed", sessionID)
	return nil
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
	return c.Display(&displayers.HostedAgentWorkspaceUpload{Uploads: []*godo.HostedAgentWorkspaceUploadResponse{{
		Path:         workspacePath,
		BytesWritten: written,
	}}})
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

	notice("Downloaded %d bytes to %s", written, saveTo)
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
	notice("HITL request %s resolved as %s", requestID, outcome)
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

// RunAgentsLogs replays the session's stored event history, then exits.
//
// One replay request only covers the newest stretch of a long transcript, so
// the rest is read by walking backwards a page at a time (see
// do.LoadSessionHistory). History is therefore buffered before anything is
// printed: the oldest events arrive last and have to be put back in order
// first.
func RunAgentsLogs(c *CmdConfig) error {
	sessionID, err := sessionIDArg(c)
	if err != nil {
		return err
	}
	tail, err := c.Doit.GetInt(c.NS, doctl.ArgAgentLogsTail)
	if err != nil {
		return err
	}
	if tail < 0 {
		return fmt.Errorf("--%s must not be negative", doctl.ArgAgentLogsTail)
	}
	pageSize, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPageSize)
	if err != nil {
		return err
	}
	if pageSize < 0 {
		return fmt.Errorf("--%s must not be negative", doctl.ArgAgentPageSize)
	}
	before, err := c.Doit.GetString(c.NS, doctl.ArgAgentLogsBefore)
	if err != nil {
		return err
	}
	stylingEnabled = detectStyling()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if before != "" {
		return runAgentsLogsPage(ctx, c, sessionID, before, pageSize)
	}

	history, err := do.LoadSessionHistory(ctx, c.HostedAgents(), sessionID, do.SessionHistoryOptions{
		PageSize:  pageSize,
		MaxEvents: tail,
	})
	if err != nil {
		return err
	}
	renderEventHistory(c.Out, history)
	return nil
}

// runAgentsLogsPage serves `agents logs --before`: one page of the events
// older than that id, no backward walk. The cursor for the page before it goes
// to stderr, so a caller stepping through history keeps a clean stdout.
func runAgentsLogsPage(ctx context.Context, c *CmdConfig, sessionID, before string, pageSize int) error {
	page, hasMore, err := do.LoadSessionHistoryPage(ctx, c.HostedAgents(), sessionID, before, pageSize)
	if err != nil {
		return err
	}
	renderEventHistory(c.Out, page)

	if !hasMore {
		fmt.Fprintln(os.Stderr, "Reached the beginning of this session's history.")
		return nil
	}
	if len(page) > 0 && page[0].EventID != "" {
		fmt.Fprintf(os.Stderr, "Older events remain. Next page: --before %s\n", page[0].EventID)
	}
	return nil
}

// renderEventHistory prints already-collected history events in order.
func renderEventHistory(out io.Writer, events []godo.HostedAgentEvent) {
	// TOKEN_CHUNK events stream token-by-token; buffer them so the whole
	// assistant message renders as markdown at once. Other event kinds are
	// discrete: flush any buffered message first, then print with a header.
	acc := &msgAccumulator{}
	for _, ev := range events {
		if ev.Kind == godo.HostedAgentEventKindTokenChunk {
			var p tokenChunkPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil {
				acc.add(p.Text)
			}
			continue
		}
		acc.flush(out)
		fmt.Fprintf(out, "[%s] %s\n", ev.At.Time.UTC().Format("2006-01-02T15:04:05Z"), ev.Kind)
		// Show the full hitl_id so request ids are copyable for out-of-band approve.
		renderEvent(out, ev)
	}
	acc.flush(out)
}

// RunAgentsAttach opens the interactive TUI: one goroutine drains the SSE
// stream (with auto-reconnect), the main goroutine reads stdin.
func RunAgentsAttach(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	svc := c.HostedAgents()
	sessionID, err := resolveSessionRef(svc, c.Args[0])
	if err != nil {
		return err
	}

	stylingEnabled = detectStyling()

	sess, err := svc.GetSession(sessionID)
	if err != nil {
		if msg, terminal := classifyStreamError(err); terminal {
			return errors.New(strings.TrimSpace(msg))
		}
		return err
	}
	if isTerminalSessionStatus(sess.Status) {
		return fmt.Errorf("session %s cannot be attached (status: %s)", sessionID, humanSessionStatus(sess.Status))
	}

	pending := &pendingHITL{}
	cursor := &eventCursor{}
	state := newAttachState(c.Out, pending)

	// All writes flow through the display so events don't clobber the user's
	// in-progress input once raw mode is on. Pass-through until raw=true.
	originalOut := c.Out
	c.Out = state.display
	defer func() { c.Out = originalOut }()

	printAttachBanner(c.Out, sessionID, sess.AgentKind)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thinking := newThinkingState(c.Out)
	defer thinking.stop()

	go streamWithReconnect(ctx, svc, sessionID, c.Out, pending, cursor, thinking)

	return runAttach(c, svc, sessionID, os.Stdin, state)
}

// runAttach dispatches to the raw-mode TTY loop or the legacy bufio line-mode
// loop based on whether stdin is an interactive terminal.
func runAttach(c *CmdConfig, svc do.HostedAgentsService, sessionID string, in io.Reader, state *attachState) error {
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return attachLoopTTY(c, svc, sessionID, f, state)
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
	display.spinnerInit(spinnerFrames[0])
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
			d.spinnerFrame(spinnerFrames[ix])
		}
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
		superseded := drainStream(stream, out, pending, cursor, thinking, dedup)
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
	var er *godo.ErrorResponse
	if !errors.As(err, &er) || er.Response == nil {
		return "", false
	}
	switch er.Response.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Sprintf("\nAuthentication failed: %s\nRun `doctl auth init` and try again.", er.Message), true
	case http.StatusForbidden:
		return fmt.Sprintf("\nAccess denied: %s", er.Message), true
	case http.StatusNotFound:
		return fmt.Sprintf("\nSession not found: %s", er.Message), true
	case http.StatusConflict:
		// V0 single-connection rejection. er.Message carries device + when.
		return fmt.Sprintf("\nSession already attached on another device: %s\nDetach there first, then re-run `doctl agents attach`.", er.Message), true
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
func drainStream(stream *godo.HostedAgentSessionStream, out io.Writer, pending *pendingHITL, cursor *eventCursor, thinking *thinkingState, dedup *tokenDeduper) (superseded bool) {
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
		if ev.Kind == hostedAgentEventKindStreamState {
			var st hostedAgentStreamState
			if err := json.Unmarshal(ev.Payload, &st); err == nil && st.State == hostedAgentStreamStateSuperseded {
				thinking.stop()
				acc.flush(out)
				flushAwaitingApproval(out, &awaiting)
				fmt.Fprintf(out, "\n%s\n", msgSuperseded)
				return true
			}
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
			thinking.stop()
			acc.flush(out)
			flushAwaitingApproval(out, &awaiting)
			dedup.reset()
			renderEvent(out, ev)
			thinking.start()
		case godo.HostedAgentEventKindTokenChunk:
			var p tokenChunkPayload
			if err := json.Unmarshal(ev.Payload, &p); err == nil && dedup.allow(p.Text) {
				acc.add(p.Text)
			}
		case godo.HostedAgentEventKindHITLRequested:
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
		default:
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
				return nil
			case hitlKeyIgnore:
				fmt.Fprintln(c.Out, "(press y, n, or d to resolve the pending approval, or Ctrl-D to detach)")
				continue
			case hitlKeyFallback:
				// Non-TTY — fall through to line mode.
			}
		}

		line, err := reader.ReadString('\n')
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
		// Ack immediately; the first agent token can be tens of seconds away
		// and without this users re-submit, spawning a duplicate run.
		if _, err := svc.SendInput(sessionID, &godo.HostedAgentSendInputRequest{Text: line}); err != nil {
			if isRunTerminalErr(err) {
				fmt.Fprintln(c.Out, "\nThis session's run has ended and can't accept new input.")
				fmt.Fprintln(c.Out, "Start a new session:  doctl agents start --spec <your-spec>.yaml")
				fmt.Fprintln(c.Out, "(detaching)")
				return nil
			}
			fmt.Fprintf(c.Out, "send failed: %v\n", err)
			continue
		}
		fmt.Fprintln(c.Out, colorize("… waiting for the agent", colMuted))
	}
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
	pending *pendingHITL
	display *promptDisplay
	mu      sync.Mutex // guards lineBuf, cursor, and hitlSel
	lineBuf []byte
	cursor  int // byte index into lineBuf (ASCII-only input today)
	hitlSel int // highlighted HITL menu option: 0 approve, 1 reject, 2 defer
	esc     int // arrow-key escape-sequence parse state (0 none, 1 ESC, 2 ESC[)
}

func newAttachState(out io.Writer, pending *pendingHITL) *attachState {
	s := &attachState{pending: pending}
	s.display = &promptDisplay{
		out:    out,
		prompt: s.promptString,
		lineBuf: func() string {
			s.mu.Lock()
			defer s.mu.Unlock()
			return string(s.lineBuf)
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
	n := s.pending.len()
	if n == 0 {
		return "> "
	}
	s.mu.Lock()
	sel := s.hitlSel
	s.mu.Unlock()
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
	p.paintPromptLocked(true)
}

// spinnerInit reserves the spinner's own line above the prompt and draws the
// first frame, then redraws the prompt below it — all in one locked write so
// the spinner row and the prompt row stay adjacent for spinnerFrame/spinnerStop.
func (p *promptDisplay) spinnerInit(frame string) {
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
	fmt.Fprintf(p.out, "%s thinking...\r\n", frame)
	p.paintPromptLocked(false)
}

// spinnerFrame redraws the spinner line one row above the prompt.
// DECSC/DECRC (\x1b7 / \x1b8) save+restore the cursor so the prompt row
// below is preserved. No-op in non-raw or mid-stream state.
func (p *promptDisplay) spinnerFrame(frame string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw || p.midLine {
		return
	}
	fmt.Fprintf(p.out, "\x1b7\x1b[A\r\x1b[K%s thinking...\x1b8", frame)
}

// spinnerStop erases the spinner line entirely so no "(thinking...)" text or
// frozen braille glyph is left behind in scrollback once the run produces output.
func (p *promptDisplay) spinnerStop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.raw || p.midLine {
		return
	}
	fmt.Fprint(p.out, "\x1b7\x1b[A\r\x1b[K\x1b8")
}

// attachLoopTTY runs the raw-mode byte-by-byte input state machine. A 50ms
// ticker polls pending-HITL so a HITL event flips the prompt instantly,
// without needing the user to press Enter to "wake up" the loop.
func attachLoopTTY(c *CmdConfig, svc do.HostedAgentsService, sessionID string, f *os.File, state *attachState) error {
	fd := int(f.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Raw mode unavailable; fall back to bufio line mode.
		return attachLoop(c, svc, sessionID, f, state)
	}
	defer term.Restore(fd, oldState)

	state.display.setRaw(true)
	defer state.display.setRaw(false)

	state.display.redraw()

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
			stop, err := handleAttachByte(c, svc, sessionID, b, state)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		case <-ticker.C:
		case err := <-readErrCh:
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// handleAttachByte is the per-byte state machine. stop=true exits the loop
// (Ctrl-C, Ctrl-D on empty line).
func handleAttachByte(c *CmdConfig, svc do.HostedAgentsService, sessionID string, b byte, state *attachState) (stop bool, err error) {
	// Arrow-key escape sequences arrive as three bytes: ESC, '[' (or 'O'), then
	// A/B/C/D. With a pending HITL they move the approve/reject/defer highlight;
	// otherwise left/right move the text-input caret.
	if state.esc == 2 {
		state.esc = 0
		switch {
		case state.pending.get() != "":
			switch b {
			case 'A', 'D': // up / left
				state.moveHITLSelection(-1)
				state.display.redraw()
			case 'B', 'C': // down / right
				state.moveHITLSelection(1)
				state.display.redraw()
			}
		case b == 'D': // left
			if state.moveLineCursor(-1) {
				state.display.redraw()
			}
		case b == 'C': // right
			if state.moveLineCursor(1) {
				state.display.redraw()
			}
			// up/down ignored for text input (no history yet)
		}
		return false, nil
	}
	if state.esc == 1 {
		state.esc = 0
		if b == '[' || b == 'O' {
			state.esc = 2
			return false, nil
		}
		// Lone ESC or unrecognized sequence: fall through and handle b normally.
	}
	if b == 0x1b { // ESC — start of a possible arrow sequence
		state.esc = 1
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
		state.mu.Lock()
		line := strings.TrimSpace(string(state.lineBuf))
		state.lineBuf = state.lineBuf[:0]
		state.cursor = 0
		state.mu.Unlock()
		state.display.echo([]byte("\r\n"))
		if line != "" {
			if detach := processAttachLine(c, svc, sessionID, line, state); detach {
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
			if atEnd {
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
		return true, nil
	case 0x04: // Ctrl-D
		state.mu.Lock()
		empty := len(state.lineBuf) == 0
		state.mu.Unlock()
		if empty {
			state.display.echo([]byte("\r\n"))
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
			if atEnd {
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
func processAttachLine(c *CmdConfig, svc do.HostedAgentsService, sessionID, line string, state *attachState) (detach bool) {
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
			fmt.Fprintln(c.Out, "\nThis session's run has ended and can't accept new input.")
			fmt.Fprintln(c.Out, "Start a new session:  doctl agents start --spec <your-spec>.yaml")
			fmt.Fprintln(c.Out, "(detaching)")
			return true
		}
		fmt.Fprintf(c.Out, "send failed: %v\n", err)
		return false
	}
	fmt.Fprintln(c.Out, colorize("… waiting for the agent", colMuted))
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

// printAttachBanner renders the styled connection header and a compact help
// key shown when an interactive session is attached.
func printAttachBanner(w io.Writer, sessionID string, kind godo.HostedAgentKind) {
	fmt.Fprintf(w, "\n%s  %s\n", boldColor("● Connected", colSuccess), colorize(sessionID, colHighlight))
	fmt.Fprintf(w, "  %s\n\n", colorize(fmt.Sprintf("agent %s · Ctrl-D to detach", prettyAgentKind(kind)), colMuted))

	row := func(label, desc string) {
		fmt.Fprintf(w, "  %s  %s\n", boldColor(fmt.Sprintf("%-8s", label), colHighlight), desc)
	}
	yn := colorize("y", colSuccess) + colorize("/a", colSuccess) + " approve · " +
		colorize("n", colError) + colorize("/r", colError) + " reject · " +
		colorize("d", colWarning) + " defer"
	row("send", "type a message and press Enter")
	row("approve", "↑/↓ then Enter, or "+yn)
	row("help", "type "+boldColor("/help", colHighlight)+" for the full command list")
	fmt.Fprintln(w)
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
