package opencode

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl/internal/agentproxy/agentproxytest"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// frameStream reads global-stream frames one at a time — needed for the
// permission tests, where the harness gates later events on the resolution
// (Event.WaitForHITL), so the stream cannot be drained to EOF before the test
// replies mid-stream.
type frameStream struct {
	t  *testing.T
	sc *bufio.Scanner
}

func openFrameStream(t *testing.T, srv *httptest.Server) *frameStream {
	t.Helper()
	resp, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	return &frameStream{t: t, sc: sc}
}

// until reads frames until one of the wanted type arrives, failing the test
// if the stream ends first.
func (fs *frameStream) until(eventType string) frame {
	fs.t.Helper()
	for fs.sc.Scan() {
		line := fs.sc.Bytes()
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		var fr frame
		require.NoError(fs.t, json.Unmarshal(bytes.TrimPrefix(line, []byte("data: ")), &fr))
		if fr.Payload.Type == eventType {
			return fr
		}
	}
	fs.t.Fatalf("stream ended before a %q frame arrived", eventType)
	return frame{}
}

// guestAskPayload is a canonical run.human_input_requested data blob shaped
// exactly like plano's opencode adapter emits it: the guest's own
// permission.asked properties, id redacted, guest session id intact.
func guestAskPayload(hitlID string) json.RawMessage {
	return json.RawMessage(`{"hitl_id":"` + hitlID + `","payload":{
		"sessionID":"ses_guestguestguest",
		"permission":"bash",
		"patterns":["echo m5"],
		"metadata":{"command":"echo m5"},
		"always":["echo *"],
		"tool":{"messageID":"msg_guestmsgid","callID":"call_abc123"}
	}}`)
}

// The M5 acceptance path: an ask-gated tool call pops permission.asked with
// the facade's ids, the TUI's reply resolves the HITL, permission.replied
// reconciles, and the gated tool then runs to completion.
func TestPermissionRoundTrip(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-perm",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindHITLRequested), Data: guestAskPayload("hitl-1")},
		// Everything past here models "the run continued because the decision
		// arrived" — gated on the actual resolution.
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindHITLResolved), WaitForHITL: "hitl-1", Data: json.RawMessage(`{"hitl_id":"hitl-1","outcome":1,"actor":"user:1"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindToolCallStarted), Data: json.RawMessage(`{"tool_call_id":"prt_05d38cd67001perm","name":"bash","input":{"command":"echo m5"}}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindToolCallCompleted), Data: json.RawMessage(`{"tool_call_id":"prt_05d38cd67001perm","ok":true,"summary":"m5\n"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	resp := postPrompt(t, srv, "run echo m5")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	fs := openFrameStream(t, srv)

	// The assistant message must precede the ask (the ask's tool reference
	// anchors to it).
	asstFrame := fs.until("message.updated")
	for {
		if info, _ := propsOf(t, asstFrame)["info"].(map[string]any); info["role"] == "assistant" {
			break
		}
		asstFrame = fs.until("message.updated")
	}
	asstID := propsOf(t, asstFrame)["info"].(map[string]any)["id"].(string)

	ask := propsOf(t, fs.until("permission.asked"))
	perID, ok := ask["id"].(string)
	require.True(t, ok)
	assert.Regexp(t, `^per_[0-9a-f]{12}`, perID)
	// The guest's session id must be rewritten to the facade's — the TUI
	// filters events by session and would drop the dialog otherwise.
	assert.Equal(t, "ses_"+ocID("", testSessionID), ask["sessionID"])
	assert.Equal(t, "bash", ask["permission"])
	assert.Equal(t, []any{"echo m5"}, ask["patterns"])
	assert.Equal(t, []any{"echo *"}, ask["always"])
	assert.Equal(t, map[string]any{"command": "echo m5"}, ask["metadata"])
	// The guest's message id would dangle; the tool reference is remapped to
	// the facade's assistant message.
	tool, ok := ask["tool"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, asstID, tool["messageID"])
	assert.Equal(t, "call_abc123", tool["callID"])

	// Reply the way the TestedVersion TUI does (captured live): the global
	// route, body {"reply":...}, response `true`.
	replyResp, err := http.Post(srv.URL+"/permission/"+perID+"/reply", "application/json",
		strings.NewReader(`{"reply":"once","message":"looks safe"}`))
	require.NoError(t, err)
	defer replyResp.Body.Close()
	assert.Equal(t, http.StatusOK, replyResp.StatusCode)
	var replyBody bool
	require.NoError(t, json.NewDecoder(replyResp.Body).Decode(&replyBody))
	assert.True(t, replyBody)

	res := h.NextHITLResolution(t, 5*time.Second)
	assert.Equal(t, "hitl-1", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeApprove), res.Outcome)
	assert.Equal(t, string(godo.HostedAgentResolutionSourceInlineKeystroke), res.Source)
	assert.Equal(t, "looks safe", res.Reason)

	replied := propsOf(t, fs.until("permission.replied"))
	assert.Equal(t, perID, replied["requestID"])
	assert.Equal(t, "once", replied["reply"])

	// The gated tool call then flows normally.
	toolFrame := propsOf(t, fs.until("message.part.updated"))
	part, _ := toolFrame["part"].(map[string]any)
	assert.Equal(t, "tool", part["type"])
	fs.until("session.idle")
}

// A reject through the older session-scoped route ({"response":...}) resolves
// the HITL as a rejection.
func TestPermissionRejectViaSessionRoute(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-rej",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindHITLRequested), Data: guestAskPayload("hitl-2")},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindHITLResolved), WaitForHITL: "hitl-2", Data: json.RawMessage(`{"hitl_id":"hitl-2","outcome":2}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	resp := postPrompt(t, srv, "run something gated")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	fs := openFrameStream(t, srv)
	ask := propsOf(t, fs.until("permission.asked"))
	perID := ask["id"].(string)

	replyResp, err := http.Post(srv.URL+"/session/ses_x/permissions/"+perID, "application/json",
		strings.NewReader(`{"response":"reject"}`))
	require.NoError(t, err)
	replyResp.Body.Close()
	assert.Equal(t, http.StatusOK, replyResp.StatusCode)

	res := h.NextHITLResolution(t, 5*time.Second)
	assert.Equal(t, "hitl-2", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeReject), res.Outcome)

	replied := propsOf(t, fs.until("permission.replied"))
	assert.Equal(t, perID, replied["requestID"])
	assert.Equal(t, "reject", replied["reply"])
}

