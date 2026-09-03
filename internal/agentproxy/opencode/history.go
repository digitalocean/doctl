package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/digitalocean/godo"
)

// M3: history. `opencode attach` replays a session by fetching
// GET /session/{id}/message?limit=N itself (client-driven — the server never
// pushes history), so re-attaching shows prior turns. The facade serves that
// endpoint from a one-shot replay_only StreamSession pass over the session's
// durable event history, reconstructed into the [{info, parts}] shape
// captured from a real server (see the capture doc). No cache: history is
// fetched per request, and the TUI asks once per attach.

// historyMessage is one reconstructed message: the {info, parts} pair the
// history endpoint returns.
type historyMessage struct {
	Info  map[string]any   `json:"info"`
	Parts []map[string]any `json:"parts"`
}

// historyTurn accumulates one hosted run while replaying.
type historyTurn struct {
	runID   string
	startMs int64
	// doneMs is the run.completed/run.failed event time — the turn's real
	// duration. lastMs (any event) is only the fallback for turns that never
	// completed: stragglers like late run.log events arrive long after
	// completion and would inflate the rendered duration (a "20m" turn
	// badge on a 2s turn, found live).
	doneMs    int64
	lastMs    int64
	userText  string
	text      []byte
	reasoning []byte
	failed    bool
}

func (ht *historyTurn) endMs() int64 {
	if ht.doneMs != 0 {
		return ht.doneMs
	}
	return ht.lastMs
}

// history returns the reconstructed message history, cached. The harness's
// replay_only stream takes ~8s to close after it has caught up (a
// server-side linger, measured against the dev stack — 68 events transfer
// instantly and the close arrives at exactly 8.0s), and an attach fetches
// history twice (list gating + the message list), so uncached history made
// re-attach feel ~16s slow. The mutex doubles as single-flight: concurrent
// callers wait for the one in-progress replay instead of starting their own.
//
// The cache is warmed at the first request the facade sees (the TUI's
// /global/health preflight fires well before the attach burst) and
// invalidated when a live turn completes (invalidateHistory).
func (f *Facade) history(ctx context.Context) ([]historyMessage, error) {
	f.histMu.Lock()
	defer f.histMu.Unlock()
	if f.histValid {
		return f.hist, nil
	}
	msgs, err := f.fetchHistory(ctx)
	if err != nil {
		return nil, err
	}
	f.hist, f.histValid = msgs, true
	return msgs, nil
}

// invalidateHistory drops the cache; the next history() call replays fresh.
func (f *Facade) invalidateHistory() {
	f.histMu.Lock()
	f.histValid = false
	f.hist = nil
	f.histMu.Unlock()
}

