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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// Agents creates the `doctl agents` command tree.
func Agents() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "agents",
			Aliases: []string{"agent"},
			Short:   "Launch and manage hosted DigitalOcean agent sessions",
			Long: `The ` + "`" + `doctl agents` + "`" + ` commands manage hosted coding-agent sessions running in DigitalOcean sandboxes.

A session is one long-lived agent process (Claude Code, OpenCode, ...) running inside a workspace sandbox. doctl drives it: starting it, attaching an interactive TUI, listing existing sessions, resolving HITL approvals out of band, and tearing it down.`,
			GroupID: manageResourcesGroup,
		},
	}

	cmdStart := CmdBuilder(cmd, RunAgentsStart, "start",
		"Start a new agent session",
		`Creates a new agent session and prints its session id and status.

The `+"`"+`--agent`+"`"+` flag selects the agent kind (`+"`"+`claude-code`+"`"+` or `+"`"+`opencode`+"`"+`). The optional `+"`"+`--repo`+"`"+` flag passes a repo hint to the agent's initial environment.

Alternatively, pass `+"`"+`--spec`+"`"+` with a YAML or JSON file describing the session. Any `+"`"+`--agent`+"`"+` / `+"`"+`--repo`+"`"+` flag given alongside `+"`"+`--spec`+"`"+` overrides the matching field in the file.`,
		Writer, aliasOpt("deploy"))
	AddStringFlag(cmdStart, "agent", "", "claude-code", "Agent kind: claude-code | opencode")
	AddStringFlag(cmdStart, "repo", "", "", "Optional repo hint (e.g. owner/repo)")
	AddStringFlag(cmdStart, doctl.ArgAgentSpec, "", "", `Path to an agent spec in JSON or YAML format. Set to "-" to read from stdin.`)
	cmdStart.Example = `doctl agents start --agent claude-code --repo acme/payments
doctl agents start --spec agent-spec.yaml`

	cmdAttach := CmdBuilder(cmd, RunAgentsAttach, "attach <session-id>",
		"Attach to an agent session",
		`Opens an interactive line-mode TUI on an existing session. Streams events from the server and accepts typed input.

Type `+"`"+`/help`+"`"+` once attached to see the inline command list. Pending HITL prompts can be resolved with `+"`"+`/a <request-id>`+"`"+`, `+"`"+`/r <request-id>`+"`"+`, or `+"`"+`/d <request-id>`+"`"+`. Ctrl-D detaches without destroying the session.`,
		Writer, aliasOpt("chat"))
	cmdAttach.Example = `doctl agents attach sess_abc123`

	cmdList := CmdBuilder(cmd, RunAgentsList, "list",
		"List agent sessions",
		"Lists all agent sessions visible to the caller.",
		Writer, aliasOpt("ls"))
	AddStringFlag(cmdList, doctl.ArgFormat, "", "", "Columns for output in a comma-separated list. Possible values: `text`")

	CmdBuilder(cmd, RunAgentsShow, "show <session-id>",
		"Show a single agent session",
		"Prints the JSON representation of one agent session.",
		Writer, aliasOpt("get"))

	CmdBuilder(cmd, RunAgentsLogs, "logs <session-id>",
		"Replay the full event history for a session",
		"Replays the full server-side event history for a session, then exits.",
		Writer)

	CmdBuilder(cmd, RunAgentsApprove, "approve <session-id> <request-id> <approve|reject|defer>",
		"Resolve a pending HITL request out of band",
		"Approves, rejects, or defers a pending HITL request without attaching the interactive TUI. The resolution source is recorded as `RESOLUTION_SOURCE_OUT_OF_BAND`.",
		Writer)

	cmd.AddCommand(agentsAuth())

	CmdBuilder(cmd, RunAgentsDestroy, "destroy <session-id>",
		"Destroy an agent session",
		"Tears down the workspace sandbox and removes the session.",
		Writer, aliasOpt("rm"))

	return cmd
}

