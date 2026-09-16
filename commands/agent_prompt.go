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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// promptStderr carries progress and diagnostics. c.Out is reserved for the
// agent's answer alone, so `$(doctl harness-runtime prompt ...)` captures the
// answer and nothing else. A var so tests can capture it.
var promptStderr io.Writer = os.Stderr

// promptStdin is where a `-` prompt is read from. A var so tests can supply one.
var promptStdin io.Reader = os.Stdin

// promptExit ends doctl with the run's status, for the same reason execExit
// exists: doctl's shared checkErr can only ever exit 1, and teaching it more
// statuses would change behaviour for every other command.
var promptExit = os.Exit

// promptTimeoutExit matches GNU timeout(1), so a caller can tell "the agent is
// still working" apart from "the agent failed" without parsing stderr.
const promptTimeoutExit = 124

// Terminal states reported in the JSON envelope and used to pick an exit code.
const (
	promptStatusCompleted  = "completed"
	promptStatusFailed     = "failed"
	promptStatusTimedOut   = "timed_out"
	promptStatusIncomplete = "incomplete"
)

// RunAgentsPrompt sends one prompt to an existing session and prints the
// answer: headless execution, with no TUI and no keyboard input.
//
// The split between the streams is the contract. The answer goes to stdout on
// its own so it can be captured directly, and everything else — progress, tool
// calls, the token and cost summary — goes to stderr, where it stays visible on
// a terminal without corrupting a pipe.
func RunAgentsPrompt(c *CmdConfig) error {
	// args[0] is the session and the rest is the prompt, matching every other
	// session command in this tree. Taking the remainder rather than exactly
	// one argument means an unquoted prompt still works.
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	text, err := promptTextFrom(c.Args[1:])
	if err != nil {
		return err
	}

	outcome, err := agentOnHITLOutcome(c)
	if err != nil {
		return err
	}
	timeout, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPromptTimeout)
	if err != nil {
		return err
	}
	includeReasoning, err := c.Doit.GetBool(c.NS, doctl.ArgAgentPromptIncludeReasoning)
	if err != nil {
		return err
	}
	quiet, err := c.Doit.GetBool(c.NS, doctl.ArgAgentPromptQuiet)
	if err != nil {
		return err
	}

	svc := c.HostedAgents()
	sessionID, err := resolveSessionRef(svc, c.Args[0])
	if err != nil {
		return err
	}

	// SIGTERM alongside SIGINT so Ctrl-C or a plain `kill` stops the wait
	// instead of hanging. Giving up locally does not stop the agent: the run
	// continues server-side and its output stays readable with `logs`.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if timeout > 0 {
		var cancelTimeout context.CancelFunc
		ctx, cancelTimeout = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancelTimeout()
	}

	if err := promptResumeIfPaused(ctx, svc, sessionID, quiet); err != nil {
		return err
	}

	col := &promptCollector{
		svc:              svc,
		sessionID:        sessionID,
		text:             text,
		outcome:          outcome,
		includeReasoning: includeReasoning,
		quiet:            quiet,
		cursor:           &eventCursor{},
		seen:             map[string]bool{},
		resolved:         map[string]bool{},
	}

	streamErr := runHeadlessStream(ctx, svc, sessionID, col.cursor,
		func(msg string) { fmt.Fprintln(promptStderr, msg) },
		col.drain)
	if streamErr != nil {
		return streamErr
	}

	return col.report(c, ctx)
}

// promptTextFrom joins the prompt arguments, or reads the prompt from stdin
// when it is exactly `-`, matching how --spec already spells "from stdin".
func promptTextFrom(args []string) (string, error) {
	joined := strings.Join(args, " ")
	if joined != "-" {
		if strings.TrimSpace(joined) == "" {
			return "", errors.New("the prompt is empty; pass the text to send, or `-` to read it from stdin")
		}
		return joined, nil
	}

	raw, err := io.ReadAll(promptStdin)
	if err != nil {
		return "", fmt.Errorf("reading the prompt from stdin: %w", err)
	}
	text := strings.TrimRight(string(raw), "\n")
	if strings.TrimSpace(text) == "" {
		return "", errors.New("the prompt read from stdin was empty")
	}
	return text, nil
}

// promptResumeIfPaused resumes a paused session on the way in and waits for it
// to come back, matching `launch`. A headless caller cannot react to "the
// session happens to be asleep", so making it sequence `resume` itself would
// just be an error every unattended script had to handle.
func promptResumeIfPaused(ctx context.Context, svc do.HostedAgentsService, sessionID string, quiet bool) error {
	sess, err := svc.GetSession(sessionID)
	if err != nil {
		return beautifyAgentError(err)
	}
	if sess.Status != godo.HostedAgentSessionStatusPaused {
		return nil
	}

	if !quiet {
		fmt.Fprintf(promptStderr, "Session %s is paused — resuming…\n", displaySessionRef(sess))
	}
	if err := svc.ResumeSession(sessionID); err != nil {
		return beautifyAgentError(err)
	}
	// Sending input to a session that is awake but not yet listening loses the
	// prompt, so wait for READY rather than racing the resume.
	if _, err := waitForSessionReady(ctx, svc, sessionID, nil); err != nil {
		return beautifyAgentError(err)
	}
	return nil
}

