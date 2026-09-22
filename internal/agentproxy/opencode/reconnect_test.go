package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/digitalocean/doctl/internal/agentproxy/agentproxytest"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// instantReconnect removes the backoff wait for the duration of a test.
func instantReconnect(t *testing.T) {
	t.Helper()
	restore := setReconnectSleepForTest(func(ctx context.Context, _ time.Duration) bool {
		return ctx.Err() == nil
	})
	t.Cleanup(restore)
}

// warmHistory forces the facade's history cache to fill synchronously. The
// facade warms history off its first request via a background replay_only
// stream, and the harness's error/drop injection applies to ALL stream opens
// — so a test arming injection before the warm has finished races it. Warming
// first (this GET blocks on the single-flight fetch) makes the next stream
// open deterministically the live one.
func warmHistory(t *testing.T, srv *httptest.Server) {
	t.Helper()
	resp, err := http.Get(srv.URL + "/session/ses_x/message")
	require.NoError(t, err)
	resp.Body.Close()
}

// A harness stream dropped mid-turn is resumed with a replay_from cursor
// inside the same client SSE response: the turn's remaining events arrive
// exactly once, with no frame duplicated and no frame lost. Dedup is the
// replay_from contract, NOT id comparison — the harness mints random event
// ids, so any max-id ratchet fails this test loudly.
func TestStreamDropMidTurnResumesWithCursor(t *testing.T) {
	instantReconnect(t)
	srv, h := newBridgedFacade(t)
	warmHistory(t, srv)
	h.QueueRun("run-drop",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"first "}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"second "}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"third"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)
	// Drop the first connection after 3 events (stream.state + run.started +
	// one delta): the run is mid-flight, so the loop must reconnect rather
	// than end the SSE response.
	h.DropConnectionAfterEvents(3)

	resp := postPrompt(t, srv, "survive a drop")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	stream, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer stream.Body.Close()
	frames := drainFrames(t, stream.Body)

	var deltas []string
	var idles int
	for _, fr := range frames {
		switch fr.Payload.Type {
		case "message.part.delta":
			deltas = append(deltas, propsOf(t, fr)["delta"].(string))
		case "session.idle":
			idles++
		}
	}
	// Exactly once each, in order: the cursor resume neither replays the
	// pre-drop delta nor skips the post-drop ones.
	assert.Equal(t, []string{"first ", "second ", "third"}, deltas)
	assert.Equal(t, 1, idles)

	// The full text survives intact into the finalized part.
	var finalText string
	for _, fr := range frames {
		if fr.Payload.Type != "message.part.updated" {
			continue
		}
		if part, _ := propsOf(t, fr)["part"].(map[string]any); part["type"] == "text" {
			if s, _ := part["text"].(string); s != "" {
				finalText = s
			}
		}
	}
	assert.Equal(t, "first second third", finalText)
}

// A stream that ends with no turn in flight ends the SSE response instead of
// reconnecting forever — the TUI's own re-attach opens a fresh handler. (This
// is also the exit every other test's drainFrames relies on; pinned here so a
// refactor can't silently turn idle streams into retry loops.)
func TestStreamEndWithNoTurnsEndsResponse(t *testing.T) {
	instantReconnect(t)
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-done",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)
	resp := postPrompt(t, srv, "quick turn")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	done := make(chan struct{})
	go func() {
		defer close(done)
		stream, err := http.Get(srv.URL + "/global/event")
		if err == nil {
			drainFrames(t, stream.Body)
			stream.Body.Close()
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("SSE response did not end after the drained stream closed with no turns in flight")
	}
}

// A terminal stream error (the session is gone, auth failed, another consumer
// holds the slot) gives up immediately and fails the in-flight turn loudly:
// dialog dismissed, assistant message closed, session.error, idle — never a
// silent spinner.
func TestTerminalStreamErrorFailsTrackedTurns(t *testing.T) {
	instantReconnect(t)
	srv, h := newBridgedFacade(t)
	warmHistory(t, srv)
	h.QueueRun("run-term",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindHITLRequested), Data: guestAskPayload("hitl-term")},
	)
	// The queue above ends the first connection mid-turn (run never
	// completed); every reconnect attempt then gets a terminal 404.
	h.SetStreamErrorStatusAfter(1, http.StatusNotFound, 1)

	resp := postPrompt(t, srv, "doomed turn")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	stream, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer stream.Body.Close()
	frames := drainFrames(t, stream.Body)

	var types []string
	var replied, errMsg map[string]any
	for _, fr := range frames {
		types = append(types, fr.Payload.Type)
		switch fr.Payload.Type {
		case "permission.replied":
			replied = propsOf(t, fr)
		case "session.error":
			errMsg = propsOf(t, fr)
		}
	}
	require.NotNil(t, replied, "the pending permission dialog must be dismissed on give-up")
	assert.Equal(t, "reject", replied["reply"])
	require.NotNil(t, errMsg, "the client must be told the stream is gone")
	assert.Contains(t, types, "session.idle")
	// The assistant message was closed (time.completed) so the turn doesn't
	// render as in-flight forever.
	var completed bool
	for _, fr := range frames {
		if fr.Payload.Type != "message.updated" {
			continue
		}
		info, _ := propsOf(t, fr)["info"].(map[string]any)
		if info["role"] != "assistant" {
			continue
		}
		if tm, _ := info["time"].(map[string]any); tm["completed"] != nil {
			completed = true
		}
	}
	assert.True(t, completed, "the assistant message must be finalized on give-up")
}

// Transient StreamSession failures are retried up to the budget, then the
// loop gives up and ends the response. (With no turn in flight nothing is
// owed the client — the point is bounded retries, not frames.)
func TestStreamOpenRetriesAreBounded(t *testing.T) {
	instantReconnect(t)
	srv, h := newBridgedFacade(t)
	warmHistory(t, srv)
	// Every connection attempt fails with a retryable 500, more times than
	// the budget allows.
	h.SetStreamErrorStatus(http.StatusInternalServerError, maxAutoReconnectAttempts+3)

	done := make(chan struct{})
	go func() {
		defer close(done)
		stream, err := http.Get(srv.URL + "/global/event")
		if err == nil {
			drainFrames(t, stream.Body)
			stream.Body.Close()
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("event loop kept retrying past its reconnect budget")
	}
}
