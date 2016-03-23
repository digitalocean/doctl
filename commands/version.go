package commands

import (
	"fmt"
	"strconv"

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
				if doit.Major != "" {
					i, _ := strconv.Atoi(doit.Major)
					doit.DoitVersion.Major = i
				}
				if doit.Minor != "" {
					i, _ := strconv.Atoi(doit.Minor)
					doit.DoitVersion.Minor = i
				}
				if doit.Patch != "" {
					i, _ := strconv.Atoi(doit.Patch)
					doit.DoitVersion.Patch = i
				}
				if doit.Label != "" {
					doit.DoitVersion.Label = doit.Label
				}

				fmt.Println(doit.DoitVersion)
			},
		},
	}
}
