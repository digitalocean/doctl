/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    10|See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import "github.com/spf13/cobra"

// Evaluation creates the top-level evaluation command for GenAI model
// evaluations and agent simulations.
func Evaluation() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "evaluation",
			Aliases: []string{"eval"},
			Short:   "Manage GenAI evaluations and simulations",
			Long: `The subcommands of ` + "`" + `doctl evaluation` + "`" + ` manage GenAI model evaluations and agent simulations.

Model evaluations score candidate models against datasets with an LLM judge.
Simulations run multi-turn scenario sets against agents and inspect journeys.`,
			GroupID: manageResourcesGroup,
		},
	}

	cmd.AddCommand(ModelEvaluationRunCmd())
	cmd.AddCommand(EvaluationDatasetCmd())
	cmd.AddCommand(ModelEvaluationPresetCmd())
	cmd.AddCommand(ModelEvaluationMetricCmd())
	cmd.AddCommand(CustomEvaluationMetricCmd())

	cmd.AddCommand(ScenarioSetCmd())
	cmd.AddCommand(ScenarioLibraryCmd())
	cmd.AddCommand(SimulationRunCmd())

	return cmd
}

// deprecatedEvaluationSubcommand is only for the simulation commands that moved
// from gradient to evaluation: scenario-set, scenario-library, and
// simulation-run. It is not a general gradient deprecator.
//
// Called from GradientAI() so `doctl gradient <name> ...` still works: it warns
// and forwards to `doctl evaluation <name> ...`. Other former gradient commands
// use their own helpers (e.g. deprecatedKnowledgeBaseCmd).
//
// It forwards rather than building a second copy of the tree: flags are bound to
// viper under `<parent>.<command>.<flag>`, so a second copy would rebind every
// flag to itself and leave the surviving tree unable to read them.
func deprecatedEvaluationSubcommand(name string, aliases ...string) *Command {
	return &Command{
		Command: &cobra.Command{
			Use:     name,
			Aliases: aliases,
			Short:   "Deprecated. Use `doctl evaluation " + name + "` instead.",
			Long:    "Deprecated. Use `doctl evaluation " + name + "` instead.",
			Hidden:  true,
			// The arguments belong to the command we forward to, not to this one.
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				warn("`doctl gradient %s` is deprecated and will be removed in a future release. Use `doctl evaluation %s` instead.", name, name)

				root := cmd.Root()
				root.SetArgs(append([]string{"evaluation", name}, args...))

				return root.Execute()
			},
		},
	}
}
