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

// Prepayment creates the prepayment commands hierarchy.
func Prepayment() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "prepayment",
			Short:   "Display commands for retrieving prepayment configuration and status",
			Long:    "The subcommands of `doctl prepayment` retrieve information about your account prepayment configuration and status.",
			GroupID: viewBillingGroup,
		},
	}

	getConfigDesc := `This command retrieves your account prepayment configuration and current prepayment status.

The configuration includes spend limit, auto prepay settings, and thresholds. The status includes
current prepayment balance, gate blocked/eligible state, and month-to-date balance.
`

	configCmd := CmdBuilder(cmd, RunPrepaymentConfigGet, "config", "Retrieve your prepayment configuration",
		getConfigDesc, Writer, displayerType(&displayers.PrepaymentConfig{}))
	configCmd.Example = `The following example retrieves your prepayment configuration: doctl prepayment config`

	getStatusDesc := `This command retrieves your account prepayment gate status and balances.

It includes your current prepayment balance, whether the gate is blocked or eligible, and your
month-to-date balance.
`

	statusCmd := CmdBuilder(cmd, RunPrepaymentStatusGet, "status", "Retrieve your prepayment status",
		getStatusDesc, Writer, displayerType(&displayers.PrepaymentStatus{}))
	statusCmd.Example = `The following example retrieves your prepayment status: doctl prepayment status`

	return cmd
}

// RunPrepaymentConfigGet runs prepayment config.
func RunPrepaymentConfigGet(c *CmdConfig) error {
	resp, err := c.Prepayment().GetConfig()
	if err != nil {
		return err
	}

	return c.Display(&displayers.PrepaymentConfig{PrepaymentConfigResponse: resp})
}

// RunPrepaymentStatusGet runs prepayment status.
func RunPrepaymentStatusGet(c *CmdConfig) error {
	resp, err := c.Prepayment().GetStatus()
	if err != nil {
		return err
	}

	return c.Display(&displayers.PrepaymentStatus{PrepaymentStatusResponse: resp})
}
