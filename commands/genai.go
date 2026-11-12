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

const harborNS = "genai.harbor"

// GenAI creates the genai command.
func GenAI() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "genai",
			Short: "Manage GenAI resources",
			Long: "The subcommands of `doctl genai` manage DigitalOcean GenAI resources.\n\n" +
				"Use `doctl genai harbor` to run Harbor agent benchmarks on DigitalOcean.",
			GroupID: manageResourcesGroup,
		},
	}

	cmd.AddCommand(Harbor())

	// `genai` used to be an alias of the hidden gradient command; keep its
	// deprecated forwarders reachable under the new name.
	cmd.AddCommand(deprecatedKnowledgeBaseCmd())
	cmd.AddCommand(deprecatedEvaluationSubcommand("scenario-set", "ss", "scenario-sets"))
	cmd.AddCommand(deprecatedEvaluationSubcommand("scenario-library", "sl"))
	cmd.AddCommand(deprecatedEvaluationSubcommand("simulation-run", "sim", "simulation-runs"))

	return cmd
}

// Harbor creates the genai harbor command.
func Harbor() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "harbor",
			Short: "Run Harbor agent benchmarks on DigitalOcean",
			Long: `The subcommands of ` + "`doctl genai harbor`" + ` publish Harbor tasks to your team, run agents against them, and download the results.

A task is a Harbor task folder (task.toml, instruction.md, environment/, tests/). A job runs one or more agent and model pairs against tasks, and each attempt is a trial scored by the task's tests. Trials run in DO Sandbox and call models through DigitalOcean inference, so no provider API keys are needed.

Author and test tasks locally with the open-source harbor CLI (harbor init, harbor lint, harbor run); use doctl to publish and run them on DigitalOcean.`,
		},
	}

	cmd.AddCommand(HarborTaskCmd())
	cmd.AddCommand(HarborDatasetCmd())
	HarborRunCmd(cmd)
	cmd.AddCommand(HarborJobCmd())
	cmd.AddCommand(HarborAgentsCmd())

	return cmd
}

// HarborAgentsCmd lists the supported agents.
func HarborAgentsCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "agents",
			Short: "Display commands that list Harbor agents",
			Long:  "The subcommands of `doctl genai harbor agents` list the agents Harbor jobs can run.",
		},
	}

	cmdList := CmdBuilder(
		cmd,
		RunHarborAgentsList,
		"list",
		"List supported Harbor agents",
		"Lists the agents a job can run, each with the version it is pinned to and the model protocol it needs.",
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HarborAgents{}),
		overrideCmdNS(harborNS+".agents"),
	)
	cmdList.Example = "The following example lists the supported agents: `doctl genai harbor agents list`"

	return cmd
}

// RunHarborAgentsList lists the supported agents.
func RunHarborAgentsList(c *CmdConfig) error {
	agents, err := c.Harbor().ListAgents()
	if err != nil {
		return err
	}
	return c.Display(&displayers.HarborAgents{Agents: agents})
}
