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

// BillingHistory creates the BillingHistory commands hierarchy.
func BillingHistory() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "billing-history",
			Short:   "Display commands for retrieving your billing history",
			Long:    "The subcommands of `doctl billing-history` are used to access the billing history for your DigitalOcean account.",
			Aliases: []string{"bh"},
		},
	}
	listBillingHistoryDesc := `This command retrieves the following details for each event in your billing history:
- The date of the event
- The type of billing event
- A description of the event
- The amount of the event in USD
- The invoice ID associated with the event, if applicable
- The invoice UUID associated with the event, if applicable

`

	CmdBuilder(cmd, RunBillingHistoryList, "list", "Retrieve a paginated billing history for a user",
		listBillingHistoryDesc, Writer, aliasOpt("ls"), displayerType(&displayers.BillingHistory{}))

	return cmd
}

// RunBillingHistoryList runs invoice list.
func RunBillingHistoryList(c *CmdConfig) error {
	billingHistory, err := c.BillingHistory().List()
	if err != nil {
		return err
	}

	return c.Display(&displayers.BillingHistory{BillingHistory: billingHistory})
}
