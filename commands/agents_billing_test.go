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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	domocks "github.com/digitalocean/doctl/do/mocks"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// harnessAPIErr builds the error godo produces from a real harness-api
// response body. httpcommon.WriteError emits {"error":{"code":N,"message":…}},
// which leaves ErrorResponse.Message empty and the reason in NestedError — so
// parsing the actual bytes is the only faithful way to construct these.
func harnessAPIErr(status int, message string) error {
	body := fmt.Sprintf(`{"error":{"code":%d,"message":%q}}`, status, message)
	return godo.CheckResponse(&http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Request:    httptest.NewRequest(http.MethodPost, "https://api.digitalocean.com/v2/agents/sessions", nil),
		Body:       io.NopCloser(strings.NewReader(body)),
	})
}

// The exact body harness-api emits when the prepayment gate is engaged.
const (
	prepayBlockedWireMessage = "team is blocked by the prepayment gate; add funds to resume"
	prepayUnknownWireMessage = "prepayment gate state is temporarily unavailable; retry"
)

func TestPrepayBlockedErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"gate 402", harnessAPIErr(http.StatusPaymentRequired, prepayBlockedWireMessage), true},
		// 402 has exactly one source in harness-api, so the status alone is
		// the signal and the message is never matched against.
		{"402 with an unfamiliar message", harnessAPIErr(http.StatusPaymentRequired, "something else"), true},
		{"gate 503", harnessAPIErr(http.StatusServiceUnavailable, prepayUnknownWireMessage), false},
		{"unrelated 409", harnessAPIErr(http.StatusConflict, "run is terminal"), false},
		{"local error", errors.New("boom"), false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, prepayBlockedErr(tc.err))
		})
	}
}

// A prepay card must survive being wrapped, because several commands beautify
// their errors before the central handler sees them.
func TestPrepayBlockedErr_ThroughPrettyError(t *testing.T) {
	wrapped := beautifyAgentError(harnessAPIErr(http.StatusPaymentRequired, prepayBlockedWireMessage))
	assert.True(t, prepayBlockedErr(wrapped))
}

func TestPrepayUnknownErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"gate 503", harnessAPIErr(http.StatusServiceUnavailable, prepayUnknownWireMessage), true},
		// An ordinary outage is not the gate failing closed, and must not be
		// reported as a billing condition.
		{"unrelated 503", harnessAPIErr(http.StatusServiceUnavailable, "upstream connect error"), false},
		{"gate 402", harnessAPIErr(http.StatusPaymentRequired, prepayBlockedWireMessage), false},
		{"local error", errors.New("boom"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, prepayUnknownErr(tc.err))
		})
	}
}

// stubPrepayment satisfies do.PrepaymentService without gomock, so the
// messaging tests can be table-driven over the billing responses.
type stubPrepayment struct {
	cfg    *do.PrepaymentConfig
	status *do.PrepaymentStatus
	err    error
	// gotCtx captures the context the caller passed, so a test can assert the
	// lookup is bounded.
	gotCtx *context.Context
}

func (s stubPrepayment) Get(ctx context.Context) (*do.PrepaymentConfig, *do.PrepaymentStatus, error) {
	if s.gotCtx != nil {
		*s.gotCtx = ctx
	}
	return s.cfg, s.status, s.err
}

func (s stubPrepayment) GetConfig() (*do.PrepaymentConfigResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &do.PrepaymentConfigResponse{
		PrepaymentConfigResponse: &godo.PrepaymentConfigResponse{Config: s.cfg, Status: s.status},
	}, nil
}

func (s stubPrepayment) GetStatus() (*do.PrepaymentStatusResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &do.PrepaymentStatusResponse{
		PrepaymentStatusResponse: &godo.PrepaymentStatusResponse{Status: s.status},
	}, nil
}

func configWithPrepayment(svc do.PrepaymentService) *CmdConfig {
	return &CmdConfig{
		Out:        &bytes.Buffer{},
		Prepayment: func() do.PrepaymentService { return svc },
	}
}

