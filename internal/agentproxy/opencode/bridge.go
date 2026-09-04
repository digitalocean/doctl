package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/digitalocean/godo"
)

func timeNowMs() int64 { return time.Now().UnixMilli() }

// M2: the harness bridge. Client prompts go to SendInput; the hosted
// session's canonical event stream comes back as opencode-native SSE frames.
// The translation tables are the inverse of plano's opencode adapter
// (crates/agent_adapters/src/opencode/raw.rs) — when unsure how a canonical
// event should look in opencode terms, that adapter is the answer key.

// turnState tracks one in-flight hosted run and the opencode ids synthesized
// for it. Created when SendInput returns the run id; deleted on
// run.completed/run.failed. Events for runs this facade isn't tracking are
// skipped, same as the codex facade — they belong to another client's turn
// or predate this connection (M3's history endpoint covers the latter).
type turnState struct {
	// userMsgID/promptText echo the user's message back on the stream when
	// the turn starts — the real server does this (message.updated role
	// "user" + a text part carrying the prompt), and the TUI renders the
	// user's side of the conversation from those frames, not optimistically.
	// userMsgID is the client-minted id from the prompt body when present.
	userMsgID  string
	promptText string
	// asstMsgID and the part ids are minted time-ordered (ocTimeID) at
	// announce time so the conversation sorts correctly; stored here so
	// every later frame references the same ids.
	asstMsgID       string
	userPartID      string
	textPartID      string
	reasoningPartID string
	// startedSent: the turn-start frames (user echo, busy, assistant
	// message.updated) went out. Normally set by run.started, but set lazily
	// by the first delta too — a turn whose run.started this stream never
	// saw (mid-turn connect) must not emit deltas for a message the TUI was
	// never told about.
	startedSent bool
	// textPartStarted / reasoningPartStarted: the empty-text
	// message.part.updated announcing each part went out. Load-bearing
	// order (confirmed by the ground-truth capture): the TUI applies a
	// message.part.delta only to a part a message.part.updated already
	// created — a delta for an unannounced part renders nothing, silently.
	textPartStarted      bool
	reasoningPartStarted bool
	// sawText: at least one non-reasoning delta was emitted, so the
	// finalizing message.part.updated has content to carry.
	sawText bool
	// text accumulates the answer for the finalizing part update. The real
	// server re-carries the full text in the finalized part (plano's adapter
	// drops it as a duplicate for exactly that reason) — the TUI expects the
	// part's final state, not only the delta trail.
	text strings.Builder
	// startMs stamps part time.start; the finalizer adds time.end.
	startMs int64
	// tools tracks in-flight tool calls by canonical tool_call_id, so the
	// completion frame can re-carry the input (the TUI keeps the part's
	// final state) and dangling tools can be closed out on turn end.
	tools map[string]*toolCall
	// Per-turn totals from run.completed's payload, rendered as the
	// step-finish part and the final assistant message's tokens/cost.
	tokensIn, tokensOut, costMicros int64
}

// toolCall is one tracked tool invocation within a turn.
type toolCall struct {
	partID  string
	name    string
	input   json.RawMessage
	startMs int64
	done    bool
}

// ocID strips the dashes from a harness UUID so synthesized ids match the
// dash-less shape opencode mints (ses_..., msg_..., prt_...), in case any
// client pattern-matches id prefixes or separators.
func ocID(prefix, uuid string) string {
	return prefix + strings.ReplaceAll(uuid, "-", "")
}

// opencode ids embed a 48-bit time value: T = ((unix_ms << 12) | counter)
// truncated to 48 bits, rendered as 12 hex chars after the prefix (sessions
// use the bitwise NOT of T, which is why ses_ ids sort newest-first). The
// encoding was decoded from a real server's ids and verified to the bit.
//
// This matters because the TUI orders messages lexicographically by id — the
// time prefix is what makes id order chronological order. The first M2 cut
// derived message ids from the run UUID (random), and the TUI rendered the
// assistant's answer ABOVE the user's prompt (found live).
const ocTimeMask = uint64(1)<<48 - 1

// ocTimeVal is the 48-bit T for a millisecond timestamp (counter zero). All
// ordering comparisons must happen on T values, never on raw milliseconds —
// T is truncated mod 2^48, so a T recovered from an id and a fresh untruncated
// ms are not comparable (that mismatch shipped once: the clock-skew guard
// below silently never fired and the conversation stayed inverted).
func ocTimeVal(ms int64) uint64 { return (uint64(ms) << 12) & ocTimeMask }