// promptCollector accumulates one run's output off the event stream.
type promptCollector struct {
	svc              do.HostedAgentsService
	sessionID        string
	text             string
	outcome          godo.HostedAgentHITLOutcome
	includeReasoning bool
	quiet            bool

	cursor *eventCursor
	// seen guards against double-counting text: a reconnect replays from the
	// cursor and the boundary event can arrive twice, which would duplicate a
	// chunk of the answer.
	seen     map[string]bool
	resolved map[string]bool

	sent   bool
	runID  string
	answer strings.Builder
	think  strings.Builder

	done    bool
	status  string
	failure string
	usage   runCompletedPayload
}

// drain consumes one connection's events, sending the prompt on the first
// connection.
func (p *promptCollector) drain(stream *godo.HostedAgentSessionStream) (bool, error) {
	// Subscribe first, then send. StreamSession returns once the connection is
	// live, so sending here cannot miss the run's opening events. Sending
	// before subscribing could, and this command would then wait out its whole
	// timeout for a terminal event it had already missed.
	if !p.sent {
		resp, err := p.svc.SendInput(p.sessionID, &godo.HostedAgentSendInputRequest{Text: p.text})
		if err != nil {
			return false, beautifyAgentError(err)
		}
		p.sent = true
		p.runID = resp.RunID
		p.progress(fmt.Sprintf("run %s started", p.runID))
	}

	for stream.Next() {
		ev := stream.Current()
		p.cursor.set(ev.EventID)

		if ev.EventID != "" {
			if p.seen[ev.EventID] {
				continue
			}
			p.seen[ev.EventID] = true
		}
		// Connection health, not session activity.
		if ev.Kind == godo.HostedAgentEventKindStreamState {
			continue
		}
		// Another run in the same session must not contaminate this answer.
		// Events predating the send carry a different run, and a session can be
		// driven from elsewhere while this runs.
		if p.runID != "" && ev.RunID != "" && ev.RunID != p.runID {
			continue
		}

		if done, err := p.handle(ev); done || err != nil {
			return done, err
		}
	}
	return false, nil
}

// handle folds one event into the result.
func (p *promptCollector) handle(ev godo.HostedAgentEvent) (bool, error) {
	switch ev.Kind {
	case godo.HostedAgentEventKindTokenChunk:
		var q tokenChunkPayload
		if err := json.Unmarshal(ev.Payload, &q); err != nil {
			return false, nil
		}
		if q.IsReasoning {
			p.think.WriteString(q.Text)
		} else {
			p.answer.WriteString(q.Text)
		}

	case godo.HostedAgentEventKindToolCallStarted:
		var q toolCallStartedPayload
		if err := json.Unmarshal(ev.Payload, &q); err == nil {
			if line := q.commandLine(); line != "" {
				p.progress(line)
			}
		}

	case godo.HostedAgentEventKindHITLRequested:
		var q hitlRequestedPayload
		if err := json.Unmarshal(ev.Payload, &q); err != nil {
			return false, nil
		}
		return p.resolveApproval(q)

	case godo.HostedAgentEventKindRunCompleted:
		var q runCompletedPayload
		if err := json.Unmarshal(ev.Payload, &q); err == nil {
			p.usage = q
		}
		p.done = true
		p.status = promptStatusCompleted
		return true, nil

	case godo.HostedAgentEventKindRunFailed:
		var q runFailedPayload
		if err := json.Unmarshal(ev.Payload, &q); err == nil {
			p.failure = q.Message
			if p.failure == "" {
				p.failure = fmt.Sprintf("code %d", q.Code)
			}
		}
		p.done = true
		p.status = promptStatusFailed
		return true, nil
	}
	return false, nil
}

