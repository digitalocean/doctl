package commands

import (
	"bytes"
	"fmt"

	"github.com/spf13/cobra"
)

type version struct {
	Major, Minor, Patch int
	Name, Build, Label  string
}

var (
	DoitVersion = version{
		Major: 0,
		Minor: 6,
		Patch: 1,
		Name:  "Maroon Marion",
		Label: "dev",
	}

	Build string
)

func Version() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "show the current version",
		Run: func(cmd *cobra.Command, args []string) {
			DoitVersion.Build = Build
			fmt.Println(DoitVersion)
		},
	}
}

func (v version) String() string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("doit version %d.%d.%d", v.Major, v.Minor, v.Patch))

	if v.Label != "" {
		buffer.WriteString("-" + v.Label)
	}

	buffer.WriteString(fmt.Sprintf(" %q", v.Name))

	if v.Build != "" {
		buffer.WriteString(fmt.Sprintf("\nGit commit hash: %s", v.Build))
	}

	return buffer.String()
}
