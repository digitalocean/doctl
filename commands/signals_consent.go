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
	"fmt"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Signals creates the top-level `doctl signals` command group.
func Signals() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "signals",
			Short: "Display commands for Signals consent, export, and data deletion management",
			Long:  "The subcommands of `doctl signals` manage Signals collection consent, bulk data exports, and data deletion.",
		},
	}

	cmd.AddCommand(SignalsConsent())
	cmd.AddCommand(SignalsSession())
	cmd.AddCommand(SignalsExport())
	cmd.AddCommand(SignalsDeletion())

	return cmd
}

// SignalsConsent creates the `doctl signals consent` subcommand group.
func SignalsConsent() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "consent",
			Short: "Display commands for Signals collection consent",
			Long: "The subcommands of `doctl signals consent` manage Signals collection consent.\n\n" +
				"Consent has a source: `agent` (per Gradient managed agent, identified by `--agent-id`) " +
				"or `inference` (team-level, covers all serverless inference traffic for the team).",
		},
	}

	consentDetails := `
		- The consent record ID
		- The team ID
		- The consent source (agent or inference)
		- The agent ID (empty for inference)
		- Whether consent is enabled
		- When the consent was last updated
	`

	cmdConsentList := CmdBuilder(
		cmd,
		RunSignalsConsentList,
		"list",
		"List all Signals consents for your team",
		"Lists all Signals collection consents for the authenticated team, where each consent contains:"+consentDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.SignalsConsent{}),
	)
	AddStringFlag(cmdConsentList, doctl.ArgSignalsConsentSource, "", "", "Only list consents for this source: agent or inference. Default lists all.")

	cmdConsentGet := CmdBuilder(
		cmd,
		RunSignalsConsentGet,
		"get",
		"Get Signals consent for an agent or for serverless inference",
		"Retrieves the current Signals collection consent status for one agent (`--source agent --agent-id <id>`) "+
			"or the team-level serverless inference consent (`--source inference`). Missing consent is reported as disabled.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.SignalsConsent{}),
	)
	AddStringFlag(cmdConsentGet, doctl.ArgSignalsConsentSource, "", godo.SignalsConsentSourceAgent, "Consent source: agent or inference.")
	AddStringFlag(cmdConsentGet, doctl.ArgSignalsAgentID, "", "", "The agent ID to query consent for (required when --source=agent).")

	cmdConsentSet := CmdBuilder(
		cmd,
		RunSignalsConsentSet,
		"set",
		"Set Signals consent for an agent or for serverless inference",
		"Enables or disables Signals collection for one agent (`--source agent --agent-id <id>`) "+
			"or for all serverless inference traffic of your team (`--source inference`). Idempotent.",
		Writer, aliasOpt("s"),
		displayerType(&displayers.SignalsConsent{}),
	)
	AddStringFlag(cmdConsentSet, doctl.ArgSignalsConsentSource, "", godo.SignalsConsentSourceAgent, "Consent source: agent or inference.")
	AddStringFlag(cmdConsentSet, doctl.ArgSignalsAgentID, "", "", "The agent ID to set consent for (required when --source=agent).")
	AddBoolFlag(cmdConsentSet, doctl.ArgSignalsEnabled, "", false, "Enable (true) or disable (false) Signals collection.", requiredOpt())

	return cmd
}

// consentTarget resolves and validates --source / --agent-id for get and set.
func consentTarget(c *CmdConfig) (source, agentID string, err error) {
	source, err = c.Doit.GetString(c.NS, doctl.ArgSignalsConsentSource)
	if err != nil {
		return "", "", err
	}
	agentID, err = c.Doit.GetString(c.NS, doctl.ArgSignalsAgentID)
	if err != nil {
		return "", "", err
	}
	source = strings.ToLower(strings.TrimSpace(source))
	agentID = strings.TrimSpace(agentID)
	if source == "" {
		source = godo.SignalsConsentSourceAgent
	}
	switch source {
	case godo.SignalsConsentSourceAgent:
		if agentID == "" {
			return "", "", fmt.Errorf("--%s is required when --%s=%s", doctl.ArgSignalsAgentID, doctl.ArgSignalsConsentSource, godo.SignalsConsentSourceAgent)
		}
	case godo.SignalsConsentSourceInference:
		if agentID != "" {
			return "", "", fmt.Errorf("--%s must not be set when --%s=%s (inference consent is team-level)", doctl.ArgSignalsAgentID, doctl.ArgSignalsConsentSource, godo.SignalsConsentSourceInference)
		}
	default:
		return "", "", fmt.Errorf("invalid --%s %q: must be %s or %s", doctl.ArgSignalsConsentSource, source, godo.SignalsConsentSourceAgent, godo.SignalsConsentSourceInference)
	}
	return source, agentID, nil
}

// RunSignalsConsentList lists Signals consents for the team, optionally filtered by source.
func RunSignalsConsentList(c *CmdConfig) error {
	source, err := c.Doit.GetString(c.NS, doctl.ArgSignalsConsentSource)
	if err != nil {
		return err
	}
	source = strings.ToLower(strings.TrimSpace(source))
	if source != "" && source != godo.SignalsConsentSourceAgent && source != godo.SignalsConsentSourceInference {
		return fmt.Errorf("invalid --%s %q: must be %s or %s", doctl.ArgSignalsConsentSource, source, godo.SignalsConsentSourceAgent, godo.SignalsConsentSourceInference)
	}

	var consents do.SignalsConsents
	if source == "" {
		consents, err = c.Signals().ListConsents()
	} else {
		consents, err = c.Signals().ListConsentsBySource(source)
	}
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsConsent{Consents: consents})
}

// RunSignalsConsentGet retrieves Signals consent for one agent or for inference.
func RunSignalsConsentGet(c *CmdConfig) error {
	source, agentID, err := consentTarget(c)
	if err != nil {
		return err
	}

	if source == godo.SignalsConsentSourceInference {
		consent, err := c.Signals().GetInferenceConsent()
		if err != nil {
			return err
		}
		return c.Display(&displayers.SignalsConsent{Consents: do.SignalsConsents{*consent}})
	}

	consent, err := c.Signals().GetConsent(agentID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsAgentConsent{Consents: []do.SignalsAgentConsent{*consent}})
}

// RunSignalsConsentSet enables or disables Signals consent for one agent or for inference.
func RunSignalsConsentSet(c *CmdConfig) error {
	source, agentID, err := consentTarget(c)
	if err != nil {
		return err
	}
	enabled, err := c.Doit.GetBool(c.NS, doctl.ArgSignalsEnabled)
	if err != nil {
		return err
	}

	var consent *do.SignalsConsent
	if source == godo.SignalsConsentSourceInference {
		consent, err = c.Signals().SetInferenceConsent(enabled)
	} else {
		consent, err = c.Signals().SetConsent(agentID, enabled)
	}
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsConsent{Consents: do.SignalsConsents{*consent}})
}
