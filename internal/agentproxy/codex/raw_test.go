// Tests for raw event passthrough (raw.go): the prefer-source_raw strategy,
// its identifier rewriting, and the per-event canonical fallback.
package codex

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/digitalocean/doctl/internal/agentproxy"
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

// TestFacade_TurnInterrupt_Relayed is the M1 acceptance case: Esc mid-turn.
// The old stub answered success while the turn kept running, so the frame
// reaching the harness — and the agent's answer reaching the client — is what
// makes the difference between an interrupt and a lie.
func TestFacade_TurnInterrupt_Relayed(t *testing.T) {
	f, h, _ := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI
	h.QueueRelayReply([]byte(`{"id":1,"result":{"aborted":true}}`))

	result, err := dispatch(t, f, "turn/interrupt", map[string]any{"turnId": "run-1"})
	require.NoError(t, err)

	var sent struct {
		ID     int             `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	require.NoError(t, json.Unmarshal(h.LastRelay(), &sent))
	assert.Equal(t, "turn/interrupt", sent.Method)
	assert.NotZero(t, sent.ID, "a relayed request needs an id; a frame without one is a notification the adapter declines")
	assert.JSONEq(t, `{"turnId":"run-1"}`, string(sent.Params))

	assert.JSONEq(t, `{"aborted":true}`, string(result.(json.RawMessage)))
}

// TestFacade_UnknownRequest_Relayed: a method this facade has never heard of
// reaches the agent instead of being refused. This is what lets codex gain
// slash commands without a doctl release.
func TestFacade_UnknownRequest_Relayed(t *testing.T) {
	f, h, _ := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI
	h.QueueRelayReply([]byte(`{"id":1,"result":{"data":["a"]}}`))

	result, err := dispatch(t, f, "some/futureMethod", map[string]any{"q": 1})
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":["a"]}`, string(result.(json.RawMessage)))
	assert.Contains(t, string(h.LastRelay()), `"some/futureMethod"`)
}

// TestFacade_Relay_AgentErrorReachesClient: an agent-side JSON-RPC error is
// the agent's verdict on the client's request, so it must arrive as that
// request's error with the agent's own code — not be swallowed into a
// success, and not be relabelled as a proxy failure.
func TestFacade_Relay_AgentErrorReachesClient(t *testing.T) {
	f, h, _ := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI
	h.QueueRelayReply([]byte(`{"id":1,"error":{"code":-32601,"message":"Method not found"}}`))

	_, err := dispatch(t, f, "some/unsupported", nil)
	var rpcErr *agentproxy.RPCError
	require.ErrorAs(t, err, &rpcErr)
	assert.Equal(t, -32601, rpcErr.Code)
	assert.Equal(t, "Method not found", rpcErr.Message)
}

// TestFacade_Relay_DeclinedFallsBackToStub: when the adapter declines, the
// facade must still answer. turn/interrupt falls back to its empty reply and
// an unknown method to method-not-found; either way the client is never left
// waiting on a request that will never be answered.
func TestFacade_Relay_DeclinedFallsBackToStub(t *testing.T) {
	f, h, _ := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI
	h.QueueRelayReply(nil)

	result, err := dispatch(t, f, "turn/interrupt", nil)
	require.NoError(t, err)
	assert.Equal(t, turnInterruptResult{}, result)

	_, err = dispatch(t, f, "some/unknown", nil)
	assert.ErrorIs(t, err, agentproxy.ErrMethodNotFound)
}

// TestFacade_Relay_TransportFailureIsReported: a relay that never reached the
// agent must not be reported to the client as a successful interrupt. The
// client gets an error it can show, which is the honest answer.
func TestFacade_Relay_TransportFailureIsReported(t *testing.T) {
	f, h, _ := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI
	h.QueueRelayError(http.StatusServiceUnavailable)

	_, err := dispatch(t, f, "turn/interrupt", nil)
	var rpcErr *agentproxy.RPCError
	require.ErrorAs(t, err, &rpcErr)
	assert.Contains(t, rpcErr.Message, "turn/interrupt")
}

