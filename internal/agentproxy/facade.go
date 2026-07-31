// Package agentproxy implements the local WebSocket facade that lets an
// unmodified coding-agent CLI (codex today, others later) drive a hosted MARS
// session as if the session were its own local backend.
package agentproxy

import (
	"context"
	"encoding/json"
)

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string { return e.Message }

// ErrMethodNotFound is returned by a Facade for any method it doesn't (yet)
// implement. The bridge logs it as "unhandled: <method>" and, for requests
// (a message with an id), turns it into a JSON-RPC error response — so the
// client never hangs waiting on a reply that was logged-and-dropped instead
// of sent.
var ErrMethodNotFound = &RPCError{Code: -32601, Message: "method not found"}

// Facade is the per-agent seam start-proxy dispatches JSON-RPC messages to.
// One implementation per --type (v1: codex; future: claude-code, opencode).
//
// Dispatch handles a single JSON-RPC message, request or notification alike;
// it doesn't need to know which one it's looking at. The bridge is the one
// that knows: a message with a non-nil id is a request and its (result, err)
// becomes a reply; a message with a nil id is a notification and (result,
// err) only affects logging, never the wire.
type Facade interface {
	Dispatch(ctx context.Context, method string, params json.RawMessage) (result any, err error)
}

// Notifier lets a Facade push a server-initiated JSON-RPC notification to the
// connected client at any time — not just synchronously in reply to a
// request. Needed for events that arrive asynchronously from the harness
// (streamed tokens, turn completion) on their own timeline, well after the
// request that kicked off the turn has already been answered.
type Notifier interface {
	Notify(method string, params any) error

	// Request sends a server-initiated JSON-RPC request and blocks until the
	// client replies or ctx is done. This is the reverse of Notify: some
	// codex methods (the item/*/requestApproval family) need an answer from
	// the client, not just to inform it of something. The bridge correlates
	// the reply by a synthesized request id — a Facade never sees or manages
	// that id itself.
	Request(ctx context.Context, method string, params any) (result json.RawMessage, err error)
}

// NotifierAware is implemented by facades that need to push asynchronous
// notifications rather than only answer synchronous requests. The bridge
// calls SetNotifier once per connection, before handing it any messages, so
// Dispatch can stash it and use it later from a background goroutine.
type NotifierAware interface {
	SetNotifier(Notifier)
}

// AfterReply is optionally implemented by a Facade that wants a hook after
// handleConn has successfully written a request's JSON-RPC reply. codex uses
// this to start --replay only once thread/start|resume has been acknowledged,
// so a replayed turn/started cannot race onto the wire before the thread it
// refers to.
type AfterReply interface {
	AfterReply(ctx context.Context, method string)
}
