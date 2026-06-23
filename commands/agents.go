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
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Agents creates the `doctl agents` command tree.
func Agents() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "agents",
			Aliases: []string{"agent"},
			Short:   "Launch and manage hosted DigitalOcean agent sessions",
			Long: `The ` + "`" + `doctl agents` + "`" + ` commands manage hosted coding-agent sessions running in DigitalOcean sandboxes.

A session is one long-lived agent process (Claude Code, OpenCode, ...) running inside a workspace sandbox. doctl drives it: starting it from an agent spec, attaching an interactive TUI, listing existing sessions, resolving HITL approvals out of band, and tearing it down.`,
			GroupID: hostedAgentsGroup,
		},
	}

	cmdStart := CmdBuilder(cmd, RunAgentsStart, "start",
		"Start a new agent session",
		`Creates a new agent session from an agent manifest file and prints its session id and status.

The `+"`"+`--spec`+"`"+` flag is required and accepts a YAML manifest matching the `+"`"+`agents.digitalocean.com/v1alpha1`+"`"+` schema. The manifest is sent verbatim to the server, which owns parsing and validation.`,
		Writer, aliasOpt("deploy"),
		displayerType(&displayers.HostedAgentSession{}))
	AddStringFlag(cmdStart, doctl.ArgAgentSpec, "", "", `Path to an agent manifest in YAML or JSON. Set to "-" to read from stdin.`, requiredOpt())
	cmdStart.Example = `doctl agents start --spec agent-spec.yaml`

	cmdAttach := CmdBuilder(cmd, RunAgentsAttach, "attach <session-id>",
		"Attach to an agent session",
		`Opens an interactive line-mode TUI on an existing session. Streams events from the server and accepts typed input.

Type `+"`"+`/help`+"`"+` once attached to see the inline command list. Pending HITL prompts can be resolved with `+"`"+`/a <request-id>`+"`"+`, `+"`"+`/r <request-id>`+"`"+`, or `+"`"+`/d <request-id>`+"`"+`. Ctrl-D detaches without destroying the session.`,
		Writer, aliasOpt("chat"))
	cmdAttach.Example = `doctl agents attach sess_abc123`

	cmdList := CmdBuilder(cmd, RunAgentsList, "list",
		"List agent sessions",
		`Lists agent sessions visible to the caller. Supports pagination and status filtering via `+"`"+`--page-size`+"`"+`, `+"`"+`--page-token`+"`"+`, and `+"`"+`--status`+"`"+`. When more pages exist, the next page token is printed after the table.`,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HostedAgentSession{}))
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of sessions to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	AddStringFlag(cmdList, doctl.ArgAgentStatus, "", "", "Filter by session status (e.g. SESSION_STATUS_READY, SESSION_STATUS_DESTROYED)")
	cmdList.Example = `doctl agents list --page-size 10 --status SESSION_STATUS_READY`

	CmdBuilder(cmd, RunAgentsShow, "show <session-id>",
		"Show a single agent session",
		"Prints details of one agent session.",
		Writer, aliasOpt("get"),
		displayerType(&displayers.HostedAgentSession{}))

	CmdBuilder(cmd, RunAgentsLogs, "logs <session-id>",
		"Replay the full event history for a session",
		"Replays the full server-side event history for a session, then exits.",
		Writer)

	CmdBuilder(cmd, RunAgentsApprove, "approve <session-id> <request-id> <approve|reject|defer>",
		"Resolve a pending HITL request out of band",
		"Approves, rejects, or defers a pending HITL request without attaching the interactive TUI. The resolution source is recorded as `RESOLUTION_SOURCE_OUT_OF_BAND`. Inside an attached session, the same outcomes are available as `/a`, `/r`, `/d` slash commands.",
		Writer)

	CmdBuilder(cmd, RunAgentsDestroy, "destroy <session-id>",
		"Destroy an agent session",
		"Tears down the workspace sandbox and removes the session.",
		Writer, aliasOpt("rm"))

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

	manifest, err := readManifest(os.Stdin, specPath)
	if err != nil {
		return err
	}

	sess, err := c.HostedAgents().CreateSessionFromManifest(manifest)
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}})
}