func TestNewPrepayBlockedError_Enrichment(t *testing.T) {
	cause := harnessAPIErr(http.StatusPaymentRequired, prepayBlockedWireMessage)

	t.Run("balance is folded into the reason", func(t *testing.T) {
		c := configWithPrepayment(stubPrepayment{
			status: &do.PrepaymentStatus{Balance: "0.00", Blocked: true, Eligible: true},
		})
		pretty := newPrepayBlockedError(c, cause)

		assert.Equal(t, prepayBlockedTitle, pretty.title)
		assert.Equal(t, http.StatusPaymentRequired, pretty.status)
		assert.Contains(t, pretty.reason, "$0.00")
		assert.Contains(t, pretty.reason, "work is saved")
		assert.Contains(t, pretty.tips, fmt.Sprintf("Add funds at %s", prepayTopUpURL))
	})

	t.Run("auto top-off replaces the manual top-up tip", func(t *testing.T) {
		c := configWithPrepayment(stubPrepayment{
			cfg:    &do.PrepaymentConfig{IsAutoPrepayEnabled: true, PrepayAmount: "25.00", PrepayThreshold: "5.00"},
			status: &do.PrepaymentStatus{Balance: "0.00", IsAutoPrepayEnabled: true, Blocked: true},
		})
		pretty := newPrepayBlockedError(c, cause)

		assert.Contains(t, pretty.reason, "Auto top-off is enabled")
		assert.Contains(t, pretty.tips, "Retry in a moment")
		assert.NotContains(t, pretty.tips, fmt.Sprintf("Add funds at %s", prepayTopUpURL))
	})

	t.Run("no billing permission redirects to the team owner", func(t *testing.T) {
		c := configWithPrepayment(stubPrepayment{err: do.ErrNoBillingPermission})
		pretty := newPrepayBlockedError(c, cause)

		assert.Equal(t, []string{"Ask your Team Owner or Biller to add funds"}, pretty.tips)
		// The caller cannot open the billing page, so it must not be offered.
		assert.NotContains(t, pretty.DisplayError(), prepayTopUpURL)
	})

	t.Run("a billing failure falls back to the plain card", func(t *testing.T) {
		c := configWithPrepayment(stubPrepayment{err: errors.New("billing is down")})
		pretty := newPrepayBlockedError(c, cause)

		assert.Equal(t, prepayBlockedTitle, pretty.title)
		assert.Contains(t, pretty.tips, fmt.Sprintf("Add funds at %s", prepayTopUpURL))
		// A billing lookup must never turn a clean 402 into a double error.
		assert.NotContains(t, pretty.DisplayError(), "billing is down")
	})

	t.Run("an uninitialized config still renders", func(t *testing.T) {
		pretty := newPrepayBlockedError(nil, cause)
		assert.Equal(t, prepayBlockedTitle, pretty.title)
		assert.Contains(t, pretty.reason, "prepayment balance has run out")
	})

	// The user has already earned this error, so a degraded billing API must
	// not be able to hold it back on doctl's usual retry schedule.
	t.Run("the balance lookup is bounded", func(t *testing.T) {
		var got context.Context
		c := configWithPrepayment(stubPrepayment{
			status: &do.PrepaymentStatus{Balance: "0.00"},
			gotCtx: &got,
		})
		newPrepayBlockedError(c, cause)

		require.NotNil(t, got)
		deadline, ok := got.Deadline()
		require.True(t, ok, "an unbounded lookup would stall the card behind billing")
		assert.LessOrEqual(t, time.Until(deadline), prepayLookupTimeout)
	})
}

// The exact session.updated body harness-api emits when the prepayment gate
// pauses a session, minus the envelope sseFrame adds.
const sessionPausedZeroBalanceData = `{"status":"paused","pause_reason":"zero_balance","changed_fields":["status","pause_reason"]}`

