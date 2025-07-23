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

import "github.com/spf13/cobra"

// GenAI creates the genai command and adds the agent subcommand.
func GenAI() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "genai",
			Aliases: []string{"ai"},
			Short:   "Manage GenAI resources",
			Long:    "The subcommands of `doctl genai` manage your GenAI resources.",
			GroupID: manageResourcesGroup,
		},
	}

	// Add the agent command as a subcommand to genai
	cmd.AddCommand(AgentCmd())
	// Add the knowledgebase command as a subcommand to genai
	cmd.AddCommand(KnowledgeBaseCmd())
	// Add the OpenAI keys command as a subcommand to genai
	cmd.AddCommand(OpenAIKeyCmd())

	return cmd
}
