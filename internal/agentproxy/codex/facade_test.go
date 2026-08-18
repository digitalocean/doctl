package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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

// recordedRequest is one server-initiated request a Facade made through
// Notifier.Request, captured the same way recordedNotification captures a
// Notify call, plus a reply channel a test uses to script what the "client"
// answers with — mirroring how a real client would eventually send a reply
// frame back over the wire.
type recordedRequest struct {
	method string
	params any
	reply  chan requestReply
}

// respond scripts this request's reply, as if the client answered with
// result (marshaled to JSON, same as a real wire reply would be).
func (r *recordedRequest) respond(result any) {
	r.reply <- requestReply{result: result}
}

// fail scripts this request's reply as a JSON-RPC error, as if the client
// (or the connection itself) never answered successfully.
func (r *recordedRequest) fail(err error) {
	r.reply <- requestReply{err: err}
}

type requestReply struct {
	result any
	err    error
}

// notifierRecorder is a fake agentproxy.Notifier that queues every call so a
// test can assert on the exact notification/request sequence a facade
// produced, in order, without racing its background event-loop goroutine.
type notifierRecorder struct {
	ch    chan recordedNotification
	reqCh chan *recordedRequest
}

func newNotifierRecorder() *notifierRecorder {
	return &notifierRecorder{
		ch:    make(chan recordedNotification, 32),
		reqCh: make(chan *recordedRequest, 32),
	}
}

func (r *notifierRecorder) Notify(method string, params any) error {
	r.ch <- recordedNotification{method: method, params: params}
	return nil
}

// Request implements agentproxy.Notifier: queue the request for a test to
// observe via nextRequest, then block until that test scripts a reply via
// recordedRequest.respond/fail, or ctx is done — same blocking contract as
// the real wsNotifier.Request.
func (r *notifierRecorder) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	req := &recordedRequest{method: method, params: params, reply: make(chan requestReply, 1)}
	r.reqCh <- req
	select {
	case rep := <-req.reply:
		if rep.err != nil {
			return nil, rep.err
		}
		return json.Marshal(rep.result)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// nextRequest blocks for the next server-initiated request, same timeout
// discipline as next().
func (r *notifierRecorder) nextRequest(t *testing.T) *recordedRequest {
	t.Helper()
	select {
	case req := <-r.reqCh:
		return req
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a request")
		return nil
	}
}

// expectNoRequest asserts no server-initiated request arrives within a short
// window — used by --replay tests to confirm historical HITLs are not
// re-prompted as live approvals.
func (r *notifierRecorder) expectNoRequest(t *testing.T) {
	t.Helper()
	select {
	case req := <-r.reqCh:
		t.Fatalf("expected no server-initiated request, got %q", req.method)
	case <-time.After(50 * time.Millisecond):
	}
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
	return dispatchCtx(t, context.Background(), f, method, params)
}

// dispatchCtx is dispatch, but with an explicit context instead of always
// context.Background() — needed by any test that starts a turn and must
// guarantee runEventLoop's background goroutine has actually stopped before
// the test returns (see stopEventLoop), rather than leaving it to exit on
// its own. A test relying on plain dispatch's hardcoded background context
// has no way to do that: since M5's reconnect loop keeps retrying after any
// stream end (even a successful turn's), an orphaned goroutine from one
// test can outlive it and race a later test's use of a shared test hook
// like reconnectSleepFn — confirmed by -race, not hypothetical.
func dispatchCtx(t *testing.T, ctx context.Context, f *Facade, method string, params any) (any, error) {
	t.Helper()
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		require.NoError(t, err)
		raw = b
	}
	result, err := f.Dispatch(ctx, method, raw)
	// Mirror handleConn: AfterReply runs only after a successful request
	// reply would have been written. Unit tests call Dispatch directly, so
	// kick it here — otherwise --replay stays armed forever and never starts.
	if err == nil {
		f.AfterReply(ctx, method)
	}
	return result, err
}

// stopEventLoop cancels cancel and blocks until f's runEventLoop goroutine
// has actually exited (observed via streamStarted resetting to false — see
// runEventLoop's own deferred cleanup), not just until cancel returns.
// Required at the end of any test that exercises the reconnect loop:
// canceling ctx only signals the goroutine to stop, asynchronously, and a
// test that returns without waiting can leave it alive to race a later
// test's mutation of a shared package-level test hook.
func stopEventLoop(t *testing.T, f *Facade, cancel context.CancelFunc) {
	t.Helper()
	cancel()
	require.Eventually(t, func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return !f.streamStarted
	}, 2*time.Second, 10*time.Millisecond, "runEventLoop should have exited after ctx cancellation")
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

// Sessions that can't relay (newTestFacade leaves AgentKind unset) keep the
// pre-relay answers for these two: method-not-found, and an empty interrupt
// reply. The relayed behaviour is covered in raw_test.go.
func TestFacade_UnhandledMethod(t *testing.T) {
	f, _, _ := newTestFacade(t)

	_, err := dispatch(t, f, "totally/unknown/method", nil)
	assert.ErrorIs(t, err, agentproxy.ErrMethodNotFound)
}

func TestFacade_TurnInterrupt_NoOpWithoutRelay(t *testing.T) {
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
	completedItem, ok := itemCompleted.params.(itemCompletedNotification).Item.(agentMessageItem)
	require.True(t, ok, "item/completed's Item should be agentMessageItem, got %T", itemCompleted.params.(itemCompletedNotification).Item)
	assert.Equal(t, "Hello", completedItem.Text)

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	tc := turnCompleted.params.(turnCompletedNotification)
	assert.Equal(t, "completed", tc.Turn.Status)
	assert.Equal(t, "run-1", tc.Turn.ID)
	assert.Nil(t, tc.Turn.Error)

	rec.expectNone(t)
}