func TestSessionUpdatedPayload_announcesPause(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    bool
	}{
		{"the gate's pause", sessionPausedZeroBalanceData, true},
		// The stream spells the status short; the session model uses the enum.
		// Both reach this code, so neither may be the only one recognized.
		{"enum spelling", `{"status":"SESSION_STATUS_PAUSED","pause_reason":"zero_balance"}`, true},
		// backward-compat: legacy servers still emit "low_balance"
		{"legacy low_balance", `{"status":"paused","pause_reason":"low_balance","changed_fields":["status","pause_reason"]}`, true},
		{"idle pause", `{"status":"paused","pause_reason":"idle","changed_fields":["status"]}`, true},
		// A pause with no reason is still a pause, and still the reason the
		// transcript is about to go quiet.
		{"pause with no reason", `{"status":"paused","changed_fields":["status"]}`, true},
		// changed_fields is optional; a paused status is then all we have.
		{"no changed_fields", `{"status":"paused","pause_reason":"zero_balance"}`, true},
		// Every update while paused still carries status=paused. Only the one
		// that changed it is news; the rest would reprint the card forever.
		{"unrelated update while paused", `{"status":"paused","pause_reason":"zero_balance","changed_fields":["name"]}`, false},
		{"running session", `{"status":"ready","changed_fields":["status"]}`, false},
		{"empty payload", `{}`, false},
		{"malformed payload", `{not-json`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var p sessionUpdatedPayload
			_ = json.Unmarshal([]byte(tc.payload), &p)
			assert.Equal(t, tc.want, p.announcesPause())
		})
	}
}

// A session paused between turns announces itself only on session.updated —
// there is no run to pause — so that event has to reach the same balance card
// run.paused does. It used to render a bare "• session updated".
func TestRenderEvent_sessionUpdatedPause(t *testing.T) {
	t.Run("a zero-balance pause renders the balance card", func(t *testing.T) {
		var buf bytes.Buffer
		renderEvent(&buf, godo.HostedAgentEvent{
			Kind:    godo.HostedAgentEventKindSessionUpdated,
			Payload: json.RawMessage(sessionPausedZeroBalanceData),
		})

		out := buf.String()
		assert.Contains(t, out, "Paused — prepayment balance exhausted")
		assert.Contains(t, out, prepayTopUpURL)
		assert.NotContains(t, out, "session updated")
	})

	t.Run("other pause reasons name the session, not the run", func(t *testing.T) {
		var buf bytes.Buffer
		renderEvent(&buf, godo.HostedAgentEvent{
			Kind:    godo.HostedAgentEventKindSessionUpdated,
			Payload: json.RawMessage(`{"status":"paused","pause_reason":"IDLE"}`),
		})
		assert.Contains(t, buf.String(), "session paused (idle)")
	})

	// The ordinary case must stay as quiet as it was.
	t.Run("a non-pause update is still one muted line", func(t *testing.T) {
		var buf bytes.Buffer
		renderEvent(&buf, godo.HostedAgentEvent{
			Kind:    godo.HostedAgentEventKindSessionUpdated,
			Payload: json.RawMessage(`{"status":"ready","changed_fields":["status"]}`),
		})
		assert.Equal(t, "\n• session updated\n", buf.String())
	})
}

// A pause arriving during warm-up used to be folded into the warm-up banner
// like the boot events it shares a code path with. That hid it twice: the
// banner is a transient one-liner, and it is dismissed by the next event —
// which, for a session the gate just stopped, never arrives.
func TestDrainStream_zeroBalancePauseSurvivesWarmup(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindSessionUpdated), sessionPausedZeroBalanceData)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	warmup := newWarmupState(&buf, time.Now())
	warmup.start()

	_, pause := drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), warmup, &tokenDeduper{})

	assert.Contains(t, buf.String(), "Paused — prepayment balance exhausted")
	assert.Equal(t, pauseOutcome{observed: true, reason: "zero_balance"}, pause,
		"the caller needs the reason to explain the stream ending")
}

