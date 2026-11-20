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
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// wakeAPIErr builds the error shape the godo client returns for an HTTP status.
func wakeAPIErr(status int, msg string) error {
	return &godo.ErrorResponse{
		Response: &http.Response{StatusCode: status, Request: &http.Request{Method: http.MethodPost, URL: &url.URL{Path: "/"}}},
		Message:  msg,
	}
}

// syncBuffer is a buffer the stage goroutine can poll while the progress loop
// writes to it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// fakeWakeClock makes the progress clock and the repeat interval fast, and
// returns a function that moves the clock forward.
func fakeWakeClock(t *testing.T) (advance func(time.Duration)) {
	t.Helper()
	var offset atomic.Int64
	base := time.Unix(1_700_000_000, 0)

	prevClock, prevInterval := creationClock, wakeProgressInterval
	creationClock = func() time.Time { return base.Add(time.Duration(offset.Load())) }
	wakeProgressInterval = 2 * time.Millisecond
	t.Cleanup(func() { creationClock, wakeProgressInterval = prevClock, prevInterval })
	return func(d time.Duration) { offset.Add(int64(d)) }
}

func fastPoll(t *testing.T) {
	t.Helper()
	prev := sessionReadyPollInterval
	sessionReadyPollInterval = time.Millisecond
	t.Cleanup(func() { sessionReadyPollInterval = prev })
}

func TestWakeSessionWithProgress_NonTerminalLines(t *testing.T) {
	advance := fakeWakeClock(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out syncBuffer
		tm.hostedAgents.EXPECT().ResumeSession("sess_1").DoAndReturn(func(string) error {
			// Hold the call open until the repeat line has been seen a few times.
			deadline := time.Now().Add(5 * time.Second)
			for strings.Count(out.String(), "waking the session") < 3 && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			advance(22 * time.Second)
			return nil
		})

		prog := newCreationProgress(&out)
		woke, err := wakeSessionWithProgress(context.Background(), tm.hostedAgents, "sess_1", prog)
		require.NoError(t, err)
		assert.True(t, woke)

		got := out.String()
		assert.True(t, strings.HasPrefix(got, "… waking the session (0s)\n"), got)
		assert.GreaterOrEqual(t, strings.Count(got, "waking the session"), 3, "one line at the start plus repeats")
		assert.True(t, strings.HasSuffix(got, "✓ Session awake (22s)\n"), got)
		assert.NotContains(t, got, "\r", "no in-place redraw off a terminal")
		assert.Nil(t, prog.spin)
	})
}

// c.Out is always wrapped in a reportingWriter, which must not hide a terminal.
func TestUnderlyingWriterSeesThroughReportingWriter(t *testing.T) {
	var buf bytes.Buffer
	assert.Same(t, &buf, underlyingWriter(&reportingWriter{out: &buf}))
	assert.Same(t, os.Stdout, underlyingWriter(&reportingWriter{out: &reportingWriter{out: os.Stdout}}))
	assert.False(t, isTerminalWriter(&reportingWriter{out: &buf}))
}

func TestWakeSessionWithProgress_NilProgressIsSilent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().ResumeSession("sess_1").Return(nil)
		woke, err := wakeSessionWithProgress(context.Background(), tm.hostedAgents, "sess_1", nil)
		require.NoError(t, err)
		assert.True(t, woke)
	})
}

// "Already running" means someone else is waking the session: wait for it to
// leave PAUSED, without claiming this run woke it.
func TestWakeSessionWithProgress_AlreadyRunningWaitsForSession(t *testing.T) {
	fastPoll(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		paused, ready := pausedAndReady()
		tm.hostedAgents.EXPECT().ResumeSession("sess_1").Return(wakeAPIErr(http.StatusConflict, "session is already running"))
		gomock.InOrder(
			tm.hostedAgents.EXPECT().GetSession("sess_1").Return(paused, nil),
			tm.hostedAgents.EXPECT().GetSession("sess_1").Return(ready, nil),
		)

		var buf bytes.Buffer
		woke, err := wakeSessionWithProgress(context.Background(), tm.hostedAgents, "sess_1", newCreationProgress(&buf))
		require.NoError(t, err)
		assert.False(t, woke)
		assert.Contains(t, buf.String(), "waiting for the session to wake")
		assert.NotContains(t, buf.String(), "Session awake")
	})
}

