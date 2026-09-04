package opencode

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/agentproxy/agentproxytest"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSessionID = "11111111-2222-3333-4444-555555555555"

// newBridgedFacade wires a Facade against a fresh agentproxytest.Harness (a
// real do.HostedAgentsService over httptest), plus an httptest server for
// the facade's own client-facing surface.
func newBridgedFacade(t *testing.T) (*httptest.Server, *agentproxytest.Harness) {
	t.Helper()
	h := agentproxytest.New(t, testSessionID)
	client, err := godo.New(nil, godo.SetBaseURL(h.Server.URL+"/"))
	require.NoError(t, err)
	f := &Facade{SessionID: testSessionID, Sessions: do.NewHostedAgentsService(client), Dir: "/tmp/ws"}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return srv, h
}

func postPrompt(t *testing.T, srv *httptest.Server, text string) *http.Response {
	t.Helper()
	body := `{"parts":[{"type":"text","text":` + string(mustJSON(t, text)) + `}]}`
	resp, err := http.Post(srv.URL+"/session/ses_x/prompt_async", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// frame is one decoded global-stream SSE envelope.
type frame struct {
	Directory string `json:"directory"`
	Project   string `json:"project"`
	Payload   struct {
		ID         string          `json:"id"`
		Type       string          `json:"type"`
		Properties json.RawMessage `json:"properties"`
	} `json:"payload"`
}

// drainFrames reads SSE frames until the stream closes (the harness closes
// its stream after serving the queued events, which ends the facade's SSE
// response) and returns them decoded.
func drainFrames(t *testing.T, body io.Reader) []frame {
	t.Helper()
	var frames []frame
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		var fr frame
		require.NoError(t, json.Unmarshal(bytes.TrimPrefix(line, []byte("data: ")), &fr))
		frames = append(frames, fr)
	}
	return frames
}

func propsOf(t *testing.T, fr frame) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(fr.Payload.Properties, &m))
	return m
}

func TestPromptAsyncBridgesToSendInput(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-1")

	resp := postPrompt(t, srv, "hello hosted agent")
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	input := h.LastInput()
	require.NotNil(t, input)
	assert.Equal(t, "hello hosted agent", input.Text)
}

func TestPromptWithoutTextIs400(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-1")

	body := `{"parts":[{"type":"file","text":""}]}`
	resp, err := http.Post(srv.URL+"/session/ses_x/prompt_async", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Nil(t, h.LastInput())
}

// The M2 acceptance path: a prompt's turn streams back as the opencode
// frame sequence message.updated → part deltas → finalized text part →
// session.status idle, with ids stamped consistently.
func TestTurnStreamsAsOpencodeFrames(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"thinking...","is_reasoning":true}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"Par","is_reasoning":false}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"is","is_reasoning":false}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	resp := postPrompt(t, srv, "capital of France?")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	stream, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer stream.Body.Close()
	frames := drainFrames(t, stream.Body)

	var types []string
	for _, fr := range frames {
		types = append(types, fr.Payload.Type)
	}
	// The ground-truth sequence from a real `opencode serve` turn: user echo
	// (message + part), busy, assistant message, part-create before each
	// part's first delta, finalized text part, then idle status + idle event.
	require.Equal(t, []string{
		"server.connected",
		"message.updated",      // user echo
		"message.part.updated", // user prompt part
		"session.status",       // busy
		"message.updated",      // assistant
		"message.part.updated", // reasoning part created
		"message.part.delta",   // reasoning
		"message.part.updated", // text part created
		"message.part.delta",   // "Par"
		"message.part.delta",   // "is"
		"message.part.updated", // text part finalized
		"message.part.updated", // step-finish (tokens/cost)
		"message.updated",      // assistant finalized (time.completed)
		"session.status",       // idle
		"session.idle",
	}, types)

	wantSession := "ses_" + strings.ReplaceAll(testSessionID, "-", "")

	// server.connected is server-scoped; session frames carry directory and
	// a top-level properties.sessionID.
	assert.Empty(t, frames[0].Directory)
	for _, fr := range frames[1:] {
		assert.Equal(t, "/tmp/ws", fr.Directory, fr.Payload.Type)
		assert.Equal(t, wantSession, propsOf(t, fr)["sessionID"], fr.Payload.Type)
	}

	userMsg := propsOf(t, frames[1])
	userInfo, ok := userMsg["info"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "user", userInfo["role"])

	userPart := propsOf(t, frames[2])
	upart, ok := userPart["part"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "capital of France?", upart["text"])
	assert.Equal(t, userInfo["id"], upart["messageID"])

	busy := propsOf(t, frames[3])
	assert.Equal(t, map[string]any{"type": "busy"}, busy["status"])

	msg := propsOf(t, frames[4])
	info, ok := msg["info"].(map[string]any)
	require.True(t, ok)
	wantMsg, ok := info["id"].(string)
	require.True(t, ok)
	assert.Equal(t, "assistant", info["role"])
	assert.Equal(t, userInfo["id"], info["parentID"])
	// Ids must sort chronologically as strings — the TUI orders messages by
	// id, and a random-prefixed assistant id rendered the answer ABOVE the
	// user's prompt (found live). User id strictly before assistant id.
	userID, ok := userInfo["id"].(string)
	require.True(t, ok)
	assert.Regexp(t, `^msg_[0-9a-f]{12}`, wantMsg)
	assert.Less(t, userID, wantMsg, "user message id must sort before the assistant's")

	reasoningCreate := propsOf(t, frames[5])
	rpart, ok := reasoningCreate["part"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "reasoning", rpart["type"])
	assert.Equal(t, "", rpart["text"])

	reasoning := propsOf(t, frames[6])
	assert.Equal(t, "reasoning", reasoning["field"])
	assert.Equal(t, "thinking...", reasoning["delta"])
	assert.Equal(t, rpart["id"], reasoning["partID"])

	textCreate := propsOf(t, frames[7])
	tpart, ok := textCreate["part"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "text", tpart["type"])
	assert.Equal(t, "", tpart["text"])

	textDelta := propsOf(t, frames[8])
	assert.Equal(t, "text", textDelta["field"])
	assert.Equal(t, "Par", textDelta["delta"])
	assert.Equal(t, wantMsg, textDelta["messageID"])
	assert.Equal(t, tpart["id"], textDelta["partID"])
	// Reasoning and text stream as distinct parts.
	assert.NotEqual(t, reasoning["partID"], textDelta["partID"])

	finalized := propsOf(t, frames[10])
	part, ok := finalized["part"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "text", part["type"])
	assert.Equal(t, "Paris", part["text"])
	assert.Equal(t, textDelta["partID"], part["id"])

	stepFinish := propsOf(t, frames[11])
	sfPart, ok := stepFinish["part"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "step-finish", sfPart["type"])
	assert.Contains(t, sfPart, "tokens")

	finalMsg := propsOf(t, frames[12])
	finalInfo, ok := finalMsg["info"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, wantMsg, finalInfo["id"])
	tm, ok := finalInfo["time"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, tm, "completed")
	// Assistant messages carry FLAT modelID/providerID (footer model display).
	assert.Equal(t, modelID, finalInfo["modelID"])

	status := propsOf(t, frames[13])
	assert.Equal(t, map[string]any{"type": "idle"}, status["status"])
}