// "always" resolves as a plain approve (the canonical outcome enum has no
// sticky approval — a documented fidelity loss) but the broadcast echoes the
// client's actual choice so the TUI reconciles what it sent.
func TestAlwaysReplyDegradesToApprove(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-alw",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindHITLRequested), Data: guestAskPayload("hitl-3")},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindHITLResolved), WaitForHITL: "hitl-3", Data: json.RawMessage(`{"hitl_id":"hitl-3","outcome":1}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	resp := postPrompt(t, srv, "run gated again")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	fs := openFrameStream(t, srv)
	perID := propsOf(t, fs.until("permission.asked"))["id"].(string)

	replyResp, err := http.Post(srv.URL+"/permission/"+perID+"/reply", "application/json",
		strings.NewReader(`{"reply":"always"}`))
	require.NoError(t, err)
	replyResp.Body.Close()

	res := h.NextHITLResolution(t, 5*time.Second)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeApprove), res.Outcome)

	replied := propsOf(t, fs.until("permission.replied"))
	assert.Equal(t, "always", replied["reply"])
}

// Question-kind HITLs (the guest's question.asked) have no opencode TUI
// surface — they are auto-rejected with a reason instead of hanging the run,
// and nothing is shown to the client.
func TestQuestionHITLAutoRejected(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-q",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindHITLRequested), Data: json.RawMessage(`{"hitl_id":"hitl-q","payload":{"category":"question","question":"which color?","options":["red","blue"]}}`)},
		// The run only ends once the auto-reject actually lands.
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), WaitForHITL: "hitl-q"},
	)

	resp := postPrompt(t, srv, "ask me something")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	stream, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer stream.Body.Close()

	res := h.NextHITLResolution(t, 5*time.Second)
	assert.Equal(t, "hitl-q", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeReject), res.Outcome)
	assert.Equal(t, string(godo.HostedAgentResolutionSourceOutOfBand), res.Source)
	assert.Contains(t, res.Reason, "not supported")

	for _, fr := range drainFrames(t, stream.Body) {
		assert.NotEqual(t, "permission.asked", fr.Payload.Type)
	}
}

// A run that dies with an ask still pending dismisses the dialog — a
// permission.asked whose permission.replied never arrives stays on screen
// forever.
func TestPendingPermissionDismissedOnRunEnd(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-die",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindHITLRequested), Data: guestAskPayload("hitl-4")},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunFailed), Data: json.RawMessage(`{"message":"session died"}`)},
	)

	resp := postPrompt(t, srv, "doomed ask")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	stream, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer stream.Body.Close()
	frames := drainFrames(t, stream.Body)

	var perID string
	var replied map[string]any
	for _, fr := range frames {
		switch fr.Payload.Type {
		case "permission.asked":
			perID, _ = propsOf(t, fr)["id"].(string)
		case "permission.replied":
			replied = propsOf(t, fr)
		}
	}
	require.NotEmpty(t, perID)
	require.NotNil(t, replied, "the pending ask must be dismissed when the run ends")
	assert.Equal(t, perID, replied["requestID"])
	assert.Equal(t, "reject", replied["reply"])
}

// Replies for unknown ids and malformed reply values fail without touching
// the harness.
func TestPermissionReplyValidation(t *testing.T) {
	srv, h := newBridgedFacade(t)

	resp, err := http.Post(srv.URL+"/permission/per_doesnotexist/reply", "application/json",
		strings.NewReader(`{"reply":"once"}`))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	resp, err = http.Post(srv.URL+"/permission/per_doesnotexist/reply", "application/json",
		strings.NewReader(`{"reply":"maybe"}`))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	h.ExpectNoHITLResolution(t, 100*time.Millisecond)
}
