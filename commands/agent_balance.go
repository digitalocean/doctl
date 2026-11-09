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
	"net/http"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

const agentsBalanceHelpMD = `Shows your team's prepayment balance and whether the prepayment gate is currently blocking Harness Runtime.

Harness Runtime sessions are paid for from a single team-wide prepayment balance shared with the rest of your DigitalOcean usage. When that balance runs out, running sessions are paused with their work saved, and commands that would spend — ` + "`create`" + `, ` + "`launch`" + `, ` + "`resume`" + `, ` + "`fork`" + `, ` + "`exec`" + `, and sending a prompt — fail until funds are added.

This is a point-in-time read, not a live meter; run it again for a fresh figure. Reading it requires a token with the ` + "`billing:read`" + ` scope. When the balance cannot be read, the command exits non-zero and explains the cause, such as a token that lacks that scope, an invalid or expired token, or a team role that cannot read billing.

Note ` + "`doctl balance get`" + ` is a different number: it reports the classic account ledger, not the prepayment wallet that gates Harness Runtime.`

// RunAgentsBalance runs `doctl harness-runtime balance`.
func RunAgentsBalance(c *CmdConfig) error {
	// No deadline of its own: reading the balance is the whole point of this
	// command, so it waits as long as any other doctl request would.
	cfg, status, err := c.Prepayment().Get(context.Background())
	if err != nil {
		return explainBalanceError(c, err)
	}

	return c.Display(&displayers.HarnessPrepayBalance{Config: cfg, Status: status})
}

// balanceTokenCheckTimeout bounds the follow-up token check, matching the
// balance lookup that enriches a 402 card. A variable so tests can shorten it.
var balanceTokenCheckTimeout = prepayLookupTimeout

// balanceUnreadableError carries the user-facing explanation while keeping the
// API error underneath, so the status and request ID still reach checkErr. It
// is an agentPrettyError with no tips, which also keeps checkErr from adding
// the generic "--help" next step.
func balanceUnreadableError(msg string, status int, cause error) error {
	return &agentPrettyError{title: msg, reason: msg, status: status, cause: cause}
}

// explainBalanceError turns a failed balance read into an error that names the
// cause. A missing scope can surface as either a 401 or a 403, and a bad token
// also surfaces as a 401, so the status alone cannot say which it is; the
// token's own scopes can. Other failures pass through unchanged.
func explainBalanceError(c *CmdConfig, err error) error {
	status, message := 0, ""
	var er *godo.ErrorResponse
	if errors.As(err, &er) && er.Response != nil &&
		(er.Response.StatusCode == http.StatusUnauthorized || er.Response.StatusCode == http.StatusForbidden) {
		status = er.Response.StatusCode
		message = serverSaid(er, status)
	} else if errors.Is(err, do.ErrNoBillingPermission) {
		status = http.StatusForbidden
		message = http.StatusText(status)
	}
	if status == 0 {
		return err
	}

	unsure := balanceUnreadableError(fmt.Sprintf(
		"Cannot read the balance: the token may lack the billing:read scope. Server said: %s.",
		strings.TrimRight(message, ". ")), status, err)

	server, serverErr := c.Doit.GetString("", doctl.ArgTokenValidationServer)
	if serverErr != nil {
		return unsure
	}
	server = strings.TrimSpace(server)
	// Never send the token to a host the user did not choose. With no
	// validation server configured, the check goes to
	// https://cloud.digitalocean.com and runs only when the API URL is the
	// default. A validation server can be set only through
	// DIGITALOCEAN_TOKEN_VALIDATION_SERVER or the top-level
	// token-validation-server config key.
	if server == "" && !isDefaultAPIURL(doctl.ResolvedAPIURL()) {
		return unsure
	}

	ctx, cancel := context.WithTimeout(context.Background(), balanceTokenCheckTimeout)
	defer cancel()
	info, infoErr := c.OAuth().TokenInfoContext(ctx, server)
	if infoErr != nil {
		var ier *godo.ErrorResponse
		if errors.As(infoErr, &ier) && ier.Response != nil && ier.Response.StatusCode == http.StatusUnauthorized {
			return balanceUnreadableError("Your token does not work: it is invalid, expired, or revoked. Run `doctl auth init` or `doctl auth login`.", status, err)
		}
		return unsure
	}
	// An answer without scopes tells us nothing, so it is not "lacks billing:read".
	if info == nil || len(info.Scopes) == 0 {
		return unsure
	}

	switch {
	case !scopesCoverBillingRead(info.Scopes):
		return balanceUnreadableError("Your token works but does not include the billing:read scope. Create a token with billing:read, or ask a Team Owner or Biller.", status, err)
	case status == http.StatusForbidden:
		return balanceUnreadableError("Your token has billing:read, but your team role or account state does not allow reading billing. Ask a Team Owner or Biller.", status, err)
	default:
		return balanceUnreadableError("Your token was refused for this team. Check that your auth context points at the right team.", status, err)
	}
}

// serverSaid picks readable text from an API error for the "Server said" part:
// the message, then the nested error message, then the status text. It never
// uses err.Error(), which carries the request URL and request ID.
func serverSaid(er *godo.ErrorResponse, status int) string {
	if isReadableMessage(er.Message) {
		return strings.TrimSpace(er.Message)
	}
	if er.NestedError != nil && isReadableMessage(er.NestedError.Message) {
		return strings.TrimSpace(er.NestedError.Message)
	}
	return http.StatusText(status)
}

// isDefaultAPIURL reports whether u is the default DigitalOcean API.
func isDefaultAPIURL(u string) bool {
	norm := func(s string) string { return strings.ToLower(strings.TrimRight(strings.TrimSpace(s), "/")) }
	return norm(u) == norm(doctl.DefaultAPIURL)
}

// scopesCoverBillingRead reports whether any scope grants billing read access.
// TODO: the exact scope names the token-info endpoint returns, and the name of
// its response field ("scopes"), have not been confirmed; check both against
// real tokens before relying on this rule.
func scopesCoverBillingRead(scopes []string) bool {
	for _, s := range scopes {
		switch s {
		case "read", "billing:read", "billing:*":
			return true
		}
	}
	return false
}
