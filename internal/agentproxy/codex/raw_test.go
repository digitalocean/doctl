// Tests for raw event passthrough (raw.go): the prefer-source_raw strategy,
// its identifier rewriting, and the per-event canonical fallback.
package codex

import (
	"encoding/json"
	"testing"

	"github.com/digitalocean/doctl/internal/agentproxy/agentproxytest"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawParams pulls the forwarded params map out of a recorded notification —
// raw passthrough notifies with the rewritten map[string]any rather than one
// of the synthesized notification structs, which is itself part of what
// these tests assert (a struct here would mean the canonical path ran).
func rawParams(t *testing.T, n recordedNotification) map[string]any {
	t.Helper()
	params, ok := n.params.(map[string]any)
	require.True(t, ok, "%s params should be the forwarded raw map, got %T (canonical reconstruction ran instead?)", n.method, n.params)
	return params
}

// TestFacade_TurnStart_RawPassthrough drives one whole turn where every
// canonical event carries its native codex frame, and asserts the client
// receives those frames — VM thread/turn ids rewritten to the proxy's,
// native item ids and raw-only detail (command output deltas, aggregated
// output) preserved — with none of the canonical path's synthesized
// notifications (no "-msg" agentMessage item lifecycle) mixed in.
func TestFacade_TurnStart_RawPassthrough(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI

	h.QueueRun("run-raw",
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindRunStarted),
			SourceEventType: "turn/started",
			SourceRaw:       []byte(`{"method":"turn/started","params":{"threadId":"thread-vm","turn":{"id":"turn-vm","items":[],"status":"inProgress","startedAt":1767225600}}}` + "\n"),
		},
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindTokenChunk),
			Data:            json.RawMessage(`{"text":"Hel"}`),
			SourceEventType: "item/agentMessage/delta",
			SourceRaw:       []byte(`{"method":"item/agentMessage/delta","params":{"threadId":"thread-vm","turnId":"turn-vm","itemId":"item_0","delta":"Hel"}}`),
		},
		// run.log has no canonical→codex mapping at all (dropped in v1);
		// with raw bytes the native command output delta reaches the TUI.
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindRunLog),
			Data:            json.RawMessage(`{"level":2,"message":"command output"}`),
			SourceEventType: "item/commandExecution/outputDelta",
			SourceRaw:       []byte(`{"method":"item/commandExecution/outputDelta","params":{"threadId":"thread-vm","turnId":"turn-vm","itemId":"item_1","delta":"total 0\n"}}`),
		},
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindToolCallCompleted),
			Data:            json.RawMessage(`{"tool_call_id":"item_1","ok":true,"duration_ms":42}`),
			SourceEventType: "item/completed",
			SourceRaw:       []byte(`{"method":"item/completed","params":{"threadId":"thread-vm","turnId":"turn-vm","item":{"type":"commandExecution","id":"item_1","command":"ls","status":"completed","aggregatedOutput":"total 0\n","exitCode":0}}}`),
		},
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindRunCompleted),
			SourceEventType: "turn/completed",
			SourceRaw:       []byte(`{"method":"turn/completed","params":{"threadId":"thread-vm","turn":{"id":"turn-vm","items":[],"status":"completed"}}}`),
		},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	started := rec.next(t)
	require.Equal(t, "turn/started", started.method)
	sp := rawParams(t, started)
	assert.Equal(t, testSessionID, sp["threadId"], "VM thread id must be rewritten to the proxy's")
	turn := sp["turn"].(map[string]any)
	assert.Equal(t, "run-raw", turn["id"], "VM turn id must be rewritten to the harness run id")
	assert.Equal(t, json.Number("1767225600"), turn["startedAt"], "numeric fields must survive the rewrite un-mangled")

	// Directly the raw delta: no synthesized item/started for a "-msg"
	// agentMessage item in between — that item id family must never appear
	// on a raw turn.
	delta := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", delta.method, "no synthesized item/started may precede the raw delta")
	dp := rawParams(t, delta)
	assert.Equal(t, testSessionID, dp["threadId"])
	assert.Equal(t, "run-raw", dp["turnId"])
	assert.Equal(t, "item_0", dp["itemId"], "native item ids are kept, not re-minted")
	assert.Equal(t, "Hel", dp["delta"])

	outputDelta := rec.next(t)
	require.Equal(t, "item/commandExecution/outputDelta", outputDelta.method, "raw-only detail (dropped by canonical translation) must be forwarded")
	assert.Equal(t, "total 0\n", rawParams(t, outputDelta)["delta"])

	itemCompleted := rec.next(t)
	require.Equal(t, "item/completed", itemCompleted.method)
	item := rawParams(t, itemCompleted)["item"].(map[string]any)
	assert.Equal(t, "item_1", item["id"])
	assert.Equal(t, "total 0\n", item["aggregatedOutput"], "canonical carries no aggregated output; the raw frame's must survive")

	completed := rec.next(t)
	require.Equal(t, "turn/completed", completed.method)
	cp := rawParams(t, completed)
	assert.Equal(t, testSessionID, cp["threadId"])
	assert.Equal(t, "run-raw", cp["turn"].(map[string]any)["id"])

	rec.expectNone(t)

	f.mu.Lock()
	_, stillTracked := f.turns["run-raw"]
	f.mu.Unlock()
	assert.False(t, stillTracked, "raw turn/completed must still untrack the run (finishTurn's bookkeeping half)")
}

