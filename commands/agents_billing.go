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
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

const (
	prepayBlockedTitle = "Prepayment balance exhausted"
	prepayUnknownTitle = "Prepayment status unavailable"

	prepayTopUpURL   = "https://cloud.digitalocean.com/account/billing"
	prepayBalanceCmd = "doctl harness-runtime balance"

	// prepaySavedWork is the sentence that answers the question users actually
	// have when a session stops: whether they lost anything.
	prepaySavedWork = "Running sessions were paused and their work is saved."
)

// prepayBlockedErr reports harness-api's 402 for a team whose prepayment gate
// is engaged. checkBillingGate is the only thing in harness-api that returns
// 402, so the status alone is the signal — no message matching needed.
func prepayBlockedErr(err error) bool {
	_, status, ok := agentAPIError(err)
	return ok && status == http.StatusPaymentRequired
}

// prepayUnknownErr reports the gate's fail-closed case: the team's mirrored
// billing state could not be resolved. Retryable, and explicitly not add-funds.
func prepayUnknownErr(err error) bool {
	msg, status, ok := agentAPIError(err)
	return ok && status == http.StatusServiceUnavailable &&
		strings.Contains(strings.ToLower(msg), "prepayment gate state")
}

// newPrepayBlockedError builds the add-funds card for a 402, enriched with the
// team's live balance and auto top-off state when billing can be read.
//
// The copy leads with the team-level cause rather than the command that
// failed, because the same card appears on every spend path. c may be nil, and
// any billing lookup failure falls back to the plain card — a billing error
// must never turn a clean 402 into a confusing double error.
func newPrepayBlockedError(c *CmdConfig, cause error) *agentPrettyError {
	reason := fmt.Sprintf("Your team's prepayment balance has run out, so Harness Runtime can't start or resume sessions. %s", prepaySavedWork)
	tips := []string{
		fmt.Sprintf("Add funds at %s", prepayTopUpURL),
		prepayBalanceCmd,
	}

	cfg, status, err := fetchPrepayment(c)
	switch {
	case errors.Is(err, do.ErrNoBillingPermission):
		// The caller can neither see nor fix the balance, so sending them to a
		// billing page they cannot open would be a dead end.
		tips = []string{"Ask your Team Owner or Biller to add funds"}
	case err != nil || status == nil:
		// Keep the plain card.
	default:
		if amount := formatPrepayMoney(status.Balance); amount != "" {
			reason = fmt.Sprintf("Your team's prepayment balance is %s, so Harness Runtime can't start or resume sessions. %s", amount, prepaySavedWork)
		}
		if status.IsAutoPrepayEnabled || (cfg != nil && cfg.IsAutoPrepayEnabled) {
			reason += " Auto top-off is enabled, so this should resolve shortly."
			tips = []string{"Retry in a moment", prepayBalanceCmd}
		}
	}

	return &agentPrettyError{
		title:  prepayBlockedTitle,
		reason: reason,
		status: http.StatusPaymentRequired,
		tips:   tips,
		cause:  cause,
	}
}

// prepayLookupTimeout bounds the balance read that enriches a 402 card. The
// user has already earned that error, so the extra detail is worth a moment
// but never a stall: without a deadline a degraded billing API would be
// retried on doctl's usual schedule and hold back an error we can already
// render. On expiry the card falls back to its plain wording.
const prepayLookupTimeout = 5 * time.Second

// fetchPrepayment reads the team's prepay config and status, tolerating a
// CmdConfig whose services were never initialized.
func fetchPrepayment(c *CmdConfig) (*do.PrepaymentConfig, *do.PrepaymentStatus, error) {
	if c == nil || c.Prepayment == nil {
		return nil, nil, errors.New("prepayment service unavailable")
	}
	svc := c.Prepayment()
	if svc == nil {
		return nil, nil, errors.New("prepayment service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), prepayLookupTimeout)
	defer cancel()
	return svc.Get(ctx)
}

// notePrepayBlocked reports whether this is the first prepay block of the
// current episode, latching it so a retry loop prints the card once.
func (s *attachState) notePrepayBlocked() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	first := !s.prepayNotified
	s.prepayNotified = true
	return first
}

// clearPrepayBlocked reopens the latch once a request succeeds, so a second
// gate episode later in the same session is announced again.
func (s *attachState) clearPrepayBlocked() {
	s.mu.Lock()
	s.prepayNotified = false
	s.mu.Unlock()
}

// printPrepayBlocked renders the add-funds card inline and leaves the session
// attached. The work is intact and the next prompt succeeds once funds land —
// harness-api resumes implicitly on the next send — so tearing the attach down
// would discard a recoverable state for a condition the user can fix from
// another window.
func printPrepayBlocked(c *CmdConfig, state *attachState, cause error) {
	if !state.notePrepayBlocked() {
		return
	}
	fmt.Fprintf(c.Out, "\n%s\n", newPrepayBlockedError(c, cause).DisplayError())
}

// reportHITLResolveErr renders a failed approval resolution. Resolving an
// approval releases a run that then spends, so the prepay gate guards this
// route too and a bare "resolve failed" would misattribute the cause.
func reportHITLResolveErr(c *CmdConfig, state *attachState, err error) {
	if prepayBlockedErr(err) {
		printPrepayBlocked(c, state, err)
		return
	}
	fmt.Fprintf(c.Out, "resolve failed: %v\n", err)
}

// pauseReasonZeroBalance is the prepay gate's pause reason. harness-api
// documents pause reasons as an open string set and tells clients to treat
// unrecognized values as opaque, so this is compared against, never switched
// on exhaustively. It is kept as a string rather than godo's typed constant
// because the run-event stream reports the reason as a bare string too, and
// both sources are normalized through the same path.
const pauseReasonZeroBalance = string(godo.HostedAgentSessionPauseReasonZeroBalance)

// normalizePauseReason folds a reason to its wire spelling. The session model
// and the run-event stream disagree on case, and the --paused-by flag reads
// better hyphenated (low-balance) than the underscored value it matches.
func normalizePauseReason(reason string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(reason), "-", "_"))
}

