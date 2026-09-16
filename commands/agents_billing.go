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

// pauseReasonLowBalance is the prepay gate's pause reason on the session
// model. harness-api documents pause reasons as an open string set and tells
// clients to treat unrecognized values as opaque, so this is compared against,
// never switched on exhaustively.
const pauseReasonLowBalance = "low_balance"

// isLowBalancePauseReason matches the gate's reason case-insensitively,
// because the session model and the run-event stream spell their reasons in
// different cases.
func isLowBalancePauseReason(reason string) bool {
	return strings.EqualFold(strings.TrimSpace(reason), pauseReasonLowBalance)
}

// renderRunPaused announces a pause the user did not ask for. Without it the
// stream just stops mid-sentence: the spinner clears, the accumulator flushes,
// and nothing on screen explains why the agent went quiet.
func renderRunPaused(w io.Writer, reason string) {
	if isLowBalancePauseReason(reason) {
		fmt.Fprintf(w, "\n%s %s\n", colorize("⏸", colWarning),
			boldColor("Paused — prepayment balance exhausted", colWarning))
		fmt.Fprintf(w, "  %s\n", colorize("Your work is saved. Add funds, then send any message to continue.", colMuted))
		fmt.Fprintf(w, "  %s\n", colorize(prepayTopUpURL, colMuted))
		fmt.Fprintln(w, colorize(runSeparator, colMuted))
		return
	}

	label := "run paused"
	if r := strings.TrimSpace(reason); r != "" {
		label = fmt.Sprintf("run paused (%s)", strings.ToLower(r))
	}
	fmt.Fprintf(w, "\n%s\n", colorize("⏸ "+label, colMuted))
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
