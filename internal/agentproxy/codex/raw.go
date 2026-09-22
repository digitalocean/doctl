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
// This file also holds the inbound halves, which come in two shapes because
// the client sends two kinds of message.
//
// Notifications and turns (v2): rawTurnStartFrame packages the TUI's
// turn/start params as SendInput's source_raw, so the in-sandbox adapter
// drives the turn from the client's actual params (input items, model,
// effort, approval policy) instead of a synthesized single-text turn.
//
// Requests (M1): relayRequest forwards a client *request* — one that blocks
// waiting for an answer — to the agent and returns its reply. turn/start is
// deliberately not one of these: it already has a canonical meaning
// (SendInput) and a run to attribute events to, so relaying it around that
// would start a turn no one is tracking. Everything else this facade can't
// answer itself goes through, which is what makes turn/interrupt real and
// lets new codex methods work without a doctl release.
//
// The direction is inverted but the id discipline is symmetric. Outbound
// rewrites VM ids to proxy ids. Inbound ships the proxy-side ids as-is and
// the adapter rewrites them to the VM's own — it is the only component that
// knows those. JSON-RPC request ids never cross a hop at all: each side mints
// its own and answers on the id it was given.
package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/digitalocean/doctl/internal/agentproxy"
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
	case godo.HostedAgentEventKindRunCompleted, godo.HostedAgentEventKindRunFailed:
		// A raw turn end on a turn whose synthesized "-msg" agentMessage
		// item was opened by the canonical path (a canonical run.started,
		// or a canonical delta after a reconnect fell this turn back per
		// event) must not leave that item dangling open: the forwarded
		// frame replaces finishTurn's turn/completed but knows nothing of
		// the item this facade announced. Close it first, mirroring
		// finishTurn's item/completed-before-turn/completed ordering.
		if ts.itemStarted {
			if !f.notify("item/completed", itemCompletedNotification{
				Item:          agentMessageItem{Type: "agentMessage", ID: ts.itemID, Text: ts.text.String()},
				ThreadID:      f.SessionID,
				TurnID:        ev.RunID,
				CompletedAtMs: eventTime(ev).UnixMilli(),
			}) {
				return true, true
			}
			ts.itemStarted = false
		}
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

// relayRequest forwards one client request this facade has no synthesized
// answer for to the session's real agent, and returns the agent's reply as
// Dispatch's result.
//
// The synthesized replies elsewhere in Dispatch exist because the harness had
// no way to ask the agent anything; this is that way. A method left to a
// stub answers plausibly but falsely (turn/interrupt returning success while
// the turn keeps running), and a method with no stub at all is answered with
// "method not found" — visible in the TUI as a broken slash command. Relaying
// puts the real agent behind both cases.
//
// The client's JSON-RPC id never travels: each hop owns its own id space.
// This facade mints one for the relayed frame, and the bridge addresses the
// reply to whatever the client sent. What comes back is unwrapped to result
// or RPCError, so an agent-side failure reaches the client as a failure of
// its own request rather than a proxy error.
//
// Returns handled=false when relaying is not available for this session or
// the agent declined the method — the caller then falls back to whatever it
// would have done before.
func (f *Facade) relayRequest(ctx context.Context, method string, params json.RawMessage) (result any, handled bool, err error) {
	if !f.rawEligible() {
		return nil, false, nil
	}
	frame, ok := rawRequestFrame(method, params)
	if !ok {
		return nil, false, nil
	}
	resp, relayErr := f.Sessions.RelayRequest(ctx, f.SessionID, &godo.HostedAgentRelayRequest{SourceRaw: frame})
	if relayErr != nil {
		// The agent is reachable in principle but this request did not get
		// through. Report it as a failure of the client's request rather than
		// silently falling back to a stub that would claim success.
		log.Printf("codex facade: relay %s failed: %v", method, relayErr)
		return nil, true, &agentproxy.RPCError{
			Code:    -32000,
			Message: fmt.Sprintf("relaying %s to the hosted agent failed: %v", method, relayErr),
		}
	}
	if resp == nil || len(resp.SourceRaw) == 0 {
		// Declined by the in-sandbox adapter (harness-owned or unknown
		// method). Not an error: the caller answers it.
		return nil, false, nil
	}
	return unwrapRelayReply(resp.SourceRaw, method)
}

// rawRequestFrame builds the JSON-RPC request frame to relay. The id is a
// fixed placeholder rather than the client's: the adapter mints its own id
// for the request it drives and echoes this one straight back, so it only has
// to be present and non-null (a frame without an id is a notification, which
// the adapter declines).
func rawRequestFrame(method string, params json.RawMessage) ([]byte, bool) {
	if method == "" {
		return nil, false
	}
	envelope := struct {
		ID     int             `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params,omitempty"`
	}{ID: 1, Method: method, Params: params}
	frame, err := json.Marshal(envelope)
	if err != nil {
		return nil, false
	}
	return frame, true
}

