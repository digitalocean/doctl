/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	domocks "github.com/digitalocean/doctl/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const cancelledRunFrame = `{"code":7,"message":"turn cancelled by me@example.com"}`

// stubPromptNotify hands the test the channel RunAgentsPrompt listens on, so
// a Ctrl-C can be delivered without signalling the test process.
func stubPromptNotify(t *testing.T) <-chan chan<- os.Signal {
	t.Helper()
	got := make(chan chan<- os.Signal, 1)
	prev := promptNotify
	promptNotify = func(ch chan<- os.Signal) func() {
		got <- ch
		return func() {}
	}
	t.Cleanup(func() { promptNotify = prev })
	return got
}

// liveSSE serves a stream that stays open: it writes whatever is pushed onto
// frames and ends only when frames closes or the client goes away. streamed
// fires after each pushed frame is flushed.
type liveSSE struct {
	frames   chan string
	streamed chan struct{}
	once     sync.Once
}

func newLiveSSE(t *testing.T) (*liveSSE, *godo.Client) {
	t.Helper()
	l := &liveSSE{frames: make(chan string, 8), streamed: make(chan struct{}, 8)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		for {
			select {
			case <-r.Context().Done():
				return
			case f, ok := <-l.frames:
				if !ok {
					return
				}
				_, _ = io.WriteString(w, f)
				w.(http.Flusher).Flush()
				l.streamed <- struct{}{}
			}
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(l.close)
	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	return l, client
}

func (l *liveSSE) close() { l.once.Do(func() { close(l.frames) }) }

func (l *liveSSE) push(t *testing.T, frame string) {
	t.Helper()
	l.frames <- frame
	select {
	case <-l.streamed:
	case <-time.After(5 * time.Second):
		t.Fatal("frame was never streamed")
	}
}

func wireLivePrompt(t *testing.T, tm *tcMocks, client *godo.Client) {
	t.Helper()
	stubReconnectSleep(t)
	tm.hostedAgents.EXPECT().GetSession(testPromptSessionID).Return(&do.HostedAgentSession{
		HostedAgentSession: &godo.HostedAgentSession{SessionID: testPromptSessionID, Status: godo.HostedAgentSessionStatusReady},
	}, nil).AnyTimes()
	tm.hostedAgents.EXPECT().
		StreamSession(gomock.Any(), testPromptSessionID, gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
			stream, _, err := client.HostedAgents.StreamSession(ctx, testPromptSessionID, opt)
			return stream, err
		}).AnyTimes()
}

func TestAgentsPromptCtrlCCancelsTheRun(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, _, exitCode := capturePrompt(t, config)
		var stderr lockedBuffer
		promptStderr = &stderr
		notify := stubPromptNotify(t)
		live, client := newLiveSSE(t)
		wireLivePrompt(t, tm, client)
		sent := make(chan struct{})
		expectSendInput(tm, "").Do(func(string, *godo.HostedAgentSendInputRequest) { close(sent) })
		tm.hostedAgents.EXPECT().CancelTurn(testPromptSessionID, testPromptRunID).
			DoAndReturn(func(string, string) (do.HostedAgentCancelTurnResult, error) {
				live.frames <- promptFrame("evt-3", testPromptRunID, godo.HostedAgentEventKindRunFailed, cancelledRunFrame)
				return do.HostedAgentCancelTurnResult{Outcome: do.HostedAgentCancelTurnAcked, RunID: testPromptRunID}, nil
			})

		done := make(chan error, 1)
		config.Args = []string{testPromptSessionID, "write an essay"}
		go func() { done <- RunAgentsPrompt(config) }()

		sigs := <-notify
		<-sent
		live.push(t, promptFrame("evt-1", testPromptRunID, godo.HostedAgentEventKindRunStarted, `{}`))
		live.push(t, promptTokenFrame("evt-2", testPromptRunID, "Rome was ", false))
		// The run becomes cancellable on the line after SendInput returns,
		// which nothing observable follows; this settles that one statement.
		time.Sleep(50 * time.Millisecond)
		sigs <- os.Interrupt

		select {
		case err := <-done:
			assert.ErrorIs(t, err, ErrExitSilently)
		case <-time.After(10 * time.Second):
			t.Fatal("prompt did not return after the cancelled run ended")
		}
		assert.Equal(t, promptInterruptExit, *exitCode)
		assert.Equal(t, "Rome was \n", stdout.String(), "what streamed before the cancel is still the answer")
		assert.Contains(t, stderr.String(), "cancelling run "+testPromptRunID)
		assert.Contains(t, stderr.String(), "run cancelled: turn cancelled by me@example.com")
	})
}

