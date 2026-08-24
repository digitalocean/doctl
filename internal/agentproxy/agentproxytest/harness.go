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
	"time"

	"github.com/digitalocean/godo"
	"github.com/google/uuid"
)

// Event is one canned SSE event the harness streams back from
// GET /v2/agents/sessions/{id}/events, matching the event-specific part of
// godo.HostedAgentEvent's wire shape (see HostedAgentEventKind's doc comment
// for the canonical type strings).
type Event struct {
	// Type is the wire "type" discriminator, e.g. "run.started",
	// "run.token_delta", "run.completed", "run.failed".
	Type string
	// Data is the event-specific payload. A nil Data marshals as "{}".
	Data json.RawMessage
	// WaitForHITL, if set, blocks this event (and every one after it) from
	// being sent until handleHITL records a resolution for this hitl_id.
	// Needed for any test where a queued event (e.g. run.tool_call_completed
	// for a gated call) must not be observed to happen before the HITL that
	// gates it is actually resolved: naively pre-queuing every event with no
	// gating, the way QueueRun otherwise works, races a Facade's
	// asynchronous approval-handling goroutine instead of modeling the real
	// harness's behavior of only continuing a gated run after the decision
	// arrives.
	WaitForHITL string
	// RunID overrides the run id this event is tagged with, for
	// QueueReplayHistory only: replayed history can span several distinct
	// past runs, unlike QueueRun's single live run sharing one runID for its
	// whole queue. Empty means "use the harness's current runID," which is
	// all QueueRun ever needs.
	RunID string
	// SourceEventType is the native event type label the in-sandbox adapter
	// recorded before canonical mapping (e.g. codex's
	// "item/agentMessage/delta"). Emitted whenever set, mirroring the real
	// SSE surface, which carries it unconditionally.
	SourceEventType string
	// SourceRaw is the exact native event bytes the adapter captured before
	// canonical mapping — for codex, one JSON-RPC frame. Only emitted when
	// the stream was opened with include_raw=true, mirroring the real
	// surface's opt-in gate ([]byte rides base64 in the JSON, matching
	// godo's wire decode).
	SourceRaw []byte

	// eventID is assigned by QueueRun/QueueReplayHistory (a random UUID,
	// like the real producers mint — see the comment at the serve loop) and
	// is stable across reconnects of the same queue, so replay_from can
	// resume mid-list the way harness-api does.
	eventID string
}

// eventWire is the on-the-wire SPI canonical event envelope this harness
// emits. It mirrors godo's unexported hostedAgentEventWire field-for-field
// (confirmed against vendor/github.com/digitalocean/godo/hosted_agents.go)
// rather than importing it, since that type isn't exported.
type eventWire struct {
	EventID         string          `json:"event_id"`
	RunID           string          `json:"run_id"`
	TenantID        string          `json:"tenant_id"`
	SessionID       string          `json:"session_id"`
	Timestamp       string          `json:"timestamp"`
	Seq             int             `json:"seq"`
	SourceEventType string          `json:"source_event_type,omitempty"`
	SourceRaw       []byte          `json:"source_raw,omitempty"`
	Type            string          `json:"type"`
	Data            json.RawMessage `json:"data"`
}

