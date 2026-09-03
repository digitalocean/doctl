package opencode

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/agentproxy/agentproxytest"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// queueTwoTurnHistory fills the harness's replay queue with two completed
// turns (the second one with reasoning), the durable-history shape M3
// reconstructs from.
func queueTwoTurnHistory(h *agentproxytest.Harness) {
	h.QueueReplayHistory(
		// Turn 1 uses the observed wire key ("agent"); turn 2 uses the proto
		// name ("user_input") — both must reconstruct.
		agentproxytest.Event{RunID: "run-1", Type: string(godo.HostedAgentEventKindRunStarted), Data: json.RawMessage(`{"agent":"first question"}`)},
		agentproxytest.Event{RunID: "run-1", Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"first "}`)},
		agentproxytest.Event{RunID: "run-1", Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"answer"}`)},
		agentproxytest.Event{RunID: "run-1", Type: string(godo.HostedAgentEventKindRunCompleted)},
		agentproxytest.Event{RunID: "run-2", Type: string(godo.HostedAgentEventKindRunStarted), Data: json.RawMessage(`{"user_input":"second question"}`)},
		agentproxytest.Event{RunID: "run-2", Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"hmm","is_reasoning":true}`)},
		agentproxytest.Event{RunID: "run-2", Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"second answer"}`)},
		agentproxytest.Event{RunID: "run-2", Type: string(godo.HostedAgentEventKindRunCompleted)},
	)
}

func getMessages(t *testing.T, url string) []historyMessage {
	t.Helper()
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var msgs []historyMessage
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&msgs))
	return msgs
}

func TestHistoryReconstructsTurns(t *testing.T) {
	srv, h := newBridgedFacade(t)
	queueTwoTurnHistory(h)

	msgs := getMessages(t, srv.URL+"/session/ses_x/message")
	require.Len(t, msgs, 4) // user+assistant per turn

	// Turn 1: user question then assistant answer, linked by parentID.
	assert.Equal(t, "user", msgs[0].Info["role"])
	require.Len(t, msgs[0].Parts, 1)
	assert.Equal(t, "first question", msgs[0].Parts[0]["text"])
	assert.Equal(t, "assistant", msgs[1].Info["role"])
	assert.Equal(t, msgs[0].Info["id"], msgs[1].Info["parentID"])
	require.Len(t, msgs[1].Parts, 1)
	assert.Equal(t, "text", msgs[1].Parts[0]["type"])
	assert.Equal(t, "first answer", msgs[1].Parts[0]["text"])
	// Completed turns carry finish + time.completed (the TUI treats a
	// message without them as still in flight).
	assert.Equal(t, "stop", msgs[1].Info["finish"])
	tm, ok := msgs[1].Info["time"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, tm, "completed")
	// The crash-critical assistant fields ride along in history too.
	assert.Contains(t, msgs[1].Info, "tokens")
	assert.Contains(t, msgs[1].Info, "cost")

	// Turn 2: reasoning reconstructs as its own part, before the text part.
	require.Len(t, msgs[3].Parts, 2)
	assert.Equal(t, "reasoning", msgs[3].Parts[0]["type"])
	assert.Equal(t, "hmm", msgs[3].Parts[0]["text"])
	assert.Equal(t, "second answer", msgs[3].Parts[1]["text"])

	// Ids sort chronologically across the whole conversation — history ids
	// must interleave correctly with live-minted ones (the TUI orders by id).
	var prev string
	for i, m := range msgs {
		id, ok := m.Info["id"].(string)
		require.True(t, ok)
		assert.Greater(t, id, prev, "message %d id must sort after its predecessor", i)
		prev = id
	}
}

func TestHistoryHonorsLimit(t *testing.T) {
	srv, h := newBridgedFacade(t)
	queueTwoTurnHistory(h)

	msgs := getMessages(t, srv.URL+"/session/ses_x/message?limit=2")
	require.Len(t, msgs, 2)
	// The newest messages win: turn 2's user+assistant.
	assert.Equal(t, "second question", msgs[0].Parts[0]["text"])
	assert.Equal(t, "assistant", msgs[1].Info["role"])
}

