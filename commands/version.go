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
				if doit.Build != "" {
					doit.DoitVersion.Build = doit.Build
				}
				if doit.Major != 0 {
					doit.DoitVersion.Major = doit.Major
				}
				if doit.Minor != 0 {
					doit.DoitVersion.Minor = doit.Minor
				}
				if doit.Patch != 0 {
					doit.DoitVersion.Patch = doit.Patch
				}
				if doit.Label != "" {
					doit.DoitVersion.Label = doit.Label
				}

				fmt.Println(doit.DoitVersion)
			},
		},
	}
}