// A tool-using turn: tool parts go out as running then completed, with the
// input re-carried on completion, and the step-finish carries the turn's
// real token/cost totals from run.completed.
func TestToolTurnStreamsToolParts(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-tool",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindToolCallStarted), Data: json.RawMessage(`{"tool_call_id":"prt_05d38cd67001guest","name":"bash","input":{"command":"echo hi"}}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindToolCallCompleted), Data: json.RawMessage(`{"tool_call_id":"prt_05d38cd67001guest","ok":true,"duration_ms":12,"summary":"hi\n"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"done"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), Data: json.RawMessage(`{"total_tokens_in":700,"total_tokens_out":30,"run_cost_micros":31500}`)},
	)

	resp := postPrompt(t, srv, "run echo hi")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	stream, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer stream.Body.Close()
	frames := drainFrames(t, stream.Body)

	var tools []map[string]any
	var stepFinish map[string]any
	for _, fr := range frames {
		if fr.Payload.Type != "message.part.updated" {
			continue
		}
		part, _ := propsOf(t, fr)["part"].(map[string]any)
		switch part["type"] {
		case "tool":
			tools = append(tools, part)
		case "step-finish":
			stepFinish = part
		}
	}
	require.Len(t, tools, 2)
	// The guest's own prt_ id is reused, so the part sorts where it ran.
	assert.Equal(t, "prt_05d38cd67001guest", tools[0]["id"])
	assert.Equal(t, "bash", tools[0]["tool"])
	running, _ := tools[0]["state"].(map[string]any)
	assert.Equal(t, "running", running["status"])
	assert.Equal(t, map[string]any{"command": "echo hi"}, running["input"])
	completed, _ := tools[1]["state"].(map[string]any)
	assert.Equal(t, "completed", completed["status"])
	assert.Equal(t, "hi\n", completed["output"])
	assert.Equal(t, map[string]any{"command": "echo hi"}, completed["input"])
	assert.Equal(t, "echo hi", completed["title"])

	require.NotNil(t, stepFinish)
	tokens, _ := stepFinish["tokens"].(map[string]any)
	assert.Equal(t, float64(700), tokens["input"])
	assert.Equal(t, float64(30), tokens["output"])
	assert.Equal(t, 0.0315, stepFinish["cost"])
}