// The gate announces one pause on both run.paused and session.updated. The
// user is looking at one stalled agent, so they get one card.
func TestDrainStream_pauseIsExplainedOnce(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindRunPaused), `{"reason":"zero_balance"}`) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindSessionUpdated), sessionPausedZeroBalanceData)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	_, pause := drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil, &tokenDeduper{})

	assert.Equal(t, 1, strings.Count(buf.String(), "Paused — prepayment balance exhausted"))
	assert.Equal(t, "zero_balance", pause.reason)
}

// A resume is the session proving it is alive again, so the next pause is a
// new fact rather than a duplicate of the one already on screen.
func TestDrainStream_pauseAfterResumeIsExplainedAgain(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindSessionUpdated), sessionPausedZeroBalanceData) +
		sseFrame("evt-2", string(godo.HostedAgentEventKindRunResumed), `{}`) +
		sseFrame("evt-3", string(godo.HostedAgentEventKindSessionUpdated), sessionPausedZeroBalanceData)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil, &tokenDeduper{})

	assert.Equal(t, 2, strings.Count(buf.String(), "Paused — prepayment balance exhausted"))
}

// A session that is merely running must not leave a pause reason behind, or
// an ordinary mid-stream drop would be reported as a billing problem.
func TestDrainStream_noPauseReportsNothing(t *testing.T) {
	body := sseFrame("evt-1", string(godo.HostedAgentEventKindSessionUpdated), `{}`)
	srv := httptest.NewServer(hostedAgentSSEHandler(body, nil))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	stream := openHostedAgentStream(t, client, nil)
	defer stream.Close()

	var buf bytes.Buffer
	_, pause := drainStream(stream, &buf, &pendingHITL{}, &eventCursor{}, newThinkingState(&buf), nil, &tokenDeduper{})

	assert.False(t, pause.observed, "a session that never paused is no news about pausing")
	assert.Empty(t, pause.reason)
	assert.Contains(t, buf.String(), "session updated")
}

// The whole point of carrying the reason out of drainStream: a stream the gate
// closed used to be reported as "Failed to reconnect to agent activity
// stream.", which sends the user to debug a network that is working and never
// mentions the balance they could top up to fix it.
func TestStreamWithReconnect_lowBalancePauseExplainsTheSilence(t *testing.T) {
	stubReconnectSleep(t)

	var (
		mu    sync.Mutex
		calls int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		calls++
		first := calls == 1
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Only the first connection carries the pause: a real server resumes
		// from the cursor set past it, so later attempts find nothing waiting
		// and drop straight away — which is exactly the silence under test.
		if first {
			_, _ = io.WriteString(w, sseFrame("evt-1",
				string(godo.HostedAgentEventKindSessionUpdated), sessionPausedZeroBalanceData))
		}
		_, _ = io.WriteString(w, "data: {not-json\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	mock := domocks.NewMockHostedAgentsService(ctrl)
	mock.EXPECT().
		StreamSession(gomock.Any(), "sess_x", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
			return openHostedAgentStream(t, client, opt), nil
		}).
		Times(maxAutoReconnectAttempts)

	var buf bytes.Buffer
	streamWithReconnect(context.Background(), mock, "sess_x", &buf, &pendingHITL{},
		&eventCursor{}, newThinkingState(&buf), nil)

	out := buf.String()
	assert.Contains(t, out, "Paused — prepayment balance exhausted")
	assert.Equal(t, 1, strings.Count(out, msgPausedStayingAttached),
		"the reason doctl is still connected is worth saying once, not once per attempt")
	assert.Contains(t, out, msgPausedStoppedWatching)
	assert.NotContains(t, out, msgReconnectFailed,
		"a session the gate stopped is not a session doctl failed to reach")
}

// The balance wording must stay on the balance path: an ordinary drop is still
// a connection problem and still says so.
func TestStreamWithReconnect_ordinaryDropKeepsGenericWording(t *testing.T) {
	stubReconnectSleep(t)

	srv := httptest.NewServer(hostedAgentSSEHandler(
		sseFrame("evt-1", string(godo.HostedAgentEventKindSessionUpdated), `{}`), errors.New("drop")))
	t.Cleanup(srv.Close)

	client, err := godo.New(nil, godo.SetBaseURL(srv.URL+"/"))
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	mock := domocks.NewMockHostedAgentsService(ctrl)
	mock.EXPECT().
		StreamSession(gomock.Any(), "sess_x", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
			return openHostedAgentStream(t, client, opt), nil
		}).
		Times(maxAutoReconnectAttempts)

	var buf bytes.Buffer
	streamWithReconnect(context.Background(), mock, "sess_x", &buf, &pendingHITL{},
		&eventCursor{}, newThinkingState(&buf), nil)

	out := buf.String()
	assert.Contains(t, out, msgReconnectFailed)
	assert.NotContains(t, out, msgPausedStoppedWatching)
	assert.NotContains(t, out, msgPausedStayingAttached)
}

