// Package codex: canonical-event-to-codex-notification translation.
//
// This file is the "translator-strategy seam" the original implementation
// plan called for: the logic that decides what a canonical event means in
// codex terms lives here, not scattered inline in runEventLoop/drainStream's
// own dispatch loop (facade.go). As of v1.5 the seam carries two strategies:
// translateEvent first offers each event to tryRawPassthrough (raw.go),
// which forwards the event's native codex frame when one is available, and
// only reconstructs from canonical — everything below — when it isn't.
//
// This is deliberately not a Go interface with multiple implementations:
// the raw strategy is one conditional at the top of translateEvent plus its
// own self-contained file, exactly the shape the v1 comment here reserved.
// The canonical reconstruction is not legacy — it is the permanent fallback
// (adapter-synthesized events carry no raw bytes; non-codex sessions never
// forward raw at all) and the only path that can ever serve cross-agent use.
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
	// Prefer the event's native codex frame when one rode along
	// (SourceRaw) — see raw.go. handled=false falls through to the
	// canonical reconstruction below, per event.
	if handled, dead := f.tryRawPassthrough(ev, ts, replay); handled {
		return dead
	}

	at := eventTime(ev)
	switch ev.Kind {
	case godo.HostedAgentEventKindRunStarted:
		ts.startedAt = at.Unix()
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
			StartedAtMs: at.UnixMilli(),
		}) {
			return true
		}
		ts.itemStarted = true

	case godo.HostedAgentEventKindTokenChunk:
		var payload struct {
			Text string `json:"text"`
			// IsReasoning marks model reasoning rather than the user-visible
			// answer (SPI TokenChunk.is_reasoning). The answer begins at the
			// first chunk where this is false.
			IsReasoning bool `json:"is_reasoning"`
		}
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return false
		}
		// Reasoning is not the answer, and this protocol has nowhere to put
		// it: item/agentMessage/delta is the only text channel the facade
		// speaks, so forwarding a thought there makes the client render the
		// thinking trace AS the reply (MARSOHS-1012). Codex sessions never
		// reach this line for reasoning — raw.go forwards their native
		// frames, which carry reasoning as its own item — so this path is
		// what non-codex runtimes (cursor) fall back to, and dropping is the
		// only faithful option until the protocol gains a reasoning item.
		//
		// Skipped ahead of the item/started below on purpose: opening an
		// agentMessage item for a turn that leads with reasoning would
		// announce an item whose only content we then refuse to send.
		if payload.IsReasoning {
			return false
		}
		// A canonical delta on a turn whose run.started went raw (or was
		// never seen at all) has no announced item to attach to: the raw
		// path deliberately never opens the synthesized "-msg" item (see
		// raw.go's package doc), and this is exactly what a mid-turn
		// reconnect produces — the server does not retain source_raw
		// durably, so events redelivered via replay_from arrive raw-less
		// and fall back here even though the turn's earlier frames were
		// forwarded raw. Open the item now so the delta below (and every
		// later one) references an item the client has actually been told
		// about, and so finishTurn closes it instead of suppressing the
		// close for an item whose deltas were delivered.
		if !ts.itemStarted {
			if !f.notify("item/started", itemStartedNotification{
				Item:        agentMessageItem{Type: "agentMessage", ID: ts.itemID, Text: ""},
				ThreadID:    f.SessionID,
				TurnID:      ev.RunID,
				StartedAtMs: at.UnixMilli(),
			}) {
				return true
			}
			ts.itemStarted = true
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
		return f.finishTurn(ev.RunID, ts, "completed", nil, at)

	case godo.HostedAgentEventKindRunFailed:
		var payload struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(ev.Payload, &payload)
		return f.finishTurn(ev.RunID, ts, "failed", &turnError{Message: payload.Message}, at)

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
			if !f.notify("item/started", itemStartedNotification{
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
				StartedAtMs: at.UnixMilli(),
			}) {
				return true
			}

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
			if !f.notify("item/started", itemStartedNotification{
				Item: fileChangeItem{
					Type:    "fileChange",
					ID:      payload.ToolCallID,
					Changes: []any{},
					Status:  "inProgress",
				},
				ThreadID:    f.SessionID,
				TurnID:      ev.RunID,
				StartedAtMs: at.UnixMilli(),
			}) {
				return true
			}

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
			if !f.notify("item/completed", itemCompletedNotification{
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
				CompletedAtMs: at.UnixMilli(),
			}) {
				return true
			}
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
			if !f.notify("item/completed", itemCompletedNotification{
				Item: fileChangeItem{
					Type:    "fileChange",
					ID:      payload.ToolCallID,
					Changes: []any{},
					Status:  status,
				},
				ThreadID:      f.SessionID,
				TurnID:        ev.RunID,
				CompletedAtMs: at.UnixMilli(),
			}) {
				return true
			}
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
		case "file_change":
			// Checked before the native path on purpose: the auto-reject
			// below is a deliberate policy about apply_patch, not a gap in
			// this proxy's protocol coverage, so being able to forward the
			// request faithfully is not a reason to start forwarding it.
			go f.autoRejectFileChangeApproval(ts, hitl)
		case "command_execution":
			// Native first: same question either way, but the agent's own
			// frame carries the fields codex sent rather than the subset
			// commandExecutionRequestApprovalParams reconstructs.
			if !f.tryNativeApproval(ctx, ev, hitl) {
				go f.requestCommandExecutionApproval(ctx, ev.RunID, hitl)
			}
		default:
			// Every other kind — an MCP elicitation, a tool asking for input,
			// or a gated request newer than this proxy — is answerable only
			// in the agent's own protocol, so the native path is the only one
			// that can do anything but refuse on the user's behalf.
			if f.tryNativeApproval(ctx, ev, hitl) {
				break
			}
			// No native frame to forward, and no way to synthesize the
			// request from the canonical payload: these kinds do not answer
			// with a simple decision enum (item/permissions/requestApproval
			// wants a grant profile — see PermissionsRequestApprovalParams),
			// so there is nothing to ask the client that it could answer.
			// Auto-reject rather than leave the harness waiting on a
			// decision that will never come — see autoRejectUnknownHITL.
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
		last, total, ok := f.accumulateUsage(ev.Payload)
		if !ok {
			return false
		}
		if !f.notify("thread/tokenUsage/updated", threadTokenUsageUpdatedNotification{
			ThreadID: f.SessionID,
			TurnID:   ev.RunID,
			TokenUsage: threadTokenUsage{
				Total: total,
				Last:  last,
			},
		}) {
			return true
		}

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

// accumulateUsage decodes one run.usage_recorded payload and folds it into
// this connection's running total under f.mu, returning the event's own
// breakdown (last) and the updated running sum (total). Shared by the
// canonical reconstruction (which reports both in its synthesized
// thread/tokenUsage/updated) and the raw passthrough (which forwards the
// VM's own notification but still keeps this total warm for any later
// canonical fallback — see tryRawPassthrough). ok=false means the payload
// didn't decode and nothing was accumulated.
func (f *Facade) accumulateUsage(payload json.RawMessage) (last, total tokenUsageBreakdown, ok bool) {
	var p struct {
		Usage struct {
			InputTokens       int64 `json:"input_tokens"`
			OutputTokens      int64 `json:"output_tokens"`
			CachedInputTokens int64 `json:"cached_input_tokens"`
			ReasoningTokens   int64 `json:"reasoning_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return tokenUsageBreakdown{}, tokenUsageBreakdown{}, false
	}
	last = tokenUsageBreakdown{
		TotalTokens:           p.Usage.InputTokens + p.Usage.OutputTokens,
		InputTokens:           p.Usage.InputTokens,
		CachedInputTokens:     p.Usage.CachedInputTokens,
		OutputTokens:          p.Usage.OutputTokens,
		ReasoningOutputTokens: p.Usage.ReasoningTokens,
	}
	f.mu.Lock()
	f.totalUsage.TotalTokens += last.TotalTokens
	f.totalUsage.InputTokens += last.InputTokens
	f.totalUsage.CachedInputTokens += last.CachedInputTokens
	f.totalUsage.OutputTokens += last.OutputTokens
	f.totalUsage.ReasoningOutputTokens += last.ReasoningOutputTokens
	total = f.totalUsage
	f.mu.Unlock()
	return last, total, true
}

// eventTime returns the canonical event's own timestamp when present, else
// time.Now. Using At (especially on the --replay path) keeps historical
// turns stamped with when they actually happened rather than when they were
// re-fed into the TUI.
func eventTime(ev godo.HostedAgentEvent) time.Time {
	if !ev.At.IsZero() {
		return ev.At.Time
	}
	return time.Now()
}
