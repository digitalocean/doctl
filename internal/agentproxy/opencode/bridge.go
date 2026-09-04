package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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

// handleSessionList answers GET /session: empty until the client has
// created/used the session, then a single-entry list. An empty list is the
// "fresh server" state the TUI expects on first attach (verified in the M0
// capture); returning the session unconditionally would make every attach
// look like a resume.
func (f *Facade) handleSessionList(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	created := f.sessionCreated
	f.mu.Unlock()
	if !created {
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

// runEventLoop is the live StreamSession drain for one connected event
// stream. It runs inside the /global/event handler goroutine — the SSE
// response writer has exactly one writer by construction, so no notifier
// mutex is needed the way the WS transport's wsNotifier is.
//
// M2 scope: no reconnect. A dropped harness stream ends the SSE response,
// and the TUI's own re-attach opens a fresh one (which re-runs this loop).
// M6 adds replay_from-cursor reconnects inside a single SSE response.
func (f *Facade) runEventLoop(ctx context.Context, ew *eventWriter) {
	stream, err := f.Sessions.StreamSession(ctx, f.SessionID, nil)
	if err != nil {
		log.Printf("agentproxy/opencode: StreamSession failed: %v", err)
		return
	}
	defer stream.Close()

	for stream.Next() {
		ev := stream.Current()
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
			return
		}
	}
	if err := stream.Err(); err != nil && ctx.Err() == nil {
		log.Printf("agentproxy/opencode: harness stream ended: %v", err)
	}
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

	case godo.HostedAgentEventKindRunCompleted:
		defer f.dropTurn(ev.RunID)
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
		// Close the assistant message with time.completed — without it the
		// TUI leaves the turn looking in-flight (a lingering QUEUED tag on
		// the user message, found live).
		if err := f.finishAssistantMessage(ev.RunID, ts, ew, at); err != nil {
			return err
		}
		return f.emitIdle(sid, ew)

	case godo.HostedAgentEventKindRunFailed:
		defer f.dropTurn(ev.RunID)
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
		// Tool calls, usage, HITL: M4/M5. The log is the backlog, exactly
		// like the codex facade's unhandled-method log was.
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
	// fine; absent is fatal. Field set mirrors the ground-truth capture.
	info := map[string]any{
		"id": ts.asstMsgID, "sessionID": sid,
		"role":  "assistant",
		"time":  map[string]any{"created": at},
		"mode":  "build",
		"agent": "build",
		"model": map[string]any{"providerID": providerID, "modelID": modelID},
		"path":  map[string]any{"cwd": f.Dir, "root": f.Dir},
		"cost":  0,
		"tokens": map[string]any{
			"input": 0, "output": 0, "reasoning": 0,
			"cache": map[string]any{"read": 0, "write": 0},
		},
	}
	if ts.userMsgID != "" {
		info["parentID"] = ts.userMsgID
	}
	return ew.session("message.updated", map[string]any{
		"sessionID": sid,
		"info":      info,
	})
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
	info := map[string]any{
		"id": ts.asstMsgID, "sessionID": sid,
		"role":  "assistant",
		"time":  map[string]any{"created": ts.startMs, "completed": at},
		"mode":  "build",
		"agent": "build",
		"model": map[string]any{"providerID": providerID, "modelID": modelID},
		"path":  map[string]any{"cwd": f.Dir, "root": f.Dir},
		"cost":  0,
		"tokens": map[string]any{
			"input": 0, "output": 0, "reasoning": 0,
			"cache": map[string]any{"read": 0, "write": 0},
		},
	}
	if ts.userMsgID != "" {
		info["parentID"] = ts.userMsgID
	}
	return ew.session("message.updated", map[string]any{"sessionID": sid, "info": info})
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
