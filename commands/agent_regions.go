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
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/spf13/cobra"
)

// AgentRegions generates the `doctl harness-runtime regions` subtree for the
// per-team region catalog returned by ListRegions. It is the discoverable
// counterpart to --region: the flag takes a slug, and this prints the slugs
// the team may actually use.
func AgentRegions() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "regions",
			Aliases: []string{"region"},
			Short:   "List available regions",
			Long:    agentsRegionsRootHelpMD,
		},
	}

	ns := agentSubNS("agents.regions")

	cmdList := CmdBuilder(cmd, RunAgentsRegionsList, "list",
		"List available regions",
		agentsRegionsListHelpMD,
		Writer, append(ns, aliasOpt("ls"),
			displayerType(&displayers.HostedAgentRegion{}))...)
	cmdList.Example = `doctl harness-runtime regions list`

	requireAgentSubcommand(cmd)
	return cmd
}

// RunAgentsRegionsList prints the regions the team may create in, marking the
// one a create uses when --region is omitted.
func RunAgentsRegionsList(c *CmdConfig) error {
	regions, err := c.HostedAgents().ListRegions()
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentRegion{Regions: regions})
}