// TestFacade_TurnStart_TokenUsage confirms run.usage_recorded maps to
// thread/tokenUsage/updated, using the real captured field names from
// harness-api's events.proto ({usage: {input_tokens, output_tokens,
// cached_input_tokens, reasoning_tokens}}). Two usage events in the same
// connection confirm "total" accumulates across turns while "last" reports
// only the most recent event's own delta — and that run.cost_accrued
// produces no notification at all, since codex has no cost-facing concept
// to translate it to.
func TestFacade_TurnStart_TokenUsage(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-usage-1",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindRunUsageRecorded),
			Data: json.RawMessage(`{"step_id":"step-1","model_id":"gpt-5.5","usage":{"input_tokens":100,"output_tokens":20,"cached_input_tokens":10,"reasoning_tokens":5}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindRunCostAccrued),
			Data: json.RawMessage(`{"running_total_micros":1500,"delta_micros":1500}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindRunUsageRecorded),
			Data: json.RawMessage(`{"step_id":"step-2","model_id":"gpt-5.5","usage":{"input_tokens":50,"output_tokens":10,"cached_input_tokens":0,"reasoning_tokens":2}}`),
		},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started

	usage1 := rec.next(t)
	require.Equal(t, "thread/tokenUsage/updated", usage1.method, "run.cost_accrued must not produce any notification")
	tu1 := usage1.params.(threadTokenUsageUpdatedNotification).TokenUsage
	assert.Equal(t, int64(120), tu1.Last.TotalTokens, "synthesized as input+output: 100+20")
	assert.Equal(t, int64(100), tu1.Last.InputTokens)
	assert.Equal(t, int64(10), tu1.Last.CachedInputTokens)
	assert.Equal(t, int64(20), tu1.Last.OutputTokens)
	assert.Equal(t, int64(5), tu1.Last.ReasoningOutputTokens)
	assert.Equal(t, tu1.Last, tu1.Total, "total equals last after the first usage event")
	assert.Nil(t, tu1.ModelContextWindow, "no source data for this field; must stay nil, not synthesized")

	usage2 := rec.next(t)
	require.Equal(t, "thread/tokenUsage/updated", usage2.method)
	tu2 := usage2.params.(threadTokenUsageUpdatedNotification).TokenUsage
	assert.Equal(t, int64(60), tu2.Last.TotalTokens, "this event's own delta only: 50+10")
	assert.Equal(t, int64(180), tu2.Total.TotalTokens, "accumulated across both events: 120+60")
	assert.Equal(t, int64(150), tu2.Total.InputTokens, "100+50")
	assert.Equal(t, int64(10), tu2.Total.CachedInputTokens, "10+0")
	assert.Equal(t, int64(30), tu2.Total.OutputTokens, "20+10")
	assert.Equal(t, int64(7), tu2.Total.ReasoningOutputTokens, "5+2")

	_ = rec.next(t) // item/completed
	_ = rec.next(t) // turn/completed
	rec.expectNone(t)
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

// TestFacade_TurnStart_ManyDeltasAllDelivered is the regression test for the
// drainStream cursor bug: event ids are random (UUIDv4 from the guest, ULIDs
// with per-id randomness from harness-api — the fixture mints random UUIDs
// to match), so any "skip events whose id sorts <= the cursor" logic
// converges on the max id seen and silently drops most of a turn. Sixteen
// deltas make a random-order id collision with an ordinal skip a
// statistical certainty; every one must be delivered, in order, and the
// turn must still complete.
func TestFacade_TurnStart_ManyDeltasAllDelivered(t *testing.T) {
	f, h, rec := newTestFacade(t)

	const deltas = 16
	events := []agentproxytest.Event{{Type: string(godo.HostedAgentEventKindRunStarted)}}
	for i := 0; i < deltas; i++ {
		events = append(events, agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindTokenChunk),
			Data: json.RawMessage(fmt.Sprintf(`{"text":"d%d "}`, i)),
		})
	}
	events = append(events, agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)})
	h.QueueRun("run-many", events...)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started

	var got strings.Builder
	for i := 0; i < deltas; i++ {
		n := rec.next(t)
		require.Equal(t, "item/agentMessage/delta", n.method, "delta %d of %d missing — events are being dropped", i+1, deltas)
		got.WriteString(n.params.(agentMessageDeltaNotification).Delta)
	}

	var want strings.Builder
	for i := 0; i < deltas; i++ {
		fmt.Fprintf(&want, "d%d ", i)
	}
	assert.Equal(t, want.String(), got.String(), "deltas must arrive complete and in order")

	itemCompleted := rec.next(t)
	require.Equal(t, "item/completed", itemCompleted.method)
	assert.Equal(t, want.String(), itemCompleted.params.(itemCompletedNotification).Item.(agentMessageItem).Text)

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	assert.Equal(t, "completed", turnCompleted.params.(turnCompletedNotification).Turn.Status)
	rec.expectNone(t)
}

// TestFacade_SessionRunFailedClosesTrackedTurns covers the fail-safe close:
// OHR establishes the session's multi-turn parent run with the session id as
// its run id and attributes transport-level failures to it (seen live: a
// rejected turn/start fails the parent run, not the per-turn run the facade
// tracks). Such a run.failed matches no tracked turn — without the
// fail-safe it would be skipped and the TUI would spin forever on a turn
// nobody will ever finish. It must close every tracked turn as failed, with
// the parent run's message.
func TestFacade_SessionRunFailedClosesTrackedTurns(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-orphaned",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type:  string(godo.HostedAgentEventKindRunFailed),
			RunID: testSessionID, // the parent run, not the tracked turn
			Data:  json.RawMessage(`{"message":"transport: turn/start: rpc error: -32600: nope"}`),
		},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started

	itemCompleted := rec.next(t)
	require.Equal(t, "item/completed", itemCompleted.method)

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	tc := turnCompleted.params.(turnCompletedNotification)
	assert.Equal(t, "run-orphaned", tc.Turn.ID, "the tracked turn must be the one closed")
	assert.Equal(t, "failed", tc.Turn.Status)
	require.NotNil(t, tc.Turn.Error)
	assert.Equal(t, "transport: turn/start: rpc error: -32600: nope", tc.Turn.Error.Message)
}

// TestFacade_TurnStart_ToolCall drives one turn with a tool call followed by
// a short message, end to end: run.tool_call_started/completed -> the
// item/started, item/completed (commandExecution) pair, interleaved with the
// message item's own started/delta/completed sequence. Event payloads match
// a real captured tool-using turn verbatim (see facade.go's
// commandExecutionItem doc comment) — not invented.
func TestFacade_TurnStart_ToolCall(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-tool-1",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallStarted),
			Data: json.RawMessage(`{"tool_call_id":"call_1","name":"command_execution","input":{"command":"/bin/bash -lc ls","cwd":"/workspace"}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallCompleted),
			Data: json.RawMessage(`{"tool_call_id":"call_1","ok":true,"duration_ms":42,"summary":""}`),
		},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"Found 2 files."}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "list files"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started (agentMessage) — always fires on run.started, tool call or not

	cmdStarted := rec.next(t)
	require.Equal(t, "item/started", cmdStarted.method)
	cmdItem, ok := cmdStarted.params.(itemStartedNotification).Item.(commandExecutionItem)
	require.True(t, ok, "item/started's Item should be commandExecutionItem, got %T", cmdStarted.params.(itemStartedNotification).Item)
	assert.Equal(t, "call_1", cmdItem.ID)
	assert.Equal(t, "/bin/bash -lc ls", cmdItem.Command)
	assert.Equal(t, "/workspace", cmdItem.Cwd)
	assert.Equal(t, "inProgress", cmdItem.Status)
	assert.Equal(t, "agent", cmdItem.Source)
	assert.NotNil(t, cmdItem.CommandActions)
	assert.Empty(t, cmdItem.CommandActions)

	cmdCompleted := rec.next(t)
	require.Equal(t, "item/completed", cmdCompleted.method)
	completedCmd, ok := cmdCompleted.params.(itemCompletedNotification).Item.(commandExecutionItem)
	require.True(t, ok, "item/completed's Item should be commandExecutionItem, got %T", cmdCompleted.params.(itemCompletedNotification).Item)
	assert.Equal(t, "call_1", completedCmd.ID)
	assert.Equal(t, "completed", completedCmd.Status)
	// command/cwd must be remembered from the started event — completed's
	// own payload never repeats them.
	assert.Equal(t, "/bin/bash -lc ls", completedCmd.Command)
	assert.Equal(t, "/workspace", completedCmd.Cwd)
	require.NotNil(t, completedCmd.ExitCode)
	assert.Equal(t, 0, *completedCmd.ExitCode)
	require.NotNil(t, completedCmd.DurationMs)
	assert.Equal(t, int64(42), *completedCmd.DurationMs)
	assert.Nil(t, completedCmd.AggregatedOutput, "summary was empty in the captured payload, so aggregatedOutput should stay nil, not a synthesized value")

	delta := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", delta.method)
	assert.Equal(t, "Found 2 files.", delta.params.(agentMessageDeltaNotification).Delta)

	_ = rec.next(t) // item/completed (agentMessage)

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	assert.Equal(t, "completed", turnCompleted.params.(turnCompletedNotification).Turn.Status)

	rec.expectNone(t)
}

// TestFacade_TurnStart_ToolCallFailed confirms ok:false maps to
// status:"failed" and a synthesized exitCode of 1 — canonical
// run.tool_call_completed carries a success boolean, not a real process exit
// code, so this is the best this facade can report today (see
// commandExecutionItem's doc comment).
func TestFacade_TurnStart_ToolCallFailed(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-tool-fail",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallStarted),
			Data: json.RawMessage(`{"tool_call_id":"call_2","name":"command_execution","input":{"command":"/bin/bash -lc false","cwd":"/workspace"}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallCompleted),
			Data: json.RawMessage(`{"tool_call_id":"call_2","ok":false,"duration_ms":5,"summary":""}`),
		},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "run a failing command"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started (agentMessage)
	_ = rec.next(t) // item/started (commandExecution)

	cmdCompleted := rec.next(t)
	require.Equal(t, "item/completed", cmdCompleted.method)
	completedCmd, ok := cmdCompleted.params.(itemCompletedNotification).Item.(commandExecutionItem)
	require.True(t, ok, "item/completed's Item should be commandExecutionItem, got %T", cmdCompleted.params.(itemCompletedNotification).Item)
	assert.Equal(t, "failed", completedCmd.Status)
	require.NotNil(t, completedCmd.ExitCode)
	assert.Equal(t, 1, *completedCmd.ExitCode)
}