func TestGiveUpMessage(t *testing.T) {
	// A session the gate stopped is not a session doctl failed to reach.
	assert.Equal(t, msgPausedStoppedWatching, giveUpMessage("zero_balance"))
	assert.Equal(t, msgPausedStoppedWatching, giveUpMessage("ZERO_BALANCE"))
	// backward-compat: legacy servers still return "low_balance"
	assert.Equal(t, msgPausedStoppedWatching, giveUpMessage("low_balance"))
	assert.Equal(t, msgPausedStoppedWatching, giveUpMessage("LOW_BALANCE"))
	// Everything else really is a connection we could not hold.
	assert.Equal(t, msgReconnectFailed, giveUpMessage("idle"))
	assert.Equal(t, msgReconnectFailed, giveUpMessage(""))
}

func TestRenderRunPaused(t *testing.T) {
	t.Run("zero balance explains itself", func(t *testing.T) {
		var buf bytes.Buffer
		renderRunPaused(&buf, "zero_balance")

		out := buf.String()
		assert.Contains(t, out, "Paused — prepayment balance exhausted")
		assert.Contains(t, out, "Your work is saved.")
		assert.Contains(t, out, prepayTopUpURL)
	})

	t.Run("legacy low_balance also explains itself (backward-compat)", func(t *testing.T) {
		var buf bytes.Buffer
		renderRunPaused(&buf, "low_balance")

		out := buf.String()
		assert.Contains(t, out, "Paused — prepayment balance exhausted")
		assert.Contains(t, out, prepayTopUpURL)
	})

	t.Run("other reasons pass through", func(t *testing.T) {
		var buf bytes.Buffer
		renderRunPaused(&buf, "INACTIVITY")
		assert.Contains(t, buf.String(), "run paused (inactivity)")
	})

	t.Run("a missing reason still announces the pause", func(t *testing.T) {
		var buf bytes.Buffer
		renderRunPaused(&buf, "")
		assert.Contains(t, buf.String(), "run paused")
	})
}

func TestIsZeroBalancePauseReason(t *testing.T) {
	// The session model and the run-event stream disagree on case, so the
	// comparison tolerates both. The function also accepts the legacy
	// "low_balance" value for backward compatibility.
	assert.True(t, isZeroBalancePauseReason("zero_balance"))
	assert.True(t, isZeroBalancePauseReason("ZERO_BALANCE"))
	assert.True(t, isZeroBalancePauseReason(" zero_balance "))
	// backward-compat: legacy servers still return "low_balance"
	assert.True(t, isZeroBalancePauseReason("low_balance"))
	assert.True(t, isZeroBalancePauseReason("LOW_BALANCE"))
	assert.True(t, isZeroBalancePauseReason(" low_balance "))
	assert.False(t, isZeroBalancePauseReason("idle"))
	assert.False(t, isZeroBalancePauseReason(""))
}