// TestFacade_Relay_NotAttemptedForNonCodexKind: relaying codex frames to a
// session running some other agent would be sending it a protocol it does not
// speak, so the pre-relay behaviour stands.
func TestFacade_Relay_NotAttemptedForNonCodexKind(t *testing.T) {
	f, h, _ := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindOpenCode

	result, err := dispatch(t, f, "turn/interrupt", nil)
	require.NoError(t, err)
	assert.Equal(t, turnInterruptResult{}, result)
	assert.Nil(t, h.LastRelay(), "a non-codex session must not relay codex frames")
}

// TestUnwrapRelayReply covers the shapes the agent can answer with, including
// the empty one: a reply carrying neither result nor error still has to
// resolve the client's request rather than becoming a second failure mode.
func TestUnwrapRelayReply(t *testing.T) {
	result, handled, err := unwrapRelayReply([]byte(`{"id":1,"result":{"ok":true}}`), "m")
	require.True(t, handled)
	require.NoError(t, err)
	assert.JSONEq(t, `{"ok":true}`, string(result.(json.RawMessage)))

	_, handled, err = unwrapRelayReply([]byte(`{"id":1}`), "m")
	assert.True(t, handled)
	assert.NoError(t, err)

	_, handled, err = unwrapRelayReply([]byte(`not json`), "m")
	assert.True(t, handled)
	assert.Error(t, err, "an unreadable reply must fail the client's request, not pass as success")
}

// queueGatedRun queues a turn that reaches a gated request of the given kind,
// carrying nativeFrame as the agent's own frame for it.
func queueGatedRun(h *agentproxytest.Harness, kind, hitlID, nativeFrame string) {
	h.QueueRun("run-native-1",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type:            string(godo.HostedAgentEventKindHITLRequested),
			Data:            json.RawMessage(`{"hitl_id":"` + hitlID + `","payload":{"kind":"` + kind + `","itemId":"call_1"}}`),
			SourceEventType: "mcpServer/elicitation/request",
			SourceRaw:       []byte(nativeFrame),
		},
	)
}

// startGatedTurn drives the turn far enough that the gated request has been
// dispatched, returning the recorder positioned at it.
func startGatedTurn(t *testing.T, f *Facade) {
	t.Helper()
	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "do the thing"}},
	})
	require.NoError(t, err)
}

// TestFacade_NativeApproval_RoundTripsAnUnmappedKind is the M2 "done when":
// a gated request of a kind this proxy has no translation for reaches the
// client as the agent's own request, and the client's own reply — content and
// all — reaches the harness. Before this path such a kind was auto-declined
// without the user ever seeing it.
func TestFacade_NativeApproval_RoundTripsAnUnmappedKind(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI

	queueGatedRun(h, "mcp_elicitation", "hitl-native-1",
		`{"id":41,"method":"mcpServer/elicitation/request","params":{"threadId":"vm-thread","turnId":"vm-turn","message":"pick a port"}}`)
	startGatedTurn(t, f)
	_ = rec.next(t) // turn/started

	req := rec.nextRequest(t)
	assert.Equal(t, "mcpServer/elicitation/request", req.method,
		"the client must be asked the agent's own question, not a synthesized approval")
	params, ok := req.params.(map[string]any)
	require.True(t, ok, "native params should be the forwarded raw map, got %T", req.params)
	assert.Equal(t, "pick a port", params["message"])
	// Same rewriting the notification path does: the client only ever knows
	// the proxy's ids, so the VM's would be meaningless to it.
	assert.Equal(t, testSessionID, params["threadId"])
	assert.Equal(t, "run-native-1", params["turnId"])

	req.respond(map[string]any{"action": "accept", "content": map[string]any{"port": 8080}})

	res := h.NextHITLResolution(t, 2*time.Second)
	assert.Equal(t, "hitl-native-1", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeApprove), res.Outcome)
	require.NotEmpty(t, res.SourceRaw, "the client's native reply must reach the harness")
	assert.JSONEq(t,
		`{"id":41,"result":{"action":"accept","content":{"port":8080}}}`,
		string(res.SourceRaw),
		"the reply must be addressed to the agent's own request id and carry the content verbatim")
}