// TestFacade_TurnStart_ApprovalAccept is the M4 "done when" sequence at the
// Dispatch level: a run.human_input_requested event produces a real
// server->client item/commandExecution/requestApproval REQUEST (not a
// notification — retrieved via rec.nextRequest, not rec.next), and an
// "accept" reply resolves the harness's HITL with Outcome "approve". Only
// through run.human_input_requested is queued after the tool call starts
// (no tool_call_completed/run.completed): this test is scoped to the
// approval round-trip itself, not the full turn lifecycle already covered by
// TestFacade_TurnStart_ToolCall.
func TestFacade_TurnStart_ApprovalAccept(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-hitl-1",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallStarted),
			Data: json.RawMessage(`{"tool_call_id":"call_1","name":"command_execution","input":{"command":"/bin/bash -lc \"find /workspace | sort\"","cwd":"/workspace"}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindHITLRequested),
			Data: json.RawMessage(`{"hitl_id":"hitl-1","payload":{"kind":"command_execution","itemId":"call_1","turnId":"harness-internal-turn-id","startedAtMs":1000,"environmentId":"local","command":"/bin/bash -lc \"find /workspace | sort\"","cwd":"/workspace","commandActions":[{"command":"find /workspace","path":"workspace","type":"listFiles"}],"proposedExecpolicyAmendment":["sort"]}}`),
		},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "find files, sorted"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started (agentMessage)
	_ = rec.next(t) // item/started (commandExecution)

	req := rec.nextRequest(t)
	assert.Equal(t, "item/commandExecution/requestApproval", req.method)
	params, ok := req.params.(commandExecutionRequestApprovalParams)
	require.True(t, ok, "request params should be commandExecutionRequestApprovalParams, got %T", req.params)
	assert.Equal(t, testSessionID, params.ThreadID)
	// The event envelope's run id, not the canonical payload's own (distinct)
	// "turnId" field — see hitlRequestedPayload's doc comment.
	assert.Equal(t, "run-hitl-1", params.TurnID)
	assert.Equal(t, "call_1", params.ItemID)
	assert.Equal(t, "local", params.EnvironmentID)
	assert.Equal(t, `/bin/bash -lc "find /workspace | sort"`, params.Command)
	assert.Equal(t, "/workspace", params.Cwd)
	assert.NotEmpty(t, params.CommandActions)

	// proposedExecpolicyAmendment must never be forwarded, even though the
	// queued HITL payload carries one (see the "sort" amendment queued
	// above): sending it re-introduces the exact "remembered approval the
	// harness can't honor" bug this facade already omits availableDecisions
	// to avoid — confirmed live (see commandExecutionRequestApprovalParams'
	// doc comment). Asserted at the marshaled-JSON level, not just "the Go
	// struct has no such field," since that's what the TUI actually sees.
	raw, err := json.Marshal(params)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "proposedExecpolicyAmendment", "must not forward proposedExecpolicyAmendment to the client")

	req.respond(map[string]any{"decision": "accept"})

	res := h.NextHITLResolution(t, 2*time.Second)
	assert.Equal(t, "hitl-1", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeApprove), res.Outcome)
	assert.Equal(t, string(godo.HostedAgentResolutionSourceInlineKeystroke), res.Source)
}

// TestFacade_TurnStart_ApprovalDecline confirms a "decline" reply resolves
// the harness's HITL with Outcome "reject", the other half of
// decodeCommandExecutionApprovalDecision's mapping exercised by
// TestFacade_TurnStart_ApprovalAccept.
func TestFacade_TurnStart_ApprovalDecline(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-hitl-2",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallStarted),
			Data: json.RawMessage(`{"tool_call_id":"call_2","name":"command_execution","input":{"command":"/bin/bash -lc rm","cwd":"/workspace"}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindHITLRequested),
			Data: json.RawMessage(`{"hitl_id":"hitl-2","payload":{"kind":"command_execution","itemId":"call_2","turnId":"harness-internal-turn-id","startedAtMs":1000,"environmentId":"local","command":"/bin/bash -lc rm","cwd":"/workspace","commandActions":[]}}`),
		},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "remove a file"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started (agentMessage)
	_ = rec.next(t) // item/started (commandExecution)

	req := rec.nextRequest(t)
	req.respond(map[string]any{"decision": "decline"})

	res := h.NextHITLResolution(t, 2*time.Second)
	assert.Equal(t, "hitl-2", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeReject), res.Outcome)
}