// ocIDWithT mints an opencode-shaped ascending id from an explicit T.
func ocIDWithT(prefix string, t uint64, counter uint16, tail string) string {
	return fmt.Sprintf("%s%012x%s", prefix, (t|uint64(counter&0xfff))&ocTimeMask, tail)
}

// ocTimeID is ocIDWithT for callers that start from a timestamp.
func ocTimeID(prefix string, ms int64, counter uint16, tail string) string {
	return ocIDWithT(prefix, ocTimeVal(ms), counter, tail)
}

// tFromOCID recovers T (counter bits cleared) from an id's 12-hex-char time
// prefix. ok=false for ids that don't parse (foreign format, too short).
func tFromOCID(id string) (uint64, bool) {
	i := strings.IndexByte(id, '_')
	if i < 0 || len(id) < i+13 {
		return 0, false
	}
	t, err := strconv.ParseUint(id[i+1:i+13], 16, 64)
	if err != nil {
		return 0, false
	}
	return t &^ 0xfff, true
}

// runTail is the id tail: a fragment of the run UUID for debuggability plus
// a role marker, standing in for the real server's random base62 tail.
func runTail(runID, marker string) string {
	s := strings.ReplaceAll(runID, "-", "")
	if len(s) > 10 {
		s = s[:10]
	}
	return s + marker
}

// ocSessionID is the opencode-side id this facade advertises for the one
// hosted session it bridges. Stable across reconnects on purpose: the TUI's
// own scrollback and session picker key off it.
func (f *Facade) ocSessionID() string { return ocID("ses_", f.SessionID) }

// sessionObject is the opencode session Info shape the TUI consumes from
// POST /session, GET /session/{id}, and the session list.
func (f *Facade) sessionObject() map[string]any {
	return map[string]any{
		"id":        f.ocSessionID(),
		"projectID": "global",
		"directory": f.Dir,
		"title":     "Hosted session " + f.SessionID,
		"version":   TestedVersion,
		"time":      map[string]any{"created": 0, "updated": 0},
	}
}

// handleSessionCreate answers POST /session. The facade bridges exactly one
// hosted session, so every create maps to it — the TUI creating "a new
// session" gets the same hosted session back. Model choice, workspace, and
// policy were all fixed when the hosted session was created; there is
// nothing a second session could bind to.
func (f *Facade) handleSessionCreate(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.sessionCreated = true
	f.mu.Unlock()
	f.writeJSON(w, f.sessionObject())
}

// handleSessionList answers GET /session: the bridged session appears when
// this proxy has used it (sessionCreated) or when the hosted session has
// replayable history — that's what lets `opencode attach --continue` and the
// session picker resume prior turns through a freshly started proxy. A truly
// fresh session lists empty, the "fresh server" state the TUI expects on
// first attach (verified in the M0 capture).
func (f *Facade) handleSessionList(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	created := f.sessionCreated
	f.mu.Unlock()
	if !created && !f.hasHistory(r.Context()) {
		f.writeJSON(w, []any{})
		return
	}
	f.writeJSON(w, []any{f.sessionObject()})
}

