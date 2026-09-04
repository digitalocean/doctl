// Package opencode implements the local HTTP facade that lets an unmodified
// opencode CLI drive a hosted MARS session as if the session were a local
// `opencode serve` backend.
//
// Unlike codex (a JSON-RPC-over-WebSocket app-server), opencode's TUI is
// natively a remote-capable HTTP client: `opencode attach <url>` points it at
// any server speaking the opencode REST + SSE surface. The facade implements
// exactly the slice of that surface a real attach exercises — captured live
// against `opencode serve` (see TestedVersion) — and will bridge prompts and
// events to the hosted session through do.HostedAgentsService in later
// milestones. M1 scope: the TUI attaches, renders, and idles cleanly.
package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/digitalocean/doctl/do"
)

// TestedVersion is the opencode CLI version the route inventory and response
// shapes below were captured against (`opencode serve` + `opencode attach`
// through a logging relay). opencode releases near-daily and a v2 API surface
// is rolling out alongside v1, so re-capture on upgrades rather than assuming
// stability; the unhandled-route log below is the drift detector.
const TestedVersion = "1.18.25"

// Facade impersonates an opencode server for one hosted session. It is the
// http.Handler ServeHTTP-transport variant of the codex Facade: dispatch is
// by method+path instead of JSON-RPC method, and server→client traffic is a
// single SSE stream instead of WebSocket frames.
//
// One Facade serves the whole listener lifetime — HTTP has no "connection" to
// scope state to the way the WebSocket transport does. The one genuinely
// connection-like resource, the /global/event stream, guards itself with a
// single-consumer slot instead (mirroring the WS transport's one-client rule).
type Facade struct {
	// SessionID is the hosted session this facade bridges to (already
	// resolved and validated by the command layer).
	SessionID string
	// Sessions is the harness bridge. Unused in M1 (stubs only); the event
	// loop and prompt path arrive in M2.
	Sessions do.HostedAgentsService
	// Dir is the directory advertised to the TUI as the workspace root
	// (opencode renders it in the footer and scopes requests to it). The
	// hosted workspace isn't locally mounted, so this is presentational —
	// the proxy's own working directory.
	Dir string

	eventSeq atomic.Uint64
	// eventSlot: at most one /global/event consumer at a time, same rule as
	// the WebSocket transport's one-client slot and for the same reason —
	// the harness allows one attached device per session, and two TUIs
	// racing one facade would each see half the stream.
	eventSlot   sync.Mutex
	eventInUse  bool
	handlerOnce sync.Once
	mux         *http.ServeMux

	// mu guards turns, sessionCreated, and the pending-permission maps: all
	// written by request goroutines (prompt/reply handlers) and read by the
	// event loop (the SSE handler goroutine).
	mu    sync.Mutex
	turns map[string]*turnState
	// perms indexes outstanding permission asks by the minted per_ id (the
	// reply routes) and permsByHitl by the harness hitl id (the resolved
	// event). Both point at the same pendingPerm entries.
	perms       map[string]*pendingPerm
	permsByHitl map[string]*pendingPerm
	// sessionCreated flips once the client has created/prompted the bridged
	// session, so the session list grows its single entry (see
	// handleSessionList).
	sessionCreated bool

	// History cache (see history() in history.go): histMu also serves as
	// single-flight for the slow replay_only fetch.
	histMu    sync.Mutex
	hist      []historyMessage
	histValid bool
}

// ServeHTTP implements http.Handler.
func (f *Facade) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.handlerOnce.Do(func() {
		f.buildMux()
		// Warm the history cache off the first request — the TUI's
		// /global/health preflight lands well before the attach burst, so
		// the slow replay (see history()) usually finishes before the burst
		// asks for the session list or messages.
		if f.Sessions != nil {
			go func() { _, _ = f.history(context.Background()) }()
		}
	})
	f.mux.ServeHTTP(w, r)
}