// TestDecodeCommandExecutionApprovalDecision table-tests every CommandExecutionApprovalDecision
// variant (both bare-string unit variants and the two struct variants) against
// the harness's 3-value HostedAgentHITLOutcome, confirming the collapse
// documented on decodeCommandExecutionApprovalDecision itself.
func TestDecodeCommandExecutionApprovalDecision(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    godo.HostedAgentHITLOutcome
		wantErr bool
	}{
		{name: "accept", raw: `"accept"`, want: godo.HostedAgentHITLOutcomeApprove},
		{name: "acceptForSession", raw: `"acceptForSession"`, want: godo.HostedAgentHITLOutcomeApprove},
		{name: "decline", raw: `"decline"`, want: godo.HostedAgentHITLOutcomeReject},
		{name: "cancel", raw: `"cancel"`, want: godo.HostedAgentHITLOutcomeReject},
		{
			name: "acceptWithExecpolicyAmendment",
			raw:  `{"acceptWithExecpolicyAmendment":{"command":["sort"]}}`,
			want: godo.HostedAgentHITLOutcomeApprove,
		},
		{
			name: "applyNetworkPolicyAmendment",
			raw:  `{"applyNetworkPolicyAmendment":{"host":"example.com","decision":"allow"}}`,
			want: godo.HostedAgentHITLOutcomeApprove,
		},
		{name: "unknown string", raw: `"somethingNew"`, wantErr: true},
		{name: "unknown object", raw: `{"somethingNew":{}}`, wantErr: true},
		{name: "malformed", raw: `123`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeCommandExecutionApprovalDecision(json.RawMessage(tt.raw))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFacade_TurnStart_FileChange confirms a file_change tool call renders
// as a fileChange item, using the real captured payload shapes verbatim
// (run.tool_call_started's input is genuinely {} for file_change — no
// command/cwd/diff data at all, unlike command_execution).
func TestFacade_TurnStart_FileChange(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-filechange-1",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallStarted),
			Data: json.RawMessage(`{"tool_call_id":"call_7IQzxBAiCb7LkLoCYhJ4a0mo","name":"file_change","input":{}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallCompleted),
			Data: json.RawMessage(`{"tool_call_id":"call_7IQzxBAiCb7LkLoCYhJ4a0mo","ok":true,"duration_ms":0,"summary":""}`),
		},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "create a file"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started (agentMessage)

	started := rec.next(t)
	require.Equal(t, "item/started", started.method)
	startedItem, ok := started.params.(itemStartedNotification).Item.(fileChangeItem)
	require.True(t, ok, "item/started's Item should be fileChangeItem, got %T", started.params.(itemStartedNotification).Item)
	assert.Equal(t, "call_7IQzxBAiCb7LkLoCYhJ4a0mo", startedItem.ID)
	assert.Equal(t, "inProgress", startedItem.Status)
	assert.NotNil(t, startedItem.Changes)
	assert.Empty(t, startedItem.Changes, "canonical carries no diff content for file_change — changes must stay empty, not synthesized")

	completed := rec.next(t)
	require.Equal(t, "item/completed", completed.method)
	completedItem, ok := completed.params.(itemCompletedNotification).Item.(fileChangeItem)
	require.True(t, ok, "item/completed's Item should be fileChangeItem, got %T", completed.params.(itemCompletedNotification).Item)
	assert.Equal(t, "completed", completedItem.Status)
}

// TestFacade_TurnStart_FileChangeAlwaysAutoRejected confirms every
// file_change HITL is rejected immediately server-side — no
// item/fileChange/requestApproval ever reaches the client — and that the
// declined-vs-failed disambiguation fileChangeState exists for still works
// on this path: once run.tool_call_completed later reports ok:false for the
// same item, the item/completed notification reports status "declined", not
// "failed". See autoRejectFileChangeApproval for why this facade never asks.
func TestFacade_TurnStart_FileChangeAlwaysAutoRejected(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-filechange-hitl-1",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallStarted),
			Data: json.RawMessage(`{"tool_call_id":"call_1","name":"file_change","input":{}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindHITLRequested),
			Data: json.RawMessage(`{"hitl_id":"hitl-fc-1","payload":{"category":"permission","grantRoot":null,"itemId":"call_1","kind":"file_change","reason":null,"startedAtMs":1000,"turnId":"harness-internal-turn-id"}}`),
		},
		agentproxytest.Event{
			// WaitForHITL: the real harness wouldn't complete a gated call
			// until it has processed our ResolveHITL — without this gate,
			// the fake harness's naive pre-queued-and-instant delivery races
			// this event against autoRejectFileChangeApproval's goroutine
			// setting fc.declined, rather than modeling that real ordering.
			Type:        string(godo.HostedAgentEventKindToolCallCompleted),
			Data:        json.RawMessage(`{"tool_call_id":"call_1","ok":false,"duration_ms":0,"summary":""}`),
			WaitForHITL: "hitl-fc-1",
		},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "create a file"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started (agentMessage)
	_ = rec.next(t) // item/started (fileChange)

	res := h.NextHITLResolution(t, 2*time.Second)
	assert.Equal(t, "hitl-fc-1", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeReject), res.Outcome)
	assert.Equal(t, string(godo.HostedAgentResolutionSourceOutOfBand), res.Source)

	// No request should ever reach the client for file_change: it's rejected
	// entirely server-side.
	select {
	case req := <-rec.reqCh:
		t.Fatalf("expected no client-facing request for file_change, got %q", req.method)
	case <-time.After(100 * time.Millisecond):
	}

	completed := rec.next(t)
	require.Equal(t, "item/completed", completed.method)
	completedItem, ok := completed.params.(itemCompletedNotification).Item.(fileChangeItem)
	require.True(t, ok, "item/completed's Item should be fileChangeItem, got %T", completed.params.(itemCompletedNotification).Item)
	assert.Equal(t, "declined", completedItem.Status)
}

// TestFacade_TurnStart_ConcurrentFileChangeToolCalls_NoRace is a regression
// test for a real data race found in review: ts.fileChanges used to be
// touched with no lock on the main event-loop goroutine (ToolCallStarted/
// ToolCallCompleted, translate.go) while autoRejectFileChangeApproval touched
// the same map under f.mu from its own spawned goroutine — a lock held by
// only one side protects nothing, and Go maps aren't safe for any concurrent
// access. Two file_change tool calls in the same turn (e.g. "create a file,
// then edit it") is exactly the scenario that overlaps the two: call_2's
// ToolCallStarted (main goroutine, map insert) is queued to fire immediately
// after call_1's HITLRequested spawns its own goroutine (which locks f.mu
// and writes fc.declined) — no artificial synchronization between them, so
// this only stays race-free because both sides now hold f.mu consistently.
// Must be run with -race to mean anything.
func TestFacade_TurnStart_ConcurrentFileChangeToolCalls_NoRace(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-filechange-concurrent",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallStarted),
			Data: json.RawMessage(`{"tool_call_id":"call_1","name":"file_change","input":{}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindHITLRequested),
			Data: json.RawMessage(`{"hitl_id":"hitl-1","payload":{"category":"permission","grantRoot":null,"itemId":"call_1","kind":"file_change","reason":null,"startedAtMs":1000,"turnId":"harness-internal-turn-id"}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindToolCallStarted),
			Data: json.RawMessage(`{"tool_call_id":"call_2","name":"file_change","input":{}}`),
		},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindHITLRequested),
			Data: json.RawMessage(`{"hitl_id":"hitl-2","payload":{"category":"permission","grantRoot":null,"itemId":"call_2","kind":"file_change","reason":null,"startedAtMs":1000,"turnId":"harness-internal-turn-id"}}`),
		},
		agentproxytest.Event{
			Type:        string(godo.HostedAgentEventKindToolCallCompleted),
			Data:        json.RawMessage(`{"tool_call_id":"call_1","ok":false,"duration_ms":0,"summary":""}`),
			WaitForHITL: "hitl-1",
		},
		agentproxytest.Event{
			Type:        string(godo.HostedAgentEventKindToolCallCompleted),
			Data:        json.RawMessage(`{"tool_call_id":"call_2","ok":false,"duration_ms":0,"summary":""}`),
			WaitForHITL: "hitl-2",
		},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "create a file, then edit it"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started (agentMessage)
	_ = rec.next(t) // item/started (fileChange call_1)
	_ = rec.next(t) // item/started (fileChange call_2)

	res1 := h.NextHITLResolution(t, 2*time.Second)
	res2 := h.NextHITLResolution(t, 2*time.Second)
	gotIDs := map[string]bool{res1.RequestID: true, res2.RequestID: true}
	assert.True(t, gotIDs["hitl-1"] && gotIDs["hitl-2"], "both HITLs should resolve, got %v", gotIDs)

	completed1 := rec.next(t)
	require.Equal(t, "item/completed", completed1.method)
	assert.Equal(t, "call_1", completed1.params.(itemCompletedNotification).Item.(fileChangeItem).ID)
	assert.Equal(t, "declined", completed1.params.(itemCompletedNotification).Item.(fileChangeItem).Status)

	completed2 := rec.next(t)
	require.Equal(t, "item/completed", completed2.method)
	assert.Equal(t, "call_2", completed2.params.(itemCompletedNotification).Item.(fileChangeItem).ID)
	assert.Equal(t, "declined", completed2.params.(itemCompletedNotification).Item.(fileChangeItem).Status)

	_ = rec.next(t) // item/completed (agentMessage, from finishTurn)

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
}