func TestNormalizePauseReason(t *testing.T) {
	// The flag reads better hyphenated than the underscored value it matches,
	// so both spellings have to land on the same reason.
	assert.Equal(t, "zero_balance", normalizePauseReason("zero-balance"))
	assert.Equal(t, "zero_balance", normalizePauseReason("ZERO_BALANCE"))
	assert.Equal(t, "zero_balance", normalizePauseReason(" zero_balance "))
	assert.Equal(t, "zero_balance", normalizePauseReason("low-balance"))
	assert.Equal(t, "zero_balance", normalizePauseReason("LOW_BALANCE"))
	assert.Equal(t, "zero_balance", normalizePauseReason(" low_balance "))
	assert.Equal(t, "idle", normalizePauseReason("Idle"))
	assert.Equal(t, "", normalizePauseReason(""))
}

func TestFilterByPauseReason(t *testing.T) {
	sessions := []do.HostedAgentSession{
		{HostedAgentSession: &godo.HostedAgentSession{SessionID: "running"}},
		{HostedAgentSession: &godo.HostedAgentSession{SessionID: "broke", PauseReason: godo.HostedAgentSessionPauseReasonZeroBalance}},
		{HostedAgentSession: &godo.HostedAgentSession{SessionID: "idle", PauseReason: godo.HostedAgentSessionPauseReasonIdle}},
		{HostedAgentSession: &godo.HostedAgentSession{SessionID: "broke2", PauseReason: godo.HostedAgentSessionPauseReasonZeroBalance}},
	}

	t.Run("matches the hyphenated flag spelling", func(t *testing.T) {
		got := filterByPauseReason(sessions, "zero-balance")
		require.Len(t, got, 2)
		assert.Equal(t, "broke", got[0].SessionID)
		assert.Equal(t, "broke2", got[1].SessionID)
	})

	// backward-compat: old servers still return "low_balance"; scripts using
	// --paused-by low-balance must continue to pick those up.
	t.Run("legacy low-balance flag spelling still matches low_balance sessions (backward-compat)", func(t *testing.T) {
		legacySessions := []do.HostedAgentSession{
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "running"}},
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "broke", PauseReason: godo.HostedAgentSessionPauseReasonLowBalance}},
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "idle", PauseReason: godo.HostedAgentSessionPauseReasonIdle}},
			{HostedAgentSession: &godo.HostedAgentSession{SessionID: "broke2", PauseReason: godo.HostedAgentSessionPauseReasonLowBalance}},
		}
		got := filterByPauseReason(legacySessions, "low-balance")
		require.Len(t, got, 2)
		assert.Equal(t, "broke", got[0].SessionID)
		assert.Equal(t, "broke2", got[1].SessionID)
	})

	t.Run("an empty filter is a no-op", func(t *testing.T) {
		assert.Len(t, filterByPauseReason(sessions, ""), len(sessions))
	})

	t.Run("an unmatched reason yields nothing rather than everything", func(t *testing.T) {
		assert.Empty(t, filterByPauseReason(sessions, "manual"))
	})
}

// The card is latched per gate episode: a user retrying in a tight loop should
// see the explanation once, not once per keystroke.
func TestPrintPrepayBlocked_OncePerEpisode(t *testing.T) {
	out := &bytes.Buffer{}
	c := configWithPrepayment(stubPrepayment{err: errors.New("unavailable")})
	c.Out = out
	state := &attachState{}
	cause := harnessAPIErr(http.StatusPaymentRequired, prepayBlockedWireMessage)

	printPrepayBlocked(c, state, cause)
	require.Contains(t, out.String(), prepayBlockedTitle)

	first := out.Len()
	printPrepayBlocked(c, state, cause)
	assert.Equal(t, first, out.Len(), "the card must not repeat within one episode")

	// A success reopens the latch so a later episode is announced again.
	state.clearPrepayBlocked()
	printPrepayBlocked(c, state, cause)
	assert.Greater(t, out.Len(), first)
}