// buildMux registers the attach-burst surface captured from a real
// `opencode attach` (TestedVersion): ~30 routes, almost all read-only
// bootstrap lookups. Anything not captured falls through to the unhandled
// logger — the discovery mechanism for TUI versions that ask for more, same
// role as the codex facade's "unhandled: <method>" log.
func (f *Facade) buildMux() {
	mux := http.NewServeMux()

	// Liveness preflight: `opencode attach` checks this before starting the
	// TUI. The version reported is this facade's tested-protocol version, not
	// the client's; attach does not require them to match.
	mux.HandleFunc("GET /global/health", f.json(map[string]any{
		"healthy": true, "version": TestedVersion,
	}))

	// The server→client event stream. The 1.18.x TUI subscribes to the
	// GLOBAL stream (/global/event), whose frames wrap the instance-stream
	// shape in a payload envelope: {"payload":{id,type,properties},...} —
	// not the bare {type,properties} the in-sandbox OHR adapter consumes
	// from /event. Both captured live; don't conflate them.
	mux.HandleFunc("GET /global/event", f.handleGlobalEvent)
	mux.HandleFunc("GET /event", f.handleGlobalEvent)

	// Workspace/project identity. All presentational in M1.
	mux.HandleFunc("GET /path", func(w http.ResponseWriter, r *http.Request) {
		home := f.Dir
		f.writeJSON(w, map[string]any{
			"home": home, "state": home, "config": home,
			"worktree": f.Dir, "directory": f.Dir,
		})
	})
	mux.HandleFunc("GET /project/current", f.json(map[string]any{
		"id": "global", "worktree": f.Dir, "vcs": "git",
		"time": map[string]any{"created": 0, "updated": 0}, "sandboxes": []any{},
	}))
	mux.HandleFunc("GET /project/global/directories", f.json([]any{}))
	mux.HandleFunc("GET /vcs", f.json(map[string]any{"branch": "hosted", "default_branch": nil}))

	// Config and the provider/model catalog. The facade advertises exactly
	// one synthetic provider/model pair representing the hosted session, so
	// the TUI has a selected model and never opens its own /connect flow —
	// model choice is a session-create-time decision on the hosted side, not
	// something this proxy can change.
	mux.HandleFunc("GET /config", f.json(map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"command": map[string]any{}, "plugin": []any{},
		"mode": map[string]any{}, "agent": map[string]any{},
	}))
	mux.HandleFunc("GET /config/providers", f.json(map[string]any{
		"default":   map[string]any{providerID: modelID},
		"providers": []any{catalogProvider()},
	}))
	mux.HandleFunc("GET /provider", f.json(map[string]any{
		"all":       []any{catalogProvider()},
		"connected": []any{providerID},
		"default":   map[string]any{providerID: modelID},
	}))
	mux.HandleFunc("GET /provider/auth", f.json(map[string]any{}))

	// Agents (opencode's build/plan/... modes, not MARS agents). One primary
	// agent whose permissions claim nothing: enforcement lives in the hosted
	// session's policy, and overstating it here would only mislead the TUI.
	mux.HandleFunc("GET /agent", f.json([]any{map[string]any{
		"name": "build", "description": "Hosted MARS session",
		"mode": "primary", "native": true,
		"permission": []any{map[string]any{"permission": "*", "pattern": "*", "action": "allow"}},
		"options":    map[string]any{},
	}}))

	// Empty-but-present bootstrap lookups, verbatim from the capture.
	mux.HandleFunc("GET /command", f.json([]any{}))
	mux.HandleFunc("GET /lsp", f.json([]any{}))
	mux.HandleFunc("GET /mcp", f.json(map[string]any{}))
	mux.HandleFunc("GET /formatter", f.json([]any{}))
	mux.HandleFunc("GET /session/status", f.json(map[string]any{}))
	mux.HandleFunc("GET /experimental/capabilities", f.json(map[string]any{"backgroundSubagents": false}))
	mux.HandleFunc("GET /experimental/console", f.json(map[string]any{
		"consoleManagedProviders": []any{}, "switchableOrgCount": 0,
	}))
	mux.HandleFunc("GET /experimental/resource", f.json(map[string]any{}))
	mux.HandleFunc("GET /experimental/workspace", f.json([]any{}))
	mux.HandleFunc("GET /experimental/workspace/status", f.json([]any{}))

	// The bridged session: list/create/get plus the prompt bridge (M2) and
	// history (M3, served from a replay_only stream pass). diff/todo are the
	// two lookups the TUI fires after every turn (post-idle refresh) and on
	// resume — real server returns empty arrays for a no-edits session.
	mux.HandleFunc("GET /session", f.handleSessionList)
	mux.HandleFunc("POST /session", f.handleSessionCreate)
	mux.HandleFunc("GET /session/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.writeJSON(w, f.sessionObject())
	})
	mux.HandleFunc("GET /session/{id}/message", f.handleMessageList)
	mux.HandleFunc("GET /session/{id}/diff", f.json([]any{}))
	mux.HandleFunc("GET /session/{id}/todo", f.json([]any{}))
	mux.HandleFunc("POST /session/{id}/prompt_async", f.handlePromptAsync)
	// The synchronous variant officially awaits the reply; bridging that
	// faithfully would block a request goroutine for a whole turn. Current
	// TUIs use prompt_async (M0 capture); if a client POSTs /message, treat
	// it as fire-and-forget and let the reply ride the stream — log it so a
	// client that genuinely blocks on the response is diagnosable.
	mux.HandleFunc("POST /session/{id}/message", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("agentproxy/opencode: POST message treated as prompt_async (reply rides the stream)")
		f.handlePromptAsync(w, r)
	})

	// The /api/* surface (new in recent opencode versions; sits alongside
	// the older paths above — the TUI queries both). Wrapped responses:
	// {"location": ..., "data": [...]}.
	mux.HandleFunc("GET /api/location", func(w http.ResponseWriter, r *http.Request) {
		f.writeJSON(w, f.apiLocation())
	})
	for _, p := range []string{"skill", "reference", "integration", "command", "agent"} {
		mux.HandleFunc("GET /api/"+p, func(w http.ResponseWriter, r *http.Request) {
			f.writeJSON(w, map[string]any{"location": f.apiLocation(), "data": []any{}})
		})
	}
	mux.HandleFunc("GET /api/provider", func(w http.ResponseWriter, r *http.Request) {
		f.writeJSON(w, map[string]any{"location": f.apiLocation(), "data": []any{apiProvider()}})
	})
	mux.HandleFunc("GET /api/model", func(w http.ResponseWriter, r *http.Request) {
		f.writeJSON(w, map[string]any{"location": f.apiLocation(), "data": []any{apiModel()}})
	})

	// Permission replies (M5). The TestedVersion TUI answers an ask with the
	// global route; the session-scoped route is the older equivalent (it's
	// what plano's adapter drives against a guest server) — same semantics,
	// different body key. Both resolve the hosted session's HITL request.
	mux.HandleFunc("POST /permission/{requestID}/reply", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Reply   string `json:"reply"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, fmt.Sprintf("malformed reply body: %v", err), http.StatusBadRequest)
			return
		}
		f.handlePermissionReply(w, r.PathValue("requestID"), body.Reply, body.Message)
	})
	mux.HandleFunc("POST /session/{id}/permissions/{permissionID}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Response string `json:"response"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, fmt.Sprintf("malformed reply body: %v", err), http.StatusBadRequest)
			return
		}
		f.handlePermissionReply(w, r.PathValue("permissionID"), body.Response, "")
	})

	// Share syncs conversation history to opencode's public share
	// infrastructure — never appropriate for a hosted session. Refuse
	// loudly rather than 404-ing quietly.
	mux.HandleFunc("POST /session/{id}/share", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sharing a hosted session is disabled", http.StatusForbidden)
	})

	// Everything else: honest 404 plus the drift log. opencode's TUI
	// tolerates 404s on optional lookups; anything it *requires* will show
	// up here first when a new client version asks for it.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("unhandled: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})

	f.mux = mux
}

