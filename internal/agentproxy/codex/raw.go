// Package codex: raw event passthrough (v1.5) — the prefer-source_raw half
// of the translator strategy translate.go's doc comment reserved a seam for.
//
// When the hosted session's agent runtime is codex itself (Facade.AgentKind),
// every canonical event the OHR adapter emitted carries the exact native
// app-server frame it was mapped from (HostedAgentEvent.SourceRaw — one
// JSON-RPC message as read off the codex app-server transport in the
// sandbox, requested via StreamSession's IncludeRaw). Forwarding that frame
// to the TUI beats reconstructing an approximation from canonical's ~15-event
// common denominator: canonical carries no file diffs, no command output, no
// reasoning deltas, no item metadata — the native frame carries all of it.
//
// "Verbatim" has exactly one deliberate exception: identifier rewriting.
// The in-sandbox adapter drove its own thread/start and turn/start against
// the VM's app-server, so the raw params reference the VM's thread and turn
// ids — but this proxy's client was told the thread id is Facade.SessionID
// (thread/start's synthesized reply) and each turn's id is the harness run
// id (turn/start's synthesized reply). A frame forwarded with the VM's ids
// would describe a thread and turn the client has never heard of, so
// threadId/turnId (and turn.id) are rewritten to the proxy's ids. Native
// item ids are kept: the proxy never promises item ids in any reply, so the
// raw frames' own item lifecycle (started → deltas → completed, all sharing
// the native id) is self-consistent as-is.
//
// The fallback is per event, as the v1.5 plan specifies: an event with no
// usable raw frame (adapter-synthesized events like a TurnError-mapped
// run.failed; a server that doesn't retain bytes; a non-codex session) takes
// translate.go's canonical reconstruction instead. The two paths keep their
// item-id families from colliding naturally: a turn whose run.started was
// forwarded raw never sets turnState.itemStarted, so finishTurn's
// synthesized agentMessage item/completed — the only canonical notification
// that could duplicate a raw item — stays suppressed for that turn.
//
// This file also holds the v2 inbound half: rawTurnStartFrame packages the
// TUI's turn/start params as SendInput's source_raw, so the in-sandbox
// adapter drives the turn from the client's actual params (input items,
// model, effort, approval policy) instead of a synthesized single-text turn.
// The direction is inverted but the id discipline is symmetric: outbound
// rewrites VM ids to proxy ids; inbound ships the proxy-side threadId as-is
// and the adapter rewrites it to the VM's own, minting its own JSON-RPC
// request id (the frame is a template, never an injected request).
package codex

import (
	"bytes"
	"encoding/json"

	"github.com/digitalocean/godo"
)

// rawSuppressedMethods is the denylist of native codex app-server
// notification methods tryRawPassthrough must NOT forward. Everything else
// with a method and no "id" forwards verbatim (ids rewritten) — a denylist
// rather than an allowlist because the frames come from the codex
// app-server's own stdout, so any notification it emits is by definition
// valid protocol for a codex client, including methods newer than this
// proxy. Paired with the OHR mapper's catch-all (plano codex/mapper.rs maps
// otherwise-untranslated notifications to a RunLog marker carrying
// source_raw), new protocol surface reaches the TUI with no doctl or
// mapper release in between.
//
// Server→client *requests* stay excluded structurally (frames with an "id"
// are never forwarded) because a request needs a proxy-minted JSON-RPC id
// and a reply pump, which the existing HITL flow already owns.
//
// thread/started is suppressed because the client's thread was created by
// this facade's own thread/start reply, and announcing the VM's would be a
// second, conflicting thread lifecycle.
var rawSuppressedMethods = map[string]bool{
	"thread/started": true,
}

