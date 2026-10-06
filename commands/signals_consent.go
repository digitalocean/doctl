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
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

// Signals creates the top-level `doctl signals` command group.
func Signals() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "signals",
			Short: "Display commands for Signals consent and export management",
			Long:  "The subcommands of `doctl signals` manage Signals collection consent and bulk data exports.",
		},
	}

	cmd.AddCommand(SignalsConsent())
	cmd.AddCommand(SignalsExport())

	return cmd
}

// SignalsConsent creates the `doctl signals consent` subcommand group.
func SignalsConsent() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "consent",
			Short: "Display commands for Signals collection consent",
			Long:  "The subcommands of `doctl signals consent` manage per-agent Signals collection consent.",
		},
	}

	consentDetails := `
		- The consent record ID
		- The team ID
		- The agent ID
		- Whether consent is enabled
		- When the consent was last updated
	`

	CmdBuilder(
		cmd,
		RunSignalsConsentList,
		"list",
		"List all Signals consents for your team",
		"Lists all Signals collection consents for the authenticated team, where each consent contains:"+consentDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.SignalsConsent{}),
	)

	cmdConsentGet := CmdBuilder(
		cmd,
		RunSignalsConsentGet,
		"get",
		"Get Signals consent for an agent",
		"Retrieves the current Signals collection consent status for one agent.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.SignalsConsent{}),
	)
	AddStringFlag(cmdConsentGet, doctl.ArgSignalsAgentID, "", "", "The agent ID to query consent for.", requiredOpt())

	cmdConsentSet := CmdBuilder(
		cmd,
		RunSignalsConsentSet,
		"set",
		"Set Signals consent for an agent",
		"Enables or disables Signals collection for one agent. Idempotent.",
		Writer, aliasOpt("s"),
		displayerType(&displayers.SignalsConsent{}),
	)
	AddStringFlag(cmdConsentSet, doctl.ArgSignalsAgentID, "", "", "The agent ID to set consent for.", requiredOpt())
	AddBoolFlag(cmdConsentSet, doctl.ArgSignalsEnabled, "", true, "Enable (true) or disable (false) Signals collection.")

	return cmd
}

// RunSignalsConsentList lists all Signals consents for the team.
func RunSignalsConsentList(c *CmdConfig) error {
	consents, err := c.Signals().ListConsents()
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsConsent{Consents: consents})
}

// RunSignalsConsentGet retrieves Signals consent for one agent.
func RunSignalsConsentGet(c *CmdConfig) error {
	agentID, err := c.Doit.GetString(c.NS, doctl.ArgSignalsAgentID)
	if err != nil {
		return err
	}

	consent, err := c.Signals().GetConsent(agentID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsConsent{Consents: do.SignalsConsents{*consent}})
}

// RunSignalsConsentSet enables or disables Signals consent for one agent.
func RunSignalsConsentSet(c *CmdConfig) error {
	agentID, err := c.Doit.GetString(c.NS, doctl.ArgSignalsAgentID)
	if err != nil {
		return err
	}
	enabled, err := c.Doit.GetBool(c.NS, doctl.ArgSignalsEnabled)
	if err != nil {
		return err
	}

	consent, err := c.Signals().SetConsent(agentID, enabled)
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsConsent{Consents: do.SignalsConsents{*consent}})
}
