package agentproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
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

// wsNotifier serializes every write to a WebSocket connection — replies from
// the main read/dispatch loop and asynchronous notifications from a facade's
// background goroutines alike — behind one mutex. gorilla/websocket conns
// aren't safe for concurrent writers; once a facade can push notifications on
// its own timeline (Notifier), there are always at least two potential
// writers, so this is required, not defensive.
type wsNotifier struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (w *wsNotifier) writeMessage(msg rpcMessage) error {
	out, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", msg.Method, err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(websocket.TextMessage, out)
}

// Notify implements agentproxy.Notifier.
func (w *wsNotifier) Notify(method string, params any) error {
	return w.writeMessage(rpcMessage{Method: method, Params: mustRawMessage(params)})
}

func mustRawMessage(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		// A facade handed us a value that can't marshal — a programmer error
		// in that facade, not a runtime condition to recover from gracefully.
		panic(fmt.Sprintf("agentproxy: params do not marshal to JSON: %v", err))
	}
	return data
}

// Serve binds a WebSocket listener on 127.0.0.1:port — never "localhost"
// (IPv6 ::1 resolution can fail to connect) or 0.0.0.0 (codex requires auth
// for non-loopback listeners, and there's no reason to expose this beyond the
// machine anyway) — and speaks facade's JSON-RPC protocol to exactly one
// connected client at a time, until ctx is canceled.
func Serve(ctx context.Context, port int, facade Facade) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	return ServeListener(ctx, ln, facade)
}

// ServeListener is Serve, except it speaks facade's protocol over an
// already-bound listener instead of binding one from a port number itself.
//
// This exists for tests: picking a free ephemeral port ahead of time (e.g.
// via net.Listen("tcp", "127.0.0.1:0")) and having a *separate* call to Serve
// re-bind that same port number is inherently racy — anything else on the
// machine could grab it in between. Handing the already-bound *net.Listener
// straight to ServeListener closes that race window entirely: the listener
// is accepting connections (into the kernel backlog, at least) from the
// moment net.Listen returns, before this function is even called.
func ServeListener(ctx context.Context, ln net.Listener, facade Facade) error {
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

		// Own context for this connection's lifetime, not r.Context(): after
		// Upgrade hijacks the connection, the request context's cancellation
		// semantics on disconnect are no longer guaranteed. This is what a
		// facade's background goroutines (e.g. streaming a turn's tokens)
		// watch to know when to stop.
		connCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		notifier := &wsNotifier{conn: conn}
		if na, ok := facade.(NotifierAware); ok {
			na.SetNotifier(notifier)
		}

		log.Printf("agentproxy: client connected from %s", r.RemoteAddr)
		handleConn(connCtx, conn, facade, notifier)
		log.Printf("agentproxy: client disconnected")
	})

	server := &http.Server{Handler: mux}

	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(ln) }()

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
func handleConn(ctx context.Context, conn *websocket.Conn, facade Facade, notifier *wsNotifier) {
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

		if err := notifier.writeMessage(reply); err != nil {
			log.Printf("agentproxy: write error: %v", err)
			return
		}
	}
}
