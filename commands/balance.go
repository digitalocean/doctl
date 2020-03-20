/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/spf13/cobra"
)

// Balance creates the balance commands hierarchy.
func Balance() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "balance",
			Short: "Display commands for retrieving your account balance",
			Long:  "The subcommands of `doctl balance` retrieve information about your account balance.",
		},
	}

	getBalanceDesc := `This command retrieves the following details about your account balance:

- Your month-to-date balance including your account
  balance and month-to-date usage.
- Your current balance as of your most recent billing activity.
- Your usage in the current billing period.
- The time at which balances were most recently generated.
`

	CmdBuilder(cmd, RunBalanceGet, "get", "Retrieve your account balance",
		getBalanceDesc, Writer, aliasOpt("g"), displayerType(&displayers.Balance{}))

	return cmd
}

// RunBalanceGet runs balance get.
func RunBalanceGet(c *CmdConfig) error {
	a, err := c.Balance().Get()
	if err != nil {
		return err
	}

	return c.Display(&displayers.Balance{Balance: a})
}
