// Package agentproxytest fakes the DigitalOcean Hosted Agents session HTTP
// API (see github.com/digitalocean/godo's hosted_agents.go) behind an
// httptest.Server, so agentproxy tests can drive a real
// do.HostedAgentsService end to end without a live harness backend.
//
// Nothing outside this package's own tests (or other packages' _test.go
// files) should ever import it: it's test scaffolding, not production code,
// even though it lives in an ordinary buildable package rather than a
// _test.go file — that's what lets other packages' tests (e.g.
// internal/agentproxy/codex's) import it across package boundaries.
package agentproxytest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Event is one canned SSE event the harness streams back from
// GET /v2/agents/sessions/{id}/stream, matching the event-specific part of
// godo.HostedAgentEvent's wire shape (see HostedAgentEventKind's doc comment
// for the canonical type strings).
type Event struct {
	// Type is the wire "type" discriminator, e.g. "run.started",
	// "run.token_delta", "run.completed", "run.failed".
	Type string
	// Data is the event-specific payload. A nil Data marshals as "{}".
	Data json.RawMessage
}

// eventWire is the on-the-wire SPI canonical event envelope this harness
// emits. It mirrors godo's unexported hostedAgentEventWire field-for-field
// (confirmed against vendor/github.com/digitalocean/godo/hosted_agents.go)
// rather than importing it, since that type isn't exported.
type eventWire struct {
	EventID   string          `json:"event_id"`
	RunID     string          `json:"run_id"`
	TenantID  string          `json:"tenant_id"`
	SessionID string          `json:"session_id"`
	Timestamp string          `json:"timestamp"`
	Seq       int             `json:"seq"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
}

// Harness fakes the four Hosted Agents session routes an agentproxy test
// needs: session lookup, SSE stream, send-input, and HITL resolve. Point a
// *godo.Client at it with godo.SetBaseURL(h.Server.URL+"/") — do's
// NewHostedAgentsService wraps that client directly, so a Facade built
// against it talks to this harness exactly as it would a real backend.
type Harness struct {
	Server *httptest.Server

	mu        sync.Mutex
	sessionID string
	runID     string  // returned by the next POST .../input call
	events    []Event // streamed, in order, by the next GET .../stream call
}

// New starts the fake harness and registers its shutdown via t.Cleanup.
// sessionID is the session id every route responds under — a test's
// Facade.SessionID should match it.
func New(t *testing.T, sessionID string) *Harness {
	t.Helper()
	h := &Harness{sessionID: sessionID}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/agents/sessions/{id}", h.handleGetSession)
	mux.HandleFunc("GET /v2/agents/sessions/{id}/stream", h.handleStream)
	mux.HandleFunc("POST /v2/agents/sessions/{id}/input", h.handleInput)
	mux.HandleFunc("POST /v2/agents/sessions/{id}/hitl/{requestID}", h.handleHITL)

	h.Server = httptest.NewServer(mux)
	t.Cleanup(h.Server.Close)
	return h
}

// QueueRun arranges for the next POST .../input call to return runID, and
// for GET .../stream to then emit events (in order, tagged with runID),
// flushing after each one so a concurrent reader observes them incrementally
// rather than all at once at EOF.
//
// One (runID, events) pair is enough for a single-turn sequence test; queue
// again before the next turn/start if a future test drives more than one
// turn over the same harness.
func (h *Harness) QueueRun(runID string, events ...Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.runID = runID
	h.events = events
}

func (h *Harness) handleGetSession(w http.ResponseWriter, _ *http.Request) {
	h.mu.Lock()
	sessionID := h.sessionID
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"session":{"session_id":%q,"team_id":15726539,`+
		`"agent_kind":"AGENT_KIND_CODEX_CLI","status":"SESSION_STATUS_READY",`+
		`"created_at":"2026-01-01T00:00:00Z","last_event_at":"2026-01-01T00:00:00Z"}}`,
		sessionID)
}

func (h *Harness) handleInput(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	runID := h.runID
	h.mu.Unlock()

	if runID == "" {
		http.Error(w, "agentproxytest: no run id queued; call Harness.QueueRun first", http.StatusInternalServerError)
		return
	}

	// Best-effort decode only — the harness doesn't assert on the input
	// text; callers that care can inspect the request in a custom handler.
	var body struct {
		Text string `json:"text"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"run_id": runID})
}

func (h *Harness) handleStream(w http.ResponseWriter, _ *http.Request) {
	h.mu.Lock()
	runID := h.runID
	events := h.events
	sessionID := h.sessionID
	h.mu.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher, canFlush := w.(http.Flusher)

	for i, ev := range events {
		data := ev.Data
		if data == nil {
			data = json.RawMessage("{}")
		}
		wire := eventWire{
			EventID:   fmt.Sprintf("evt-%d", i),
			RunID:     runID,
			TenantID:  "15726539",
			SessionID: sessionID,
			Timestamp: "2026-01-01T00:00:00Z",
			Seq:       i,
			Type:      ev.Type,
			Data:      data,
		}
		encoded, err := json.Marshal(wire)
		if err != nil {
			// A test handed the harness a payload that can't marshal — a
			// programmer error in the test, not a runtime condition.
			panic(fmt.Sprintf("agentproxytest: event does not marshal to JSON: %v", err))
		}

		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, encoded)
		if canFlush {
			flusher.Flush()
		}
	}
}

func (h *Harness) handleHITL(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
