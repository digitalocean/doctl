package opencode

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/digitalocean/godo"
)

// M5: the approval round-trip. The hosted session's policy gates a tool call
// and the adapter surfaces the guest's own permission.asked as
// run.human_input_requested, whose payload is that event's properties verbatim
// (minus the guest's request id — plano's raw.rs redacts it). The facade
// re-minted the ask for the TUI, the TUI's reply resolves the HITL through the
// harness, and permission.replied closes the loop:
//
//	run.human_input_requested → permission.asked (per_ id minted here)
//	POST /permission/{id}/reply {"reply":...}  → ResolveHITL(outcome)
//	run.human_input_received → permission.replied
//
// Fidelity notes, both documented behavior rather than bugs:
//   - "always" degrades to a one-time approve. The canonical outcome enum has
//     no sticky-approval; the adapter turns HITL_OUTCOME_APPROVE into a "once"
//     reply to the guest, so the guest asks again next time. (The raw
//     passthrough section of the plan is the eventual fix.)
//   - The reply's optional free-text message rides ResolveHITL.reason. That
//     reaches the harness's audit trail and — for question-kind HITLs — the
//     guest's answer text, but opencode's permission-reply API has no note
//     field, so for permission asks it is audit-only.

// pendingPerm is one outstanding ask bridged to the TUI, keyed both by the
// minted per_ id (the reply routes look up by it) and by the harness hitl id
// (run.human_input_received resolves by it).
type pendingPerm struct {
	hitlID string
	perID  string
	runID  string
	// reply records the client's exact reply string ("once"/"always"/
	// "reject") once it has replied, so the permission.replied broadcast
	// echoes what the client actually chose; empty until then (an ask
	// resolved out-of-band maps from the canonical outcome instead).
	reply string
}

// hitlRequestedPayload is the canonical run.human_input_requested data: the
// harness-minted hitl id plus the adapter's passthrough of the guest event's
// properties (spi/events.go HumanInputRequestedData).
type hitlRequestedPayload struct {
	HitlID  string         `json:"hitl_id"`
	Payload map[string]any `json:"payload"`
}

// handleHITLRequested bridges one ask to the TUI. Question-kind HITLs (the
// guest's question.asked, payload {"category":"question",...}) are
// auto-rejected with a reason: opencode's TUI has no question dialog to drive,
// and leaving the HITL pending would hang the run silently (opencode HITLs
// carry no deadline, so nothing would ever time it out).
func (f *Facade) handleHITLRequested(ev godo.HostedAgentEvent, ts *turnState, ew *eventWriter, at int64) error {
	var payload hitlRequestedPayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil || payload.HitlID == "" {
		log.Printf("agentproxy/opencode: unparseable human_input_requested payload")
		return nil
	}
	if payload.Payload["category"] == "question" {
		// Off the event-loop goroutine: ResolveHITL is an HTTP round-trip and
		// must not stall stream translation (same shape as the codex facade's
		// auto-reject goroutines).
		go f.autoRejectHITL(payload.HitlID, "question-style prompts are not supported by the opencode proxy yet; re-run the request without requiring an answer")
		return nil
	}
	// An ask for a turn whose start this stream never saw (mid-turn connect):
	// announce the turn first so the ask's tool reference has a message.
	if !ts.startedSent {
		if err := f.announceTurn(ev.RunID, ts, ew, at); err != nil {
			return err
		}
	}

	props := make(map[string]any, len(payload.Payload)+2)
	for k, v := range payload.Payload {
		props[k] = v
	}
	// The payload's sessionID is the guest opencode's own ses_ id — the TUI
	// filters events by session, so it must be rewritten to the facade's.
	sid := f.ocSessionID()
	props["sessionID"] = sid
	// Mint the client-facing request id (per_ prefix per the TUI's schema
	// pattern). The eventSeq counter keeps two asks in the same millisecond
	// distinct — the time+run tail alone would collide.
	perID := ocTimeID("per_", at, uint16(f.eventSeq.Add(1)), runTail(ev.RunID, "pe"))
	props["id"] = perID
	// Schema-required fields the details may lack (id/sessionID/permission/
	// patterns/metadata/always are all required at TestedVersion).
	if _, ok := props["permission"]; !ok {
		props["permission"] = "unknown"
	}
	if _, ok := props["patterns"]; !ok {
		props["patterns"] = []any{}
	}
	if _, ok := props["metadata"]; !ok {
		props["metadata"] = map[string]any{}
	}
	if _, ok := props["always"]; !ok {
		props["always"] = []any{}
	}
	// tool.messageID references a guest-side message id this TUI has never
	// seen; remap it to the facade's current assistant message (the dialog
	// anchors to it) or drop the reference entirely.
	if tool, ok := props["tool"].(map[string]any); ok {
		if ts.asstMsgID != "" {
			remapped := map[string]any{"messageID": ts.asstMsgID}
			if callID, ok := tool["callID"]; ok {
				remapped["callID"] = callID
			}
			props["tool"] = remapped
		} else {
			delete(props, "tool")
		}
	}

	p := &pendingPerm{hitlID: payload.HitlID, perID: perID, runID: ev.RunID}
	f.mu.Lock()
	if f.perms == nil {
		f.perms = map[string]*pendingPerm{}
		f.permsByHitl = map[string]*pendingPerm{}
	}
	f.perms[perID] = p
	f.permsByHitl[payload.HitlID] = p
	f.mu.Unlock()

	return ew.session("permission.asked", props)
}

