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
	facade := &Facade{SessionID: sessionID, Sessions: do.NewHostedAgentsService(godoClient)}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	serveErr := make(chan error, 1)
	go func() { serveErr <- agentproxy.ServeListener(ctx, ln, facade) }()

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