// TestFacade_TurnStart_UnknownHITLKindAutoRejected confirms a HITL of a kind
// this facade doesn't implement (e.g. "permissions") is resolved as Reject
// automatically, rather than left unanswered forever — see
// autoRejectUnknownHITL. No client-facing request should ever be sent: the
// resolution happens entirely server-side, since an unrecognized kind's
// requestApproval shape isn't known well enough to forward to the client.
func TestFacade_TurnStart_UnknownHITLKindAutoRejected(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueRun("run-unknown-hitl",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{
			Type: string(godo.HostedAgentEventKindHITLRequested),
			Data: json.RawMessage(`{"hitl_id":"hitl-unknown-1","payload":{"category":"permission","kind":"permissions","itemId":"call_1","turnId":"harness-internal-turn-id","startedAtMs":1000}}`),
		},
	)

	_, err := dispatch(t, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "do something requiring elevated permissions"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started (agentMessage)

	res := h.NextHITLResolution(t, 2*time.Second)
	assert.Equal(t, "hitl-unknown-1", res.RequestID)
	assert.Equal(t, string(godo.HostedAgentHITLOutcomeReject), res.Outcome)
	assert.Equal(t, string(godo.HostedAgentResolutionSourceOutOfBand), res.Source)

	// No request should ever reach the client for a kind this facade never
	// learned how to forward.
	select {
	case req := <-rec.reqCh:
		t.Fatalf("expected no client-facing request for an unknown HITL kind, got %q", req.method)
	case <-time.After(100 * time.Millisecond):
	}
}

// stubReconnectSleep replaces reconnectSleepFn for the duration of a test
// with one that never actually sleeps (always returns true immediately) and
// counts how many times it was called — letting a test exercise several
// reconnect attempts without waiting through real backoff durations, while
// still being able to assert on how many reconnect attempts happened.
// Restores the original via t.Cleanup.
func stubReconnectSleep(t *testing.T) *int32 {
	t.Helper()
	var calls int32
	restore := setReconnectSleepForTest(func(ctx context.Context, d time.Duration) bool {
		atomic.AddInt32(&calls, 1)
		return true
	})
	t.Cleanup(restore)
	return &calls
}

// TestFacade_RunEventLoop_TerminalStreamErrorFailsTurnStartImmediately
// confirms a terminal StreamSession error (401/403/404/409 — auth, gone, or
// a conflicting single-connection consumer) fails turn/start itself via
// ensureEventLoop, rather than returning an in-progress turn nothing will
// ever finish — and, critically, without ever retrying: sleepCalls must stay
// 0, proving the open failure short-circuits before the reconnect loop runs.
func TestFacade_RunEventLoop_TerminalStreamErrorFailsTurnStartImmediately(t *testing.T) {
	sleepCalls := stubReconnectSleep(t)

	f, h, _ := newTestFacade(t)
	h.SetStreamErrorStatus(404, -1)  // permanent
	h.QueueRun("run-terminal-error") // never actually served

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := dispatchCtx(t, ctx, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.Error(t, err)

	var rpcErr *agentproxy.RPCError
	require.ErrorAs(t, err, &rpcErr)
	assert.Equal(t, -32000, rpcErr.Code)
	assert.Contains(t, rpcErr.Message, "opening event stream failed")

	assert.Equal(t, int32(0), atomic.LoadInt32(sleepCalls), "a terminal open error must fail turn/start immediately, never retry")
}

// TestFacade_FailAllTrackedTurns_ConcurrentTurnsWrite is a regression test:
// failAllTrackedTurns used to alias f.turns rather than copy it under f.mu,
// then range the live map while finishTurn deleted from it — so anything else
// touching f.mu-guarded f.turns meanwhile (a turn/start registering a new
// turn, or a --replay goroutine's own finishTurn) raced the open iterator.
// That's a "concurrent map iteration and map write" fatal error, which no
// amount of locking on only the writer's side prevents. Run under -race.
func TestFacade_FailAllTrackedTurns_ConcurrentTurnsWrite(t *testing.T) {
	f, _, rec := newTestFacade(t)

	// failAllTrackedTurns emits two notifications per turn, well past
	// notifierRecorder's buffer, so drain rather than assert on the sequence
	// — the ordering isn't what's under test here.
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for range rec.ch {
		}
	}()

	f.mu.Lock()
	f.turns = make(map[string]*turnState)
	for i := 0; i < 32; i++ {
		runID := fmt.Sprintf("run-%d", i)
		f.turns[runID] = &turnState{itemID: runID + "-msg"}
	}
	f.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			f.mu.Lock()
			if f.turns != nil {
				f.turns["concurrent"] = &turnState{}
				delete(f.turns, "concurrent")
			}
			f.mu.Unlock()
		}
	}()

	f.failAllTrackedTurns("lost connection to hosted session")
	wg.Wait()

	close(rec.ch)
	<-drained
}