// Harness fakes the four Hosted Agents session routes an agentproxy test
// needs: session lookup, SSE stream, send-input, and HITL resolve. Point a
// *godo.Client at it with godo.SetBaseURL(h.Server.URL+"/") — do's
// NewHostedAgentsService wraps that client directly, so a Facade built
// against it talks to this harness exactly as it would a real backend.
type Harness struct {
	Server *httptest.Server

	mu                   sync.Mutex
	hangAfterEvents      bool
	sessionID            string
	runID                string        // returned by the next POST .../input call
	lastInput            *InputRequest // most recent POST .../input body, for test assertions (LastInput)
	lastRelay            []byte        // most recent POST .../request frame, for test assertions (LastRelay)
	relayReply           []byte        // answered by the next POST .../request call; empty = adapter declined
	relayStatus          int           // 0 = answer normally; nonzero = fail POST .../request with this status
	events               []Event       // streamed, in order, by the next GET .../events call
	replayEvents         []Event       // streamed instead of events when GET .../events carries replay_only=true
	streamErrorStatus    int           // 0 = serve normally; nonzero = return this HTTP status instead
	streamErrorRemaining int           // >0: decrement per hit, clearing streamErrorStatus at 0; <=0 with status set: permanent
	streamErrorSkip      int           // succeed this many opens before applying streamErrorStatus (reconnect-path tests)
	dropAfterEvents      int           // >0: end the very next stream connection after this many events (one-shot)

	// hitlCh delivers every POST .../hitl/{requestID} call this harness
	// receives, in order. A channel (rather than a "last resolution" field)
	// because ResolveHITL is typically called from a Facade's own background
	// goroutine on its own timeline — a test needs to block for it the same
	// way notifierRecorder.next blocks for a notification, not poll a field
	// that could race with that goroutine.
	hitlCh chan HITLResolution

	// hitlResolved holds one channel per hitl_id, closed by handleHITL the
	// moment a resolution for that id is received — see Event.WaitForHITL.
	// Lazily created (by hitlResolvedChan) so an event can reference a
	// hitl_id that hasn't been requested yet at QueueRun time.
	hitlResolved map[string]chan struct{}
}

// HITLResolution is one POST .../hitl/{requestID} call the harness received,
// recorded so a test can assert not just that ResolveHITL was called, but
// that it resolved the right request with the right outcome.
type HITLResolution struct {
	RequestID string
	Outcome   string
	Reason    string
	Source    string
	// SourceRaw is the client's native reply frame, when the resolver
	// answered in the agent's own protocol rather than with an outcome
	// alone. Empty for every non-native resolution.
	SourceRaw []byte
}

// New starts the fake harness and registers its shutdown via t.Cleanup.
// sessionID is the session id every route responds under — a test's
// Facade.SessionID should match it.
func New(t *testing.T, sessionID string) *Harness {
	t.Helper()
	h := &Harness{sessionID: sessionID, hitlCh: make(chan HITLResolution, 8)}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/agents/sessions/{id}", h.handleGetSession)
	// One SSE surface, matching godo's StreamSession: both the live and the
	// replay-only read go to the data plane at .../events, the latter carrying
	// replay_only=true (see QueueReplayHistory). handleStream serves both,
	// branching on that query parameter. The control plane's .../stream is
	// deliberately not registered, so a request landing there fails the test
	// instead of quietly passing against a route the client no longer uses.
	mux.HandleFunc("GET /v2/agents/sessions/{id}/events", h.handleStream)
	mux.HandleFunc("POST /v2/agents/sessions/{id}/input", h.handleInput)
	mux.HandleFunc("POST /v2/agents/sessions/{id}/request", h.handleRelay)
	mux.HandleFunc("POST /v2/agents/sessions/{id}/hitl/{requestID}", h.handleHITL)

	h.Server = httptest.NewServer(mux)
	t.Cleanup(h.Server.Close)
	return h
}

// QueueRun arranges for the next POST .../input call to return runID, and
// for GET .../events to then emit events (in order, tagged with runID),
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
	h.events = assignEventIDs(events)
}

// assignEventIDs gives every queued event a stable random-UUID event id (in
// place; the slice is returned for convenience). Stability across
// connections is what lets handleStream honor replay_from on a reconnect;
// randomness (rather than a monotonic scheme) is what keeps facade logic
// honest about id ordering — see the serve loop's comment.
func assignEventIDs(events []Event) []Event {
	for i := range events {
		if events[i].eventID == "" {
			events[i].eventID = uuid.NewString()
		}
	}
	return events
}

