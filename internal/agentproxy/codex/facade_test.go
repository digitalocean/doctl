package codex

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/agentproxy"
	"github.com/digitalocean/doctl/internal/agentproxy/agentproxytest"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSessionID = "sess_test123"

// recordedNotification is one call a Facade made through Notifier.Notify,
// captured verbatim (not round-tripped through JSON) so assertions can type
// assert the concrete notification struct directly.
type recordedNotification struct {
	method string
	params any
}

// notifierRecorder is a fake agentproxy.Notifier that queues every call so a
// test can assert on the exact notification sequence a facade produced,
// in order, without racing its background event-loop goroutine.
type notifierRecorder struct {
	ch chan recordedNotification
}

func newNotifierRecorder() *notifierRecorder {
	return &notifierRecorder{ch: make(chan recordedNotification, 32)}
}

func (r *notifierRecorder) Notify(method string, params any) error {
	r.ch <- recordedNotification{method: method, params: params}
	return nil
}

// next blocks for the next notification, failing the test rather than
// hanging the suite if the facade's event loop never produces one.
func (r *notifierRecorder) next(t *testing.T) recordedNotification {
	t.Helper()
	select {
	case n := <-r.ch:
		return n
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a notification")
		return recordedNotification{}
	}
}

func (r *notifierRecorder) expectNone(t *testing.T) {
	t.Helper()
	select {
	case n := <-r.ch:
		t.Fatalf("expected no further notifications, got %q", n.method)
	case <-time.After(100 * time.Millisecond):
	}
}

var _ agentproxy.Notifier = (*notifierRecorder)(nil)

// newTestFacade wires a Facade against a fresh agentproxytest.Harness (a real
// do.HostedAgentsService talking to an httptest.Server) and installs a
// notifierRecorder, mirroring how agentproxy.Serve wires a real *wsNotifier
// via NotifierAware before handing the facade any messages.
func newTestFacade(t *testing.T) (*Facade, *agentproxytest.Harness, *notifierRecorder) {
	t.Helper()
	h := agentproxytest.New(t, testSessionID)
	client, err := godo.New(nil, godo.SetBaseURL(h.Server.URL+"/"))
	require.NoError(t, err)
	svc := do.NewHostedAgentsService(client)

	f := &Facade{SessionID: testSessionID, Sessions: svc}
	rec := newNotifierRecorder()
	f.SetNotifier(rec)
	return f, h, rec
}

// dispatch marshals params (if any) to json.RawMessage and calls Dispatch,
// matching what proxy.go's handleConn does with an incoming JSON-RPC frame's
// params field.
func dispatch(t *testing.T, f *Facade, method string, params any) (any, error) {
	t.Helper()
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		require.NoError(t, err)
		raw = b
	}
	return f.Dispatch(context.Background(), method, raw)
}

func TestFacade_Initialize(t *testing.T) {
	f, _, _ := newTestFacade(t)

	result, err := dispatch(t, f, "initialize", nil)
	require.NoError(t, err)

	init, ok := result.(initializeResult)
	require.True(t, ok, "result should be initializeResult, got %T", result)
	assert.Equal(t, "unix", init.PlatformFamily)
	assert.Equal(t, "linux", init.PlatformOs)
	assert.Contains(t, init.UserAgent, TestedVersion)
}

func TestFacade_Initialized_NoOp(t *testing.T) {
	f, _, _ := newTestFacade(t)

	result, err := dispatch(t, f, "initialized", nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestFacade_AccountRead(t *testing.T) {
	f, _, _ := newTestFacade(t)

	result, err := dispatch(t, f, "account/read", nil)
	require.NoError(t, err)

	acct, ok := result.(accountReadResult)
	require.True(t, ok, "result should be accountReadResult, got %T", result)
	assert.False(t, acct.RequiresOpenaiAuth)
	assert.Equal(t, "chatgpt", acct.Account.Type)
}

func TestFacade_ModelList(t *testing.T) {
	f, _, _ := newTestFacade(t)

	result, err := dispatch(t, f, "model/list", nil)
	require.NoError(t, err)

	list, ok := result.(modelListResult)
	require.True(t, ok, "result should be modelListResult, got %T", result)
	require.Len(t, list.Data, 1)
	assert.Equal(t, defaultModel, list.Data[0].ID)
	assert.True(t, list.Data[0].IsDefault)
}

func TestFacade_ThreadStart(t *testing.T) {
	f, _, _ := newTestFacade(t)

	result, err := dispatch(t, f, "thread/start", nil)
	require.NoError(t, err)

	start, ok := result.(threadStartResult)
	require.True(t, ok, "result should be threadStartResult, got %T", result)
	assert.Equal(t, testSessionID, start.Thread.ID)
	assert.Equal(t, testSessionID, start.Thread.SessionID)
	assert.Equal(t, "idle", start.Thread.Status.Type)
}

func TestFacade_ThreadResume(t *testing.T) {
	f, _, _ := newTestFacade(t)

	t.Run("matching thread id", func(t *testing.T) {
		result, err := dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
		require.NoError(t, err)

		resume, ok := result.(threadResumeResult)
		require.True(t, ok, "result should be threadResumeResult, got %T", result)
		assert.Equal(t, testSessionID, resume.Thread.ID)
	})

	t.Run("unknown thread id is a real not-found, not a stub gap", func(t *testing.T) {
		_, err := dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: "not-this-session"})
		require.Error(t, err)

		var rpcErr *agentproxy.RPCError
		require.ErrorAs(t, err, &rpcErr)
		assert.Equal(t, -32001, rpcErr.Code)
	})
}

