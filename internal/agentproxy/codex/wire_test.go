package codex

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/agentproxy"
	"github.com/digitalocean/doctl/internal/agentproxy/agentproxytest"
	"github.com/digitalocean/godo"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file is the wire-level complement to facade_test.go: those tests call
// Facade.Dispatch directly with a fake Notifier, which is fast and thorough
// for facade logic but never exercises proxy.go's actual JSON-RPC/WebSocket
// framing (ServeListener, handleConn's request/notification split, the
// SetNotifier/NotifierAware wiring, or whether a struct's json tags really
// serialize the way the Go-level tests assume). wsTestClient below drives a
// real agentproxy.ServeListener over a real *websocket.Conn, so a struct-tag
// typo or a framing regression in proxy.go would fail here even if every
// Dispatch-level test still passes.

// wsTestClient is a minimal scripted WebSocket JSON-RPC client: send one
// message, read the next one back, with a short deadline so a regression
// that breaks the reply/notification sequence fails fast and loud instead of
// hanging the test suite.
type wsTestClient struct {
	t    *testing.T
	conn *websocket.Conn
}

func dialTestClient(t *testing.T, url string) *wsTestClient {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err, "dial %s", url)
	t.Cleanup(func() { conn.Close() })
	return &wsTestClient{t: t, conn: conn}
}

func (c *wsTestClient) send(id, method string, params any) {
	c.t.Helper()
	msg := map[string]any{"method": method}
	if id != "" {
		msg["id"] = id
	}
	if params != nil {
		msg["params"] = params
	}
	require.NoError(c.t, c.conn.WriteJSON(msg), "send %s", method)
}

