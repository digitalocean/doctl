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

// AgentSizes generates the `doctl open-harness-runtime sizes` subtree for the
// sandbox (microVM) size catalog returned by ListSandboxSizes.
func AgentSizes() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "sizes",
			Aliases: []string{"size"},
			Short:   "List available sandbox sizes",
			Long:    agentsSizesRootHelpMD,
		},
	}

	ns := agentSubNS("agents.sizes")

	cmdList := CmdBuilder(cmd, RunAgentsSizesList, "list",
		"List available sandbox sizes",
		agentsSizesListHelpMD,
		Writer, append(ns, aliasOpt("ls"),
			displayerType(&displayers.HostedAgentSandboxSize{}))...)
	cmdList.Example = `doctl open-harness-runtime sizes list`

	return cmd
}

// RunAgentsSizesList prints the sandbox size catalog (slug, vCPUs, memory).
func RunAgentsSizesList(c *CmdConfig) error {
	sizes, err := c.HostedAgents().ListSandboxSizes()
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentSandboxSize{Sizes: sizes})
}
