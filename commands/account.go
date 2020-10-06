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

// Account creates the account commands hierarchy.
func Account() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "account",
			Short: "Display commands that retrieve account details",
			Long: `The subcommands of ` + "`" + `doctl account` + "`" + ` retrieve information about DigitalOcean accounts.

For example, ` + "`" + `doctl account get` + "`" + ` retrieves account profile details, and ` + "`" + `doctl account ratelimit` + "`" + ` retrieves API usage details.`,
		},
	}

	CmdBuilder(cmd, RunAccountGet, "get", "Retrieve account profile details", `Retrieve the following details from your account profile:

- Email address
- Account Droplet limit
- Email verification status
- Account status (active or disabled)
- UUID for the account.`, Writer,
		aliasOpt("g"), displayerType(&displayers.Account{}))

	CmdBuilder(cmd, RunAccountRateLimit, "ratelimit", "Retrieve your API usage and the remaining quota", `Retrieve the following details about your account's API usage:

- The current limit on your account for API calls (5,000 per hour per OAuth token)
- The number of API calls you have made in the last hour
- When the API call count is due to reset to zero, which happens hourly

Note that these details are per OAuth token and are tied to the token you used when calling `+"`"+`doctl auth init`+"`"+` at setup time.`, Writer,
		aliasOpt("rl"), displayerType(&displayers.RateLimit{}))

	return cmd
}

// RunAccountGet runs account get.
func RunAccountGet(c *CmdConfig) error {
	a, err := c.Account().Get()
	if err != nil {
		return err
	}

	return c.Display(&displayers.Account{Account: a})
}

// RunAccountRateLimit retrieves API rate limits for the account.
func RunAccountRateLimit(c *CmdConfig) error {
	rl, err := c.Account().RateLimit()
	if err != nil {
		return err
	}

	return c.Display(&displayers.RateLimit{RateLimit: rl})
}