// recv reads the next frame within 5s, or fails the test — never polls or
// sleeps to wait for a notification.
func (c *wsTestClient) recv() map[string]any {
	c.t.Helper()
	require.NoError(c.t, c.conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	var msg map[string]any
	require.NoError(c.t, c.conn.ReadJSON(&msg), "recv")
	return msg
}

// result returns msg["result"] as a map, failing the test if msg carries a
// JSON-RPC error instead or result isn't an object.
func (c *wsTestClient) result(msg map[string]any) map[string]any {
	c.t.Helper()
	if errObj, ok := msg["error"]; ok {
		c.t.Fatalf("got error reply instead of result: %v", errObj)
	}
	result, ok := msg["result"].(map[string]any)
	require.True(c.t, ok, "result is not an object: %v", msg)
	return result
}

// reply sends a bare JSON-RPC reply — id + result, no method — the shape a
// real client sends back to a server-initiated request (see
// item/commandExecution/requestApproval), as opposed to send, which always
// includes a method since only requests/notifications originate client-side.
func (c *wsTestClient) reply(id any, result any) {
	c.t.Helper()
	require.NoError(c.t, c.conn.WriteJSON(map[string]any{"id": id, "result": result}), "reply")
}

// TestWire_InitializeThreadStartTurnStartDeltasTurnCompleted is the M2 "done
// when" sequence test at the wire level: initialize -> thread/start ->
// turn/start -> deltas -> turn/completed, over a real WebSocket connection to
// a real agentproxy.ServeListener, against a fake harness instead of a live
// one. facade_test.go's TestFacade_TurnStart_StreamsToCompletion covers the
// same sequence at the Dispatch level; this one additionally proves the JSON
// wire shapes and proxy.go's framing are correct. Extend this file (or
// facade_test.go) for M3 (tool-call items) and M4 (approval round-trip)
// rather than retrofitting tests later.
func TestWire_InitializeThreadStartTurnStartDeltasTurnCompleted(t *testing.T) {
	const sessionID = "sess-wire-test"
	const runID = "run-wire-1"

	harness := agentproxytest.New(t, sessionID)
	harness.QueueRun(runID,
		agentproxytest.Event{Type: "run.started", Data: json.RawMessage(`{"agent":"codex"}`)},
		agentproxytest.Event{Type: "run.token_delta", Data: json.RawMessage(`{"text":"Four"}`)},
		agentproxytest.Event{Type: "run.token_delta", Data: json.RawMessage(`{"text":"!"}`)},
		agentproxytest.Event{Type: "run.completed", Data: json.RawMessage(`{}`)},
	)

	godoClient, err := godo.New(http.DefaultClient, godo.SetBaseURL(harness.Server.URL+"/"))
	require.NoError(t, err)
	svc := do.NewHostedAgentsService(godoClient)
	newFacade := func() agentproxy.Facade {
		return &Facade{SessionID: sessionID, Sessions: svc}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	serveErr := make(chan error, 1)
	go func() { serveErr <- agentproxy.ServeListener(ctx, ln, newFacade) }()

	client := dialTestClient(t, "ws://"+addr+"/")

	// initialize
	client.send("1", "initialize", map[string]any{
		"clientInfo":   map[string]any{"name": "test", "version": "0"},
		"capabilities": map[string]any{},
	})
	initResult := client.result(client.recv())
	require.NotEmpty(t, initResult["userAgent"], "initialize result missing userAgent: %v", initResult)

	// thread/start
	client.send("2", "thread/start", map[string]any{})
	threadResult := client.result(client.recv())
	thread, ok := threadResult["thread"].(map[string]any)
	require.True(t, ok, "thread/start result missing thread object: %v", threadResult)
	require.Equal(t, sessionID, thread["id"])

	// turn/start
	client.send("3", "turn/start", map[string]any{
		"threadId": sessionID,
		"input":    []any{map[string]any{"type": "text", "text": "test prompt"}},
	})
	turnStartResult := client.result(client.recv())
	turn, ok := turnStartResult["turn"].(map[string]any)
	require.True(t, ok, "turn/start result missing turn object: %v", turnStartResult)
	require.Equal(t, runID, turn["id"])
	require.Equal(t, "inProgress", turn["status"])

	// Async notification sequence, in order, with the fields that matter.
	notif := client.recv()
	require.Equal(t, "turn/started", notif["method"], "full: %v", notif)

	notif = client.recv()
	require.Equal(t, "item/started", notif["method"], "full: %v", notif)
	params, _ := notif["params"].(map[string]any)
	item, _ := params["item"].(map[string]any)
	itemID, _ := item["id"].(string)
	require.NotEmpty(t, itemID, "item/started missing item.id: %v", notif)

	for _, want := range []string{"Four", "!"} {
		notif = client.recv()
		require.Equal(t, "item/agentMessage/delta", notif["method"], "full: %v", notif)
		params, _ = notif["params"].(map[string]any)
		require.Equal(t, want, params["delta"])
		require.Equal(t, itemID, params["itemId"])
	}

	notif = client.recv()
	require.Equal(t, "item/completed", notif["method"], "full: %v", notif)
	params, _ = notif["params"].(map[string]any)
	item, _ = params["item"].(map[string]any)
	require.Equal(t, "Four!", item["text"])

	notif = client.recv()
	require.Equal(t, "turn/completed", notif["method"], "full: %v", notif)
	params, _ = notif["params"].(map[string]any)
	finalTurn, _ := params["turn"].(map[string]any)
	require.Equal(t, "completed", finalTurn["status"])

	cancel()
	select {
	case err := <-serveErr:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("ServeListener did not shut down within 5s of ctx cancellation")
	}
}

// TestWire_ApprovalRoundTrip is the M4 "done when" sequence at the wire
// level: a run.human_input_requested event produces a real
// item/commandExecution/requestApproval frame carrying both a method AND an
// id — a genuine JSON-RPC request, not a notification — over a real
// WebSocket connection to a real agentproxy.ServeListener. The test client
// replies with a bare {"id":..., "result": {...}} frame (no method, exactly
// what a real client sends back), and asserts the harness actually received
// a POST .../hitl/{id} resolving it as approved. Complements
// TestFacade_TurnStart_ApprovalAccept (facade_test.go), which proves the
// same logic at the Dispatch level without exercising proxy.go's framing.
func TestWire_ApprovalRoundTrip(t *testing.T) {
	const sessionID = "sess-wire-approval"
	const runID = "run-wire-approval"

	harness := agentproxytest.New(t, sessionID)
	harness.QueueRun(runID,
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallStarted),
			Data: json.RawMessage(`{"tool_call_id":"call_1","name":"command_execution","input":{"command":"/bin/bash -lc \"find /workspace | sort\"","cwd":"/workspace"}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindHITLRequested),
			Data: json.RawMessage(`{"hitl_id":"hitl-wire-1","payload":{"kind":"command_execution","itemId":"call_1","turnId":"harness-internal-turn-id","startedAtMs":1000,"environmentId":"local","command":"/bin/bash -lc \"find /workspace | sort\"","cwd":"/workspace","commandActions":[],"proposedExecpolicyAmendment":["sort"]}}`),
		},
	)

	godoClient, err := godo.New(http.DefaultClient, godo.SetBaseURL(harness.Server.URL+"/"))
	require.NoError(t, err)
	svc := do.NewHostedAgentsService(godoClient)
	newFacade := func() agentproxy.Facade {
		return &Facade{SessionID: sessionID, Sessions: svc}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	serveErr := make(chan error, 1)
	go func() { serveErr <- agentproxy.ServeListener(ctx, ln, newFacade) }()

	client := dialTestClient(t, "ws://"+addr+"/")

	client.send("1", "initialize", map[string]any{
		"clientInfo":   map[string]any{"name": "test", "version": "0"},
		"capabilities": map[string]any{},
	})
	client.result(client.recv())

	client.send("2", "thread/start", map[string]any{})
	client.result(client.recv())

	client.send("3", "turn/start", map[string]any{
		"threadId": sessionID,
		"input":    []any{map[string]any{"type": "text", "text": "find files, sorted"}},
	})
	client.result(client.recv())

	notif := client.recv()
	require.Equal(t, "turn/started", notif["method"], "full: %v", notif)
	notif = client.recv()
	require.Equal(t, "item/started", notif["method"], "full: %v", notif) // agentMessage
	notif = client.recv()
	require.Equal(t, "item/started", notif["method"], "full: %v", notif) // commandExecution

	// The approval request itself: has both a method AND an id, unlike every
	// notification above (method only) — that's what makes it a request a
	// reply is expected for, not just an FYI.
	approvalReq := client.recv()
	require.Equal(t, "item/commandExecution/requestApproval", approvalReq["method"], "full: %v", approvalReq)
	reqID, ok := approvalReq["id"]
	require.True(t, ok, "requestApproval frame missing id, so it's indistinguishable from a notification: %v", approvalReq)
	params, _ := approvalReq["params"].(map[string]any)
	assert.Equal(t, "call_1", params["itemId"])
	assert.Equal(t, runID, params["turnId"])
	// proposedExecpolicyAmendment must never reach the wire — even though the
	// queued HITL carries one — since sending it re-introduces the "always
	// allow" option the harness can't actually honor (see
	// commandExecutionRequestApprovalParams' doc comment).
	_, present := params["proposedExecpolicyAmendment"]
	assert.False(t, present, "proposedExecpolicyAmendment must not be forwarded to the client")

	client.reply(reqID, map[string]any{"decision": "accept"})

	res := harness.NextHITLResolution(t, 2*time.Second)
	assert.Equal(t, "hitl-wire-1", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeApprove), res.Outcome)
	assert.Equal(t, string(godo.HostedAgentResolutionSourceInlineKeystroke), res.Source)

	cancel()
	select {
	case err := <-serveErr:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("ServeListener did not shut down within 5s of ctx cancellation")
	}
}

// dialTestClientRetry dials, retrying on failure, up to timeout. Needed for
// TestWire_Reconnect_StartsFreshEventLoop: ServeListener's "one slot" accept
// loop only releases the slot after the previous connection's handler
// (including all its defers) fully unwinds, so dialing immediately after
// closing the first connection can race a 409 "already connected" — retrying
// waits out that teardown instead of guessing a sleep duration.
func dialTestClientRetry(t *testing.T, url string, timeout time.Duration) *wsTestClient {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err == nil {
			t.Cleanup(func() { conn.Close() })
			return &wsTestClient{t: t, conn: conn}
		}
		if time.Now().After(deadline) {
			t.Fatalf("dial %s: timed out retrying: %v", url, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestWire_Reconnect_StartsFreshEventLoop confirms a second, separate
// `codex --remote` connection to an already-used proxy gets a working
// turn/start. ServeListener's per-connection factory (newFacade) is what
// makes this safe: each connection gets a fresh Facade with its own
// notifier and event-loop state, so a still-unwinding previous connection's
// goroutines cannot leak notifications onto the new socket or leave
// streamStarted stuck true. Before that factory existed, reusing one Facade
// across disconnect/reconnect was a real live bug.
func TestWire_Reconnect_StartsFreshEventLoop(t *testing.T) {
	const sessionID = "sess-wire-reconnect"

	harness := agentproxytest.New(t, sessionID)
	godoClient, err := godo.New(http.DefaultClient, godo.SetBaseURL(harness.Server.URL+"/"))
	require.NoError(t, err)
	svc := do.NewHostedAgentsService(godoClient)
	newFacade := func() agentproxy.Facade {
		return &Facade{SessionID: sessionID, Sessions: svc}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	serveErr := make(chan error, 1)
	go func() { serveErr <- agentproxy.ServeListener(ctx, ln, newFacade) }()

	// First connection: a full turn, then disconnect without ever finishing
	// the second turn — mirrors a user quitting codex mid-session.
	harness.QueueRun("run-1",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"first"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	client1 := dialTestClient(t, "ws://"+addr+"/")
	client1.send("1", "initialize", map[string]any{"clientInfo": map[string]any{"name": "test", "version": "0"}, "capabilities": map[string]any{}})
	client1.result(client1.recv())
	client1.send("2", "thread/start", map[string]any{})
	client1.result(client1.recv())
	client1.send("3", "turn/start", map[string]any{
		"threadId": sessionID,
		"input":    []any{map[string]any{"type": "text", "text": "first turn"}},
	})
	client1.result(client1.recv())

	notif := client1.recv()
	require.Equal(t, "turn/started", notif["method"], "full: %v", notif)
	notif = client1.recv()
	require.Equal(t, "item/started", notif["method"], "full: %v", notif)
	notif = client1.recv()
	require.Equal(t, "item/agentMessage/delta", notif["method"], "full: %v", notif)
	notif = client1.recv()
	require.Equal(t, "item/completed", notif["method"], "full: %v", notif)
	notif = client1.recv()
	require.Equal(t, "turn/completed", notif["method"], "full: %v", notif)

	require.NoError(t, client1.conn.Close())

	// Second, separate connection — factory must hand it a fresh Facade.
	// Queue a fresh run and confirm this connection's own turn/start gets
	// its own turn/started, not silence.
	harness.QueueRun("run-2",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"second"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	client2 := dialTestClientRetry(t, "ws://"+addr+"/", 5*time.Second)
	client2.send("1", "initialize", map[string]any{"clientInfo": map[string]any{"name": "test", "version": "0"}, "capabilities": map[string]any{}})
	client2.result(client2.recv())
	client2.send("2", "thread/start", map[string]any{})
	client2.result(client2.recv())
	client2.send("3", "turn/start", map[string]any{
		"threadId": sessionID,
		"input":    []any{map[string]any{"type": "text", "text": "second turn, after a reconnect"}},
	})
	client2.result(client2.recv())

	notif = client2.recv()
	require.Equal(t, "turn/started", notif["method"], "second connection got no notifications at all: %v", notif)

	notif = client2.recv()
	require.Equal(t, "item/started", notif["method"], "full: %v", notif)
	notif = client2.recv()
	require.Equal(t, "item/agentMessage/delta", notif["method"], "full: %v", notif)
	params, _ := notif["params"].(map[string]any)
	assert.Equal(t, "second", params["delta"])
	notif = client2.recv()
	require.Equal(t, "item/completed", notif["method"], "full: %v", notif)
	notif = client2.recv()
	require.Equal(t, "turn/completed", notif["method"], "full: %v", notif)

	cancel()
	select {
	case err := <-serveErr:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("ServeListener did not shut down within 5s of ctx cancellation")
	}
}

// TestWire_FileChangeAlwaysAutoRejected is the FileChange sibling of
// TestWire_ApprovalRoundTrip, but for the opposite behavior: a
// run.human_input_requested (kind file_change) must NOT produce a
// client-facing item/fileChange/requestApproval frame at all over a real
// WebSocket connection — see autoRejectFileChangeApproval — and the harness
// must still receive an auto-rejected resolution.
func TestWire_FileChangeAlwaysAutoRejected(t *testing.T) {
	const sessionID = "sess-wire-filechange"
	const runID = "run-wire-filechange"

	harness := agentproxytest.New(t, sessionID)
	harness.QueueRun(runID,
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallStarted),
			Data: json.RawMessage(`{"tool_call_id":"call_1","name":"file_change","input":{}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindHITLRequested),
			Data: json.RawMessage(`{"hitl_id":"hitl-wire-fc-1","payload":{"category":"permission","grantRoot":null,"itemId":"call_1","kind":"file_change","reason":null,"startedAtMs":1000,"turnId":"harness-internal-turn-id"}}`),
		},
	)

	godoClient, err := godo.New(http.DefaultClient, godo.SetBaseURL(harness.Server.URL+"/"))
	require.NoError(t, err)
	svc := do.NewHostedAgentsService(godoClient)
	newFacade := func() agentproxy.Facade {
		return &Facade{SessionID: sessionID, Sessions: svc}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	serveErr := make(chan error, 1)
	go func() { serveErr <- agentproxy.ServeListener(ctx, ln, newFacade) }()

	client := dialTestClient(t, "ws://"+addr+"/")

	client.send("1", "initialize", map[string]any{
		"clientInfo":   map[string]any{"name": "test", "version": "0"},
		"capabilities": map[string]any{},
	})
	client.result(client.recv())

	client.send("2", "thread/start", map[string]any{})
	client.result(client.recv())

	client.send("3", "turn/start", map[string]any{
		"threadId": sessionID,
		"input":    []any{map[string]any{"type": "text", "text": "create a file"}},
	})
	client.result(client.recv())

	notif := client.recv()
	require.Equal(t, "turn/started", notif["method"], "full: %v", notif)
	notif = client.recv()
	require.Equal(t, "item/started", notif["method"], "full: %v", notif) // agentMessage
	notif = client.recv()
	require.Equal(t, "item/started", notif["method"], "full: %v", notif) // fileChange

	// No item/fileChange/requestApproval should ever arrive: file_change is
	// rejected entirely server-side (see autoRejectFileChangeApproval), so
	// the harness resolution below must show up with no client frame at all
	// in between.
	res := harness.NextHITLResolution(t, 2*time.Second)
	assert.Equal(t, "hitl-wire-fc-1", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeReject), res.Outcome)
	assert.Equal(t, string(godo.HostedAgentResolutionSourceOutOfBand), res.Source)

	cancel()
	select {
	case err := <-serveErr:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("ServeListener did not shut down within 5s of ctx cancellation")
	}
}

// TestWire_ShutdownCancelsConnectionGoroutines is the regression test for a
// real bug found live-verifying M5: connCtx used to be rooted in
// context.Background(), so canceling ServeListener's own ctx (e.g.
// SIGINT/SIGTERM) never signaled a live connection's background goroutines
// (runEventLoop) to stop — only that connection's own handler returning
// normally did. ServeListener itself always returned quickly regardless
// (Go's http.Server.Shutdown doesn't track hijacked WebSocket connections),
// which is what made this bug easy to miss: the server-level shutdown looked
// fine even though a facade's goroutines kept running with no stop signal.
//
// This uses HangStreamAfterEvents so the fake harness's SSE stream would
// hang open forever on its own — the only thing that can end it is ctx
// cancellation actually propagating all the way down to the underlying
// StreamSession HTTP request, exactly the path the bug broke.
func TestWire_ShutdownCancelsConnectionGoroutines(t *testing.T) {
	const sessionID = "sess-wire-shutdown"
	const runID = "run-wire-shutdown"

	harness := agentproxytest.New(t, sessionID)
	harness.HangStreamAfterEvents(true)
	harness.QueueRun(runID,
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
	)

	godoClient, err := godo.New(http.DefaultClient, godo.SetBaseURL(harness.Server.URL+"/"))
	require.NoError(t, err)
	svc := do.NewHostedAgentsService(godoClient)
	var facade *Facade
	newFacade := func() agentproxy.Facade {
		facade = &Facade{SessionID: sessionID, Sessions: svc}
		return facade
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	serveErr := make(chan error, 1)
	go func() { serveErr <- agentproxy.ServeListener(ctx, ln, newFacade) }()

	client := dialTestClient(t, "ws://"+addr+"/")
	client.send("1", "initialize", map[string]any{
		"clientInfo":   map[string]any{"name": "test", "version": "0"},
		"capabilities": map[string]any{},
	})
	client.result(client.recv())
	client.send("2", "thread/start", map[string]any{})
	client.result(client.recv())
	client.send("3", "turn/start", map[string]any{
		"threadId": sessionID,
		"input":    []any{map[string]any{"type": "text", "text": "hi"}},
	})
	client.result(client.recv())

	notif := client.recv()
	require.Equal(t, "turn/started", notif["method"], "full: %v", notif)

	// Confirm the precondition: runEventLoop is actually active (and, with
	// the stream hung open, would stay that way forever without the fix)
	// before testing that shutdown stops it.
	require.NotNil(t, facade, "factory should have produced a Facade for the live connection")
	facade.mu.Lock()
	started := facade.streamStarted
	facade.mu.Unlock()
	require.True(t, started, "expected streamStarted to be true while runEventLoop is active")

	// Cancel the SERVER's ctx, not the client connection — simulates
	// SIGINT/SIGTERM arriving while a connection is active.
	cancel()

	select {
	case err := <-serveErr:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("ServeListener did not shut down within 5s of ctx cancellation")
	}

	// The real assertion: runEventLoop must have actually been signaled to
	// stop and reset its state — not just that ServeListener's own
	// HTTP-level Shutdown returned quickly, which it always did even with
	// the bug present.
	require.Eventually(t, func() bool {
		facade.mu.Lock()
		defer facade.mu.Unlock()
		return !facade.streamStarted
	}, 5*time.Second, 10*time.Millisecond, "runEventLoop should have exited and reset streamStarted after ctx cancellation")
}