// isZeroBalancePauseReason reports whether the prepay gate is what paused a
// session. It accepts both "zero_balance" (current) and "low_balance"
// (deprecated) so that existing sessions and scripts continue to work.
func isZeroBalancePauseReason(reason string) bool {
	r := normalizePauseReason(reason)
	return r == pauseReasonZeroBalance || r == "low_balance"
}

// filterByPauseReason narrows a page of sessions to one pause reason.
//
// ListSessions accepts only page_size, page_token, status, name, and
// parent_session_id — there is no server-side pause-reason filter — so this
// runs over whatever a page returned. Callers pair it with an implied
// status=paused, which is as much narrowing as the API can do for us.
func filterByPauseReason(sessions []do.HostedAgentSession, reason string) []do.HostedAgentSession {
	want := normalizePauseReason(reason)
	if want == "" {
		return sessions
	}

	out := make([]do.HostedAgentSession, 0, len(sessions))
	for _, s := range sessions {
		if normalizePauseReason(string(s.PauseReason)) == want {
			out = append(out, s)
		}
	}
	return out
}

// sessionStatusWithReason renders a session's status, naming why it is paused
// when the server said. Unrecognized reasons are shown verbatim rather than
// flattened to "unknown", since the API reserves the right to add values.
func sessionStatusWithReason(sess *do.HostedAgentSession) string {
	status := colorizeSessionStatus(sess.Status)
	reason := strings.TrimSpace(string(sess.PauseReason))
	if reason == "" {
		return status
	}
	return status + colorize(fmt.Sprintf(" (%s)", reason), colMuted)
}

// The thing a pause notice says stopped. run.paused pauses a turn in flight;
// session.updated pauses the whole session, often with no run to speak of, and
// calling that one "run paused" would point the user at the wrong scope.
const (
	pauseSubjectRun     = "run"
	pauseSubjectSession = "session"
)

// renderPauseNotice announces a pause the user did not ask for. Without it the
// stream just stops mid-sentence: the spinner clears, the accumulator flushes,
// and nothing on screen explains why the agent went quiet.
//
// The balance card is deliberately identical whichever event carried the
// reason. The user is looking at one stalled agent, and the run/session
// distinction that matters to the API is not one they can act on differently.
func renderPauseNotice(w io.Writer, subject, reason string) {
	if isZeroBalancePauseReason(reason) {
		fmt.Fprintf(w, "\n%s %s\n", colorize("⏸", colWarning),
			boldColor("Paused — prepayment balance exhausted", colWarning))
		fmt.Fprintf(w, "  %s\n", colorize("Your work is saved. Add funds, then send any message to continue.", colMuted))
		fmt.Fprintf(w, "  %s\n", colorize(prepayTopUpURL, colMuted))
		fmt.Fprintln(w, colorize(runSeparator, colMuted))
		return
	}

	label := subject + " paused"
	if r := strings.TrimSpace(reason); r != "" {
		label = fmt.Sprintf("%s paused (%s)", subject, strings.ToLower(r))
	}
	fmt.Fprintf(w, "\n%s\n", colorize("⏸ "+label, colMuted))
}

// renderRunPaused announces a paused run.
func renderRunPaused(w io.Writer, reason string) {
	renderPauseNotice(w, pauseSubjectRun, reason)
}

// pauseOutcome is what one connection learned about the session being paused.
//
// Knowing nothing is deliberately distinct from knowing it is not paused: a
// connection that drops before delivering an event carries no news, and must
// not erase what an earlier connection established. Collapsing the two would
// make a paused session look healthy again after one silent reconnect.
type pauseOutcome struct {
	// observed is set when this connection saw the session pause or resume.
	observed bool
	// reason is the pause in effect when the connection ended, empty if the
	// session was last seen running.
	reason string
}

// pauseTracker latches the pause a stream has already explained.
//
// The gate announces one pause on both run.paused and session.updated, and a
// session that stays paused keeps saying so, so the card is printed once per
// pause rather than once per event.
type pauseTracker struct {
	observed bool
	notified bool
	reason   string
}

// note records a pause and reports whether it still needs rendering. A
// different reason always renders: it is a new fact about the session, not a
// repeat of the one already on screen.
func (p *pauseTracker) note(reason string) bool {
	fresh := !p.notified || normalizePauseReason(p.reason) != normalizePauseReason(reason)
	p.observed = true
	p.notified = true
	p.reason = reason
	return fresh
}

// clear records that the session is running again. It reopens the latch, so a
// second pause later in the same attach is announced rather than swallowed as
// a duplicate of the first, and it counts as news in its own right: a resume
// is how the caller learns a pause it was told about is over.
func (p *pauseTracker) clear() {
	p.observed = true
	p.notified = false
	p.reason = ""
}

func (p *pauseTracker) outcome() pauseOutcome {
	return pauseOutcome{observed: p.observed, reason: p.reason}
}

// formatPrepayMoney renders billing's decimal-dollar string as a display
// amount, returning "" for values it cannot vouch for.
func formatPrepayMoney(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "-") {
		return "-$" + strings.TrimPrefix(s, "-")
	}
	return "$" + s
}