func agentsAuth() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "auth",
			Short: "Authorize external providers for a session",
			Long:  "Manage per-session OAuth links (currently only `github`). Tokens are stored server-side; the developer never sees a token value.",
		},
	}

	cmdGitHub := CmdBuilder(cmd, RunAgentsAuthGitHub, "github",
		"Authorize GitHub for a session",
		`Starts the GitHub OAuth flow for an existing session, opens the authorize URL in the default browser, and waits for the server-side completion event.

The command returns as soon as the session reports `+"`"+`provider_auth["github"] == PROVIDER_AUTH_STATE_AUTHORIZED`+"`"+`.`,
		Writer)
	AddStringFlag(cmdGitHub, "session", "", "", "Session id", requiredOpt())
	AddBoolFlag(cmdGitHub, "no-open", "", false, "Do not auto-open the authorize URL in a browser")
	AddBoolFlag(cmdGitHub, "no-wait", "", false, "Return as soon as the authorize URL is known; do not wait for completion")
	cmdGitHub.Example = `doctl agents auth github --session sess_abc123`

	return cmd
}

// --- runners ----------------------------------------------------------------

// RunAgentsStart creates a new hosted agent session.
//
// Inputs come from one of two sources:
//   - `--spec <path>`: parses a YAML/JSON file matching godo.HostedAgentSessionCreateRequest.
//   - `--agent` + `--repo`: build the request inline from individual flags.
//
// If both are provided, the flags override matching fields in the spec.
func RunAgentsStart(c *CmdConfig) error {
	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAgentSpec)
	if err != nil {
		return err
	}
	agent, err := c.Doit.GetString(c.NS, "agent")
	if err != nil {
		return err
	}
	repo, err := c.Doit.GetString(c.NS, "repo")
	if err != nil {
		return err
	}

	var req *godo.HostedAgentSessionCreateRequest
	if specPath != "" {
		req, err = readAgentSpec(os.Stdin, specPath)
		if err != nil {
			return err
		}
		// Flags override spec fields when both are provided. Note: --agent has
		// a default of "claude-code", so we only override when the user passed
		// it explicitly (i.e. the spec lacks an agent_kind).
		if req.AgentKind == "" {
			kind, err := agentKindFor(agent)
			if err != nil {
				return err
			}
			req.AgentKind = kind
		}
		if repo != "" {
			req.RepoHint = repo
		}
	} else {
		kind, err := agentKindFor(agent)
		if err != nil {
			return err
		}
		req = &godo.HostedAgentSessionCreateRequest{
			AgentKind: kind,
			RepoHint:  repo,
		}
	}

	sess, err := c.HostedAgents().CreateSession(req)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.Out, "Session created: %s\n", sess.SessionID)
	fmt.Fprintf(c.Out, "  Agent:   %s\n", sess.AgentKind)
	fmt.Fprintf(c.Out, "  Status:  %s\n", sess.Status)
	if sess.SandboxID != "" {
		fmt.Fprintf(c.Out, "  Sandbox: %s\n", sess.SandboxID)
	}
	return nil
}

func agentKindFor(s string) (godo.HostedAgentKind, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "claude-code", "claude":
		return godo.HostedAgentKindClaudeCode, nil
	case "opencode":
		return godo.HostedAgentKindOpenCode, nil
	case "none":
		return godo.HostedAgentKindNone, nil
	default:
		return "", fmt.Errorf("unknown --agent value %q; expected claude-code or opencode", s)
	}
}

// RunAgentsList lists hosted agent sessions visible to the caller.
func RunAgentsList(c *CmdConfig) error {
	sessions, err := c.HostedAgents().ListSessions(nil)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		fmt.Fprintln(c.Out, "(no sessions)")
		return nil
	}
	fmt.Fprintf(c.Out, "%-26s %-22s %-28s %s\n", "SESSION", "AGENT", "STATUS", "CREATED")
	for _, s := range sessions {
		fmt.Fprintf(c.Out, "%-26s %-22s %-28s %s\n",
			s.SessionID, s.AgentKind, s.Status, s.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"))
	}
	return nil
}