// handleGlobalEvent serves the SSE event stream: server.connected first (the
// TUI's liveness signal), then the live harness event loop until the client
// disconnects, the harness stream ends, or the server shuts down.
func (f *Facade) handleGlobalEvent(w http.ResponseWriter, r *http.Request) {
	f.eventSlot.Lock()
	if f.eventInUse {
		f.eventSlot.Unlock()
		http.Error(w, "an event-stream client is already connected", http.StatusConflict)
		return
	}
	f.eventInUse = true
	f.eventSlot.Unlock()
	defer func() {
		f.eventSlot.Lock()
		f.eventInUse = false
		f.eventSlot.Unlock()
	}()

	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	ew := &eventWriter{f: f, w: w, fl: fl}
	if err := ew.global("server.connected", map[string]any{}); err != nil {
		return
	}

	// r.Context() is rooted in the transport's ctx (ServeHTTPListener's
	// BaseContext), so both a client disconnect and a proxy shutdown cancel
	// the harness stream — no separate signal plumbing needed.
	f.runEventLoop(r.Context(), ew)
}

// eventWriter frames global-stream SSE events. Envelope shape per the M0
// capture: data: {"payload":{"id":"evt_<n>","type":T,"properties":P}}, with
// top-level directory/project keys on directory-scoped (session) events —
// mirroring how the real global stream tags them (project.updated carries
// them; server.connected doesn't).
//
// Only the /global/event handler goroutine writes, by construction — the
// prompt and permission-reply handlers never touch the stream — so unlike the
// WS transport's wsNotifier there is no write mutex here. M5 kept the rule on
// purpose: permission.replied is emitted by the event loop when
// run.human_input_received comes back (which also covers out-of-band
// resolutions), not by the reply handler. Anything that breaks the
// single-writer property must add the mutex.
type eventWriter struct {
	f  *Facade
	w  http.ResponseWriter
	fl http.Flusher
}