// handleHITLResolved broadcasts permission.replied for a resolved ask. This is
// the single place the replied frame is emitted — the reply HTTP handler does
// not write to the stream (single-writer rule, see eventWriter) — and it also
// covers out-of-band resolutions (another device's `doctl agents attach`, a
// policy auto-decision), which the TUI must reconcile too.
func (f *Facade) handleHITLResolved(ev godo.HostedAgentEvent, ew *eventWriter) error {
	var payload struct {
		HitlID string `json:"hitl_id"`
		// Proto enum on the wire: 1=APPROVE, 2=REJECT, 3=DEFER.
		Outcome int32 `json:"outcome"`
	}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		return nil
	}
	f.mu.Lock()
	p := f.permsByHitl[payload.HitlID]
	if p != nil {
		delete(f.permsByHitl, p.hitlID)
		delete(f.perms, p.perID)
	}
	f.mu.Unlock()
	if p == nil {
		// An ask this facade never showed: a question auto-reject's ack, or a
		// request that predates this proxy. Nothing to reconcile.
		return nil
	}
	reply := p.reply
	if reply == "" {
		reply = "reject"
		if payload.Outcome == 1 {
			reply = "once"
		}
	}
	return f.emitPermissionReplied(ew, p.perID, reply)
}

func (f *Facade) emitPermissionReplied(ew *eventWriter, perID, reply string) error {
	return ew.session("permission.replied", map[string]any{
		"sessionID": f.ocSessionID(),
		"requestID": perID,
		"reply":     reply,
	})
}

// closePendingPerms dismisses asks still pending when their run ends — the
// run died with the dialog up, and a dialog whose permission.replied never
// arrives stays on screen forever.
func (f *Facade) closePendingPerms(runID string, ew *eventWriter) error {
	f.mu.Lock()
	var stale []*pendingPerm
	for _, p := range f.perms {
		if p.runID == runID {
			stale = append(stale, p)
		}
	}
	for _, p := range stale {
		delete(f.perms, p.perID)
		delete(f.permsByHitl, p.hitlID)
	}
	f.mu.Unlock()
	for _, p := range stale {
		reply := p.reply
		if reply == "" {
			reply = "reject"
		}
		if err := f.emitPermissionReplied(ew, p.perID, reply); err != nil {
			return err
		}
	}
	return nil
}

// handlePermissionReply is the shared reply path behind both client routes:
// POST /permission/{requestID}/reply {"reply":...,"message":...} (what the
// TestedVersion TUI sends — captured live) and the session-scoped
// POST /session/{id}/permissions/{permissionID} {"response":...} (the older
// route plano's adapter drives; kept for older clients). Replies resolve the
// HITL with the harness; the permission.replied broadcast rides the event
// stream when run.human_input_received comes back.
func (f *Facade) handlePermissionReply(w http.ResponseWriter, perID, reply, message string) {
	var outcome godo.HostedAgentHITLOutcome
	switch reply {
	case "once", "always":
		// "always" is approve-without-stickiness — see the fidelity note atop
		// this file.
		outcome = godo.HostedAgentHITLOutcomeApprove
	case "reject":
		outcome = godo.HostedAgentHITLOutcomeReject
	default:
		http.Error(w, fmt.Sprintf("unknown permission reply %q", reply), http.StatusBadRequest)
		return
	}
	// Record the client's reply string BEFORE resolving: the harness can
	// deliver run.human_input_received (whose handler echoes p.reply in
	// permission.replied) the instant the resolve call lands, racing a
	// write placed after it. Rolled back if the resolve fails — the ask is
	// then still pending and unreplied.
	f.mu.Lock()
	p := f.perms[perID]
	if p != nil {
		p.reply = reply
	}
	f.mu.Unlock()
	if p == nil {
		// Matches the real server's PermissionNotFoundError behavior for
		// stale/foreign ids.
		http.Error(w, "permission request not found", http.StatusNotFound)
		return
	}
	if err := f.Sessions.ResolveHITL(f.SessionID, p.hitlID, &godo.HostedAgentResolveHITLRequest{
		Outcome: outcome,
		Reason:  message,
		Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
	}); err != nil {
		f.mu.Lock()
		p.reply = ""
		f.mu.Unlock()
		http.Error(w, fmt.Sprintf("resolving the permission with the hosted session failed: %v", err), http.StatusBadGateway)
		return
	}
	// The real server answers the reply POST with a bare `true` (captured).
	f.writeJSON(w, true)
}

// autoRejectHITL resolves a HITL this facade can't surface to the client,
// with a reason for the audit trail. Runs on its own goroutine.
func (f *Facade) autoRejectHITL(hitlID, reason string) {
	log.Printf("agentproxy/opencode: auto-rejecting HITL %s: %s", hitlID, reason)
	if err := f.Sessions.ResolveHITL(f.SessionID, hitlID, &godo.HostedAgentResolveHITLRequest{
		Outcome: godo.HostedAgentHITLOutcomeReject,
		Reason:  reason,
		Source:  godo.HostedAgentResolutionSourceOutOfBand,
	}); err != nil {
		log.Printf("agentproxy/opencode: auto-reject of HITL %s failed: %v", hitlID, err)
	}
}
