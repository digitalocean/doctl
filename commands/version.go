package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version string

func Version() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "show the version",
		Run: func(cmd *cobra.Command, args []string) {
			currentVer := version
			if currentVer == "" {
				currentVer = "dev"
			}

			fmt.Println(currentVer)
		},
	}
}