// TestFacade_TurnStart_RawFallsBackPerEvent starts a turn raw, then ends it
// with a run.failed that carries no raw frame — exactly what an
// adapter-synthesized failure (plano's TurnError path) looks like. The
// failure must arrive via the canonical reconstruction (synthesized
// turn/completed with status failed), and the canonical path must not also
// invent an agentMessage item/completed for a turn whose items were all
// raw-native (itemStarted was never set).
func TestFacade_TurnStart_RawFallsBackPerEvent(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI

	h.QueueRun("run-mixed",
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindRunStarted),
			SourceEventType: "turn/started",
			SourceRaw:       []byte(`{"method":"turn/started","params":{"threadId":"thread-vm","turn":{"id":"turn-vm","items":[],"status":"inProgress"}}}`),
		},
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindTokenChunk),
			Data:            json.RawMessage(`{"text":"partial"}`),
			SourceEventType: "item/agentMessage/delta",
			SourceRaw:       []byte(`{"method":"item/agentMessage/delta","params":{"threadId":"thread-vm","turnId":"turn-vm","itemId":"item_0","delta":"partial"}}`),
		},
		// No SourceRaw: adapter-synthesized failures never carry one.
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindRunFailed),
			Data: json.RawMessage(`{"message":"boom"}`),
		},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	require.Equal(t, "turn/started", rec.next(t).method)
	require.Equal(t, "item/agentMessage/delta", rec.next(t).method)

	failed := rec.next(t)
	require.Equal(t, "turn/completed", failed.method, "the raw-less run.failed must fall back to canonical reconstruction — and no synthesized item/completed may precede it on a raw turn")
	tc, ok := failed.params.(turnCompletedNotification)
	require.True(t, ok, "fallback must be the synthesized struct, got %T", failed.params)
	assert.Equal(t, "failed", tc.Turn.Status)
	assert.Equal(t, "run-mixed", tc.Turn.ID)
	require.NotNil(t, tc.Turn.Error)
	assert.Equal(t, "boom", tc.Turn.Error.Message)

	rec.expectNone(t)
}

// TestFacade_TurnStart_AgentKindMismatchStaysCanonical runs a session whose
// agent is NOT codex: the facade must not request include_raw (so the
// harness withholds the queued SourceRaw, mirroring the real surface's
// opt-in) and the whole turn must arrive via canonical reconstruction —
// raw bytes from a different agent's protocol must never reach a codex TUI.
func TestFacade_TurnStart_AgentKindMismatchStaysCanonical(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindOpenCode

	h.QueueRun("run-oc",
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindRunStarted),
			// An OpenCode-native frame; must never be forwarded.
			SourceEventType: "message.updated",
			SourceRaw:       []byte(`{"type":"message.updated","properties":{}}`),
		},
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindTokenChunk),
			Data:            json.RawMessage(`{"text":"Hi"}`),
			SourceEventType: "message.part.delta",
			SourceRaw:       []byte(`{"type":"message.part.delta","properties":{"delta":"Hi"}}`),
		},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	started := rec.next(t)
	require.Equal(t, "turn/started", started.method)
	_, ok := started.params.(turnStartedNotification)
	require.True(t, ok, "non-codex sessions must take the canonical path, got %T", started.params)

	require.Equal(t, "item/started", rec.next(t).method, "canonical path synthesizes the agentMessage item lifecycle")

	delta := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", delta.method)
	dn, ok := delta.params.(agentMessageDeltaNotification)
	require.True(t, ok, "delta must be synthesized, got %T", delta.params)
	assert.Equal(t, "Hi", dn.Delta)

	require.Equal(t, "item/completed", rec.next(t).method)
	require.Equal(t, "turn/completed", rec.next(t).method)
	rec.expectNone(t)
}

