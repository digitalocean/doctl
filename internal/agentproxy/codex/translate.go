// Package codex: canonical-event-to-codex-notification translation.
//
// This file is the "translator-strategy seam" the original implementation
// plan called for: the logic that decides what a canonical event means in
// codex terms lives here, not scattered inline in runEventLoop/drainStream's
// own dispatch loop (facade.go). A future raw-passthrough strategy —
// preferring an event's native codex bytes when the harness exposes them,
// falling back to this reconstruction otherwise — would extend or replace
// translateEvent, not runEventLoop's own retry/reconnect machinery.
//
// This is deliberately not a Go interface with multiple implementations:
// there is exactly one strategy today (canonical reconstruction), and
// godo.HostedAgentEvent doesn't even have a source_raw field yet (see the
// RFC's own gap analysis — the harness doesn't expose it on the client
// stream yet either). Building a pluggable-strategy interface with nothing
// real to plug in would be speculative abstraction; the seam that matters
// right now is that this logic is self-contained in its own file, ready to
// grow a conditional (prefer source_raw when present) once the data exists.
package codex

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/digitalocean/godo"
)

// translateEvent is runEventLoop's / replaySessionHistory's per-event
// translation step (see drainStream, facade.go / replay.go): given one
// canonical event and the turnState it belongs to, decide what (if anything)
// to tell the codex client, mutating ts and pushing notifications via
// f.notify along the way. Returns true if a notify failure indicates the
// client connection itself is dead, in which case the caller must stop
// entirely rather than try to reconnect to the harness — the problem isn't
// the harness stream.
//
// replay is true when this event is being fed from durable session history
// (--replay) rather than the live SSE tail. Historical hitl.requested
// records are already settled (a matching hitl.resolved follows in the same
// history); re-driving them would re-prompt the user for command_execution
// and POST ResolveHITL against hitl_ids the harness closed long ago. When
// replay is set, those side effects are skipped — tool_call_started /
// tool_call_completed in the same history already render the items.
func (f *Facade) translateEvent(ctx context.Context, ev godo.HostedAgentEvent, ts *turnState, replay bool) (clientDead bool) {
	switch ev.Kind {
	case godo.HostedAgentEventKindRunStarted:
		ts.startedAt = time.Now().Unix()
		if !f.notify("turn/started", turnStartedNotification{
			ThreadID: f.SessionID,
			Turn:     turnObj{ID: ev.RunID, Items: []any{}, Status: "inProgress", StartedAt: &ts.startedAt},
		}) {
			return true
		}
		if !f.notify("item/started", itemStartedNotification{
			Item:        agentMessageItem{Type: "agentMessage", ID: ts.itemID, Text: ""},
			ThreadID:    f.SessionID,
			TurnID:      ev.RunID,
			StartedAtMs: ts.startedAt * 1000,
		}) {
			return true
		}
		ts.itemStarted = true

	case godo.HostedAgentEventKindTokenChunk:
		var payload struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return false
		}
		ts.text.WriteString(payload.Text)
		if !f.notify("item/agentMessage/delta", agentMessageDeltaNotification{
			ThreadID: f.SessionID,
			TurnID:   ev.RunID,
			ItemID:   ts.itemID,
			Delta:    payload.Text,
		}) {
			return true
		}

	case godo.HostedAgentEventKindRunCompleted:
		f.finishTurn(ev.RunID, ts, "completed", nil)

	case godo.HostedAgentEventKindRunFailed:
		var payload struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(ev.Payload, &payload)
		f.finishTurn(ev.RunID, ts, "failed", &turnError{Message: payload.Message})

	case godo.HostedAgentEventKindToolCallStarted:
		var payload struct {
			ToolCallID string `json:"tool_call_id"`
			Name       string `json:"name"`
			Input      struct {
				Command string `json:"command"`
				Cwd     string `json:"cwd"`
			} `json:"input"`
		}
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return false
		}
		switch payload.Name {
		case "command_execution":
			// ts.commands is also touched by autoRejectFileChangeApproval's
			// sibling for file_change (ts.fileChanges) from a separately
			// spawned goroutine — f.mu must guard every touch of either map,
			// not just the ones on that goroutine's side, since a lock held
			// by only one side of a shared map access protects nothing (see
			// ts.fileChanges's own comment below for the concrete race this
			// was found to cause).
			f.mu.Lock()
			if ts.commands == nil {
				ts.commands = make(map[string]*commandState)
			}
			ts.commands[payload.ToolCallID] = &commandState{command: payload.Input.Command, cwd: payload.Input.Cwd}
			f.mu.Unlock()
			f.notify("item/started", itemStartedNotification{
				Item: commandExecutionItem{
					Type:           "commandExecution",
					ID:             payload.ToolCallID,
					Command:        payload.Input.Command,
					Cwd:            payload.Input.Cwd,
					Source:         "agent",
					Status:         "inProgress",
					CommandActions: []any{},
				},
				ThreadID:    f.SessionID,
				TurnID:      ev.RunID,
				StartedAtMs: time.Now().UnixMilli(),
			})

		case "file_change":
			// input is always {} for file_change (confirmed via a live
			// capture) — nothing to remember here, unlike commandState;
			// this map entry exists only to mark the tool_call_id as a
			// file_change so ToolCallCompleted renders the right item
			// type and fileChangeState can record a declined outcome.
			//
			// f.mu guards this map: autoRejectFileChangeApproval sets
			// fc.declined from a separately spawned goroutine (see the
			// HITLRequested case below), so every touch here — including
			// map creation and insertion, not just field reads — must hold
			// the same lock. A concurrent unlocked map write racing a
			// locked-on-only-one-side access is a real, confirmed
			// "concurrent map read and map write" hazard whenever two
			// file_change tool calls are in flight in the same turn (e.g.
			// "create a file, then edit it").
			f.mu.Lock()
			if ts.fileChanges == nil {
				ts.fileChanges = make(map[string]*fileChangeState)
			}
			ts.fileChanges[payload.ToolCallID] = &fileChangeState{}
			f.mu.Unlock()
			f.notify("item/started", itemStartedNotification{
				Item: fileChangeItem{
					Type:    "fileChange",
					ID:      payload.ToolCallID,
					Changes: []any{},
					Status:  "inProgress",
				},
				ThreadID:    f.SessionID,
				TurnID:      ev.RunID,
				StartedAtMs: time.Now().UnixMilli(),
			})

		default:
			// Only "command_execution" and "file_change" have been
			// observed and mapped so far; any other tool kind falls
			// through here logged, same as any other unhandled case,
			// rather than guessed at.
			log.Printf("codex facade: unhandled tool call kind %q (tool_call_id=%s)", payload.Name, payload.ToolCallID)
		}

	case godo.HostedAgentEventKindToolCallCompleted:
		var payload struct {
			ToolCallID string `json:"tool_call_id"`
			OK         bool   `json:"ok"`
			DurationMs int64  `json:"duration_ms"`
			Summary    string `json:"summary"`
		}
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return false
		}
		// Lock held only around the map lookup/delete, not the notify below
		// — same narrow-critical-section discipline as
		// HostedAgentEventKindRunUsageRecorded's totalUsage update further
		// down, so a slow client write never blocks a concurrent
		// autoRejectFileChangeApproval/requestCommandExecutionApproval
		// goroutine that also needs f.mu.
		f.mu.Lock()
		cmd, cmdOK := ts.commands[payload.ToolCallID]
		if cmdOK {
			delete(ts.commands, payload.ToolCallID)
		}
		f.mu.Unlock()
		if cmdOK {
			status := "completed"
			exitCode := 0
			if !payload.OK {
				status = "failed"
				exitCode = 1
			}
			var output *string
			if payload.Summary != "" {
				output = &payload.Summary
			}
			durationMs := payload.DurationMs
			f.notify("item/completed", itemCompletedNotification{
				Item: commandExecutionItem{
					Type:             "commandExecution",
					ID:               payload.ToolCallID,
					Command:          cmd.command,
					Cwd:              cmd.cwd,
					Source:           "agent",
					Status:           status,
					CommandActions:   []any{},
					AggregatedOutput: output,
					ExitCode:         &exitCode,
					DurationMs:       &durationMs,
				},
				ThreadID:      f.SessionID,
				TurnID:        ev.RunID,
				CompletedAtMs: time.Now().UnixMilli(),
			})
			return false
		}

		// PatchApplyStatus has a 4th value (Declined) that
		// CommandExecutionStatus doesn't — canonical only carries a success
		// boolean, so ok:false is ambiguous between "failed to apply" and
		// "user declined" unless this facade already remembered which one
		// happened (see fileChangeState). declined is copied out under the
		// same lock as the map lookup/delete, not read again afterward,
		// since fc.declined is itself written by
		// autoRejectFileChangeApproval from another goroutine under f.mu.
		f.mu.Lock()
		fc, fcOK := ts.fileChanges[payload.ToolCallID]
		var declined bool
		if fcOK {
			declined = fc.declined
			delete(ts.fileChanges, payload.ToolCallID)
		}
		f.mu.Unlock()
		if fcOK {
			status := "completed"
			if !payload.OK {
				if declined {
					status = "declined"
				} else {
					status = "failed"
				}
			}
			f.notify("item/completed", itemCompletedNotification{
				Item: fileChangeItem{
					Type:    "fileChange",
					ID:      payload.ToolCallID,
					Changes: []any{},
					Status:  status,
				},
				ThreadID:      f.SessionID,
				TurnID:        ev.RunID,
				CompletedAtMs: time.Now().UnixMilli(),
			})
			return false
		}

		// A completed event for a call this facade never saw start
		// (e.g. an unhandled tool kind skipped above) — nothing tracked
		// to report, so there's nothing safe to send.

	case godo.HostedAgentEventKindHITLRequested:
		var hitl hitlRequestedPayload
		if err := json.Unmarshal(ev.Payload, &hitl); err != nil {
			return false
		}
		if replay {
			// Already settled in history — see translateEvent's doc comment.
			// For file_change, still record declined when the historical
			// resolution was a reject so the later tool_call_completed
			// reports PatchApplyStatus::Declined rather than ::Failed; the
			// outcome itself arrives as HITLResolved below. Remember the
			// hitl→item mapping here so that event can find the fileChangeState.
			if hitl.Payload.Kind == "file_change" {
				f.mu.Lock()
				if ts.replayHITLs == nil {
					ts.replayHITLs = make(map[string]string)
				}
				ts.replayHITLs[hitl.HitlID] = hitl.Payload.ItemID
				f.mu.Unlock()
			}
			return false
		}
		// Spawned, not called inline: requestCommandExecutionApproval
		// blocks on real human interaction time in the TUI, and this loop
		// must keep reading the one shared stream for every other
		// in-flight turn/tool-call meanwhile (see the Facade.mu doc
		// comment on why there's only ever one StreamSession call).
		// autoRejectFileChangeApproval doesn't block on anything, but is
		// spawned the same way for consistency.
		//
		// autoRejectFileChangeApproval takes ts directly (this
		// translateEvent call's own turnState) rather than re-deriving it
		// via a f.turns[ev.RunID] lookup: ts is already the correct object
		// whether this call came from the live event loop or from
		// replaySessionHistory, which never registers its synthesized
		// turnStates in the shared f.turns map at all (see replay.go) — a
		// lookup-by-id would simply fail to find them during replay.
		switch hitl.Payload.Kind {
		case "command_execution":
			go f.requestCommandExecutionApproval(ctx, ev.RunID, hitl)
		case "file_change":
			go f.autoRejectFileChangeApproval(ts, hitl)
		default:
			// item/permissions/requestApproval exists in the codex
			// protocol too, but its canonical trigger has never been
			// observed, and its requestApproval shape isn't known well
			// enough to forward to the client (see
			// PermissionsRequestApprovalParams/Response — a grant
			// profile, not a simple decision enum, unlike the other two
			// kinds). Auto-reject rather than leave the harness waiting
			// on a decision that will never come — see
			// autoRejectUnknownHITL.
			go f.autoRejectUnknownHITL(hitl.HitlID, hitl.Payload.Kind)
		}

	case godo.HostedAgentEventKindHITLResolved:
		// Live: the harness's own ack that a resolution from
		// requestCommandExecutionApproval/autoRejectFileChangeApproval/
		// autoRejectUnknownHITL landed. codex has no client-facing
		// "approval acknowledged" notification — for command_execution the
		// client already got its answer accepted the moment its
		// requestApproval reply was read, and file_change/unknown kinds
		// never had a client-facing request to acknowledge in the first
		// place — so there's nothing to forward here.
		//
		// Replay: apply a historical reject onto fileChangeState.declined
		// so tool_call_completed can distinguish Declined from Failed —
		// the live path set that flag inside autoRejectFileChangeApproval,
		// which replay deliberately does not call.
		if !replay {
			return false
		}
		var resolved struct {
			HitlID  string `json:"hitl_id"`
			Outcome int32  `json:"outcome"`
		}
		if err := json.Unmarshal(ev.Payload, &resolved); err != nil {
			return false
		}
		// Proto enum on the wire: 1=APPROVE, 2=REJECT, 3=DEFER (same as
		// doctl agents attach's hitlResolvedPayload).
		if resolved.Outcome != 2 {
			return false
		}
		f.mu.Lock()
		itemID, ok := ts.replayHITLs[resolved.HitlID]
		if ok {
			delete(ts.replayHITLs, resolved.HitlID)
			if fc, fcOK := ts.fileChanges[itemID]; fcOK {
				fc.declined = true
			}
		}
		f.mu.Unlock()

	case godo.HostedAgentEventKindRunUsageRecorded:
		if replay {
			// Historical usage must not inflate this connection's live
			// thread total — the TUI's "total" is for traffic on this
			// thread/connection, and --replay already reconstructs past
			// turns as ordinary notifications without needing their
			// token counts folded into the running sum.
			return false
		}
		var payload struct {
			Usage struct {
				InputTokens       int64 `json:"input_tokens"`
				OutputTokens      int64 `json:"output_tokens"`
				CachedInputTokens int64 `json:"cached_input_tokens"`
				ReasoningTokens   int64 `json:"reasoning_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return false
		}
		last := tokenUsageBreakdown{
			TotalTokens:           payload.Usage.InputTokens + payload.Usage.OutputTokens,
			InputTokens:           payload.Usage.InputTokens,
			CachedInputTokens:     payload.Usage.CachedInputTokens,
			OutputTokens:          payload.Usage.OutputTokens,
			ReasoningOutputTokens: payload.Usage.ReasoningTokens,
		}
		f.mu.Lock()
		f.totalUsage.TotalTokens += last.TotalTokens
		f.totalUsage.InputTokens += last.InputTokens
		f.totalUsage.CachedInputTokens += last.CachedInputTokens
		f.totalUsage.OutputTokens += last.OutputTokens
		f.totalUsage.ReasoningOutputTokens += last.ReasoningOutputTokens
		total := f.totalUsage
		f.mu.Unlock()
		f.notify("thread/tokenUsage/updated", threadTokenUsageUpdatedNotification{
			ThreadID: f.SessionID,
			TurnID:   ev.RunID,
			TokenUsage: threadTokenUsage{
				Total: total,
				Last:  last,
			},
		})

	case godo.HostedAgentEventKindRunCostAccrued:
		// No codex-facing equivalent: this canonical event carries a
		// running dollar total from the billing layer
		// (running_total_micros/delta_micros), not token counts, and
		// codex's ThreadTokenUsageUpdatedNotification has no cost field
		// to put it in. Cased explicitly, same reasoning as
		// HITLResolved above — a documented no-op, not an oversight.
	}
	return false
}
