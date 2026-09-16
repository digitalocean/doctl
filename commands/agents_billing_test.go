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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRenderRunPaused(t *testing.T) {
	t.Run("low balance explains itself", func(t *testing.T) {
		var buf bytes.Buffer
		renderRunPaused(&buf, "low_balance")

		out := buf.String()
		assert.Contains(t, out, "Paused — prepayment balance exhausted")
		assert.Contains(t, out, "Your work is saved.")
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

func TestIsLowBalancePauseReason(t *testing.T) {
	// The session model and the run-event stream disagree on case, so the
	// comparison tolerates both.
	assert.True(t, isLowBalancePauseReason("low_balance"))
	assert.True(t, isLowBalancePauseReason("LOW_BALANCE"))
	assert.True(t, isLowBalancePauseReason(" low_balance "))
	assert.False(t, isLowBalancePauseReason("idle"))
	assert.False(t, isLowBalancePauseReason(""))
}

func TestNormalizePauseReason(t *testing.T) {
	// The flag reads better hyphenated than the underscored value it matches,
	// so both spellings have to land on the same reason.
	assert.Equal(t, "low_balance", normalizePauseReason("low-balance"))
	assert.Equal(t, "low_balance", normalizePauseReason("LOW_BALANCE"))
	assert.Equal(t, "low_balance", normalizePauseReason(" low_balance "))
	assert.Equal(t, "idle", normalizePauseReason("Idle"))
	assert.Equal(t, "", normalizePauseReason(""))
}

func TestFilterByPauseReason(t *testing.T) {
	sessions := []do.HostedAgentSession{
		{HostedAgentSession: &godo.HostedAgentSession{SessionID: "running"}},
		{HostedAgentSession: &godo.HostedAgentSession{SessionID: "broke", PauseReason: godo.HostedAgentSessionPauseReasonLowBalance}},
		{HostedAgentSession: &godo.HostedAgentSession{SessionID: "idle", PauseReason: godo.HostedAgentSessionPauseReasonIdle}},
		{HostedAgentSession: &godo.HostedAgentSession{SessionID: "broke2", PauseReason: godo.HostedAgentSessionPauseReasonLowBalance}},
	}

	t.Run("matches the hyphenated flag spelling", func(t *testing.T) {
		got := filterByPauseReason(sessions, "low-balance")
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
