package commands

import "github.com/spf13/cobra"

func GenAI() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "genai",
			Aliases: []string{"ai"},
			Short:   "Display commands that manage DigitalOcean GenAI Agents.",
			Long:    "The subcommands of `doctl agents` allow you to access and manage GenAI Agents.",
			GroupID: manageResourcesGroup,
		},
	}

	cmd.AddCommand(KnowledgeBase())

	return cmd
}