// fetchHistory replays the durable event history and reconstructs completed
// turns. Only text is reconstructed in M3 — tool-call parts are M4. Callers
// go through history() for the cache; this always hits the harness.
func (f *Facade) fetchHistory(ctx context.Context) ([]historyMessage, error) {
	stream, err := f.Sessions.StreamSession(ctx, f.SessionID, &godo.HostedAgentSessionStreamOptions{
		ReplayOnly: true,
	})
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var order []string
	turns := map[string]*historyTurn{}
	for stream.Next() {
		ev := stream.Current()
		if ev.RunID == "" {
			continue
		}
		ht, ok := turns[ev.RunID]
		if !ok {
			ht = &historyTurn{runID: ev.RunID}
			turns[ev.RunID] = ht
			order = append(order, ev.RunID)
		}
		at := eventTimeMs(ev)
		if ht.startMs == 0 {
			ht.startMs = at
		}
		if at > ht.lastMs {
			ht.lastMs = at
		}
		switch ev.Kind {
		case godo.HostedAgentEventKindRunStarted:
			// The proto names this field user_input, but the SSE wire's SPI
			// envelope delivers it as "agent" (observed live against the dev
			// stack: data:{"agent":"<prompt text>"}). Accept both so a wire
			// rename toward the proto name doesn't silently drop history.
			var payload struct {
				UserInput string `json:"user_input"`
				Agent     string `json:"agent"`
			}
			_ = json.Unmarshal(ev.Payload, &payload)
			ht.userText = payload.UserInput
			if ht.userText == "" {
				ht.userText = payload.Agent
			}
		case godo.HostedAgentEventKindTokenChunk:
			var payload struct {
				Text        string `json:"text"`
				IsReasoning bool   `json:"is_reasoning"`
			}
			if err := json.Unmarshal(ev.Payload, &payload); err != nil {
				continue
			}
			if payload.IsReasoning {
				ht.reasoning = append(ht.reasoning, payload.Text...)
			} else {
				ht.text = append(ht.text, payload.Text...)
			}
		case godo.HostedAgentEventKindRunCompleted:
			ht.doneMs = at
		case godo.HostedAgentEventKindRunFailed:
			ht.failed = true
			ht.doneMs = at
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}

	// T allocation is monotonic across turns: two turns whose events share a
	// millisecond would otherwise interleave (turn N+1's counter-0 user id
	// sorts below turn N's counter-2 assistant id — caught by test, and real
	// replays can burst events into one ms). Each turn gets a fresh ms slot
	// in T-space at minimum.
	var msgs []historyMessage
	var lastT uint64
	for _, runID := range order {
		ht := turns[runID]
		t := ocTimeVal(ht.startMs)
		if t <= lastT {
			t = lastT + 0x1000
		}
		lastT = t
		msgs = append(msgs, f.turnMessages(ht, t)...)
	}
	return msgs, nil
}

// turnMessages renders one replayed turn as its user and assistant messages.
// Ids are minted from the caller-allocated T (turn event time, forced
// monotonic across turns) with the real id encoding — history ids sort
// against live-minted ids in the same conversation, so a random id here
// reintroduces the shuffled-rendering bug the live path fixed.
func (f *Facade) turnMessages(ht *historyTurn, t uint64) []historyMessage {
	sid := f.ocSessionID()
	userMsgID := ocIDWithT("msg_", t, 0, runTail(ht.runID, "hu"))
	asstMsgID := ocIDWithT("msg_", t, 2, runTail(ht.runID, "ha"))

	var msgs []historyMessage
	if ht.userText != "" {
		msgs = append(msgs, historyMessage{
			Info: map[string]any{
				"id": userMsgID, "sessionID": sid,
				"role":    "user",
				"time":    map[string]any{"created": ht.startMs},
				"agent":   "build",
				"model":   map[string]any{"providerID": providerID, "modelID": modelID},
				"summary": map[string]any{"diffs": []any{}},
			},
			Parts: []map[string]any{{
				"id": ocIDWithT("prt_", t, 1, runTail(ht.runID, "hu")), "messageID": userMsgID, "sessionID": sid,
				"type": "text", "text": ht.userText,
			}},
		})
	}

	if len(ht.text) == 0 && len(ht.reasoning) == 0 && !ht.failed {
		return msgs
	}
	info := map[string]any{
		"id": asstMsgID, "sessionID": sid,
		"role":  "assistant",
		"time":  map[string]any{"created": ht.startMs, "completed": ht.endMs()},
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
	if ht.userText != "" {
		info["parentID"] = userMsgID
	}
	if !ht.failed {
		info["finish"] = "stop"
	}
	var parts []map[string]any
	partT := t
	if len(ht.reasoning) > 0 {
		parts = append(parts, map[string]any{
			"id": ocIDWithT("prt_", partT, 3, runTail(ht.runID, "hr")), "messageID": asstMsgID, "sessionID": sid,
			"type": "reasoning", "text": string(ht.reasoning),
			"time": map[string]any{"start": ht.startMs, "end": ht.endMs()},
		})
	}
	if len(ht.text) > 0 {
		parts = append(parts, map[string]any{
			"id": ocIDWithT("prt_", partT, 4, runTail(ht.runID, "ht")), "messageID": asstMsgID, "sessionID": sid,
			"type": "text", "text": string(ht.text),
			"time": map[string]any{"start": ht.startMs, "end": ht.endMs()},
		})
	}
	msgs = append(msgs, historyMessage{Info: info, Parts: parts})
	return msgs
}

// handleMessageList serves GET /session/{id}/message from a fresh replay.
// The TUI sends ?limit=N (100 on resume); the newest N messages win, order
// preserved (oldest first), matching a real server's response.
func (f *Facade) handleMessageList(w http.ResponseWriter, r *http.Request) {
	msgs, err := f.history(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("replaying session history failed: %v", err), http.StatusBadGateway)
		return
	}
	if s := r.URL.Query().Get("limit"); s != "" {
		if limit, err := strconv.Atoi(s); err == nil && limit >= 0 && len(msgs) > limit {
			msgs = msgs[len(msgs)-limit:]
		}
	}
	if msgs == nil {
		msgs = []historyMessage{}
	}
	f.writeJSON(w, msgs)
}

// hasHistory reports whether the session has any replayable turns — the
// session list includes the bridged session when it does, so `--continue`
// and the session picker can find it on a fresh proxy.
func (f *Facade) hasHistory(ctx context.Context) bool {
	msgs, err := f.history(ctx)
	return err == nil && len(msgs) > 0
}