func TestHistoryEmptyForFreshSession(t *testing.T) {
	srv, _ := newBridgedFacade(t)
	msgs := getMessages(t, srv.URL+"/session/ses_x/message")
	assert.Empty(t, msgs)
}

// A fresh proxy on a session with prior turns must list the session, or
// `opencode attach --continue` has nothing to resume.
func TestSessionListIncludesSessionWithHistory(t *testing.T) {
	srv, h := newBridgedFacade(t)
	queueTwoTurnHistory(h)

	resp, err := http.Get(srv.URL + "/session")
	require.NoError(t, err)
	defer resp.Body.Close()
	var list []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	require.Len(t, list, 1)
	assert.Equal(t, "ses_"+ocID("", testSessionID), list[0]["id"])
}

// History is fetched once and cached (the harness replay stream lingers ~8s
// before closing, and an attach asks twice) until a completed turn
// invalidates it.
func TestHistoryIsCachedUntilInvalidated(t *testing.T) {
	h := agentproxytest.New(t, testSessionID)
	client, err := godo.New(nil, godo.SetBaseURL(h.Server.URL+"/"))
	require.NoError(t, err)
	f := &Facade{SessionID: testSessionID, Sessions: do.NewHostedAgentsService(client), Dir: "/tmp/ws"}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)

	h.QueueReplayHistory(
		agentproxytest.Event{RunID: "run-1", Type: string(godo.HostedAgentEventKindRunStarted), Data: json.RawMessage(`{"user_input":"old"}`)},
		agentproxytest.Event{RunID: "run-1", Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"old answer"}`)},
		agentproxytest.Event{RunID: "run-1", Type: string(godo.HostedAgentEventKindRunCompleted)},
	)
	first := getMessages(t, srv.URL+"/session/ses_x/message")
	require.Len(t, first, 2)

	// The harness's durable history changes, but the cache still serves the
	// old view...
	h.QueueReplayHistory(
		agentproxytest.Event{RunID: "run-2", Type: string(godo.HostedAgentEventKindRunStarted), Data: json.RawMessage(`{"user_input":"new"}`)},
		agentproxytest.Event{RunID: "run-2", Type: string(godo.HostedAgentEventKindRunCompleted)},
	)
	cached := getMessages(t, srv.URL+"/session/ses_x/message")
	require.Len(t, cached, 2)
	assert.Equal(t, "old", cached[0].Parts[0]["text"])

	// ...until a completed live turn invalidates it (translateEvent does
	// this; exercised directly here).
	f.invalidateHistory()
	fresh := getMessages(t, srv.URL+"/session/ses_x/message")
	require.Len(t, fresh, 1) // run-2 has no answer text: user message only
	assert.Equal(t, "new", fresh[0].Parts[0]["text"])
}

// Failed turns keep the user's message visible but don't fabricate a
// completed answer.
func TestHistoryFailedTurn(t *testing.T) {
	srv, h := newBridgedFacade(t)
	h.QueueReplayHistory(
		agentproxytest.Event{RunID: "run-f", Type: string(godo.HostedAgentEventKindRunStarted), Data: json.RawMessage(`{"user_input":"doomed"}`)},
		agentproxytest.Event{RunID: "run-f", Type: string(godo.HostedAgentEventKindRunFailed), Data: json.RawMessage(`{"message":"boom"}`)},
	)

	msgs := getMessages(t, srv.URL+"/session/ses_x/message")
	require.Len(t, msgs, 2)
	assert.Equal(t, "user", msgs[0].Info["role"])
	assert.Equal(t, "assistant", msgs[1].Info["role"])
	assert.NotContains(t, msgs[1].Info, "finish")
}