// QueueReplayHistory arranges for a GET .../events call carrying
// replay_only=true to return these events instead of whatever QueueRun set
// up, then end — mirrors harness-api's own handleStreamSession, which never
// continues a replay_only request into a live tail (see
// streamsvc.Service.SubscribeEvents). Each event's own RunID is used as-is
// (unlike QueueRun's events, which all share one runID), since replayed
// history can span several distinct past runs.
func (h *Harness) QueueReplayHistory(events ...Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.replayEvents = assignEventIDs(events)
}

// SetStreamErrorStatus makes GET .../events return status immediately
// (instead of opening an SSE stream) for the next `times` calls, then
// resume normal behavior (serving whatever's queued via QueueRun) —
// simulating a StreamSession failure rather than a mid-stream drop. times
// <= 0 means permanent: every call fails until this is changed again — for
// a test verifying a terminal error (401/403/404/409) makes the caller give
// up immediately, with no retries. times > 0 fails exactly that many calls
// before clearing automatically — for a test verifying reconnect-with-backoff
// actually recovers after N transient failures.
func (h *Harness) SetStreamErrorStatus(status, times int) {
	h.SetStreamErrorStatusAfter(0, status, times)
}

// SetStreamErrorStatusAfter is SetStreamErrorStatus, but the first `skip`
// stream opens succeed normally before the injected failures begin. Needed
// once turn/start opens the stream synchronously before SendInput
// (ensureEventLoop): a reconnect-path test must let that first attach
// succeed, then fail subsequent opens, rather than failing turn/start itself.
func (h *Harness) SetStreamErrorStatusAfter(skip, status, times int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.streamErrorSkip = skip
	h.streamErrorStatus = status
	h.streamErrorRemaining = times
}

// DropConnectionAfterEvents makes the very next GET .../events call return
// after sending only the first n queued events (a clean end, no error —
// exactly how a genuine mid-stream drop/idle-timeout looks from the
// client's side) instead of the whole list, then resets to 0 so any later
// connection serves normally. One-shot: simulates a real drop-and-resume for
// a test verifying a reconnect actually resumes (via Last-Event-ID) and dedups
// whatever prefix gets redelivered, rather than only ever seeing a stream
// that's already run to completion.
func (h *Harness) DropConnectionAfterEvents(n int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.dropAfterEvents = n
}

// hitlResolvedChan returns (lazily creating) the channel that handleHITL
// closes once a resolution for hitlID is received — see Event.WaitForHITL.
func (h *Harness) hitlResolvedChan(hitlID string) chan struct{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hitlResolved == nil {
		h.hitlResolved = make(map[string]chan struct{})
	}
	ch, ok := h.hitlResolved[hitlID]
	if !ok {
		ch = make(chan struct{})
		h.hitlResolved[hitlID] = ch
	}
	return ch
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

	// Best-effort decode — the harness never fails a test on a bad body,
	// but it records what it saw so tests can assert on the input request
	// (LastInput), e.g. that the v2 raw turn/start frame rode along.
	var body InputRequest
	_ = json.NewDecoder(r.Body).Decode(&body)
	h.mu.Lock()
	h.lastInput = &body
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"run_id": runID})
}

// InputRequest is the POST .../input body as this harness decoded it —
// the same wire shape godo.HostedAgentSendInputRequest marshals to.
type InputRequest struct {
	Text string `json:"text"`
	// SourceRaw is the client's native protocol frame, when the caller
	// attached one (v2 inbound raw passthrough). encoding/json handles the
	// wire base64 for []byte, so this holds the decoded frame bytes.
	SourceRaw []byte `json:"source_raw"`
}

// LastInput returns the most recent POST .../input body this harness
// received, or nil if none arrived yet.
func (h *Harness) LastInput() *InputRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastInput
}

// QueueRelayReply arranges for the next POST .../request call to answer with
// replyFrame — the agent's native reply, as the in-sandbox adapter would have
// produced it. An empty frame reproduces the adapter *declining* the method,
// which is a successful call with nothing in it, not an error.
func (h *Harness) QueueRelayReply(replyFrame []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.relayReply = replyFrame
	h.relayStatus = 0
}

