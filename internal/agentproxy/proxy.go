package agentproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// rpcMessage is a JSON-RPC 2.0 envelope, minus the "jsonrpc" field itself —
// the real codex app-server protocol omits "jsonrpc":"2.0" on the wire
// (confirmed by the protocol capture), and codex --remote doesn't require it
// either, so the proxy matches what's actually sent instead of adding a
// field nothing checks.
type rpcMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result any             `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return allowedProxyOrigin(r.Header.Get("Origin"))
	},
}

// allowedProxyOrigin reports whether a WebSocket Origin is acceptable for
// this loopback-only proxy. Empty Origin (websocat, codex --remote) and
// loopback Origins are allowed; any other present Origin is rejected so a
// browser tab on an unrelated site cannot drive the session.
func allowedProxyOrigin(origin string) bool {
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

// wsNotifier serializes every write to a WebSocket connection — replies from
// the main read/dispatch loop and asynchronous notifications from a facade's
// background goroutines alike — behind one mutex. gorilla/websocket conns
// aren't safe for concurrent writers; once a facade can push notifications on
// its own timeline (Notifier), there are always at least two potential
// writers, so this is required, not defensive.
//
// pending tracks this connection's own in-flight server-initiated requests
// (see Request), keyed by the JSON encoding of the id this facade minted for
// each one — handleConn's read loop checks incoming frames against this map
// (see deliverReply) before ever reaching facade.Dispatch.
type wsNotifier struct {
	mu      sync.Mutex
	conn    *websocket.Conn
	nextID  uint64
	pending map[string]chan rpcMessage
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

// Request implements agentproxy.Notifier: mint an id this connection hasn't
// used before, send it as a request, and block until deliverReply routes a
// matching reply back (or ctx is done).
//
// Ids are prefixed "srv-" to keep this side's ids out of the client's own id
// namespace. Nothing on the wire requires the two spaces to be disjoint —
// each id is just an opaque correlation token — but a distinct prefix makes
// a collision with a client-minted id (e.g. "1", "2", ...) impossible rather
// than merely unlikely.
func (w *wsNotifier) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	w.mu.Lock()
	w.nextID++
	idJSON, err := json.Marshal(fmt.Sprintf("srv-%d", w.nextID))
	if err != nil {
		w.mu.Unlock()
		return nil, err
	}
	key := string(idJSON)
	if w.pending == nil {
		w.pending = make(map[string]chan rpcMessage)
	}
	replyCh := make(chan rpcMessage, 1)
	w.pending[key] = replyCh
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		delete(w.pending, key)
		w.mu.Unlock()
	}()

	if err := w.writeMessage(rpcMessage{ID: idJSON, Method: method, Params: mustRawMessage(params)}); err != nil {
		return nil, err
	}

	select {
	case reply := <-replyCh:
		if reply.Error != nil {
			return nil, reply.Error
		}
		raw, err := json.Marshal(reply.Result)
		if err != nil {
			return nil, fmt.Errorf("agentproxy: reply result did not remarshal: %w", err)
		}
		return raw, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// deliverReply routes an incoming frame that matches one of this
// connection's own pending server-initiated requests (see Request) to
// whoever's waiting on it. Returns false if the id doesn't match anything
// pending — e.g. the reply arrived after Request already gave up on ctx
// cancellation — so handleConn can log an orphaned reply instead of silently
// dropping it.
//
// The send into the reply channel is non-blocking: the channel is buffer-1
// and Request is the only consumer. A second frame with the same id before
// Request wakes and deletes the pending entry would otherwise block forever
// inside handleConn's read loop and stop the connection from being read at
// all. Logging the duplicate and moving on keeps a misbehaving client from
// wedging the pump.
func (w *wsNotifier) deliverReply(msg rpcMessage) bool {
	key := string(msg.ID)
	w.mu.Lock()
	ch, ok := w.pending[key]
	w.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case ch <- msg:
	default:
		log.Printf("agentproxy: duplicate reply for id %s, ignoring", key)
	}
	return true
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
// machine anyway) — and speaks newFacade's JSON-RPC protocol to exactly one
// connected client at a time, until ctx is canceled.
func Serve(ctx context.Context, port int, newFacade func() Facade) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	return ServeListener(ctx, ln, newFacade)
}

// ServeListener is Serve, except it speaks newFacade's protocol over an
// already-bound listener instead of binding one from a port number itself.
//
// This exists for tests: picking a free ephemeral port ahead of time (e.g.
// via net.Listen("tcp", "127.0.0.1:0")) and having a *separate* call to Serve
// re-bind that same port number is inherently racy — anything else on the
// machine could grab it in between. Handing the already-bound *net.Listener
// straight to ServeListener closes that race window entirely: the listener
// is accepting connections (into the kernel backlog, at least) from the
// moment net.Listen returns, before this function is even called.
//
// newFacade is called once per accepted connection, not once for the whole
// listener: a facade like codex's carries per-connection state (in-flight
// turns, whether its event loop is running, the connection's Notifier), and
// only one client connects at a time anyway (see the slot below), so reusing
// one Facade instance across a disconnect/reconnect would leak that state
// into the new connection — notifications from a still-unwinding previous
// connection's background goroutine could even land on the new socket via a
// shared notifier. A fresh Facade per connection starts clean and makes the
// previous one's goroutines (if still winding down) entirely self-contained.
func ServeListener(ctx context.Context, ln net.Listener, newFacade func() Facade) error {
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

		// Derived from ServeListener's ctx, not r.Context() and not
		// context.Background(): after Upgrade hijacks the connection, the
		// request context's cancellation semantics on disconnect are no
		// longer guaranteed, so this can't be r.Context(). It also can't be
		// rooted in Background() — that was a real bug (found live): on
		// shutdown, http.Server.Shutdown returns quickly since it doesn't
		// track hijacked connections, but nothing ever cancels a
		// Background()-rooted connCtx, so a facade's background goroutines
		// (e.g. runEventLoop) never get a stop signal from server shutdown at
		// all, only from the connection's own handler returning normally.
		// Deriving from ctx here means canceling the server's ctx
		// deterministically stops every live connection's goroutines too.
		connCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		facade := newFacade()
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

		// A frame with an id but no method is a reply to one of this
		// connection's own server-initiated requests (see wsNotifier.Request),
		// not a new client request/notification — route it there instead of
		// through facade.Dispatch, which has no method to switch on for an
		// empty string and would otherwise answer it with a bogus
		// "unhandled: " error reply, corrupting the exchange it's actually a
		// reply to.
		if msg.Method == "" && len(msg.ID) > 0 {
			if !notifier.deliverReply(msg) {
				log.Printf("agentproxy: reply with unrecognized id %s, dropping", msg.ID)
			}
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
		// Only after a successful Dispatch — matches unit-test dispatchCtx
		// and AfterReply's contract (acknowledged result, not an error reply).
		if dispatchErr == nil {
			if ar, ok := facade.(AfterReply); ok {
				ar.AfterReply(ctx, msg.Method)
			}
		}
	}
}
