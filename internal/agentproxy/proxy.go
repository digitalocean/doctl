package agentproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// rpcMessage is a JSON-RPC 2.0 envelope, minus the "jsonrpc" field itself —
// the real codex app-server protocol omits "jsonrpc":"2.0" on the wire
// (confirmed by the M0 protocol capture), and codex --remote doesn't require
// it either, so the proxy matches what's actually sent instead of adding a
// field nothing checks.
type rpcMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result any             `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

var upgrader = websocket.Upgrader{
	// Loopback-only listener (see Serve); no browser reaches this, so origin
	// checking buys nothing and would only get in websocat's way.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Serve binds a WebSocket listener on 127.0.0.1:port — never "localhost"
// (IPv6 ::1 resolution can fail to connect) or 0.0.0.0 (codex requires auth
// for non-loopback listeners, and there's no reason to expose this beyond the
// machine anyway) — and speaks facade's JSON-RPC protocol to exactly one
// connected client at a time, until ctx is canceled.
func Serve(ctx context.Context, port int, facade Facade) error {
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	// One slot: the first upgrade takes it, a second concurrent attempt is
	// refused until the first disconnects and returns it.
	slot := make(chan struct{}, 1)
	slot <- struct{}{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-slot:
		default:
			http.Error(w, "a client is already connected", http.StatusConflict)
			return
		}
		defer func() { slot <- struct{}{} }()

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("agentproxy: upgrade error: %v", err)
			return
		}
		defer conn.Close()

		log.Printf("agentproxy: client connected from %s", r.RemoteAddr)
		handleConn(r.Context(), conn, facade)
		log.Printf("agentproxy: client disconnected")
	})

	server := &http.Server{Addr: addr, Handler: mux}

	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

// handleConn pumps one WebSocket connection: read a text frame, dispatch it
// to facade, write a response if (and only if) the message was a request.
//
// Every request gets *some* reply, even an error — that invariant lives here,
// structurally, rather than in each facade's Dispatch, so a facade can never
// recreate the "TUI hangs forever on a logged-and-dropped request" bug by
// forgetting to answer.
func handleConn(ctx context.Context, conn *websocket.Conn, facade Facade) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("agentproxy: read error: %v", err)
			}
			return
		}

		var msg rpcMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("agentproxy: malformed JSON-RPC frame, dropping: %v", err)
			continue
		}

		result, dispatchErr := facade.Dispatch(ctx, msg.Method, msg.Params)

		if errors.Is(dispatchErr, ErrMethodNotFound) {
			log.Printf("unhandled: %s", msg.Method)
		} else if dispatchErr != nil {
			log.Printf("agentproxy: %s: %v", msg.Method, dispatchErr)
		}

		if len(msg.ID) == 0 {
			// Notification: no id, no reply — even on error.
			continue
		}

		reply := rpcMessage{ID: msg.ID}
		if dispatchErr != nil {
			var rpcErr *RPCError
			if !errors.As(dispatchErr, &rpcErr) {
				rpcErr = &RPCError{Code: -32603, Message: dispatchErr.Error()}
			}
			reply.Error = rpcErr
		} else {
			reply.Result = result
		}

		out, err := json.Marshal(reply)
		if err != nil {
			log.Printf("agentproxy: failed to marshal reply for %s: %v", msg.Method, err)
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, out); err != nil {
			log.Printf("agentproxy: write error: %v", err)
			return
		}
	}
}