// QueueRelayError arranges for the next POST .../request call to fail with an
// HTTP status, covering the transport-failure half of the relay contract
// (session gone, agent unreachable) as opposed to a protocol-level error,
// which arrives as a normal reply frame via QueueRelayReply.
func (h *Harness) QueueRelayError(status int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.relayStatus = status
}

// LastRelay returns the most recent POST .../request frame this harness
// received, or nil if none arrived yet.
func (h *Harness) LastRelay() []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastRelay
}

func (h *Harness) handleRelay(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SourceRaw []byte `json:"source_raw"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	h.mu.Lock()
	h.lastRelay = body.SourceRaw
	status, reply := h.relayStatus, h.relayReply
	h.mu.Unlock()

	if status != 0 {
		http.Error(w, "agentproxytest: queued relay error", status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// source_raw is omitempty on the wire, so a declined relay is a 200 with
	// no frame at all — exactly what the real API sends.
	_ = json.NewEncoder(w).Encode(struct {
		SourceRaw []byte `json:"source_raw,omitempty"`
	}{SourceRaw: reply})
}

func (h *Harness) handleStream(w http.ResponseWriter, r *http.Request) {
	replayOnly := r.URL.Query().Get("replay_only") == "true"
	// Resume delivery strictly after the named event, matching godo's
	// StreamSession: live reconnects send Last-Event-ID, replay-only reads
	// send replay_from as a query parameter. An unknown id serves the whole
	// queue — matching a real stream whose retention no longer covers the
	// cursor.
	replayFrom := r.URL.Query().Get("replay_from")
	if replayFrom == "" {
		replayFrom = r.Header.Get("Last-Event-ID")
	}
	// Raw source bytes are opt-in per connection (they fatten every event);
	// without the flag a queued Event's SourceRaw is withheld, exactly like
	// the real surface — so a test can prove the canonical fallback engages
	// when the server has raw bytes but the client didn't ask.
	includeRaw := r.URL.Query().Get("include_raw") == "true"

	h.mu.Lock()
	runID := h.runID
	events := h.events
	if replayOnly {
		events = h.replayEvents
	}
	sessionID := h.sessionID
	hang := h.hangAfterEvents
	errStatus := h.streamErrorStatus
	if h.streamErrorSkip > 0 {
		h.streamErrorSkip--
		errStatus = 0 // this open is exempt; failures start on a later call
	} else if errStatus != 0 && h.streamErrorRemaining > 0 {
		h.streamErrorRemaining--
		if h.streamErrorRemaining == 0 {
			h.streamErrorStatus = 0
		}
	}
	dropAfter := h.dropAfterEvents
	h.dropAfterEvents = 0
	h.mu.Unlock()

	if replayFrom != "" {
		for i, ev := range events {
			if ev.eventID == replayFrom {
				events = events[i+1:]
				break
			}
		}
	}

	if errStatus != 0 {
		http.Error(w, "agentproxytest: simulated stream error", errStatus)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	flusher, canFlush := w.(http.Flusher)

	// The data plane opens every stream with a stream.state control frame. It
	// belongs to no run, so consumers must skip it rather than mistake it for
	// session activity — emit it here so tests exercise that.
	streamState, err := json.Marshal(eventWire{
		TenantID:  "15726539",
		SessionID: sessionID,
		Timestamp: "2026-01-01T00:00:00Z",
		Type:      string(godo.HostedAgentEventKindStreamState),
		Data:      json.RawMessage(`{"state":"live","cursor":""}`),
	})
	if err != nil {
		panic(fmt.Sprintf("agentproxytest: stream.state does not marshal to JSON: %v", err))
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", godo.HostedAgentEventKindStreamState, streamState)
	if canFlush {
		flusher.Flush()
	}

	for i, ev := range events {
		if dropAfter > 0 && i >= dropAfter {
			return // simulate a clean mid-stream drop
		}

		if ev.WaitForHITL != "" {
			<-h.hitlResolvedChan(ev.WaitForHITL)
		}

		data := ev.Data
		if data == nil {
			data = json.RawMessage("{}")
		}
		evRunID := runID
		if ev.RunID != "" {
			evRunID = ev.RunID
		}
		// Random UUIDs (assigned at queue time — see assignEventIDs), like
		// the real producers: the guest harness mints UUIDv4s, and
		// harness-api's ULIDs use fresh randomness per id, not
		// ulid.Monotonic — so ids do NOT sort in emission order. Any facade
		// logic that assumes ordinal event ids (a past, shipped bug:
		// drainStream's `<= cursor` skip) fails loudly under this fixture
		// instead of passing on synthetic monotonic ids.
		wire := eventWire{
			EventID:         ev.eventID,
			RunID:           evRunID,
			TenantID:        "15726539",
			SessionID:       sessionID,
			Timestamp:       "2026-01-01T00:00:00Z",
			Seq:             i,
			SourceEventType: ev.SourceEventType,
			Type:            ev.Type,
			Data:            data,
		}
		if includeRaw {
			wire.SourceRaw = ev.SourceRaw
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

	if replayOnly {
		return // matches harness-api: replay_only never continues to a live tail
	}

	if hang {
		// Block on the request's own context instead of returning — lets a
		// test prove that canceling a Facade's connection context actually
		// aborts the in-flight StreamSession call, rather than the fake
		// stream just naturally running out of queued events regardless of
		// whether anything was ever canceled (see HangStreamAfterEvents).
		<-r.Context().Done()
	}
}

// HangStreamAfterEvents controls whether handleStream blocks on the
// request's context after sending all queued events instead of returning
// immediately once they're exhausted (the default). Needed for tests that
// must distinguish "the stream ended because ctx was canceled" from "the
// stream ended because it ran out of pre-queued events" — the latter would
// happen regardless of whether a connection-lifecycle fix actually works.
func (h *Harness) HangStreamAfterEvents(hang bool) {
	h.mu.Lock()
	h.hangAfterEvents = hang
	h.mu.Unlock()
}

func (h *Harness) handleHITL(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Outcome   string `json:"outcome"`
		Reason    string `json:"reason"`
		Source    string `json:"source"`
		SourceRaw []byte `json:"source_raw"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	requestID := r.PathValue("requestID")

	select {
	case h.hitlCh <- HITLResolution{
		RequestID: requestID,
		Outcome:   body.Outcome,
		Reason:    body.Reason,
		Source:    body.Source,
		SourceRaw: body.SourceRaw,
	}:
	default:
		// Buffer full: a test that cares should be draining via
		// NextHITLResolution already. Don't block the HTTP handler on a test
		// that forgot to.
	}

	// Release any queued event(s) waiting on this specific resolution (see
	// Event.WaitForHITL) — safe to close even if nothing is currently
	// waiting; hitlResolvedChan lazily creates it either way, and a channel
	// close is a no-op wait for anyone who checks it later.
	close(h.hitlResolvedChan(requestID))

	w.WriteHeader(http.StatusNoContent)
}

// NextHITLResolution blocks (up to timeout) for the next POST .../hitl call
// the harness received, failing the test rather than hanging the suite if
// none arrives — same discipline as notifierRecorder.next in the codex
// package's tests.
func (h *Harness) NextHITLResolution(t *testing.T, timeout time.Duration) HITLResolution {
	t.Helper()
	select {
	case res := <-h.hitlCh:
		return res
	case <-time.After(timeout):
		t.Fatal("agentproxytest: timed out waiting for a HITL resolution")
		return HITLResolution{}
	}
}

// ExpectNoHITLResolution asserts no POST .../hitl call arrives within wait —
// used by --replay tests to confirm historical HITLs are not re-resolved.
func (h *Harness) ExpectNoHITLResolution(t *testing.T, wait time.Duration) {
	t.Helper()
	select {
	case res := <-h.hitlCh:
		t.Fatalf("agentproxytest: expected no HITL resolution, got %+v", res)
	case <-time.After(wait):
	}
}