// RunAgentsShow prints one session as indented JSON.
func RunAgentsShow(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sess, err := c.HostedAgents().GetSession(c.Args[0])
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(sess.HostedAgentSession, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(c.Out, string(out))
	return nil
}

// RunAgentsDestroy tears down a session.
func RunAgentsDestroy(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	if err := c.HostedAgents().DestroySession(c.Args[0]); err != nil {
		return err
	}
	notice("Session %s destroyed", c.Args[0])
	return nil
}

// RunAgentsApprove resolves a pending HITL request out of band.
func RunAgentsApprove(c *CmdConfig) error {
	if len(c.Args) < 3 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	if len(c.Args) > 3 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	sessionID, requestID := c.Args[0], c.Args[1]
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

// RunAgentsLogs replays the full event history for a session, then exits.
func RunAgentsLogs(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := c.HostedAgents().StreamSession(ctx, c.Args[0], &godo.HostedAgentSessionStreamOptions{
		ReplayOnly: true,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	for stream.Next() {
		ev := stream.Current()
		// TOKEN_CHUNK events stream token-by-token without trailing newlines so
		// they render as one continuous line; printing a per-event header here
		// would land mid-line. Other event kinds are discrete and get a header.
		if ev.Kind != godo.HostedAgentEventKindTokenChunk {
			fmt.Fprintf(c.Out, "[%s] %s\n", ev.At.Time.UTC().Format("2006-01-02T15:04:05Z"), ev.Kind)
		}
		renderEvent(c.Out, ev)
	}
	return stream.Err()
}

// RunAgentsAttach opens the interactive TUI for an existing session. One
// goroutine pumps the SSE iterator, the main goroutine reads stdin. Typed text
// becomes a SendInput call; `/a`, `/r`, `/d` followed by a request id resolves
// a HITL prompt; Ctrl-D detaches.
func RunAgentsAttach(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sessionID := c.Args[0]
	svc := c.HostedAgents()

	sess, err := svc.GetSession(sessionID)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.Out, "Connected to %s (%s)\n", sessionID, sess.AgentKind)
	if state, ok := sess.ProviderAuth["github"]; ok && state == godo.HostedAgentProviderAuthStateAuthorized {
		fmt.Fprintln(c.Out, "GitHub: authorized")
	} else {
		fmt.Fprintf(c.Out, "GitHub: not authorized (run `doctl agents auth github --session %s`)\n", sessionID)
	}
	fmt.Fprintln(c.Out, "Type a message and press Enter to send. Ctrl-D to detach. Type `/help` for HITL commands.")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pending := &pendingHITL{}
	stream, err := svc.StreamSession(ctx, sessionID, nil)
	if err != nil {
		return err
	}
	defer stream.Close()

	go func() {
		for stream.Next() {
			ev := stream.Current()
			switch ev.Kind {
			case godo.HostedAgentEventKindHITLRequested:
				var p hitlRequestedPayload
				if err := json.Unmarshal(ev.Payload, &p); err == nil {
					pending.set(p.Request.RequestID)
				}
			case godo.HostedAgentEventKindHITLResolved:
				var p hitlResolvedPayload
				if err := json.Unmarshal(ev.Payload, &p); err == nil {
					pending.clearIf(p.Decision.RequestID)
				}
			}
			renderEvent(c.Out, ev)
		}
	}()

	return attachLoop(c, svc, sessionID, os.Stdin, pending)
}

type pendingHITL struct {
	mu sync.Mutex
	id string
}

func (p *pendingHITL) set(id string) {
	p.mu.Lock()
	p.id = id
	p.mu.Unlock()
}

func (p *pendingHITL) get() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.id
}

func (p *pendingHITL) clearIf(id string) {
	p.mu.Lock()
	if p.id == id {
		p.id = ""
	}
	p.mu.Unlock()
}

func attachLoop(c *CmdConfig, svc do.HostedAgentsService, sessionID string, in io.Reader, pending *pendingHITL) error {
	reader := bufio.NewReader(in)
	for {
		fmt.Fprint(c.Out, "\n> ")
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
		if strings.HasPrefix(line, "/") {
			if err := handleAttachCommand(c, svc, sessionID, line, pending); err != nil {
				fmt.Fprintf(c.Out, "error: %v\n", err)
			}
			continue
		}
		if _, err := svc.SendInput(sessionID, &godo.HostedAgentSendInputRequest{Text: line}); err != nil {
			fmt.Fprintf(c.Out, "send failed: %v\n", err)
		}
	}
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
		fmt.Fprintln(c.Out, "  /a [request-id]   approve a pending HITL request (defaults to most recent)")
		fmt.Fprintln(c.Out, "  /r [request-id]   reject a pending HITL request")
		fmt.Fprintln(c.Out, "  /d [request-id]   defer a pending HITL request")
		return nil
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
	return svc.ResolveHITL(sessionID, id, &godo.HostedAgentResolveHITLRequest{
		Outcome: outcome,
		Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
	})
}

// RunAgentsAuthGitHub starts the GitHub OAuth flow for a session and (unless
// --no-wait) blocks until provider_auth["github"] flips to AUTHORIZED.
func RunAgentsAuthGitHub(c *CmdConfig) error {
	sessionID, err := c.Doit.GetString(c.NS, "session")
	if err != nil {
		return err
	}
	if sessionID == "" {
		return errors.New("--session is required")
	}
	noOpen, _ := c.Doit.GetBool(c.NS, "no-open")
	noWait, _ := c.Doit.GetBool(c.NS, "no-wait")

	svc := c.HostedAgents()
	resp, err := svc.StartOAuthFlow(sessionID, "github", &godo.HostedAgentStartOAuthFlowRequest{})
	if err != nil {
		return err
	}
	fmt.Fprintln(c.Out, "Initiating GitHub OAuth flow...")
	fmt.Fprintf(c.Out, "  -> %s\n", resp.AuthorizeURL)
	fmt.Fprintln(c.Out, "  (paste this URL if your browser did not open)")

	if !noOpen {
		_ = browser.OpenURL(resp.AuthorizeURL)
	}
	if noWait {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := svc.StreamSession(ctx, sessionID, nil)
	if err != nil {
		return err
	}
	defer stream.Close()

	fmt.Fprintln(c.Out, "Waiting for authorization...")
	for stream.Next() {
		ev := stream.Current()
		if ev.Kind != godo.HostedAgentEventKindSessionUpdated {
			continue
		}
		var p sessionUpdatedPayload
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			continue
		}
		if state, ok := p.SessionDelta.ProviderAuth["github"]; ok && state == godo.HostedAgentProviderAuthStateAuthorized {
			notice("GitHub authorized (token stored server-side, scope: session %s)", sessionID)
			return nil
		}
	}
	return stream.Err()
}

// --- event payload helpers --------------------------------------------------
//
// godo defines the generic HostedAgentEvent envelope but leaves the per-kind
// payload as json.RawMessage. The shapes below are local mirrors used only for
// rendering and are not on any HTTP wire — they exist to keep the rendering
// switch tidy.

type tokenChunkPayload struct {
	Text string `json:"text"`
}

type runStartedPayload struct {
	Run godo.HostedAgentRun `json:"run"`
}

type toolCallStartedPayload struct {
	ToolName string `json:"tool_name"`
}

type toolCallCompletedPayload struct {
	DurationMS int64  `json:"duration_ms"`
	Summary    string `json:"summary,omitempty"`
}

type hitlRequestedPayload struct {
	Request godo.HostedAgentHITLRequest `json:"request"`
}

type hitlResolvedPayload struct {
	Decision godo.HostedAgentHITLDecision `json:"decision"`
}

type runCompletedPayload struct {
	TotalTokensIn  int64 `json:"total_tokens_in"`
	TotalTokensOut int64 `json:"total_tokens_out"`
	RunCostMicros  int64 `json:"run_cost_micros"`
}

type runFailedPayload struct {
	Code    godo.HostedAgentRunFailureCode `json:"code"`
	Message string                         `json:"message,omitempty"`
}

type sessionUpdatedPayload struct {
	SessionDelta  godo.HostedAgentSession `json:"session_delta"`
	ChangedFields []string                `json:"changed_fields"`
}

func renderEvent(w io.Writer, ev godo.HostedAgentEvent) {
	switch ev.Kind {
	case godo.HostedAgentEventKindTokenChunk:
		var p tokenChunkPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprint(w, p.Text)
		}
	case godo.HostedAgentEventKindRunStarted:
		var p runStartedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n[run %s started]\n", p.Run.RunID)
		}
	case godo.HostedAgentEventKindToolCallStarted:
		var p toolCallStartedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n> %s ...\n", p.ToolName)
		}
	case godo.HostedAgentEventKindToolCallCompleted:
		var p toolCallCompletedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			if p.Summary != "" {
				fmt.Fprintf(w, "  %s (%dms)\n", p.Summary, p.DurationMS)
			} else {
				fmt.Fprintf(w, "  ok (%dms)\n", p.DurationMS)
			}
		}
	case godo.HostedAgentEventKindHITLRequested:
		var p hitlRequestedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			renderHITLRequest(w, p.Request)
		}
	case godo.HostedAgentEventKindHITLResolved:
		var p hitlResolvedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n[HITL %s -> %s]\n", p.Decision.RequestID, p.Decision.Outcome)
		}
	case godo.HostedAgentEventKindRunCompleted:
		var p runCompletedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n[run done: %d in / %d out tokens, $%.4f]\n",
				p.TotalTokensIn, p.TotalTokensOut, float64(p.RunCostMicros)/1_000_000)
		}
	case godo.HostedAgentEventKindRunFailed:
		var p runFailedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n[run failed: %s %s]\n", p.Code, p.Message)
		}
	case godo.HostedAgentEventKindSessionUpdated:
		var p sessionUpdatedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n[session updated: %v]\n", p.ChangedFields)
		}
	}
}