// TestFacade_RunEventLoop_ReconnectsAfterTransientStreamError confirms a
// non-terminal StreamSession error (500, here) on the reconnect path is
// retried rather than immediately failing the turn — and that once the
// harness recovers, the turn proceeds completely normally.
//
// ensureEventLoop opens the first stream synchronously before SendInput, so
// the injected failures start only after that first attach (and a deliberate
// mid-stream drop) — otherwise turn/start itself would fail before any
// reconnect loop ran.
func TestFacade_RunEventLoop_ReconnectsAfterTransientStreamError(t *testing.T) {
	stubReconnectSleep(t)

	f, h, rec := newTestFacade(t)
	h.DropConnectionAfterEvents(1)         // first connection: RunStarted, then clean drop
	h.SetStreamErrorStatusAfter(1, 500, 2) // skip ensureEventLoop's open; fail the next 2 reconnect opens
	h.QueueRun("run-transient-error",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"hi"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	ctx, cancel := context.WithCancel(context.Background())
	// Not deferred: this loop keeps retrying to reconnect after any stream
	// end, even a successful turn's (it doesn't know no more turns are
	// coming) — an orphaned goroutine left running past this test would
	// otherwise race a later test's stubReconnectSleep write (confirmed by
	// -race). Stop it explicitly once this test's own assertions are done,
	// not deferred to the end of the function.

	_, err := dispatchCtx(t, ctx, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started — from the first connection, before the drop
	_ = rec.next(t) // item/started

	delta := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", delta.method)
	assert.Equal(t, "hi", delta.params.(agentMessageDeltaNotification).Delta)

	_ = rec.next(t) // item/completed

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	assert.Equal(t, "completed", turnCompleted.params.(turnCompletedNotification).Turn.Status)

	stopEventLoop(t, f, cancel)
}

// TestFacade_RunEventLoop_ReconnectDedupesRedeliveredPrefix confirms a
// reconnect after a genuine mid-stream drop (see DropConnectionAfterEvents)
// resumes cleanly from the cursor: the facade reconnects with
// replay_from=<last processed event id> and the harness (like the real one)
// serves only what comes after it. The already-processed prefix must not
// repeat turn/started, item/started, or the first delta a second time.
// Dedup is the replay_from contract, NOT id comparison — event ids are
// random and unordered (see drainStream's cursor comment), so the facade
// must never try to detect redelivery by sorting ids.
func TestFacade_RunEventLoop_ReconnectDedupesRedeliveredPrefix(t *testing.T) {
	stubReconnectSleep(t)

	f, h, rec := newTestFacade(t)
	h.DropConnectionAfterEvents(2) // first connection only serves RunStarted + the "A" delta
	h.QueueRun("run-dedup",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"A"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"B"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	ctx, cancel := context.WithCancel(context.Background())
	// Not deferred — see the identical note in
	// TestFacade_RunEventLoop_ReconnectsAfterTransientStreamError.

	_, err := dispatchCtx(t, ctx, f, "turn/start", turnStartParams{
		ThreadID: testSessionID,
		Input:    []userInputItem{{Type: "text", Text: "hi"}},
	})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started (from the first connection)
	_ = rec.next(t) // item/started

	deltaA := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", deltaA.method)
	assert.Equal(t, "A", deltaA.params.(agentMessageDeltaNotification).Delta)

	// The reconnect happens here. Without dedup, the redelivered RunStarted/
	// "A" delta would produce a second turn/started, a second item/started,
	// and a duplicate "A" delta before ever reaching the new "B" delta.
	deltaB := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", deltaB.method, "reconnect must not re-emit turn/started or item/started for the already-processed prefix")
	assert.Equal(t, "B", deltaB.params.(agentMessageDeltaNotification).Delta)

	completed := rec.next(t)
	require.Equal(t, "item/completed", completed.method)
	assert.Equal(t, "AB", completed.params.(itemCompletedNotification).Item.(agentMessageItem).Text)

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	assert.Equal(t, "completed", turnCompleted.params.(turnCompletedNotification).Turn.Status)

	rec.expectNone(t)
	stopEventLoop(t, f, cancel)
}

// TestFacade_Replay_ThreadResume drives thread/resume on a Facade with
// Replay: true against a harness queued with two distinct completed
// historical runs (via QueueReplayHistory, not QueueRun — a replay_only
// fetch, independent of the live turn/start path), confirming each arrives
// as an ordinary turn/started -> item/started -> item/agentMessage/delta ->
// item/completed -> turn/completed sequence, in order, exactly like a live
// turn would produce — the codex protocol has no separate "history" shape.
func TestFacade_Replay_ThreadResume(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.Replay = true

	h.QueueReplayHistory(
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: "hist-1"},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), RunID: "hist-1", Data: json.RawMessage(`{"text":"Hi"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), RunID: "hist-1"},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: "hist-2"},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), RunID: "hist-2", Data: json.RawMessage(`{"text":"Bye"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), RunID: "hist-2"},
	)

	_, err := dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)

	started1 := rec.next(t)
	assert.Equal(t, "turn/started", started1.method)
	assert.Equal(t, "hist-1", started1.params.(turnStartedNotification).Turn.ID)

	_ = rec.next(t) // item/started

	delta1 := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", delta1.method)
	assert.Equal(t, "Hi", delta1.params.(agentMessageDeltaNotification).Delta)

	itemCompleted1 := rec.next(t)
	require.Equal(t, "item/completed", itemCompleted1.method)
	assert.Equal(t, "Hi", itemCompleted1.params.(itemCompletedNotification).Item.(agentMessageItem).Text)

	turnCompleted1 := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted1.method)
	tc1 := turnCompleted1.params.(turnCompletedNotification)
	assert.Equal(t, "hist-1", tc1.Turn.ID)
	assert.Equal(t, "completed", tc1.Turn.Status)

	started2 := rec.next(t)
	assert.Equal(t, "turn/started", started2.method)
	assert.Equal(t, "hist-2", started2.params.(turnStartedNotification).Turn.ID)

	_ = rec.next(t) // item/started

	delta2 := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", delta2.method)
	assert.Equal(t, "Bye", delta2.params.(agentMessageDeltaNotification).Delta)

	itemCompleted2 := rec.next(t)
	require.Equal(t, "item/completed", itemCompleted2.method)

	turnCompleted2 := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted2.method)
	tc2 := turnCompleted2.params.(turnCompletedNotification)
	assert.Equal(t, "hist-2", tc2.Turn.ID)
	assert.Equal(t, "completed", tc2.Turn.Status)

	rec.expectNone(t)
}

// TestFacade_Replay_Disabled confirms Replay: false (the default) never
// touches replay-only history, even when a session has some queued — the
// flag must actually gate the behavior, not just always run it. Asserts
// both "no notifications" and "replay never started": a silent fetch
// failure (e.g. the harness 404ing .../events) would satisfy expectNone
// alone and hide a broken gate.
func TestFacade_Replay_Disabled(t *testing.T) {
	f, h, rec := newTestFacade(t)

	h.QueueReplayHistory(
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: "hist-1"},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), RunID: "hist-1"},
	)

	_, err := dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)

	// Give a wrongly-started replay goroutine a moment to either emit or
	// mark itself in flight — then confirm neither happened.
	time.Sleep(50 * time.Millisecond)
	f.replayMu.Lock()
	replaying, done := f.replaying, f.replayDone
	f.replayMu.Unlock()
	assert.False(t, replaying, "Replay:false must not start replaySessionHistory")
	assert.False(t, done, "Replay:false must not mark replayDone")
	rec.expectNone(t)
}

