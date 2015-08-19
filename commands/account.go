package commands

import (
	"github.com/bryanl/doit"
	"github.com/spf13/cobra"
)

func Account() *cobra.Command {
	cmdAccount := &cobra.Command{
		Use:   "account",
		Short: "account commands",
		Long:  "account is used to access account commands",
	}

	cmdAccountGet := &cobra.Command{
		Use:   "get",
		Short: "account info",
		Long:  "get account details",
		Run: func(cmd *cobra.Command, args []string) {
			doit.NewAccountGet()
		},
	}

	cmdAccount.AddCommand(cmdAccountGet)

	return cmdAccount
}
