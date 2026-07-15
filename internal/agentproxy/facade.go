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