func TestFacade_UnhandledMethod(t *testing.T) {
	f, _, _ := newTestFacade(t)

	_, err := dispatch(t, f, "totally/unknown/method", nil)
	assert.ErrorIs(t, err, agentproxy.ErrMethodNotFound)
}

func TestFacade_TurnInterrupt_NoOp(t *testing.T) {
	f, _, _ := newTestFacade(t)

	result, err := dispatch(t, f, "turn/interrupt", nil)
	require.NoError(t, err)
	assert.Equal(t, turnInterruptResult{}, result)
}

func TestFacade_TurnStart_WrongThreadID(t *testing.T) {
	f, _, _ := newTestFacade(t)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: "not-this-session",
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.Error(t, err)

	var rpcErr *agentproxy.RPCError
	require.ErrorAs(t, err, &rpcErr)
	assert.Equal(t, -32001, rpcErr.Code)
}

// TestFacade_TurnStart_SendInputError exercises the harness's "no run queued"
// failure path (no QueueRun call before turn/start) to confirm a SendInput
// failure surfaces as a JSON-RPC error rather than a hang or panic.
func TestFacade_TurnStart_SendInputError(t *testing.T) {
	f, _, _ := newTestFacade(t)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.Error(t, err)

	var rpcErr *agentproxy.RPCError
	require.ErrorAs(t, err, &rpcErr)
	assert.Equal(t, -32000, rpcErr.Code)
}

// TestFacade_TurnStart_StreamsToCompletion drives one full "one-way text"
// turn end to end: turn/start -> SendInput against the fake harness -> the
// harness's queued run.started/run.token_delta/run.completed events ->
// translated into the turn/started, item/started, item/agentMessage/delta*,
// item/completed, turn/completed notification sequence the design doc's
// event-mapping table calls for.
func TestFacade_TurnStart_StreamsToCompletion(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-1",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"He"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"llo"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	result, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	start, ok := result.(turnStartResult)
	require.True(t, ok, "result should be turnStartResult, got %T", result)
	assert.Equal(t, "run-1", start.Turn.ID)
	assert.Equal(t, "inProgress", start.Turn.Status)

	started := rec.next(t)
	assert.Equal(t, "turn/started", started.method)
	assert.Equal(t, "run-1", started.params.(turnStartedNotification).Turn.ID)

	itemStarted := rec.next(t)
	assert.Equal(t, "item/started", itemStarted.method)

	delta1 := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", delta1.method)
	assert.Equal(t, "He", delta1.params.(agentMessageDeltaNotification).Delta)

	delta2 := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", delta2.method)
	assert.Equal(t, "llo", delta2.params.(agentMessageDeltaNotification).Delta)

	itemCompleted := rec.next(t)
	require.Equal(t, "item/completed", itemCompleted.method)
	assert.Equal(t, "Hello", itemCompleted.params.(itemCompletedNotification).Item.Text)

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	tc := turnCompleted.params.(turnCompletedNotification)
	assert.Equal(t, "completed", tc.Turn.Status)
	assert.Equal(t, "run-1", tc.Turn.ID)
	assert.Nil(t, tc.Turn.Error)

	rec.expectNone(t)
}

// waitEventLoopStopped polls until the facade's event-loop goroutine has
// exited and stopEventLoop has reset streamStarted. Observing a
// turn/completed notification only means the loop finished that turn, not
// that it has since hit EOF on the underlying stream and stopped — a test
// that starts a second turn right after turn/completed would otherwise race
// stopEventLoop rather than deterministically exercising life after it.
func waitEventLoopStopped(t *testing.T, f *Facade) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		stopped := !f.streamStarted
		f.mu.Unlock()
		if stopped {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for the event loop to stop")
}

// TestFacade_TurnStart_ReopensStreamAfterPreviousLoopEnded confirms a second
// turn/start can still get notified after the harness's SSE stream for the
// first turn ends (agentproxytest.Harness.handleStream closes the response
// once its queued events run out) — regression coverage for streamStarted
// getting stuck true forever, which left every later turn/start returning
// "inProgress" with no event-loop goroutine left running to ever notify it.
func TestFacade_TurnStart_ReopensStreamAfterPreviousLoopEnded(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-1",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)
	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started
	_ = rec.next(t) // item/completed
	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	waitEventLoopStopped(t, f)

	h.QueueRun("run-2",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)
	_, err = dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "again"}},
	})
	require.NoError(t, err)

	started := rec.next(t)
	require.Equal(t, "turn/started", started.method)
	assert.Equal(t, "run-2", started.params.(turnStartedNotification).Turn.ID)
}

// TestFacade_TurnStart_Failure confirms a canonical run.failed event maps to
// turn/completed with status "failed" and the error populated — there is no
// turn/failed method in the codex protocol (see the facade.go comment above
// turnStartParams), so this is the only failure signal the client ever sees.
func TestFacade_TurnStart_Failure(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-fail",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunFailed), Data: json.RawMessage(`{"message":"boom"}`)},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started

	// finishTurn always closes a started item before completing the turn,
	// success or failure — an item that started should never dangle "open".
	itemCompleted := rec.next(t)
	require.Equal(t, "item/completed", itemCompleted.method)

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	tc := turnCompleted.params.(turnCompletedNotification)
	assert.Equal(t, "failed", tc.Turn.Status)
	require.NotNil(t, tc.Turn.Error)
	assert.Equal(t, "boom", tc.Turn.Error.Message)
}
