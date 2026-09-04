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
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// Everything else about driving an agent unattended already works: the global
// --interactive flag makes env prompts and the GitHub-connect confirm fail fast
// instead of hanging, and `create` returns as soon as the session is ready.
// Approvals were the one thing that still needed a human — either at the chat
// TUI or running `approve` out of band. --on-hitl closes that: one fixed
// policy, applied to every request, with no terminal involved.

// agentOnHITLOutcome reads --on-hitl. An empty flag means "not unattended",
// which is the default and returns an empty outcome.
func agentOnHITLOutcome(c *CmdConfig) (godo.HostedAgentHITLOutcome, error) {
	raw, err := c.Doit.GetString(c.NS, doctl.ArgAgentOnHITL)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	outcome, err := hitlOutcomeFor(raw)
	if err != nil {
		return "", fmt.Errorf("--%s: %w", doctl.ArgAgentOnHITL, err)
	}
	return outcome, nil
}

// hitlWatchTerminalRun reports whether an event ends the run we are watching.
func hitlWatchTerminalRun(kind godo.HostedAgentEventKind) bool {
	switch kind {
	case godo.HostedAgentEventKindRunCompleted, godo.HostedAgentEventKindRunFailed:
		return true
	default:
		return false
	}
}

// watchSessionHeadless follows the session's event stream with no TUI and no
// keyboard input, resolving every approval request with the fixed outcome, and
// returns when the run reaches a terminal state. Ctrl-C detaches without
// touching the session, matching `launch`'s detach semantics: the caller is
// watching a run, not owning it.
//
// Output is one plain line per event so it reads correctly in a CI log, where
// this is almost always running — the chat renderer's cursor control and
// spinners would be noise there.
func watchSessionHeadless(c *CmdConfig, sessionID string, outcome godo.HostedAgentHITLOutcome) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	svc := c.HostedAgents()
	cursor := &eventCursor{}
	backoff := initialReconnectBackoff
	failures := 0
	// Resolving the same request twice is harmless server-side but noisy in the
	// log, and SSE replay after a reconnect makes it likely.
	resolved := map[string]bool{}

	fmt.Fprintf(c.Out, "%s\n", colorize(fmt.Sprintf("Watching %s · approvals will be %s automatically · Ctrl-C detaches",
		sessionID, humanHITLOutcome(outcome)), colMuted))

	for {
		if ctx.Err() != nil {
			return nil
		}

		stream, err := svc.StreamSession(ctx, sessionID, &godo.HostedAgentSessionStreamOptions{
			ReplayFrom: cursor.get(),
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if msg, terminal := classifyStreamError(err); terminal {
				fmt.Fprintln(c.Out, msg)
				return nil
			}
			failures++
			if failures >= maxAutoReconnectAttempts {
				return fmt.Errorf("lost the event stream for session %s after %d attempts: %w", sessionID, failures, err)
			}
			if !reconnectSleepFn(ctx, backoff) {
				return nil
			}
			backoff = nextBackoff(backoff)
			continue
		}

		connectedAt := streamClock()
		done, watchErr := drainHeadless(c, svc, sessionID, stream, cursor, outcome, resolved)
		streamErr := stream.Err()
		stream.Close()

		if watchErr != nil {
			return watchErr
		}
		if done || ctx.Err() != nil {
			return nil
		}

		// A stream that stayed up is a server idle timeout, not a fault, so it
		// must not consume the failure budget — an unattended run can sit quiet
		// for a long time while the agent thinks.
		if streamClock().Sub(connectedAt) >= healthyStreamDuration {
			failures = 0
			backoff = initialReconnectBackoff
		} else {
			failures++
		}
		if streamErr != nil {
			if msg, terminal := classifyStreamError(streamErr); terminal {
				fmt.Fprintln(c.Out, msg)
				return nil
			}
		}
		if failures >= maxAutoReconnectAttempts {
			return fmt.Errorf("lost the event stream for session %s after %d attempts", sessionID, failures)
		}
		if !reconnectSleepFn(ctx, backoff) {
			return nil
		}
		backoff = nextBackoff(backoff)
	}
}

// drainHeadless consumes one connection's events. done is true when the run
// finished, which ends the watch; a non-nil error means resolving an approval
// failed, which must not be swallowed — an unattended run that silently stops
// getting approvals looks identical to one that hung.
func drainHeadless(
	c *CmdConfig,
	svc do.HostedAgentsService,
	sessionID string,
	stream *godo.HostedAgentSessionStream,
	cursor *eventCursor,
	outcome godo.HostedAgentHITLOutcome,
	resolved map[string]bool,
) (done bool, err error) {
	for stream.Next() {
		ev := stream.Current()
		cursor.set(ev.EventID)

		// Connection health, not session activity.
		if ev.Kind == godo.HostedAgentEventKindStreamState {
			continue
		}
		// Token deltas would flood a CI log one word at a time; the assistant's
		// output is recoverable in full with `logs`.
		if ev.Kind == godo.HostedAgentEventKindTokenChunk {
			continue
		}

		fmt.Fprintf(c.Out, "[%s] %s%s\n",
			ev.At.Time.UTC().Format(time.RFC3339), ev.Kind, headlessEventDetail(ev))

		if ev.Kind == godo.HostedAgentEventKindHITLRequested {
			var p hitlRequestedPayload
			if jsonErr := json.Unmarshal(ev.Payload, &p); jsonErr != nil {
				continue
			}
			requestID := p.id()
			if requestID == "" || resolved[requestID] {
				continue
			}
			if err := svc.ResolveHITL(sessionID, requestID, &godo.HostedAgentResolveHITLRequest{
				Outcome: outcome,
				Source:  godo.HostedAgentResolutionSourceOutOfBand,
			}); err != nil {
				return false, fmt.Errorf("auto-resolving approval %s as %s: %w", requestID, humanHITLOutcome(outcome), err)
			}
			resolved[requestID] = true
			fmt.Fprintf(c.Out, "  %s %s\n",
				colorize("auto-"+humanHITLOutcome(outcome), colMuted), requestID)
		}

		if hitlWatchTerminalRun(ev.Kind) {
			return true, nil
		}
	}
	return false, nil
}

// headlessEventDetail adds the one field that makes an event line actionable
// (the command awaiting approval, the failure message), and nothing more.
func headlessEventDetail(ev godo.HostedAgentEvent) string {
	switch ev.Kind {
	case godo.HostedAgentEventKindHITLRequested:
		var p hitlRequestedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			if summary := p.commandSummary(); summary != "" {
				return " " + summary
			}
		}
	case godo.HostedAgentEventKindRunFailed:
		var p runFailedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil && p.Message != "" {
			return " " + p.Message
		}
	case godo.HostedAgentEventKindToolCallStarted:
		var p toolCallStartedPayload
		if err := json.Unmarshal(ev.Payload, &p); err == nil {
			if line := p.commandLine(); line != "" {
				return " " + line
			}
		}
	}
	return ""
}

// humanHITLOutcome renders an outcome for prose, e.g. "approve".
func humanHITLOutcome(o godo.HostedAgentHITLOutcome) string {
	return strings.ToLower(strings.TrimPrefix(string(o), "HITL_OUTCOME_"))
}
