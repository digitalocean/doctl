package opencode

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

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
// Fidelity notes, documented behavior rather than bugs:
//   - "Allow always" is emulated proxy-side. The canonical HITL outcome enum
//     has no sticky approval, so an "always" reply reaches the guest as a
//     one-time approve and the guest would re-ask next time. To make the
//     button do what users expect, the facade remembers the ask's own
//     persist-patterns (the guest-supplied `always` globs) for the proxy's
//     lifetime and auto-approves later matching asks itself, without a dialog
//     — see allowAlways / matchesAllowAlways. Caveats, all deliberate: the
//     memory is in-process (a proxy restart forgets it), the auto-approvals
//     resolve out-of-band (the audit trail attributes them to the proxy, not
//     a per-ask keystroke), and the glob match (globMatch) is the facade's
//     own approximation of the guest's matcher. The durable fix is the raw
//     ResolveHITL.source_raw passthrough so the guest persists the rule
//     itself (plan's passthrough section).
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
	// permission and always carry what an "always" reply should remember: the
	// permission type (e.g. "bash") and the guest-supplied persist-patterns
	// from this ask (the `always` globs). See allowAlways.
	permission string
	always     []string
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
		log.Printf("agentproxy/opencode: unparsable human_input_requested payload")
		return nil
	}
	if payload.Payload["category"] == "question" {
		// Off the event-loop goroutine: ResolveHITL is an HTTP round-trip and
		// must not stall stream translation (same shape as the codex facade's
		// auto-reject goroutines).
		go f.autoRejectHITL(payload.HitlID, "question-style prompts are not supported by the opencode proxy yet; re-run the request without requiring an answer")
		return nil
	}

	permission, _ := payload.Payload["permission"].(string)
	command := askCommand(payload.Payload)
	// A prior "Allow always" for a matching command auto-approves this ask
	// with no dialog — the proxy-side emulation of sticky approval (see the
	// fidelity note atop this file).
	if f.matchesAllowAlways(permission, command) {
		go f.autoApproveHITL(payload.HitlID, permission, command)
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

	p := &pendingPerm{
		hitlID: payload.HitlID, perID: perID, runID: ev.RunID,
		permission: permission, always: alwaysPatterns(payload.Payload),
	}
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
	// "Allow always": now that the resolve succeeded, remember this ask's
	// persist-patterns so future matching asks auto-approve without a dialog
	// (proxy-side sticky-approval emulation — see the fidelity note atop this
	// file). p.reply was already recorded before the resolve (race fix above).
	if reply == "always" && p.permission != "" && len(p.always) > 0 {
		f.mu.Lock()
		if f.allowAlways == nil {
			f.allowAlways = map[string][]string{}
		}
		f.allowAlways[p.permission] = append(f.allowAlways[p.permission], p.always...)
		f.mu.Unlock()
	}
	// The real server answers the reply POST with a bare `true` (captured).
	f.writeJSON(w, true)
}

// matchesAllowAlways reports whether a prior "Allow always" covers this
// command: any remembered glob for this permission type matches it. A blank
// command never matches (nothing to test against) — the ask is surfaced.
func (f *Facade) matchesAllowAlways(permission, command string) bool {
	if permission == "" || command == "" {
		return false
	}
	f.mu.Lock()
	globs := f.allowAlways[permission]
	f.mu.Unlock()
	for _, g := range globs {
		if globMatch(g, command) {
			return true
		}
	}
	return false
}

// autoApproveHITL approves an ask a prior "Allow always" already covered,
// out-of-band and with no client-facing dialog. Runs on its own goroutine
// (ResolveHITL is an HTTP round-trip). The guest's run.human_input_received
// then has no pendingPerm to reconcile, so no permission.replied is emitted —
// the tool simply runs, which is the point.
func (f *Facade) autoApproveHITL(hitlID, permission, command string) {
	log.Printf("agentproxy/opencode: auto-approving HITL %s (%s %q matched an Allow-always rule)", hitlID, permission, command)
	if err := f.Sessions.ResolveHITL(f.SessionID, hitlID, &godo.HostedAgentResolveHITLRequest{
		Outcome: godo.HostedAgentHITLOutcomeApprove,
		Source:  godo.HostedAgentResolutionSourceOutOfBand,
	}); err != nil {
		log.Printf("agentproxy/opencode: auto-approve of HITL %s failed: %v", hitlID, err)
	}
}

// askCommand pulls the human-readable command an ask is gating: bash-shaped
// asks carry it in metadata.command, otherwise the first pattern stands in.
// Empty when neither is present (a permission type this heuristic doesn't
// cover) — such asks never auto-approve.
func askCommand(props map[string]any) string {
	if md, ok := props["metadata"].(map[string]any); ok {
		if cmd, ok := md["command"].(string); ok && cmd != "" {
			return cmd
		}
	}
	for _, p := range toStrings(props["patterns"]) {
		if p != "" {
			return p
		}
	}
	return ""
}

// alwaysPatterns is the ask's guest-supplied persist-globs (the `always`
// field): what an "Allow always" on this ask should remember.
func alwaysPatterns(props map[string]any) []string {
	return toStrings(props["always"])
}

// toStrings coerces a JSON array-of-strings (as decoded into []any or already
// []string) to []string, dropping non-strings.
func toStrings(v any) []string {
	switch xs := v.(type) {
	case []string:
		return xs
	case []any:
		out := make([]string, 0, len(xs))
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// globMatch matches a shell-style glob (as opencode's `always` patterns use)
// against a command: `*` spans any run of characters (including spaces and
// `/`, unlike filepath.Match, so "cat *" covers "cat /etc/passwd") and `?`
// matches one. It is the facade's own approximation of the guest's matcher —
// deliberately permissive, since a false match only ever broadens what a user
// already chose to always-allow within this proxy's lifetime.
func globMatch(glob, s string) bool {
	re := globToRegexp(glob)
	return re.MatchString(s)
}

func globToRegexp(glob string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for _, r := range glob {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	// Cannot fail: QuoteMeta-escaped literals plus .*/. are always valid.
	re, _ := regexp.Compile(b.String())
	return re
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