func (ew *eventWriter) send(envelope map[string]any) error {
	frame, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(ew.w, "data: %s\n\n", frame); err != nil {
		return err
	}
	ew.fl.Flush()
	return nil
}

func (ew *eventWriter) payload(eventType string, properties any) map[string]any {
	return map[string]any{
		"id":         fmt.Sprintf("evt_%d", ew.f.eventSeq.Add(1)),
		"type":       eventType,
		"properties": properties,
	}
}

// global emits a server-scoped frame (server.connected).
func (ew *eventWriter) global(eventType string, properties any) error {
	return ew.send(map[string]any{"payload": ew.payload(eventType, properties)})
}

// session emits a directory-scoped frame (everything about the bridged
// session).
func (ew *eventWriter) session(eventType string, properties any) error {
	return ew.send(map[string]any{
		"directory": ew.f.Dir,
		"project":   "global",
		"payload":   ew.payload(eventType, properties),
	})
}

// json returns a handler that writes a fixed JSON body. For bodies that
// depend on request or facade state, write inline handlers instead.
func (f *Facade) json(v any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { f.writeJSON(w, v) }
}

func (f *Facade) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("agentproxy/opencode: encode response: %v", err)
	}
}

func (f *Facade) apiLocation() map[string]any {
	return map[string]any{
		"directory": f.Dir,
		"project":   map[string]any{"id": "global", "directory": f.Dir},
	}
}

// The synthetic provider/model catalog entry. The hosted session already has
// its model pinned at create time; this exists so the TUI has something to
// display and select, not to offer a real choice.
const (
	providerID   = "digitalocean"
	providerName = "DigitalOcean"
	modelID      = "hosted"
	modelName    = "Hosted Session"
)

func catalogProvider() map[string]any {
	return map[string]any{
		"id": providerID, "name": providerName, "api": nil, "env": []any{},
		"models": map[string]any{modelID: map[string]any{
			"id": modelID, "name": modelName, "providerID": providerID,
			"family": "hosted", "status": "active",
			"capabilities": map[string]any{"tools": true, "input": []any{"text"}, "output": []any{"text"}},
			"limit":        map[string]any{"context": 262144, "output": 32768},
			"options":      map[string]any{}, "variants": []any{},
		}},
	}
}

func apiProvider() map[string]any {
	return map[string]any{
		"id": providerID, "name": providerName,
		"api":     map[string]any{"type": "aisdk", "package": "@ai-sdk/openai-compatible", "url": "http://127.0.0.1"},
		"request": map[string]any{"headers": map[string]any{}, "body": map[string]any{}},
	}
}

func apiModel() map[string]any {
	return map[string]any{
		"id": modelID, "providerID": providerID, "family": "hosted", "name": modelName,
		"api":          map[string]any{"id": modelID, "type": "aisdk", "package": "@ai-sdk/openai-compatible", "url": "http://127.0.0.1"},
		"capabilities": map[string]any{"tools": true, "input": []any{"text"}, "output": []any{"text"}},
		"request":      map[string]any{"headers": map[string]any{}, "body": map[string]any{}},
		"variants":     []any{}, "time": map[string]any{"released": 0},
		"cost":   []any{map[string]any{"input": 0, "output": 0, "cache": map[string]any{"read": 0, "write": 0}}},
		"status": "active", "enabled": true,
		"limit": map[string]any{"context": 262144, "output": 32768},
	}
}