// TestFacade_NativeApproval_RefusalIsReportedAsReject: the raw reply is what
// the agent acts on, but the harness still records a verdict, and a decline
// must not be audited as an approval.
func TestFacade_NativeApproval_RefusalIsReportedAsReject(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI

	queueGatedRun(h, "mcp_elicitation", "hitl-native-2",
		`{"id":42,"method":"mcpServer/elicitation/request","params":{"threadId":"vm-thread","message":"pick a port"}}`)
	startGatedTurn(t, f)
	_ = rec.next(t)

	rec.nextRequest(t).respond(map[string]any{"action": "decline"})

	res := h.NextHITLResolution(t, 2*time.Second)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeReject), res.Outcome)
}

// TestFacade_NativeApproval_FileChangeStaysAutoRejected: being able to forward
// the request faithfully is not a reason to start forwarding it. The
// apply_patch auto-reject is a deliberate policy (see
// autoRejectFileChangeApproval), so it must survive this path existing.
func TestFacade_NativeApproval_FileChangeStaysAutoRejected(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI

	queueGatedRun(h, "file_change", "hitl-fc-1",
		`{"id":43,"method":"item/fileChange/requestApproval","params":{"threadId":"vm-thread","itemId":"call_1"}}`)
	startGatedTurn(t, f)
	_ = rec.next(t)

	res := h.NextHITLResolution(t, 2*time.Second)
	assert.Equal(t, "hitl-fc-1", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeReject), res.Outcome)
	assert.Empty(t, res.SourceRaw, "an auto-reject answers on the user's behalf; it has no native reply")
	rec.expectNoRequest(t)
}

// TestFacade_NativeApproval_NotAttemptedWithoutTheNativeFrame: with no frame
// to forward there is nothing to ask, so the pre-M2 behaviour stands rather
// than the request being left unanswered.
func TestFacade_NativeApproval_NotAttemptedWithoutTheNativeFrame(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.AgentKind = godo.HostedAgentKindCodexCLI

	h.QueueRun("run-native-2",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindHITLRequested),
			Data: json.RawMessage(`{"hitl_id":"hitl-bare","payload":{"kind":"mcp_elicitation"}}`),
		},
	)
	startGatedTurn(t, f)
	_ = rec.next(t)

	res := h.NextHITLResolution(t, 2*time.Second)
	assert.Equal(t, "hitl-bare", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeReject), res.Outcome)
	rec.expectNoRequest(t)
}

// TestNativeApprovalOutcome covers the verdict the audit trail records for
// each reply shape codex can answer with. Only the classification is under
// test — the reply itself always reaches the agent verbatim.
func TestNativeApprovalOutcome(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   godo.HostedAgentHITLOutcome
	}{
		{name: "command decision accept", result: `{"decision":"accept"}`, want: godo.HostedAgentHITLOutcomeApprove},
		{name: "command decision decline", result: `{"decision":"decline"}`, want: godo.HostedAgentHITLOutcomeReject},
		{name: "elicitation accept", result: `{"action":"accept","content":{}}`, want: godo.HostedAgentHITLOutcomeApprove},
		{name: "elicitation decline", result: `{"action":"decline"}`, want: godo.HostedAgentHITLOutcomeReject},
		{name: "elicitation cancel", result: `{"action":"cancel"}`, want: godo.HostedAgentHITLOutcomeReject},
		// No verdict field at all: the user answered, so it is an approval.
		{name: "tool answers", result: `{"answers":["yes"]}`, want: godo.HostedAgentHITLOutcomeApprove},
		{name: "empty object", result: `{}`, want: godo.HostedAgentHITLOutcomeApprove},
		{name: "not an object", result: `"whatever"`, want: godo.HostedAgentHITLOutcomeApprove},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, nativeApprovalOutcome(json.RawMessage(tt.result)))
		})
	}
}
