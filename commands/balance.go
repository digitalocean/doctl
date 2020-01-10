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
			Short: "balance commands",
			Long:  "balance is used to access balance commands",
		},
	}

	CmdBuilder(cmd, RunBalanceGet, "get", "get balance", Writer,
		aliasOpt("g"), displayerType(&displayers.Balance{}))

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