func TestWakeSessionWithProgress_AlreadyRunningStillPausedFails(t *testing.T) {
	fastPoll(t)
	prev := wakePollTimeout
	wakePollTimeout = 30 * time.Millisecond
	t.Cleanup(func() { wakePollTimeout = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		paused, _ := pausedAndReady()
		tm.hostedAgents.EXPECT().ResumeSession("sess_1").Return(wakeAPIErr(http.StatusConflict, "session is already running"))
		tm.hostedAgents.EXPECT().GetSession("sess_1").Return(paused, nil).AnyTimes()

		woke, err := wakeSessionWithProgress(context.Background(), tm.hostedAgents, "sess_1", nil)
		assert.False(t, woke)
		require.Error(t, err)
		assert.Equal(t, "the session could not be woken: it did not wake within 30ms. It is still paused; try again.", err.Error())
	})
}

func TestWakeSessionWithProgress_OtherConflictIsAnError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().ResumeSession("sess_1").Return(wakeAPIErr(http.StatusConflict, "session is destroyed"))

		var buf bytes.Buffer
		woke, err := wakeSessionWithProgress(context.Background(), tm.hostedAgents, "sess_1", newCreationProgress(&buf))
		assert.False(t, woke)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "could not be woken", "a 4xx is not fixed by trying again")
		assert.NotContains(t, buf.String(), "Session awake")
	})
}

func TestWakeSessionWithProgress_FailureMessages(t *testing.T) {
	const want = "the session could not be woken: %s. It is still paused; try again."
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"server error", wakeAPIErr(http.StatusBadGateway, "sandbox did not start."), "sandbox did not start"},
		{"transport error", errors.New("connection reset"), "connection reset"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.hostedAgents.EXPECT().ResumeSession("sess_1").Return(tc.err)
				_, err := wakeSessionWithProgress(context.Background(), tm.hostedAgents, "sess_1", nil)
				require.Error(t, err)
				assert.Equal(t, strings.Replace(want, "%s", tc.want, 1), err.Error())
			})
		})
	}
}

// Auth, not-found and billing failures keep the titles and tips the rest of
// the agent commands give them.
func TestWakeSessionWithProgress_OtherStatusesKeepTheirHandling(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantTitle string
	}{
		{"unauthorized", wakeAPIErr(http.StatusUnauthorized, "bad token"), "Authentication failed"},
		{"not found", wakeAPIErr(http.StatusNotFound, "no such session"), "Not found"},
		{"payment required", wakeAPIErr(http.StatusPaymentRequired, "balance"), prepayBlockedTitle},
		{"prepay unknown", wakeAPIErr(http.StatusServiceUnavailable, "prepayment gate state unavailable"), prepayUnknownTitle},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.hostedAgents.EXPECT().ResumeSession("sess_1").Return(tc.err)
				_, err := wakeSessionWithProgress(context.Background(), tm.hostedAgents, "sess_1", nil)
				require.Error(t, err)
				assert.NotContains(t, err.Error(), "could not be woken")

				var ape *agentPrettyError
				require.True(t, errors.As(beautifyAgentError(err), &ape))
				assert.Equal(t, tc.wantTitle, ape.title)
			})
		})
	}
}

func TestWakeSessionWithProgress_DeadlineExceeded(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })
		tm.hostedAgents.EXPECT().ResumeSession("sess_1").DoAndReturn(func(string) error {
			<-release
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		var buf bytes.Buffer
		woke, err := wakeSessionWithProgress(ctx, tm.hostedAgents, "sess_1", newCreationProgress(&buf))
		assert.False(t, woke)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.NotContains(t, buf.String(), "Session awake")
	})
}

// A failure that arrives after the deadline is the deadline, not the failure.
func TestRunStage_ErrorAfterDeadlineReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	err := runStage(ctx, nil, "stage", func() error {
		cancel()
		return errors.New("late failure")
	})
	assert.ErrorIs(t, err, context.Canceled)
}

// --- prompt ----------------------------------------------------------------

func pausedAndReady() (paused, ready *do.HostedAgentSession) {
	paused = &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
		SessionID: testPromptSessionID, Status: godo.HostedAgentSessionStatusPaused,
	}}
	ready = &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
		SessionID: testPromptSessionID, Status: godo.HostedAgentSessionStatusReady,
	}}
	return paused, ready
}

