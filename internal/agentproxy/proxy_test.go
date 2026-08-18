package agentproxy

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// stubFacade answers every request with an empty result — just enough to
// prove a connection is being served.
type stubFacade struct{}

func (stubFacade) Dispatch(ctx context.Context, method string, params json.RawMessage) (any, error) {
	return struct{}{}, nil
}

// TestServeListener_SecondClientRefusedWhileFirstConnected documents the
// multi-client attach behavior: the proxy serves exactly one WebSocket
// client at a time. A second concurrent connection attempt is refused at
// the HTTP layer with 409 Conflict — before any upgrade, so it can never
// touch the first connection's facade state — and the slot is handed back
// once the first client disconnects, letting a new client (e.g. a restarted
// TUI) connect to the same still-running proxy.
func TestServeListener_SecondClientRefusedWhileFirstConnected(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() { serveDone <- ServeListener(ctx, ln, func() Facade { return stubFacade{} }) }()

	url := "ws://" + ln.Addr().String() + "/"

	first, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("first client should connect: %v", err)
	}
	defer first.Close()

	// Prove the first connection is actually being served (request in,
	// reply out) — not merely accepted.
	if err := first.WriteMessage(websocket.TextMessage, []byte(`{"id":1,"method":"anything"}`)); err != nil {
		t.Fatalf("first client write: %v", err)
	}
	if _, _, err := first.ReadMessage(); err != nil {
		t.Fatalf("first client should get a reply: %v", err)
	}

	// A second concurrent client must be refused with 409, without
	// disturbing the first connection.
	second, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		second.Close()
		t.Fatal("second concurrent client should be refused")
	}
	if resp == nil || resp.StatusCode != http.StatusConflict {
		t.Fatalf("second client should be refused with 409 Conflict, got %+v", resp)
	}

	// The first connection must have survived the refused attempt.
	if err := first.WriteMessage(websocket.TextMessage, []byte(`{"id":2,"method":"anything"}`)); err != nil {
		t.Fatalf("first client write after refused second: %v", err)
	}
	if _, _, err := first.ReadMessage(); err != nil {
		t.Fatalf("first client should still be served after the refused attempt: %v", err)
	}

	// Once the first client disconnects, the slot is returned and a new
	// client can connect. The return happens after handleConn unwinds, so
	// poll briefly rather than racing it.
	first.Close()
	deadline := time.Now().Add(2 * time.Second)
	for {
		replacement, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err == nil {
			replacement.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("a new client should be able to connect after the first disconnected: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case <-serveDone:
	case <-time.After(5 * time.Second):
		t.Fatal("ServeListener did not shut down")
	}
}