// readManifest returns the spec file as raw bytes. path "-" reads from stdin.
// The only client-side validation is "non-empty after trim" so a stray
// `--spec /dev/null` fails fast instead of hitting the server.
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
	return raw, nil
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
		fmt.Fprintf(c.Out, "Next page token: %s\n", nextPageToken)
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
	if pageSize == 0 && pageToken == "" && status == "" {
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
	return opt, nil
}

// RunAgentsShow prints one session.
func RunAgentsShow(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sess, err := c.HostedAgents().GetSession(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}})
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
					pending.set(p.HitlID)
				}
			case godo.HostedAgentEventKindHITLResolved:
				var p hitlResolvedPayload
				if err := json.Unmarshal(ev.Payload, &p); err == nil {
					pending.clearIf(p.HitlID)
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
		resp, err := svc.SendInput(sessionID, &godo.HostedAgentSendInputRequest{Text: line})
		if err != nil {
			fmt.Fprintf(c.Out, "send failed: %v\n", err)
			continue
		}
		// Acknowledge the submit right away. The agent runtime can take tens of
		// seconds to boot and produce its first token; without this line the
		// wait looks like a hang and users re-submit, spawning a second run.
		if resp != nil && resp.RunID != "" {
			fmt.Fprintf(c.Out, "(queued as %s; waiting for the agent...)\n", resp.RunID)
		} else {
			fmt.Fprintln(c.Out, "(queued; waiting for the agent...)")
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

// --- event payload helpers --------------------------------------------------
//
// godo decodes the SPI canonical event envelope and leaves the per-kind body
// (the wire's `data` object) as json.RawMessage on HostedAgentEvent.Payload.
// The shapes below mirror the SPI data structs (spi/events.go) for the kinds
// doctl renders; they exist to keep the rendering switch tidy.

type tokenChunkPayload struct {
	Text string `json:"text"`
}

type runStartedPayload struct {
	Agent string `json:"agent"`
}

type toolCallStartedPayload struct {
	Name string `json:"name"`
}

type toolCallCompletedPayload struct {
	OK         bool   `json:"ok"`
	DurationMS int64  `json:"duration_ms"`
	Summary    string `json:"summary,omitempty"`
}

type hitlRequestedPayload struct {
	HitlID  string         `json:"hitl_id"`
	Payload map[string]any `json:"payload"`
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
		var p runStartedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			if p.Agent != "" {
				fmt.Fprintf(w, "\n[run %s started (%s)]\n", ev.RunID, p.Agent)
			} else {
				fmt.Fprintf(w, "\n[run %s started]\n", ev.RunID)
			}
		}
	case godo.HostedAgentEventKindToolCallStarted:
		var p toolCallStartedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n> %s ...\n", p.Name)
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
			renderHITLRequest(w, p.HitlID, p.Payload)
		}
	case godo.HostedAgentEventKindHITLResolved:
		var p hitlResolvedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			fmt.Fprintf(w, "\n[HITL %s -> %s]\n", p.HitlID, hitlOutcomeLabel(p.Outcome))
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
			if p.Message != "" {
				fmt.Fprintf(w, "\n[run failed: %s (code %d)]\n", p.Message, p.Code)
			} else {
				fmt.Fprintf(w, "\n[run failed: code %d]\n", p.Code)
			}
		}
	case godo.HostedAgentEventKindSessionUpdated:
		fmt.Fprint(w, "\n[session updated]\n")
	}
}

// hitlOutcomeLabel maps the proto HITLOutcome int carried on the SPI
// human_input_received event to a human-readable verb.
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

func renderHITLRequest(w io.Writer, hitlID string, payload map[string]any) {
	fmt.Fprintln(w, "\n\n[HITL] Action requires approval:")
	if len(payload) > 0 {
		// MarshalIndent sorts map keys, so output is stable run-to-run.
		if b, err := json.MarshalIndent(payload, "  ", "  "); err == nil {
			fmt.Fprintf(w, "  %s\n", b)
		}
	}
	fmt.Fprintf(w, "  hitl_id: %s\n", hitlID)
	fmt.Fprintf(w, "  (resolve with `/a %s`, `/r %s`, or `/d %s`)\n", hitlID, hitlID, hitlID)
}