func renderHITLRequest(w io.Writer, req godo.HostedAgentHITLRequest) {
	fmt.Fprintln(w, "\n\n[HITL] Action requires approval:")
	switch req.Action {
	case godo.HostedAgentHITLActionBash:
		if cmd, ok := req.Details["command"].(string); ok {
			fmt.Fprintf(w, "  bash: %s\n", cmd)
		}
		if req.Workdir != "" {
			fmt.Fprintf(w, "  workdir: %s\n", req.Workdir)
		}
	case godo.HostedAgentHITLActionGitHubCreatePR:
		fmt.Fprintln(w, "  github.create_pr")
		for _, k := range []string{"title", "branch", "base", "repo"} {
			if v, ok := req.Details[k].(string); ok {
				fmt.Fprintf(w, "    %s: %s\n", k, v)
			}
		}
	default:
		fmt.Fprintf(w, "  action: %s\n", req.Action)
		for k, v := range req.Details {
			fmt.Fprintf(w, "    %s: %v\n", k, v)
		}
	}
	fmt.Fprintf(w, "  request_id: %s\n", req.RequestID)
	fmt.Fprintf(w, "  (resolve with `/a %s`, `/r %s`, or `/d %s`)\n", req.RequestID, req.RequestID, req.RequestID)
}

// readAgentSpec reads a hosted-agent session spec from a file path (or stdin
// when path is "-"), normalizes YAML to JSON, and strictly decodes it into a
// godo.HostedAgentSessionCreateRequest.
func readAgentSpec(stdin io.Reader, path string) (*godo.HostedAgentSessionCreateRequest, error) {
	var src io.Reader
	if path == "-" && stdin != nil {
		src = stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("opening spec: %s does not exist", path)
			}
			return nil, fmt.Errorf("opening spec: %w", err)
		}
		defer f.Close()
		src = f
	}

	raw, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("reading spec: %w", err)
	}

	jsonBytes, err := yaml.YAMLToJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(jsonBytes))
	dec.DisallowUnknownFields()

	var req godo.HostedAgentSessionCreateRequest
	if err := dec.Decode(&req); err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}
	return &req, nil
}