// expectPausedPrompt registers the paused read before promptStream's
// open-ended ready answer, which would otherwise match first.
func expectPausedPrompt(t *testing.T, tm *tcMocks) {
	t.Helper()
	paused, _ := pausedAndReady()
	tm.hostedAgents.EXPECT().GetSession(testPromptSessionID).Return(paused, nil).Times(1)
	promptStream(t, tm, completedRun("Paris"))
}

func TestAgentsPromptShowsWakeAndSendStages(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, stderr, _ := capturePrompt(t, config)
		expectPausedPrompt(t, tm)
		tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).Return(nil)
		expectSendInput(tm, "hi")

		config.Args = []string{testPromptSessionID, "hi"}
		require.NoError(t, RunAgentsPrompt(config))

		assert.Equal(t, "Paris\n", stdout.String(), "stdout carries only the answer")
		out := stderr.String()
		wake := strings.Index(out, "waking the session")
		awake := strings.Index(out, "Session awake")
		send := strings.Index(out, "sending the prompt")
		started := strings.Index(out, "run "+testPromptRunID+" started")
		assert.True(t, 0 <= wake && wake < awake && awake < send && send < started, out)
		assert.NotContains(t, out, "Prompt sent")
	})
}

func TestAgentsPromptSendFailureAfterWake(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, stderr, _ := capturePrompt(t, config)
		expectPausedPrompt(t, tm)
		tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).Return(nil)
		tm.hostedAgents.EXPECT().SendInput(testPromptSessionID, gomock.Any()).
			Return(nil, wakeAPIErr(http.StatusInternalServerError, "boom"))

		config.Args = []string{testPromptSessionID, "hi"}
		err := RunAgentsPrompt(config)
		require.Error(t, err)
		assert.Equal(t,
			"the prompt was not delivered: the agent in the session did not start listening in time. Send it again.",
			err.Error())
		assert.Contains(t, stderr.String(), "sending the prompt")
		assert.Empty(t, stdout.String())
	})
}

// Only a 500 after a wake means "the agent was not listening yet".
func TestAgentsPromptOtherSendFailureAfterWakeIsOrdinary(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		capturePrompt(t, config)
		expectPausedPrompt(t, tm)
		tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).Return(nil)
		tm.hostedAgents.EXPECT().SendInput(testPromptSessionID, gomock.Any()).
			Return(nil, wakeAPIErr(http.StatusBadGateway, "bad gateway"))

		config.Args = []string{testPromptSessionID, "hi"}
		err := RunAgentsPrompt(config)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "not delivered")
	})
}

// Without a wake, a 500 is an ordinary API error.
func TestAgentsPromptSendFailureWithoutWake(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		capturePrompt(t, config)
		promptStream(t, tm, completedRun("Paris"))
		tm.hostedAgents.EXPECT().SendInput(testPromptSessionID, gomock.Any()).
			Return(nil, wakeAPIErr(http.StatusInternalServerError, "boom"))

		config.Args = []string{testPromptSessionID, "hi"}
		err := RunAgentsPrompt(config)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "not delivered")
	})
}

func TestAgentsPromptQuietHidesProgressKeepsFailure(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, stderr, _ := capturePrompt(t, config)
		config.Doit.Set(config.NS, doctl.ArgAgentPromptQuiet, true)
		expectPausedPrompt(t, tm)
		tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).Return(nil)
		tm.hostedAgents.EXPECT().SendInput(testPromptSessionID, gomock.Any()).
			Return(nil, wakeAPIErr(http.StatusInternalServerError, "boom"))

		config.Args = []string{testPromptSessionID, "hi"}
		err := RunAgentsPrompt(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not delivered")
		assert.Empty(t, stderr.String())
	})
}

func TestAgentsPromptWakeFailure(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, _, exitCode := capturePrompt(t, config)
		expectPausedPrompt(t, tm)
		tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).Return(wakeAPIErr(http.StatusBadGateway, "no capacity"))

		config.Args = []string{testPromptSessionID, "hi"}
		err := RunAgentsPrompt(config)
		require.Error(t, err)
		assert.Equal(t, "the session could not be woken: no capacity. It is still paused; try again.", err.Error())
		assert.Equal(t, noExit, *exitCode)
	})
}

