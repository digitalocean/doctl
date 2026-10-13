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

	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
)

const agentsBalanceHelpMD = `Shows your team's prepayment balance and whether the prepayment gate is currently blocking Harness Runtime.

Harness Runtime sessions are paid for from a single team-wide prepayment balance shared with the rest of your DigitalOcean usage. When that balance runs out, running sessions are paused with their work saved, and commands that would spend — ` + "`create`" + `, ` + "`launch`" + `, ` + "`resume`" + `, ` + "`fork`" + `, ` + "`exec`" + `, and sending a prompt — fail until funds are added.

This is a point-in-time read, not a live meter; run it again for a fresh figure. Reading it requires a token with the ` + "`billing:read`" + ` scope. Without that scope the balance shows as unknown, which is expected rather than an error — ask a Team Owner or Biller to add funds.

Note ` + "`doctl balance get`" + ` is a different number: it reports the classic account ledger, not the prepayment wallet that gates Harness Runtime.`

// RunAgentsBalance runs `doctl harness-runtime balance`.
func RunAgentsBalance(c *CmdConfig) error {
	// No deadline of its own: reading the balance is the whole point of this
	// command, so it waits as long as any other doctl request would.
	cfg, status, err := c.Prepayment().Get(context.Background())
	if err != nil {
		// A token without billing:read cannot read the balance and cannot fix
		// it either. That is a state worth displaying, not a failed command.
		if errors.Is(err, do.ErrNoBillingPermission) {
			return c.Display(&displayers.HarnessPrepayBalance{NoPermission: true})
		}
		return err
	}

	return c.Display(&displayers.HarnessPrepayBalance{Config: cfg, Status: status})
}
