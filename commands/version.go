package commands

import (
	"fmt"

	"github.com/bryanl/doit"
	"github.com/spf13/cobra"
)

type version struct {
	Major, Minor, Patch int
	Name, Build, Label  string
}

// Version creates a version command.
func Version() *Command {
	return &Command{
		Command: &cobra.Command{
			Use:   "version",
			Short: "show the current version",
			Run: func(cmd *cobra.Command, args []string) {
				doit.DoitVersion.Build = doit.Build
				fmt.Println(doit.DoitVersion)
			},
		},
	}
}