func TestAgentsPromptRunCancelledElsewhereIsAFailure(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, stderr, exitCode := capturePrompt(t, config)
		promptStream(t, tm, promptFrame("evt-1", testPromptRunID, godo.HostedAgentEventKindRunFailed, cancelledRunFrame))
		expectSendInput(tm, "")

		config.Args = []string{testPromptSessionID, "hi"}
		assert.ErrorIs(t, RunAgentsPrompt(config), ErrExitSilently)
		assert.Equal(t, 1, *exitCode, "130 is for this caller's own Ctrl-C only")
		assert.Contains(t, stderr.String(), "run cancelled: turn cancelled by me@example.com")
	})
}

func TestPromptWatchInterrupts(t *testing.T) {
	cases := []struct {
		name        string
		live        string
		signals     []os.Signal
		outcome     do.HostedAgentCancelTurnOutcome
		cancelErr   error
		wantCancel  bool
		wantStopped bool
		wantSent    bool
		wantStderr  string
	}{
		{name: "ctrl-c with a live run cancels and keeps waiting", live: "run-1", signals: []os.Signal{os.Interrupt},
			outcome: do.HostedAgentCancelTurnAcked, wantCancel: true, wantSent: true, wantStderr: "cancelling run run-1"},
		{name: "second ctrl-c stops waiting", live: "run-1", signals: []os.Signal{os.Interrupt, os.Interrupt},
			outcome: do.HostedAgentCancelTurnAcked, wantCancel: true, wantSent: true, wantStopped: true},
		{name: "ctrl-c before the prompt is sent stops waiting", signals: []os.Signal{os.Interrupt}, wantStopped: true},
		{name: "sigterm never cancels", live: "run-1", signals: []os.Signal{syscall.SIGTERM}, wantStopped: true},
		{name: "unsupported leaves the run going", live: "run-1", signals: []os.Signal{os.Interrupt},
			outcome: do.HostedAgentCancelTurnUnsupported, wantCancel: true, wantStopped: true, wantStderr: "can't cancel a run; run run-1 keeps going"},
		{name: "no turn stops waiting", live: "run-1", signals: []os.Signal{os.Interrupt},
			outcome: do.HostedAgentCancelTurnNoTurn, wantCancel: true, wantStopped: true},
		{name: "a failed cancel stops waiting and says why", live: "run-1", signals: []os.Signal{os.Interrupt},
			cancelErr: errors.New("boom"), wantCancel: true, wantStopped: true, wantStderr: "cancelling run run-1 failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stderr lockedBuffer
			prev := promptStderr
			promptStderr = &stderr
			t.Cleanup(func() { promptStderr = prev })

			ctrl := gomock.NewController(t)
			svc := domocks.NewMockHostedAgentsService(ctrl)
			if tc.wantCancel {
				svc.EXPECT().CancelTurn("sess_x", tc.live).Return(do.HostedAgentCancelTurnResult{Outcome: tc.outcome, RunID: tc.live}, tc.cancelErr)
			}
			col := &promptCollector{svc: svc, sessionID: "sess_x"}
			col.setLive(tc.live)

			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			sigs := make(chan os.Signal, len(tc.signals))
			for _, s := range tc.signals {
				sigs <- s
			}
			exited := make(chan struct{})
			go func() { col.watchInterrupts(ctx, sigs, stop); close(exited) }()

			if tc.wantStopped {
				select {
				case <-ctx.Done():
				case <-time.After(5 * time.Second):
					t.Fatal("the wait was never stopped")
				}
			} else {
				select {
				case <-ctx.Done():
					t.Fatal("the wait stopped although the run was cancelled and is ending")
				case <-time.After(100 * time.Millisecond):
				}
				stop()
			}
			<-exited
			assert.Equal(t, tc.wantSent, col.sentCancel())
			if tc.wantStderr != "" {
				assert.Contains(t, stderr.String(), tc.wantStderr)
			}
		})
	}
}

// lockedBuffer is a bytes.Buffer safe to write from the interrupt goroutine
// while the test reads it.
type lockedBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}
