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
			Long: `The subcommands of ` + "`doctl genai harbor`" + ` publish Harbor datasets, run them on DigitalOcean models, and download the results.

Author and check a task on your own machine with the harbor CLI: ` + "`harbor task init`" + `, ` + "`harbor task start-env`" + `, ` + "`harbor run -a oracle`" + `, and ` + "`harbor lint`" + `. Then use doctl to publish the dataset, run agents, and download the job. Open the downloaded folder with ` + "`harbor view`" + `.

A task passed to ` + "`run`" + ` uses ` + "`-T`" + ` or ` + "`--task`" + `. ` + "`-t`" + ` is doctl's access token.`,
		},
	}

	cmd.AddCommand(HarborDatasetCmd())
	HarborRunCmd(cmd)
	cmd.AddCommand(HarborJobCmd())

	return cmd
}