// TestFacade_UnknownMethodCatchAllForwards drives the contract the OHR
// mapper's catch-all creates: a notification the mapper has no translation
// for arrives as a run.log marker event ("unhandled:<method>") whose
// source_raw carries the native frame. Because forwarding is deny-listed
// rather than allow-listed, a method this proxy has never heard of still
// reaches the TUI — new codex protocol surface needs no doctl release.
func TestFacade_UnknownMethodCatchAllForwards(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI

	h.QueueRun("run-catchall",
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindRunStarted),
			SourceEventType: "turn/started",
			SourceRaw:       []byte(`{"method":"turn/started","params":{"threadId":"thread-vm","turn":{"id":"turn-vm","items":[],"status":"inProgress"}}}`),
		},
		// The catch-all shape plano's mapper emits for an untranslated
		// notification: canonical payload is only a debug log marker; the
		// native frame — a method minted after this proxy shipped — rides
		// in source_raw.
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindRunLog),
			Data:            json.RawMessage(`{"level":1,"message":"unhandled:item/todoList/updated"}`),
			SourceEventType: "item/todoList/updated",
			SourceRaw:       []byte(`{"method":"item/todoList/updated","params":{"threadId":"thread-vm","turnId":"turn-vm","itemId":"todo-1","items":[{"text":"write tests","done":true}]}}`),
		},
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindRunCompleted),
			SourceEventType: "turn/completed",
			SourceRaw:       []byte(`{"method":"turn/completed","params":{"threadId":"thread-vm","turn":{"id":"turn-vm","items":[],"status":"completed"}}}`),
		},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	require.Equal(t, "turn/started", rec.next(t).method)

	unknown := rec.next(t)
	require.Equal(t, "item/todoList/updated", unknown.method,
		"a method absent from any proxy-side list must forward — that's the denylist's whole point")
	up := rawParams(t, unknown)
	assert.Equal(t, testSessionID, up["threadId"], "ids are rewritten even on unknown methods")
	assert.Equal(t, "run-catchall", up["turnId"])
	assert.Equal(t, "todo-1", up["itemId"], "native item ids pass through")
	items := up["items"].([]any)
	assert.Equal(t, "write tests", items[0].(map[string]any)["text"], "unknown payload shape survives untouched")

	require.Equal(t, "turn/completed", rec.next(t).method)
	rec.expectNone(t)
}