// unwrapRelayReply turns the agent's JSON-RPC reply frame into Dispatch's
// (result, error) pair. An error object becomes an RPCError carrying the
// agent's own code and message, so the client sees the agent's verdict rather
// than a proxy-flavoured translation of it.
func unwrapRelayReply(frame []byte, method string) (any, bool, error) {
	var reply struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(frame), &reply); err != nil {
		return nil, true, &agentproxy.RPCError{
			Code:    -32000,
			Message: fmt.Sprintf("hosted agent returned an unreadable reply to %s: %v", method, err),
		}
	}
	if reply.Error != nil {
		return nil, true, &agentproxy.RPCError{Code: reply.Error.Code, Message: reply.Error.Message}
	}
	if len(reply.Result) == 0 {
		// A reply with neither result nor error still resolves the client's
		// request; an empty object is the honest rendering of "done, nothing
		// to report".
		return struct{}{}, true, nil
	}
	return json.RawMessage(reply.Result), true, nil
}

// tryNativeApproval takes ownership of a gated request when the native frame
// the agent sent is available and the client speaks its protocol, returning
// true once it has (the round-trip itself runs in its own goroutine, for the
// reason given on translateEvent's HITLRequested case).
//
// This is the server→client direction of the passthrough that tryRawPassthrough
// does for notifications, and it exists because the canonical HITL payload is
// a lossy rendering of the request: it reduces the agent's question to a kind
// and a redacted detail map, which is enough to render an approve/decline
// prompt and not enough for anything else. An MCP elicitation wants content
// back; a tool asking for user input wants answers. Forwarding the agent's own
// frame and returning the client's own reply keeps both intact, and means a
// gated request this proxy has never heard of still reaches the user.
//
// false means "not this path's event": no request was sent, and the caller
// must fall back to synthesizing one from the canonical payload.
func (f *Facade) tryNativeApproval(ctx context.Context, ev godo.HostedAgentEvent, hitl hitlRequestedPayload) bool {
	if !f.rawEligible() || len(ev.SourceRaw) == 0 {
		return false
	}
	var frame rawFrame
	if err := json.Unmarshal(bytes.TrimSpace(ev.SourceRaw), &frame); err != nil {
		return false
	}
	// The mirror of tryRawPassthrough's test: there a frame with an id is
	// rejected because it is not a notification, here one without an id is,
	// because only a request has a reply to carry back.
	if frame.ID == nil || frame.Method == "" {
		return false
	}
	params, ok := rewriteRawParams(frame.Params, f.SessionID, ev.RunID)
	if !ok {
		return false
	}
	go f.answerNativeApproval(ctx, hitl.HitlID, frame, params)
	return true
}

// answerNativeApproval asks the client the agent's own question and sends the
// client's own answer back, resolving the harness's HITL request with both the
// raw reply and the coarse verdict the audit trail records.
func (f *Facade) answerNativeApproval(ctx context.Context, hitlID string, frame rawFrame, params map[string]any) {
	result, err := f.notifier.Request(ctx, frame.Method, params)
	if err != nil {
		f.rejectHITLAfterRequestFailure(hitlID, frame.Method, err)
		return
	}
	reply, err := json.Marshal(struct {
		ID     json.RawMessage `json:"id"`
		Result json.RawMessage `json:"result"`
	}{ID: frame.ID, Result: result})
	if err != nil {
		// Nothing sensible left to forward, but the agent is still blocked, so
		// resolve on the verdict alone rather than abandoning the request.
		log.Printf("codex facade: native reply %s: re-encode failed: %v", hitlID, err)
		reply = nil
	}
	if err := f.Sessions.ResolveHITL(f.SessionID, hitlID, &godo.HostedAgentResolveHITLRequest{
		Outcome:   nativeApprovalOutcome(result),
		Source:    godo.HostedAgentResolutionSourceInlineKeystroke,
		SourceRaw: reply,
	}); err != nil {
		log.Printf("codex facade: ResolveHITL %s failed: %v", hitlID, err)
	}
}

// nativeApprovalOutcome reduces a native reply to the three-way verdict the
// control plane persists.
//
// Only the audit trail and the agent's own fallback depend on this: the reply
// the agent acts on is the raw frame, which is forwarded verbatim. Codex
// spells a refusal in one of two fields depending on which surface asked, so
// both are checked. Anything else is an answer the user chose to give, which
// is an approval — including reply shapes with no verdict field at all, like
// a tool's `answers`.
func nativeApprovalOutcome(result json.RawMessage) godo.HostedAgentHITLOutcome {
	var body struct {
		Decision json.RawMessage `json:"decision"`
		Action   string          `json:"action"`
	}
	if err := json.Unmarshal(result, &body); err != nil {
		return godo.HostedAgentHITLOutcomeApprove
	}
	if len(body.Decision) > 0 {
		if outcome, err := decodeCommandExecutionApprovalDecision(body.Decision); err == nil {
			return outcome
		}
	}
	switch body.Action {
	case "decline", "cancel", "reject":
		return godo.HostedAgentHITLOutcomeReject
	}
	return godo.HostedAgentHITLOutcomeApprove
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