// promptInput is the POST /session/{id}/prompt_async body (opencode's
// PromptInput, minus fields this facade has no use for). Only text parts
// are bridged in M2; file/agent/subtask parts are logged and dropped.
type promptInput struct {
	MessageID string `json:"messageID"`
	Parts     []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"parts"`
	// Model is accepted and ignored: the hosted session's model was pinned
	// at create time. The synthetic catalog only advertises one model, so a
	// well-behaved client can't ask for anything else anyway.
	Model json.RawMessage `json:"model"`
}

// handlePromptAsync bridges a prompt to the hosted session: concatenate the
// text parts, SendInput, track the returned per-turn run id so the event
// loop can translate that run's events back. 204 on success — the real
// server answers prompt_async with 204 NO_CONTENT and results ride the
// event stream.
func (f *Facade) handlePromptAsync(w http.ResponseWriter, r *http.Request) {
	var body promptInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("malformed prompt body: %v", err), http.StatusBadRequest)
		return
	}
	var texts []string
	for _, p := range body.Parts {
		switch p.Type {
		case "text":
			if p.Text != "" {
				texts = append(texts, p.Text)
			}
		default:
			log.Printf("agentproxy/opencode: dropping unsupported prompt part type %q", p.Type)
		}
	}
	text := strings.Join(texts, "\n\n")
	if text == "" {
		http.Error(w, "prompt has no text parts", http.StatusBadRequest)
		return
	}

	resp, err := f.Sessions.SendInput(f.SessionID, &godo.HostedAgentSendInputRequest{Text: text})
	if err != nil {
		// 502, not 500: the facade is fine, the harness leg failed. The TUI
		// surfaces the body text.
		http.Error(w, fmt.Sprintf("sending input to the hosted session failed: %v", err), http.StatusBadGateway)
		return
	}

	userMsgID := body.MessageID
	if userMsgID == "" {
		userMsgID = ocTimeID("msg_", timeNowMs(), 0, runTail(resp.RunID, "us"))
	}
	f.mu.Lock()
	if f.turns == nil {
		f.turns = make(map[string]*turnState)
	}
	f.turns[resp.RunID] = &turnState{userMsgID: userMsgID, promptText: text}
	f.sessionCreated = true
	f.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (f *Facade) lookupTurn(runID string) (*turnState, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ts, ok := f.turns[runID]
	return ts, ok
}

func (f *Facade) dropTurn(runID string) {
	f.mu.Lock()
	delete(f.turns, runID)
	f.mu.Unlock()
}

// Reconnect schedule for runEventLoop's StreamSession loop. Mirrors the codex
// facade's constants (which themselves mirror doctl agents attach) —
// duplicated a third time now; if these ever need to change, extract a shared
// package rather than editing three copies.
const (
	maxAutoReconnectAttempts = 5
	initialReconnectBackoff  = 1 * time.Second
	maxReconnectBackoff      = 30 * time.Second
)

// healthyStreamDuration is how long a harness stream must stay connected
// before a drop counts as a normal idle timeout (resetting the reconnect
// budget) rather than a failing connection. Var, not const, so tests can
// shrink it.
var healthyStreamDuration = 30 * time.Second

func nextReconnectBackoff(cur time.Duration) time.Duration {
	next := cur * 2
	if next > maxReconnectBackoff {
		return maxReconnectBackoff
	}
	return next
}

// sleepCtx waits for d or returns false immediately if ctx is done.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// reconnectSleep is the backoff wait between reconnect attempts, behind an
// atomic.Pointer for the same reason as the codex facade's identical hook: a
// test's event loop can outlive the test that started it and race a later
// test's reassignment under -race.
var reconnectSleep atomic.Pointer[func(context.Context, time.Duration) bool]

func init() {
	fn := sleepCtx
	reconnectSleep.Store(&fn)
}

func setReconnectSleepForTest(fn func(context.Context, time.Duration) bool) (restore func()) {
	old := reconnectSleep.Load()
	reconnectSleep.Store(&fn)
	return func() { reconnectSleep.Store(old) }
}

// isTerminalStreamError reports whether reconnecting is pointless — auth
// failure, session gone, or a conflicting single-connection consumer (a
// concurrent `doctl agents attach` holds the same per-session slot this
// proxy's stream needs).
func isTerminalStreamError(err error) bool {
	var er *godo.ErrorResponse
	if !errors.As(err, &er) || er.Response == nil {
		return false
	}
	switch er.Response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict:
		return true
	}
	return false
}

// runEventLoop is the live StreamSession drain for one connected event
// stream. It runs inside the /global/event handler goroutine — the SSE
// response writer has exactly one writer by construction, so no notifier
// mutex is needed the way the WS transport's wsNotifier is.
//
// M6: the loop reconnects across transient harness-stream drops inside the
// one client SSE response, resuming with a replay_from cursor so no event is
// lost or duplicated. The cursor is the last event id SEEN, recorded
// unconditionally and never compared — event ids are random UUIDv4s, so a
// max-id ratchet would skip events (the codex facade's shipped drainStream
// bug; the test harness mints random ids for exactly this reason).
//
// The loop exits (ending the SSE response, which the TUI answers with its own
// re-attach) when the client is gone, the proxy is shutting down, a terminal
// stream error makes retrying pointless, the reconnect budget is exhausted,
// or — the common case — the harness stream ends with no turn in flight:
// nothing is owed to the client, and the TUI's next prompt arrives on a fresh
// handler anyway. Turns still in flight at give-up are failed loudly
// (failTrackedTurns) rather than left spinning.
func (f *Facade) runEventLoop(ctx context.Context, ew *eventWriter) {
	var cursor string
	backoff := initialReconnectBackoff
	failures := 0

	for {
		if ctx.Err() != nil {
			return
		}
		var opts *godo.HostedAgentSessionStreamOptions
		if cursor != "" {
			opts = &godo.HostedAgentSessionStreamOptions{ReplayFrom: cursor}
		}
		stream, err := f.Sessions.StreamSession(ctx, f.SessionID, opts)
		if err != nil {
			if isTerminalStreamError(err) {
				log.Printf("agentproxy/opencode: StreamSession failed (terminal), giving up: %v", err)
				f.failTrackedTurns(ew, "hosted session stream unavailable: "+err.Error())
				return
			}
			failures++
			if failures >= maxAutoReconnectAttempts {
				log.Printf("agentproxy/opencode: StreamSession failed %d times in a row, giving up: %v", failures, err)
				f.failTrackedTurns(ew, "lost connection to the hosted session")
				return
			}
			log.Printf("agentproxy/opencode: StreamSession failed, reconnecting: %v", err)
			if !(*reconnectSleep.Load())(ctx, backoff) {
				return
			}
			backoff = nextReconnectBackoff(backoff)
			continue
		}

		connectedAt := time.Now()
		// Next() reads the HTTP body with no select on ctx; close the body
		// when ctx ends so the drain returns and shutdown can finish.
		drainDone := make(chan struct{})
		go func(s *godo.HostedAgentSessionStream) {
			select {
			case <-ctx.Done():
				_ = s.Close()
			case <-drainDone:
			}
		}(stream)
		clientDead := f.drainStream(stream, ew, &cursor)
		close(drainDone)
		streamErr := stream.Err()
		stream.Close()

		if clientDead || ctx.Err() != nil {
			return
		}
		if streamErr != nil && isTerminalStreamError(streamErr) {
			log.Printf("agentproxy/opencode: stream error (terminal), giving up: %v", streamErr)
			f.failTrackedTurns(ew, "hosted session stream unavailable: "+streamErr.Error())
			return
		}

		// Nothing in flight: a stream end here owes the client nothing, and
		// the next prompt opens its own fresh stream state. This is also the
		// exit that ends the SSE response when the harness closes a drained
		// stream normally.
		f.mu.Lock()
		noTurnsLeft := len(f.turns) == 0
		f.mu.Unlock()
		if noTurnsLeft {
			if streamErr != nil {
				log.Printf("agentproxy/opencode: harness stream ended: %v", streamErr)
			}
			return
		}

		// A drop after a healthy, long-lived connection is a normal idle
		// timeout, not a failing session: reset the budget so a long-lived
		// proxy survives unbounded idle drops while still giving up on a
		// session that keeps failing immediately.
		if time.Since(connectedAt) >= healthyStreamDuration {
			failures = 0
			backoff = initialReconnectBackoff
		} else {
			failures++
		}
		if failures >= maxAutoReconnectAttempts {
			log.Printf("agentproxy/opencode: reconnected %d times without staying healthy, giving up", failures)
			f.failTrackedTurns(ew, "lost connection to the hosted session")
			return
		}
		log.Printf("agentproxy/opencode: harness stream dropped mid-turn (err=%v), reconnecting from cursor", streamErr)
		if !(*reconnectSleep.Load())(ctx, backoff) {
			return
		}
		backoff = nextReconnectBackoff(backoff)
	}
}

// drainStream reads one harness stream until it ends, translating events for
// tracked turns. cursor is updated with every event's id as it is seen,
// before processing — even an event this facade fails to handle must never be
// re-requested on reconnect (re-receiving it wouldn't fix it). Returns true
// when a client write failed — the SSE consumer is gone and reconnecting to
// the harness would help nothing.
func (f *Facade) drainStream(stream *godo.HostedAgentSessionStream, ew *eventWriter, cursor *string) bool {
	for stream.Next() {
		ev := stream.Current()
		if ev.EventID != "" {
			*cursor = ev.EventID
		}
		// Control frames (stream.state) and anything else without a run id
		// belong to no turn.
		if ev.RunID == "" {
			continue
		}
		ts, ok := f.lookupTurn(ev.RunID)
		if !ok {
			continue
		}
		if err := f.translateEvent(ev, ts, ew); err != nil {
			log.Printf("agentproxy/opencode: client write failed, closing stream: %v", err)
			return true
		}
	}
	return false
}

// failTrackedTurns ends every in-flight turn on the client when the event
// loop gives up for good: dismiss pending permission dialogs, close dangling
// tool parts and assistant messages, then surface one session.error and go
// idle. Best-effort — the client may itself be the casualty — so write
// errors are ignored; the SSE response is about to end either way.
func (f *Facade) failTrackedTurns(ew *eventWriter, message string) {
	f.mu.Lock()
	turns := f.turns
	f.turns = nil
	perms := f.perms
	f.perms = nil
	f.permsByHitl = nil
	f.mu.Unlock()
	if len(turns) == 0 && len(perms) == 0 {
		return
	}

	for _, p := range perms {
		_ = f.emitPermissionReplied(ew, p.perID, "reject")
	}
	sid := f.ocSessionID()
	at := timeNowMs()
	for runID, ts := range turns {
		_ = f.closeDanglingTools(ts, ew, sid, at)
		_ = f.finishAssistantMessage(runID, ts, ew, at)
	}
	_ = ew.session("session.error", map[string]any{
		"sessionID": sid,
		"error": map[string]any{
			"name": "UnknownError",
			"data": map[string]any{"message": message},
		},
	})
	_ = f.emitIdle(sid, ew)
}

// translateEvent maps one canonical event to opencode frames on the
// connected stream, following the ground-truth sequence captured from a real
// `opencode serve` turn (see the M0/M2 capture doc): user echo → busy →
// assistant message.updated → empty part.updated → deltas → finalized
// part.updated → status idle → session.idle. Returns a write error when the
// client is gone.
func (f *Facade) translateEvent(ev godo.HostedAgentEvent, ts *turnState, ew *eventWriter) error {
	sid := f.ocSessionID()
	at := eventTimeMs(ev)

	switch ev.Kind {
	case godo.HostedAgentEventKindRunStarted:
		return f.announceTurn(ev.RunID, ts, ew, at)

	case godo.HostedAgentEventKindTokenChunk:
		var payload struct {
			Text        string `json:"text"`
			IsReasoning bool   `json:"is_reasoning"`
		}
		if err := json.Unmarshal(ev.Payload, &payload); err != nil || payload.Text == "" {
			return nil
		}
		// A delta for a message the TUI hasn't been told about yet (stream
		// connected mid-turn, or run.started was lost) — announce it first.
		if !ts.startedSent {
			if err := f.announceTurn(ev.RunID, ts, ew, at); err != nil {
				return err
			}
		}
		// Unlike codex's protocol, opencode has a first-class reasoning
		// channel: same flat delta frame, field "reasoning", its own part.
		field := "text"
		part := &ts.textPartID
		started := &ts.textPartStarted
		marker := "tx"
		if payload.IsReasoning {
			field, part, started, marker = "reasoning", &ts.reasoningPartID, &ts.reasoningPartStarted, "re"
		} else {
			ts.sawText = true
			ts.text.WriteString(payload.Text)
		}
		// The part must exist before its first delta: the TUI applies
		// deltas only to parts a message.part.updated already created —
		// verified live, a delta for an unannounced part renders nothing.
		if !*started {
			*part = ocTimeID("prt_", at, 2, runTail(ev.RunID, marker))
			if err := ew.session("message.part.updated", map[string]any{
				"sessionID": sid,
				"part": map[string]any{
					"id": *part, "messageID": ts.asstMsgID, "sessionID": sid,
					"type": field, "text": "",
					"time": map[string]any{"start": at},
				},
			}); err != nil {
				return err
			}
			*started = true
		}
		return ew.session("message.part.delta", map[string]any{
			"sessionID": sid,
			"messageID": ts.asstMsgID,
			"partID":    *part,
			"field":     field,
			"delta":     payload.Text,
		})

	case godo.HostedAgentEventKindToolCallStarted:
		var payload struct {
			ToolCallID string          `json:"tool_call_id"`
			Name       string          `json:"name"`
			Input      json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return nil
		}
		if !ts.startedSent {
			if err := f.announceTurn(ev.RunID, ts, ew, at); err != nil {
				return err
			}
		}
		// The canonical tool_call_id is the guest opencode's own part id
		// (prt_..., time-encoded by the guest's clock) — reuse it so the
		// part sorts where the tool actually ran; mint only when a non-
		// opencode-shaped id shows up.
		partID := payload.ToolCallID
		if _, ok := tFromOCID(partID); !ok {
			partID = ocTimeID("prt_", at, 5, runTail(ev.RunID, "tc"))
		}
		tc := &toolCall{partID: partID, name: payload.Name, input: payload.Input, startMs: at}
		if ts.tools == nil {
			ts.tools = map[string]*toolCall{}
		}
		ts.tools[payload.ToolCallID] = tc
		return ew.session("message.part.updated", map[string]any{
			"sessionID": sid,
			"part":      f.toolPart(ts, tc, "running", "", at),
		})

	case godo.HostedAgentEventKindToolCallCompleted:
		var payload struct {
			ToolCallID string `json:"tool_call_id"`
			OK         bool   `json:"ok"`
			Summary    string `json:"summary"`
		}
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return nil
		}
		tc, ok := ts.tools[payload.ToolCallID]
		if !ok {
			// Completion for a start this stream never saw (mid-turn
			// connect): synthesize what we can.
			tc = &toolCall{
				partID:  ocTimeID("prt_", at, 5, runTail(ev.RunID, "tc")),
				name:    "tool",
				startMs: at,
			}
		}
		tc.done = true
		status := "completed"
		if !payload.OK {
			status = "error"
		}
		return ew.session("message.part.updated", map[string]any{
			"sessionID": sid,
			"part":      f.toolPart(ts, tc, status, payload.Summary, at),
		})

	case godo.HostedAgentEventKindHITLRequested:
		return f.handleHITLRequested(ev, ts, ew, at)

	case godo.HostedAgentEventKindHITLResolved:
		return f.handleHITLResolved(ev, ew)

	case godo.HostedAgentEventKindRunUsageRecorded, godo.HostedAgentEventKindRunCostAccrued, godo.HostedAgentEventKindRunLog:
		// Deliberate no-ops: per-turn token/cost totals ride run.completed's
		// own payload (used below), and run.log is guest debug noise. Listed
		// explicitly so they don't spam the unhandled-kind log.
		return nil

	case godo.HostedAgentEventKindRunCompleted:
		defer f.dropTurn(ev.RunID)
		// The finished turn is durable history now; the cache predates it.
		f.invalidateHistory()
		// Close any dangling tool part first — a part stuck "running"
		// renders as in-flight forever — and dismiss any permission dialog
		// still up for this run.
		if err := f.closeDanglingTools(ts, ew, sid, at); err != nil {
			return err
		}
		if err := f.closePendingPerms(ev.RunID, ew); err != nil {
			return err
		}
		var totals struct {
			TokensIn   int64 `json:"total_tokens_in"`
			TokensOut  int64 `json:"total_tokens_out"`
			CostMicros int64 `json:"run_cost_micros"`
		}
		_ = json.Unmarshal(ev.Payload, &totals)
		ts.tokensIn, ts.tokensOut, ts.costMicros = totals.TokensIn, totals.TokensOut, totals.CostMicros
		// Finalize the text part with its full content before idling — the
		// real server does (deltas stream, then the part's final state
		// re-carries the whole text; plano's adapter must drop that as a
		// duplicate for exactly this reason).
		if ts.sawText {
			if err := ew.session("message.part.updated", map[string]any{
				"sessionID": sid,
				"part": map[string]any{
					"id": ts.textPartID, "messageID": ts.asstMsgID, "sessionID": sid,
					"type": "text", "text": ts.text.String(),
					"time": map[string]any{"start": ts.startMs, "end": at},
				},
			}); err != nil {
				return err
			}
		}
		// step-finish carries the turn's real token/cost numbers — the TUI's
		// token counter and cost display key off it (real server emits one
		// per inference step; the canonical stream only has per-turn totals,
		// so one closing step-finish stands in for the lot).
		if ts.startedSent {
			if err := ew.session("message.part.updated", map[string]any{
				"sessionID": sid,
				"part": map[string]any{
					"id": ocTimeID("prt_", at, 6, runTail(ev.RunID, "sf")), "messageID": ts.asstMsgID, "sessionID": sid,
					"type": "step-finish", "reason": "stop",
					"tokens": tokensObject(ts.tokensIn, ts.tokensOut),
					"cost":   float64(ts.costMicros) / 1e6,
				},
			}); err != nil {
				return err
			}
		}
		// Close the assistant message with time.completed — without it the
		// TUI leaves the turn looking in-flight (a lingering QUEUED tag on
		// the user message, found live).
		if err := f.finishAssistantMessage(ev.RunID, ts, ew, at); err != nil {
			return err
		}
		return f.emitIdle(sid, ew)

	case godo.HostedAgentEventKindRunFailed:
		defer f.dropTurn(ev.RunID)
		f.invalidateHistory()
		if err := f.closeDanglingTools(ts, ew, sid, at); err != nil {
			return err
		}
		if err := f.closePendingPerms(ev.RunID, ew); err != nil {
			return err
		}
		var payload struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(ev.Payload, &payload)
		if payload.Message == "" {
			payload.Message = "hosted session run failed"
		}
		if err := ew.session("session.error", map[string]any{
			"sessionID": sid,
			"error": map[string]any{
				"name": "UnknownError",
				"data": map[string]any{"message": payload.Message},
			},
		}); err != nil {
			return err
		}
		return f.emitIdle(sid, ew)

	default:
		// The log is the backlog, exactly like the codex facade's
		// unhandled-method log was.
		log.Printf("unhandled event kind: %s", ev.Kind)
		return nil
	}
}

// announceTurn emits the turn-start frames in the real server's order: the
// user message echo (message.updated role "user" + a text part carrying the
// prompt — the TUI renders the user's side of the conversation from these),
// then session.status busy, then the assistant message.updated every later
// part delta hangs off. Idempotent per turn via ts.startedSent.
func (f *Facade) announceTurn(runID string, ts *turnState, ew *eventWriter, at int64) error {
	if ts.startedSent {
		return nil
	}
	ts.startedSent = true
	ts.startMs = at
	sid := f.ocSessionID()

	// The assistant id must sort strictly after the user id (the TUI orders
	// by id string). The user id's time prefix comes from a different clock
	// — the client's when client-minted, the proxy's on the fallback — while
	// `at` is the harness event time (the dev stack's container clock runs
	// ~half a second behind the host, observed live). Mint the assistant id
	// off whichever is later so skew can't invert the conversation.
	asstT := ocTimeVal(at)
	if userT, ok := tFromOCID(ts.userMsgID); ok && userT >= asstT {
		asstT = userT + 0x1000 // +1ms in T-space
	}
	ts.asstMsgID = ocIDWithT("msg_", asstT, 1, runTail(runID, "as"))
	ts.userPartID = ocIDWithT("prt_", asstT, 0, runTail(runID, "up"))

	if ts.userMsgID != "" {
		if err := ew.session("message.updated", map[string]any{
			"sessionID": sid,
			"info": map[string]any{
				"id": ts.userMsgID, "sessionID": sid,
				"role":    "user",
				"time":    map[string]any{"created": at},
				"agent":   "build",
				"model":   map[string]any{"providerID": providerID, "modelID": modelID},
				"summary": map[string]any{"diffs": []any{}},
			},
		}); err != nil {
			return err
		}
		if err := ew.session("message.part.updated", map[string]any{
			"sessionID": sid,
			"part": map[string]any{
				"id": ts.userPartID, "messageID": ts.userMsgID, "sessionID": sid,
				"type": "text", "text": ts.promptText,
			},
		}); err != nil {
			return err
		}
	}
	if err := ew.session("session.status", map[string]any{
		"sessionID": sid,
		"status":    map[string]any{"type": "busy"},
	}); err != nil {
		return err
	}
	// The assistant info's cost/tokens are not optional decoration: the TUI
	// crashes rendering a message list whose assistant entries lack
	// `tokens.output` ("undefined is not an object", found live). Zeros are
	// fine; absent is fatal. Shared shape: assistantInfo.
	return ew.session("message.updated", map[string]any{
		"sessionID": sid,
		"info":      f.assistantInfo(ts, map[string]any{"created": at}),
	})
}

// toolPart renders a tool part's state frame. Shape per the real-server tool
// capture: state{status, input, output/metadata on completion, title,
// time{start[,end]}}; callID mirrors the part id (canonical has no separate
// provider call id to carry).
func (f *Facade) toolPart(ts *turnState, tc *toolCall, status, output string, at int64) map[string]any {
	input := any(map[string]any{})
	if len(tc.input) > 0 {
		input = json.RawMessage(tc.input)
	}
	state := map[string]any{
		"status": status,
		"input":  input,
		"time":   map[string]any{"start": tc.startMs},
	}
	if status != "running" {
		state["time"] = map[string]any{"start": tc.startMs, "end": at}
		state["output"] = output
		state["metadata"] = map[string]any{"output": output}
		if title := toolTitle(tc); title != "" {
			state["title"] = title
		}
		if status == "error" {
			state["error"] = output
		}
	}
	return map[string]any{
		"id": tc.partID, "messageID": ts.asstMsgID, "sessionID": f.ocSessionID(),
		"type": "tool", "tool": tc.name, "callID": tc.partID,
		"state": state,
	}
}

// toolTitle is the human title the TUI shows on a finished tool part — the
// command for bash-shaped inputs, nothing otherwise.
func toolTitle(tc *toolCall) string {
	var input struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(tc.input, &input)
	return input.Command
}

// closeDanglingTools finishes any tool part still "running" when the turn
// ends — the completion event was lost or the run died mid-tool, and a part
// stuck running renders as in-flight forever.
func (f *Facade) closeDanglingTools(ts *turnState, ew *eventWriter, sid string, at int64) error {
	for _, tc := range ts.tools {
		if tc.done {
			continue
		}
		tc.done = true
		if err := ew.session("message.part.updated", map[string]any{
			"sessionID": sid,
			"part":      f.toolPart(ts, tc, "error", "turn ended before the tool completed", at),
		}); err != nil {
			return err
		}
	}
	return nil
}

// tokensObject is the tokens map used on step-finish parts and assistant
// info; reasoning/cache are not broken out by the canonical totals, so they
// render as zero.
func tokensObject(in, out int64) map[string]any {
	return map[string]any{
		"total": in + out, "input": in, "output": out, "reasoning": 0,
		"cache": map[string]any{"read": 0, "write": 0},
	}
}

// finishAssistantMessage re-sends the assistant message.updated with
// time.completed stamped, mirroring the real server's end-of-turn frame.
func (f *Facade) finishAssistantMessage(runID string, ts *turnState, ew *eventWriter, at int64) error {
	if ts.asstMsgID == "" {
		// The turn was never announced (no run.started or delta seen) —
		// there is no assistant message to close.
		return nil
	}
	sid := f.ocSessionID()
	info := f.assistantInfo(ts, map[string]any{"created": ts.startMs, "completed": at})
	info["tokens"] = tokensObject(ts.tokensIn, ts.tokensOut)
	info["cost"] = float64(ts.costMicros) / 1e6
	info["finish"] = "stop"
	return ew.session("message.updated", map[string]any{"sessionID": sid, "info": info})
}

// assistantInfo is the shared assistant message info shape. modelID and
// providerID are FLAT fields on assistant messages (the nested model{} form
// is user-message shape — mixing them up leaves the TUI's footer model
// segment blank, observed live against the real server's history payloads).
func (f *Facade) assistantInfo(ts *turnState, timeObj map[string]any) map[string]any {
	info := map[string]any{
		"id": ts.asstMsgID, "sessionID": f.ocSessionID(),
		"role":       "assistant",
		"time":       timeObj,
		"mode":       "build",
		"agent":      "build",
		"modelID":    modelID,
		"providerID": providerID,
		"path":       map[string]any{"cwd": f.Dir, "root": f.Dir},
		"cost":       0,
		"tokens":     tokensObject(0, 0),
	}
	if ts.userMsgID != "" {
		info["parentID"] = ts.userMsgID
	}
	return info
}

// emitIdle ends a turn on the stream the way the real server does: a
// session.status idle followed by the dedicated session.idle event (both
// exist on the wire; the TUI's spinner keys off them).
func (f *Facade) emitIdle(sid string, ew *eventWriter) error {
	if err := ew.session("session.status", map[string]any{
		"sessionID": sid,
		"status":    map[string]any{"type": "idle"},
	}); err != nil {
		return err
	}
	return ew.session("session.idle", map[string]any{"sessionID": sid})
}

// eventTimeMs is the event's timestamp in Unix milliseconds, falling back to
// now — opencode part/message times are millisecond epochs.
func eventTimeMs(ev godo.HostedAgentEvent) int64 {
	if !ev.At.IsZero() {
		return ev.At.UnixMilli()
	}
	return timeNowMs()
}