// With no --timeout and no Ctrl-C, a failure that merely wraps a context error
// is still just a failure: it must not exit 124.
func TestAgentsPromptWakeTransportErrorWithoutTimeoutIsNotA124(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, _, exitCode := capturePrompt(t, config)
		expectPausedPrompt(t, tm)
		tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).
			Return(&wrappedErr{context.DeadlineExceeded})

		config.Args = []string{testPromptSessionID, "hi"}
		require.Error(t, RunAgentsPrompt(config))
		assert.Equal(t, noExit, *exitCode)
	})
}

type wrappedErr struct{ cause error }

func (e *wrappedErr) Error() string { return "request failed: " + e.cause.Error() }
func (e *wrappedErr) Unwrap() error { return e.cause }

func TestAgentsPromptWakeDeadlineExits124(t *testing.T) {
	for _, quiet := range []bool{false, true} {
		name := "default"
		if quiet {
			name = "quiet"
		}
		t.Run(name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				stdout, stderr, exitCode := capturePrompt(t, config)
				config.Doit.Set(config.NS, doctl.ArgAgentPromptTimeout, 1)
				config.Doit.Set(config.NS, doctl.ArgAgentPromptQuiet, quiet)
				release := make(chan struct{})
				t.Cleanup(func() { close(release) })
				expectPausedPrompt(t, tm)
				tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).DoAndReturn(func(string) error {
					<-release
					return nil
				})

				config.Args = []string{testPromptSessionID, "hi"}
				assert.ErrorIs(t, RunAgentsPrompt(config), ErrExitSilently)
				assert.Equal(t, promptTimeoutExit, *exitCode)
				assert.Empty(t, stdout.String())
				assert.Contains(t, stderr.String(), "timed out waiting for the session to wake")
			})
		})
	}
}

func TestAgentsPromptWakeDeadlineJSON(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, _, exitCode := capturePrompt(t, config)
		config.Doit.Set(config.NS, doctl.ArgAgentPromptTimeout, 1)
		prev := Output
		Output = "json"
		t.Cleanup(func() { Output = prev })
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })
		expectPausedPrompt(t, tm)
		tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).DoAndReturn(func(string) error {
			<-release
			return nil
		})

		config.Args = []string{testPromptSessionID, "hi"}
		assert.ErrorIs(t, RunAgentsPrompt(config), ErrExitSilently)
		assert.Equal(t, promptTimeoutExit, *exitCode)
		assert.Contains(t, stdout.String(), `"status": "timed_out"`)
	})
}

// stubPromptSignals lets a test deliver a signal to the running prompt.
func stubPromptSignals(t *testing.T) <-chan chan<- os.Signal {
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

func TestAgentsPromptCtrlCWhileWakingExits130(t *testing.T) {
	for _, structured := range []bool{false, true} {
		name := "text"
		if structured {
			name = "json"
		}
		t.Run(name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				stdout, stderr, exitCode := capturePrompt(t, config)
				sigs := stubPromptSignals(t)
				if structured {
					prev := Output
					Output = "json"
					t.Cleanup(func() { Output = prev })
				}
				release := make(chan struct{})
				t.Cleanup(func() { close(release) })
				expectPausedPrompt(t, tm)
				tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).DoAndReturn(func(string) error {
					(<-sigs) <- os.Interrupt
					<-release
					return nil
				})

				config.Args = []string{testPromptSessionID, "hi"}
				assert.ErrorIs(t, RunAgentsPrompt(config), ErrExitSilently)
				assert.Equal(t, promptInterruptExit, *exitCode)
				if structured {
					assert.Contains(t, stdout.String(), `"status": "cancelled"`)
				} else {
					assert.Contains(t, stderr.String(), "interrupted")
				}
			})
		})
	}
}

// SIGTERM keeps the exit status it always had (1), unlike Ctrl-C.
func TestAgentsPromptSigtermWhileWakingExits1(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, _, exitCode := capturePrompt(t, config)
		sigs := stubPromptSignals(t)
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })
		expectPausedPrompt(t, tm)
		tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).DoAndReturn(func(string) error {
			(<-sigs) <- syscall.SIGTERM
			<-release
			return nil
		})

		config.Args = []string{testPromptSessionID, "hi"}
		assert.ErrorIs(t, RunAgentsPrompt(config), ErrExitSilently)
		assert.Equal(t, 1, *exitCode)
	})
}