// rawFrame is the JSON-RPC envelope of one native codex app-server message.
// ID is captured only to detect that a frame is a request/response rather
// than a notification — it is never forwarded.
type rawFrame struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// tryRawPassthrough is translateEvent's prefer-raw step: if ev carries a
// forwardable native codex frame, push it to the client (ids rewritten, see
// the package doc above) and report handled=true so the canonical
// reconstruction is skipped for this event. handled=false means "not this
// path's event" — no notification was sent and the caller must translate
// canonically. clientDead mirrors translateEvent's contract: a notify
// failure means the client connection is gone and the caller must stop.
//
// Turn bookkeeping the canonical path does inside its notifications is
// replicated here where it matters: run.started stamps ts.startedAt (so a
// canonical-fallback finishTurn later in the same turn can still compute a
// duration), and run.completed/run.failed untrack the run from f.turns —
// finishTurn's map delete without finishTurn's synthesized notifications,
// which the forwarded frame replaces.
func (f *Facade) tryRawPassthrough(ev godo.HostedAgentEvent, ts *turnState, replay bool) (handled, clientDead bool) {
	if !f.rawEligible() || len(ev.SourceRaw) == 0 {
		return false, false
	}

	// Historical usage is skipped on --replay by the canonical path so a
	// past session's token counts don't inflate this connection's running
	// total — the raw equivalent (the VM's thread/tokenUsage/updated, whose
	// "total" spans the VM thread's whole lifetime) is skipped for the same
	// reason. Handled (not fallback): the canonical case would only
	// re-decide the same skip.
	if replay && ev.Kind == godo.HostedAgentEventKindRunUsageRecorded {
		return true, false
	}

	var frame rawFrame
	if err := json.Unmarshal(bytes.TrimSpace(ev.SourceRaw), &frame); err != nil {
		return false, false
	}
	// Frames with an id are requests (or responses), not notifications —
	// see rawSuppressedMethods on why those never forward verbatim. A
	// missing method means the frame isn't a notification at all.
	if frame.ID != nil || frame.Method == "" || rawSuppressedMethods[frame.Method] {
		return false, false
	}

	params, ok := rewriteRawParams(frame.Params, f.SessionID, ev.RunID)
	if !ok {
		return false, false
	}

	switch ev.Kind {
	case godo.HostedAgentEventKindRunStarted:
		ts.startedAt = eventTime(ev).Unix()
	case godo.HostedAgentEventKindRunUsageRecorded:
		// Keep the connection-scoped running total warm even though the
		// forwarded frame carries the VM's own totals: if a later event in
		// this session has to fall back to canonical reconstruction, its
		// synthesized notification reports f.totalUsage — an accurate
		// fallback beats one that silently restarts from zero.
		f.accumulateUsage(ev.Payload)
	}

	if !f.notify(frame.Method, params) {
		return true, true
	}

	switch ev.Kind {
	case godo.HostedAgentEventKindRunCompleted, godo.HostedAgentEventKindRunFailed:
		// The forwarded turn/completed|failed replaced finishTurn's
		// notifications; its f.turns cleanup still has to happen so
		// drainStream's noTurnsLeft accounting sees the turn end. A no-op
		// for replayed history, which never registers turns in the shared
		// map (see replaySessionHistory).
		f.mu.Lock()
		delete(f.turns, ev.RunID)
		f.mu.Unlock()
	}
	return true, false
}

// rawTurnStartFrame rebuilds the client's turn/start message from the params
// Dispatch received, for SendInput's source_raw. The params bytes pass
// through untouched — that byte-fidelity is the whole point. The JSON-RPC
// envelope is rebuilt rather than captured because Dispatch never sees the
// original frame, and the two dropped envelope fields are dropped
// deliberately: the client's request id must not travel (the adapter's
// transport owns its own id space and would discard it anyway), and this
// facade already answers the TUI's turn/start itself with the synthesized
// run-id turn — upstream only ever needs the intent, not the RPC.
func rawTurnStartFrame(params json.RawMessage) []byte {
	if len(params) == 0 {
		return nil
	}
	frame := make([]byte, 0, len(params)+len(`{"method":"turn/start","params":}`))
	frame = append(frame, `{"method":"turn/start","params":`...)
	frame = append(frame, params...)
	frame = append(frame, '}')
	return frame
}

// rewriteRawParams decodes a native frame's params object and rewrites the
// VM-scoped identifiers to the ones this proxy's client knows (see the
// package doc comment): threadId → sessionID, and turnId / turn.id → runID.
// Numbers are decoded with UseNumber so re-encoding doesn't mangle int64
// timestamps into float notation. ok=false (nil, non-object, or unparsable
// params) tells the caller to fall back to canonical reconstruction.
func rewriteRawParams(raw json.RawMessage, sessionID, runID string) (map[string]any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var params map[string]any
	if err := dec.Decode(&params); err != nil || params == nil {
		return nil, false
	}
	if _, ok := params["threadId"]; ok {
		params["threadId"] = sessionID
	}
	if _, ok := params["turnId"]; ok && runID != "" {
		params["turnId"] = runID
	}
	if turn, ok := params["turn"].(map[string]any); ok && runID != "" {
		if _, ok := turn["id"]; ok {
			turn["id"] = runID
		}
	}
	return params, true
}
