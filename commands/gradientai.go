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

// GradientAI creates the gradient command and adds the knowledge base and
// deprecated simulation aliases.
func GradientAI() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "gradient",
			Aliases: []string{"ai", "genai", "gradientai"},
			Short:   "Manage Gradient AI resources",
			Long: `doctl gradient is deprecated and hidden from help.

Public commands moved to:
  doctl knowledge-base
  doctl inference
  doctl evaluation`,
			GroupID: manageResourcesGroup,
			// All public children moved to knowledge-base / inference / evaluation;
			// keep this parent only for hidden/deprecated aliases.
			Hidden: true,
		},
	}

	// Kept under gradient but hidden (same pattern as scenario/simulation cmds).
	// Public home is now doctl knowledge-base.
	cmd.AddCommand(deprecatedKnowledgeBaseCmd())

	// Simulation APIs moved to doctl evaluation; keep hidden deprecated aliases.
	cmd.AddCommand(deprecatedEvaluationSubcommand("scenario-set", "ss", "scenario-sets"))
	cmd.AddCommand(deprecatedEvaluationSubcommand("scenario-library", "sl"))
	cmd.AddCommand(deprecatedEvaluationSubcommand("simulation-run", "sim", "simulation-runs"))

	return cmd
}