// Ctrl-C while the prompt is being sent says the prompt may not have arrived.
func TestAgentsPromptCtrlCDuringSend(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, stderr, exitCode := capturePrompt(t, config)
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })
		tm.hostedAgents.EXPECT().SendInput(testPromptSessionID, gomock.Any()).
			DoAndReturn(func(string, *godo.HostedAgentSendInputRequest) (*godo.HostedAgentSendInputResponse, error) {
				<-release
				return nil, errors.New("late")
			})

		ctx, cancel := context.WithCancel(context.Background())
		col := &promptCollector{svc: tm.hostedAgents, sessionID: testPromptSessionID, text: "hi", ctx: ctx, sigInt: true}
		go func() { time.Sleep(20 * time.Millisecond); cancel() }()

		done, err := col.drain(nil)
		require.NoError(t, err)
		assert.False(t, done)

		require.ErrorIs(t, col.report(config, ctx), ErrExitSilently)
		assert.Equal(t, promptInterruptExit, *exitCode)
		assert.Contains(t, stderr.String(), "before the prompt was delivered")
		assert.NotContains(t, stderr.String(), "did not finish")
	})
}

func TestPromptResumeIfPausedDeadline(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		capturePrompt(t, config)
		paused, _ := pausedAndReady()
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })
		tm.hostedAgents.EXPECT().GetSession(testPromptSessionID).Return(paused, nil)
		tm.hostedAgents.EXPECT().ResumeSession(testPromptSessionID).DoAndReturn(func(string) error {
			<-release
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		woke, err := promptResumeIfPaused(ctx, tm.hostedAgents, testPromptSessionID, false)
		assert.False(t, woke)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

// A send that outlasts --timeout ends the command with the timeout status and
// says the prompt may not have arrived, rather than claiming a run was left going.
func TestAgentsPromptSendDeadline(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, stderr, exitCode := capturePrompt(t, config)
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })
		tm.hostedAgents.EXPECT().SendInput(testPromptSessionID, gomock.Any()).
			DoAndReturn(func(string, *godo.HostedAgentSendInputRequest) (*godo.HostedAgentSendInputResponse, error) {
				<-release
				return nil, errors.New("late")
			})

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		col := &promptCollector{svc: tm.hostedAgents, sessionID: testPromptSessionID, text: "hi", ctx: ctx, quiet: true}

		done, err := col.drain(nil)
		require.NoError(t, err)
		assert.False(t, done)

		<-ctx.Done()
		require.ErrorIs(t, col.report(config, ctx), ErrExitSilently)
		assert.Equal(t, promptTimeoutExit, *exitCode)
		assert.Contains(t, stderr.String(), "before the prompt was delivered")
	})
}

// Only the wake stages see through the reporting wrapper on c.Out; every other
// user of creationProgress keeps its earlier behaviour.
func TestCreationProgressDefaultDoesNotSeeThroughReportingWriter(t *testing.T) {
	var buf bytes.Buffer
	out := &reportingWriter{out: &buf}
	assert.False(t, newCreationProgress(out).seeThrough)
	assert.True(t, newStageProgress(out).seeThrough)
	assert.False(t, isTerminalWriter(&reportingWriter{out: os.Stdout}), "the shared check does not unwrap")

	// Off a terminal both print one line per call.
	p := newCreationProgress(out)
	p.wait("creating")
	assert.Contains(t, buf.String(), "creating")
	assert.Nil(t, p.spin)
}

// A Ctrl-C that lands before the wake or send stages, on a session that was
// never paused, keeps the earlier behaviour: exit 1, "did not finish".
func TestAgentsPromptCtrlCBeforeSendOnReadySessionKeepsOldBehaviour(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, stderr, exitCode := capturePrompt(t, config)
		sigs := stubPromptSignals(t)
		_, ready := pausedAndReady()
		tm.hostedAgents.EXPECT().GetSession(testPromptSessionID).Return(ready, nil).AnyTimes()
		tm.hostedAgents.EXPECT().StreamSession(gomock.Any(), testPromptSessionID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, _ string, _ *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
				(<-sigs) <- os.Interrupt
				<-ctx.Done()
				return nil, ctx.Err()
			})

		config.Args = []string{testPromptSessionID, "hi"}
		assert.ErrorIs(t, RunAgentsPrompt(config), ErrExitSilently)
		assert.Equal(t, 1, *exitCode)
		assert.Contains(t, stderr.String(), "did not finish")
	})
}