// TestFacade_Replay_FiresOnlyOnce confirms replay runs at most once per
// Facade instance: a second thread/resume call (modeling a client
// reconnecting to an already-bootstrapped proxy) must not replay the same
// history again.
func TestFacade_Replay_FiresOnlyOnce(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.Replay = true

	h.QueueReplayHistory(
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: "hist-1"},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), RunID: "hist-1"},
	)

	_, err := dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started
	_ = rec.next(t) // item/completed
	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	rec.expectNone(t)

	_, err = dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)
	rec.expectNone(t)
}

// TestFacade_Replay_UnaffectedByConcurrentLiveTurnsReset is a regression
// test: replaySessionHistory used to look its turnStates up in the shared
// f.turns map, which runEventLoop's own exit path unconditionally nils out
// on every exit (including an ordinary "no more live turns" exit) — so a
// live turn finishing while a --replay fetch was still mid-flight could
// silently wipe replay's accumulated state. replay now keeps its own local
// map, so manipulating f.turns concurrently (simulating exactly what
// runEventLoop's defer does) must have no effect on an in-flight replay.
func TestFacade_Replay_UnaffectedByConcurrentLiveTurnsReset(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.Replay = true

	h.QueueReplayHistory(
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: "hist-1"},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), RunID: "hist-1", Data: json.RawMessage(`{"text":"Hi"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), RunID: "hist-1"},
	)

	_, err := dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)

	// Simulate a live turn's runEventLoop exiting concurrently, mid-replay.
	f.mu.Lock()
	f.turns = nil
	f.mu.Unlock()

	started := rec.next(t)
	assert.Equal(t, "turn/started", started.method)
	_ = rec.next(t) // item/started

	delta := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", delta.method)
	assert.Equal(t, "Hi", delta.params.(agentMessageDeltaNotification).Delta)

	itemCompleted := rec.next(t)
	require.Equal(t, "item/completed", itemCompleted.method)
	assert.Equal(t, "Hi", itemCompleted.params.(itemCompletedNotification).Item.(agentMessageItem).Text)

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	assert.Equal(t, "completed", turnCompleted.params.(turnCompletedNotification).Turn.Status)
}

// TestFacade_Replay_RetriesAfterAbortedAttempt confirms an aborted replay
// attempt (the replay-only StreamSession call itself failing) does not
// permanently foreclose --replay for the rest of this Facade's lifetime: a
// later thread/resume (e.g. after a reconnect) must retry, not silently
// no-op forever the way a plain sync.Once would have.
//
// The first attempt must fail for a reason that only hits once the
// data-plane .../events route is actually registered (injected 500) —
// a missing-route 404 would also unwind without marking replayDone and
// make this half of the test pass for the wrong reason.
func TestFacade_Replay_RetriesAfterAbortedAttempt(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.Replay = true

	// First attempt: the replay-only StreamSession call itself fails.
	h.SetStreamErrorStatus(500, 1)

	_, err := dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		f.replayMu.Lock()
		defer f.replayMu.Unlock()
		return !f.replaying && !f.replayDone
	}, 2*time.Second, 10*time.Millisecond, "an aborted replay attempt should unwind without ever marking replayDone")

	rec.expectNone(t)

	// Second attempt: the stream error has cleared and real history is
	// queued, so this time replay should actually succeed.
	h.QueueReplayHistory(
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: "hist-1"},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), RunID: "hist-1"},
	)

	_, err = dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)

	started := rec.next(t)
	assert.Equal(t, "turn/started", started.method)
	_ = rec.next(t) // item/started
	_ = rec.next(t) // item/completed
	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)

	require.Eventually(t, func() bool {
		f.replayMu.Lock()
		defer f.replayMu.Unlock()
		return f.replayDone
	}, 2*time.Second, 10*time.Millisecond, "a successfully completed replay should mark replayDone")
}

// TestFacade_Replay_AdoptsInProgressRunAndStreamsContinuation is the
// "proxy killed and restarted mid-turn" regression test: a run that history
// shows started but never finished must be adopted by the live event loop
// once the replay completes — attached at the replay's own cursor — so the
// turn's continuation streams into the same item the replay just rendered,
// and the turn completes for real instead of sitting "inProgress" forever.
func TestFacade_Replay_AdoptsInProgressRunAndStreamsContinuation(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.Replay = true

	h.QueueReplayHistory(
		// A finished historical run: must NOT be adopted.
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: "hist-done"},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), RunID: "hist-done"},
		// The run that was mid-flight when the previous proxy died: started,
		// partial text, no terminal event.
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: "run-live"},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), RunID: "run-live", Data: json.RawMessage(`{"text":"Hel"}`)},
	)
	// The continuation the live stream serves after adoption.
	h.QueueRun("run-live",
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindTokenChunk), Data: json.RawMessage(`{"text":"lo"}`)},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted)},
	)

	ctx, cancel := context.WithCancel(context.Background())

	_, err := dispatchCtx(t, ctx, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)

	// Replayed history: the finished run in full…
	require.Equal(t, "turn/started", rec.next(t).method)
	_ = rec.next(t) // item/started (hist-done)
	_ = rec.next(t) // item/completed (hist-done)
	require.Equal(t, "turn/completed", rec.next(t).method)

	// …then the in-flight run's prefix.
	started := rec.next(t)
	require.Equal(t, "turn/started", started.method)
	assert.Equal(t, "run-live", started.params.(turnStartedNotification).Turn.ID)
	_ = rec.next(t) // item/started (run-live)
	deltaHel := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", deltaHel.method)
	assert.Equal(t, "Hel", deltaHel.params.(agentMessageDeltaNotification).Delta)

	// Adoption: the live loop attaches and the SAME turn continues — no
	// second turn/started, no second item/started, text accumulated across
	// the restart boundary.
	deltaLo := rec.next(t)
	require.Equal(t, "item/agentMessage/delta", deltaLo.method,
		"the adopted turn's continuation must not re-announce the turn or item")
	dn := deltaLo.params.(agentMessageDeltaNotification)
	assert.Equal(t, "lo", dn.Delta)
	assert.Equal(t, "run-live-msg", dn.ItemID, "the continuation must append to the item the replay rendered")

	itemCompleted := rec.next(t)
	require.Equal(t, "item/completed", itemCompleted.method)
	assert.Equal(t, "Hello", itemCompleted.params.(itemCompletedNotification).Item.(agentMessageItem).Text,
		"the item's final text must span the restart boundary")

	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	tc := turnCompleted.params.(turnCompletedNotification)
	assert.Equal(t, "run-live", tc.Turn.ID)
	assert.Equal(t, "completed", tc.Turn.Status)

	rec.expectNone(t)

	f.mu.Lock()
	cursor := f.streamCursor
	f.mu.Unlock()
	assert.NotEmpty(t, cursor, "adoption must seed the live cursor from the replay so the continuation resumes without a gap")

	stopEventLoop(t, f, cancel)
}

// TestFacade_Replay_DoesNotAdoptSessionParentRun: OHR establishes the
// session-wide multi-turn run with the session id as its run id, and that
// run has no terminal event while the session lives — so it always looks
// "in progress" in history. Adopting it would hold the live event loop open
// (and reconnecting) forever on a turn that can never complete.
func TestFacade_Replay_DoesNotAdoptSessionParentRun(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.Replay = true

	h.QueueReplayHistory(
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: testSessionID},
	)

	_, err := dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)

	// The parent run's replayed events still render (pre-existing behavior);
	// what must NOT happen is adoption.
	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started

	require.Eventually(t, func() bool {
		f.replayMu.Lock()
		defer f.replayMu.Unlock()
		return f.replayDone
	}, 2*time.Second, 10*time.Millisecond)

	f.mu.Lock()
	streamStarted := f.streamStarted
	tracked := len(f.turns)
	f.mu.Unlock()
	assert.False(t, streamStarted, "the parent run must not start a live event loop")
	assert.Zero(t, tracked, "the parent run must not be adopted into f.turns")
}

// TestFacade_Replay_HistoricalHITLsNotReDriven confirms --replay does not
// re-prompt the client or re-POST ResolveHITL for approvals already settled
// in durable history. translateEvent used to treat historical
// human_input_requested like a live one: command_execution opened a modal
// for a command that already ran, and file_change/unknown kinds POSTed
// ResolveHITL against hitl_ids the harness closed long ago.
func TestFacade_Replay_HistoricalHITLsNotReDriven(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.Replay = true

	h.QueueReplayHistory(
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: "hist-hitl"},
		// command_execution that was already approved historically
		agentproxytest.Event{
			Type:  string(godo.HostedAgentEventKindToolCallStarted),
			RunID: "hist-hitl",
			Data:  json.RawMessage(`{"tool_call_id":"call_cmd","name":"command_execution","input":{"command":"rm -rf /tmp/x","cwd":"/workspace"}}`),
		},
		agentproxytest.Event{
			Type:  string(godo.HostedAgentEventKindHITLRequested),
			RunID: "hist-hitl",
			Data:  json.RawMessage(`{"hitl_id":"hitl-old-cmd","payload":{"kind":"command_execution","itemId":"call_cmd","turnId":"harness-internal","startedAtMs":1000,"environmentId":"local","command":"rm -rf /tmp/x","cwd":"/workspace","commandActions":[]}}`),
		},
		agentproxytest.Event{
			Type:  string(godo.HostedAgentEventKindHITLResolved),
			RunID: "hist-hitl",
			Data:  json.RawMessage(`{"hitl_id":"hitl-old-cmd","outcome":1}`), // APPROVE
		},
		agentproxytest.Event{
			Type:  string(godo.HostedAgentEventKindToolCallCompleted),
			RunID: "hist-hitl",
			Data:  json.RawMessage(`{"tool_call_id":"call_cmd","ok":true,"duration_ms":5,"summary":"removed"}`),
		},
		// file_change that was already rejected historically
		agentproxytest.Event{
			Type:  string(godo.HostedAgentEventKindToolCallStarted),
			RunID: "hist-hitl",
			Data:  json.RawMessage(`{"tool_call_id":"call_fc","name":"file_change","input":{}}`),
		},
		agentproxytest.Event{
			Type:  string(godo.HostedAgentEventKindHITLRequested),
			RunID: "hist-hitl",
			Data:  json.RawMessage(`{"hitl_id":"hitl-old-1","payload":{"category":"permission","grantRoot":null,"itemId":"call_fc","kind":"file_change","reason":null,"startedAtMs":1000,"turnId":"harness-internal"}}`),
		},
		agentproxytest.Event{
			Type:  string(godo.HostedAgentEventKindHITLResolved),
			RunID: "hist-hitl",
			Data:  json.RawMessage(`{"hitl_id":"hitl-old-1","outcome":2}`), // REJECT
		},
		agentproxytest.Event{
			Type:  string(godo.HostedAgentEventKindToolCallCompleted),
			RunID: "hist-hitl",
			Data:  json.RawMessage(`{"tool_call_id":"call_fc","ok":false,"duration_ms":1,"summary":""}`),
		},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), RunID: "hist-hitl"},
	)

	_, err := dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started (agentMessage)

	cmdStarted := rec.next(t)
	require.Equal(t, "item/started", cmdStarted.method)
	assert.Equal(t, "commandExecution", cmdStarted.params.(itemStartedNotification).Item.(commandExecutionItem).Type)

	cmdCompleted := rec.next(t)
	require.Equal(t, "item/completed", cmdCompleted.method)
	assert.Equal(t, "completed", cmdCompleted.params.(itemCompletedNotification).Item.(commandExecutionItem).Status)

	fcStarted := rec.next(t)
	require.Equal(t, "item/started", fcStarted.method)
	assert.Equal(t, "fileChange", fcStarted.params.(itemStartedNotification).Item.(fileChangeItem).Type)

	fcCompleted := rec.next(t)
	require.Equal(t, "item/completed", fcCompleted.method)
	assert.Equal(t, "declined", fcCompleted.params.(itemCompletedNotification).Item.(fileChangeItem).Status,
		"historical file_change reject must still render as declined, not failed")

	_ = rec.next(t) // item/completed agentMessage
	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)

	rec.expectNone(t)
	rec.expectNoRequest(t)
	h.ExpectNoHITLResolution(t, 50*time.Millisecond)

	require.Eventually(t, func() bool {
		f.replayMu.Lock()
		defer f.replayMu.Unlock()
		return f.replayDone
	}, 2*time.Second, 10*time.Millisecond)
}

// TestFacade_Replay_UsageDoesNotInflateLiveTotal confirms historical
// run.usage_recorded events from --replay do not emit
// thread/tokenUsage/updated or accumulate into f.totalUsage — otherwise a
// reconnecting client's live total would include (and re-include) history.
func TestFacade_Replay_UsageDoesNotInflateLiveTotal(t *testing.T) {
	f, h, rec := newTestFacade(t)
	f.Replay = true

	h.QueueReplayHistory(
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunStarted), RunID: "hist-usage"},
		agentproxytest.Event{
			Type:  string(godo.HostedAgentEventKindRunUsageRecorded),
			RunID: "hist-usage",
			Data:  json.RawMessage(`{"usage":{"input_tokens":100,"output_tokens":20,"cached_input_tokens":0,"reasoning_tokens":0}}`),
		},
		agentproxytest.Event{Type: string(godo.HostedAgentEventKindRunCompleted), RunID: "hist-usage"},
	)

	_, err := dispatch(t, f, "thread/resume", threadResumeParams{ThreadID: testSessionID})
	require.NoError(t, err)

	_ = rec.next(t) // turn/started
	_ = rec.next(t) // item/started
	_ = rec.next(t) // item/completed
	turnCompleted := rec.next(t)
	require.Equal(t, "turn/completed", turnCompleted.method)
	rec.expectNone(t)

	f.mu.Lock()
	total := f.totalUsage
	f.mu.Unlock()
	assert.Equal(t, tokenUsageBreakdown{}, total, "replayed usage must not inflate the live thread total")
}