// resolveApproval applies the --on-hitl policy, or stops when there is none.
//
// Blocking here would be the worst option: a headless command has no way to ask
// and would simply hang until its timeout, looking identical to a slow agent.
// Picking a default silently is worse still, since approving means running a
// command the caller never saw.
func (p *promptCollector) resolveApproval(q hitlRequestedPayload) (bool, error) {
	requestID := q.id()
	if p.outcome == "" {
		what := q.commandSummary()
		if what == "" {
			what = "an action"
		}
		return false, fmt.Errorf(
			"the agent needs approval to run %s, and `prompt` is headless so there is nobody to ask.\n"+
				"Re-run with --on-hitl approve|reject|defer to answer every request the same way, "+
				"or attach with `%s launch %s` to answer them yourself.\n"+
				"The run is still waiting; resolve it with `%s approve %s %s <approve|reject|defer>`",
			what, agentCLI, p.sessionID, agentCLI, p.sessionID, requestID)
	}
	if requestID == "" || p.resolved[requestID] {
		return false, nil
	}
	if err := p.svc.ResolveHITL(p.sessionID, requestID, &godo.HostedAgentResolveHITLRequest{
		Outcome: p.outcome,
		Source:  godo.HostedAgentResolutionSourceOutOfBand,
	}); err != nil {
		return false, fmt.Errorf("auto-resolving approval %s as %s: %w", requestID, humanHITLOutcome(p.outcome), err)
	}
	p.resolved[requestID] = true
	p.progress(fmt.Sprintf("auto-%s %s", humanHITLOutcome(p.outcome), requestID))
	return false, nil
}

// progress writes one activity line to stderr, keeping stdout clean.
func (p *promptCollector) progress(msg string) {
	if p.quiet {
		return
	}
	fmt.Fprintf(promptStderr, "· %s\n", msg)
}

// report emits the answer and translates the run's outcome into an exit code.
func (p *promptCollector) report(c *CmdConfig, ctx context.Context) error {
	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
	switch {
	case p.done:
		// status already set by handle
	case timedOut:
		p.status = promptStatusTimedOut
	default:
		p.status = promptStatusIncomplete
	}

	if agentStructuredOutput(c) {
		if err := c.Display(&displayers.HostedAgentPrompt{
			Prompts: []*displayers.HostedAgentPromptResult{{
				SessionID:      p.sessionID,
				RunID:          p.runID,
				Text:           p.answer.String(),
				Reasoning:      p.reasoningForOutput(),
				Status:         p.status,
				TotalTokensIn:  p.usage.TotalTokensIn,
				TotalTokensOut: p.usage.TotalTokensOut,
				RunCostMicros:  p.usage.RunCostMicros,
				Error:          p.failure,
			}},
			Single: true,
		}); err != nil {
			return err
		}
		return p.exitFor()
	}

	if reasoning := p.reasoningForOutput(); reasoning != "" {
		fmt.Fprintf(promptStderr, "%s\n", reasoning)
	}

	// The answer, then a newline only when the agent did not end with one. A
	// trailing newline leaves the shell prompt on its own line and is stripped
	// by `$(...)`, so it cannot corrupt a captured value.
	answer := p.answer.String()
	if _, err := io.WriteString(c.Out, answer); err != nil {
		return err
	}
	if answer != "" && !strings.HasSuffix(answer, "\n") {
		if _, err := io.WriteString(c.Out, "\n"); err != nil {
			return err
		}
	}

	p.summarize()
	return p.exitFor()
}

func (p *promptCollector) reasoningForOutput() string {
	if !p.includeReasoning {
		return ""
	}
	return p.think.String()
}

// summarize prints the closing status line to stderr.
func (p *promptCollector) summarize() {
	if p.quiet {
		return
	}
	switch p.status {
	case promptStatusCompleted:
		summary := "run complete"
		if p.usage.TotalTokensIn > 0 || p.usage.TotalTokensOut > 0 {
			summary = fmt.Sprintf("run complete · %d in / %d out tokens",
				p.usage.TotalTokensIn, p.usage.TotalTokensOut)
		}
		if p.usage.RunCostMicros > 0 {
			summary += fmt.Sprintf(" · $%.4f", float64(p.usage.RunCostMicros)/1_000_000)
		}
		fmt.Fprintf(promptStderr, "· %s\n", summary)
	case promptStatusFailed:
		fmt.Fprintf(promptStderr, "run failed: %s\n", p.failure)
	case promptStatusTimedOut:
		fmt.Fprintf(promptStderr,
			"timed out waiting for the run to finish; it is still going and its output stays readable with `%s logs %s`\n",
			agentCLI, p.sessionID)
	case promptStatusIncomplete:
		fmt.Fprintf(promptStderr,
			"the run did not finish; read what it produced with `%s logs %s`\n",
			agentCLI, p.sessionID)
	}
}

// exitFor reproduces the run's outcome as doctl's exit status so the command
// composes in `&&` chains and `if` tests. A run that ran and failed is not a
// doctl error, so nothing is printed on top of what the run already reported.
func (p *promptCollector) exitFor() error {
	switch p.status {
	case promptStatusCompleted:
		return nil
	case promptStatusTimedOut:
		promptExit(promptTimeoutExit)
	default:
		promptExit(1)
	}
	// Unreachable in a real run — os.Exit does not return. Under test the stub
	// does, and the sentinel keeps the runner honest about having failed
	// without printing a second error on top.
	return ErrExitSilently
}