// TestTryRawPassthrough_RejectsNonNotificationFrames unit-tests the
// structural guards: frames that must never be forwarded verbatim fall back
// to canonical (handled=false) without notifying anything.
func TestTryRawPassthrough_RejectsNonNotificationFrames(t *testing.T) {
	rec := newNotifierRecorder()
	f := &Facade{SessionID: testSessionID, AgentKind: godo.HostedAgentKindCodexCLI}
	f.SetNotifier(rec)
	ts := &turnState{itemID: "run-1-msg"}

	cases := []struct {
		name string
		raw  string
	}{
		{"server request (has id)", `{"id":7,"method":"item/commandExecution/requestApproval","params":{"threadId":"t","turnId":"u","itemId":"i"}}`},
		{"response (id, no method)", `{"id":7,"result":{}}`},
		{"suppressed method (conflicting thread lifecycle)", `{"method":"thread/started","params":{"threadId":"t"}}`},
		{"no method (not a notification)", `{"params":{"threadId":"t"}}`},
		{"not json", `data: not-a-frame`},
		{"no params", `{"method":"turn/started"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev := godo.HostedAgentEvent{
				RunID:     "run-1",
				Kind:      godo.HostedAgentEventKindRunStarted,
				SourceRaw: []byte(tc.raw),
			}
			handled, clientDead := f.tryRawPassthrough(ev, ts, false)
			assert.False(t, handled, "must fall back to canonical")
			assert.False(t, clientDead)
			rec.expectNone(t)
		})
	}
}

// TestTryRawPassthrough_ReplaySkipsUsage: historical usage is suppressed on
// --replay by the canonical path (a past session's counts must not inflate
// this connection's running total); the raw equivalent is suppressed too,
// as handled — not forwarded, not reconstructed.
func TestTryRawPassthrough_ReplaySkipsUsage(t *testing.T) {
	rec := newNotifierRecorder()
	f := &Facade{SessionID: testSessionID, AgentKind: godo.HostedAgentKindCodexCLI}
	f.SetNotifier(rec)

	ev := godo.HostedAgentEvent{
		RunID:     "run-old",
		Kind:      godo.HostedAgentEventKindRunUsageRecorded,
		Payload:   json.RawMessage(`{"usage":{"input_tokens":10,"output_tokens":2}}`),
		SourceRaw: []byte(`{"method":"thread/tokenUsage/updated","params":{"threadId":"thread-vm","turnId":"turn-vm","tokenUsage":{}}}`),
	}
	handled, clientDead := f.tryRawPassthrough(ev, &turnState{}, true)
	assert.True(t, handled)
	assert.False(t, clientDead)
	rec.expectNone(t)
	assert.Zero(t, f.totalUsage.TotalTokens, "replayed usage must not touch the live running total")
}

// TestRewriteRawParams covers the id-rewrite rules in isolation: threadId
// and turnId/turn.id are replaced, everything else — nested objects, native
// item ids, numbers — passes through untouched.
func TestRewriteRawParams(t *testing.T) {
	params, ok := rewriteRawParams(
		json.RawMessage(`{"threadId":"thread-vm","turnId":"turn-vm","itemId":"item_9","startedAtMs":1767225600123,"turn":{"id":"turn-vm","status":"inProgress"},"item":{"id":"item_9"}}`),
		"session-1", "run-9",
	)
	require.True(t, ok)
	assert.Equal(t, "session-1", params["threadId"])
	assert.Equal(t, "run-9", params["turnId"])
	assert.Equal(t, "run-9", params["turn"].(map[string]any)["id"])
	assert.Equal(t, "item_9", params["itemId"], "item ids are never rewritten")
	assert.Equal(t, "item_9", params["item"].(map[string]any)["id"], "nested item ids are never rewritten")
	assert.Equal(t, json.Number("1767225600123"), params["startedAtMs"], "int64-scale numbers must not decay to float64 notation")

	_, ok = rewriteRawParams(nil, "s", "r")
	assert.False(t, ok, "missing params can't be forwarded")
	_, ok = rewriteRawParams(json.RawMessage(`null`), "s", "r")
	assert.False(t, ok, "null params can't be forwarded")
	_, ok = rewriteRawParams(json.RawMessage(`[1,2]`), "s", "r")
	assert.False(t, ok, "non-object params can't be forwarded")
}

// ── v2 inbound raw passthrough ──────────────────────────────────────────

// TestFacade_TurnStart_SendsRawFrameInbound drives turn/start with params
// far richer than the two fields the facade parses (extra input item types,
// model, effort, approvalPolicy — everything the v1 text reduction drops)
// and asserts SendInput carried the client's exact params to the harness as
// source_raw, with the plain-text reduction still riding along for
// canonical consumers.
func TestFacade_TurnStart_SendsRawFrameInbound(t *testing.T) {
	f, h, _ := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI

	h.QueueRun("run-inbound",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)})

	params := json.RawMessage(`{"threadId":"` + testSessionID + `","input":[{"type":"text","text":"fix the bug"},{"type":"image","url":"file:///tmp/shot.png"}],"model":"gpt-5.3-codex","effort":"high","approvalPolicy":"never"}`)
	_, err := dispatch(t, f, "turn/start", params)
	require.NoError(t, err)

	in := h.LastInput()
	require.NotNil(t, in, "turn/start must POST .../input")
	assert.Equal(t, "fix the bug", in.Text, "the text reduction still rides along (canonical previews, non-passthrough consumers)")
	assert.Equal(t,
		`{"method":"turn/start","params":`+string(params)+`}`,
		string(in.SourceRaw),
		"source_raw must carry the client's params byte-identical inside a rebuilt turn/start envelope")
}

// TestFacade_TurnStart_NoInboundRawForNonCodexKind: a session whose runtime
// isn't codex can't consume codex frames — the inbound gate is the same
// AgentKind check as the outbound one, so only plain text is sent.
func TestFacade_TurnStart_NoInboundRawForNonCodexKind(t *testing.T) {
	f, h, _ := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindOpenCode

	h.QueueRun("run-oc-inbound",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)})

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	in := h.LastInput()
	require.NotNil(t, in)
	assert.Equal(t, "hi", in.Text)
	assert.Nil(t, in.SourceRaw, "non-codex sessions must not ship codex frames upstream")
}

// TestRawTurnStartFrame unit-tests the envelope builder: params bytes pass
// through untouched, and empty params produce no frame at all rather than a
// malformed one.
func TestRawTurnStartFrame(t *testing.T) {
	params := json.RawMessage(`{"threadId":"s","input":[{"type":"text","text":"x"}],  "model":"m"}`)
	got := rawTurnStartFrame(params)
	assert.Equal(t, `{"method":"turn/start","params":`+string(params)+`}`, string(got),
		"params must not be re-encoded, reordered, or re-spaced")
	assert.True(t, json.Valid(got), "the rebuilt frame must be valid JSON")

	assert.Nil(t, rawTurnStartFrame(nil))
	assert.Nil(t, rawTurnStartFrame(json.RawMessage{}))
}