// A tool left running when the turn ends is closed out as an error — a part
// stuck "running" renders as in-flight forever.
func TestDanglingToolClosedOnTurnEnd(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-dangle",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindToolCallStarted), Data: json.RawMessage(`{"tool_call_id":"prt_05d38cd67001dngl","name":"bash","input":{"command":"sleep 999"}}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunFailed), Data: json.RawMessage(`{"message":"timeout"}`)},
	)

	resp := postPrompt(t, srv, "hang")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	stream, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer stream.Body.Close()
	frames := drainFrames(t, stream.Body)

	var last map[string]any
	for _, fr := range frames {
		if fr.Payload.Type != "message.part.updated" {
			continue
		}
		if part, _ := propsOf(t, fr)["part"].(map[string]any); part["type"] == "tool" {
			last = part
		}
	}
	require.NotNil(t, last)
	state, _ := last["state"].(map[string]any)
	assert.Equal(t, "error", state["status"])
}

// The assistant id must sort after the user id even when the event clock
// runs behind the clock that minted the user id (client-minted ids, or
// container-vs-host skew on the dev stack — both observed live). The
// comparison must happen on 48-bit T values: comparing a truncated T against
// an untruncated millisecond count silently never fires (shipped once).
func TestAssistantIDSortsAfterUserIDDespiteClockSkew(t *testing.T) {
	srv, h := newBridgedFacade(t)
	// Harness events are stamped 2026-01-01, far behind the wall clock that
	// mints the fallback user id — exactly the inversion scenario.
	h.QueueRun("run-skew",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	// Client-minted user id far in the future of the event clock.
	futureID := ocTimeID("msg_", timeNowMs()+3_600_000, 0, "clientmint")
	body := `{"messageID":` + string(mustJSON(t, futureID)) + `,"parts":[{"type":"text","text":"skew test"}]}`
	resp, err := http.Post(srv.URL+"/session/ses_x/prompt_async", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	stream, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer stream.Body.Close()
	frames := drainFrames(t, stream.Body)

	var userID, asstID string
	for _, fr := range frames {
		if fr.Payload.Type != "message.updated" {
			continue
		}
		info, ok := propsOf(t, fr)["info"].(map[string]any)
		require.True(t, ok)
		switch info["role"] {
		case "user":
			userID = info["id"].(string)
		case "assistant":
			if asstID == "" {
				asstID = info["id"].(string)
			}
		}
	}
	require.NotEmpty(t, userID)
	require.NotEmpty(t, asstID)
	assert.Equal(t, futureID, userID)
	assert.Less(t, userID, asstID, "assistant id must sort after the user id")
}

func TestRunFailedEmitsSessionErrorThenIdle(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-f",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunFailed), Data: json.RawMessage(`{"message":"provider exploded"}`)},
	)

	resp := postPrompt(t, srv, "boom")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	stream, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer stream.Body.Close()
	frames := drainFrames(t, stream.Body)

	var types []string
	for _, fr := range frames {
		types = append(types, fr.Payload.Type)
	}
	require.Equal(t, []string{
		"server.connected",
		"message.updated",      // user echo
		"message.part.updated", // user prompt part
		"session.status",       // busy
		"message.updated",      // assistant
		"session.error",
		"session.status", // idle
		"session.idle",
	}, types)

	errProps := propsOf(t, frames[5])
	errObj, ok := errProps["error"].(map[string]any)
	require.True(t, ok)
	data, ok := errObj["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "provider exploded", data["message"])
}

// Events for runs this facade didn't start (another device's turns) are not
// translated onto the client stream.
func TestUntrackedRunsAreSkipped(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueRun("run-foreign",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"not yours"}`)},
	)
	// No prompt sent — nothing tracked.

	stream, err := http.Get(srv.URL + "/global/event")
	require.NoError(t, err)
	defer stream.Body.Close()
	frames := drainFrames(t, stream.Body)

	require.Len(t, frames, 1)
	assert.Equal(t, "server.connected", frames[0].Payload.Type)
}

// Session lifecycle: list is empty until create/prompt, then single-entry;
// create always returns the one bridged session.
func TestSessionListGrowsAfterCreate(t *testing.T) {
	srv, _ := newBridgedFacade(t)

	resp, err := http.Get(srv.URL + "/session")
	require.NoError(t, err)
	var list []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	resp.Body.Close()
	assert.Empty(t, list)

	created, err := http.Post(srv.URL+"/session", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	var sess map[string]any
	require.NoError(t, json.NewDecoder(created.Body).Decode(&sess))
	created.Body.Close()
	assert.Equal(t, "ses_"+strings.ReplaceAll(testSessionID, "-", ""), sess["id"])

	resp, err = http.Get(srv.URL + "/session")
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	resp.Body.Close()
	require.Len(t, list, 1)
	assert.Equal(t, sess["id"], list[0]["id"])
}
