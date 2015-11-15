package commands

import (
	"io"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/cobra"
)

// Account creates the account commands heirarchy.
func Account() *cobra.Command {
	cmdAccount := &cobra.Command{
		Use:   "account",
		Short: "account commands",
		Long:  "account is used to access account commands",
	}

	cmdAccountGet := cmdBuilder(RunAccountGet, "get", "get account", writer, aliasOpt("g"))
	cmdAccount.AddCommand(cmdAccountGet)

	return cmdAccount
}

// RunAccountGet runs account get.
func RunAccountGet(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	a, _, err := client.Account.Get()
	if err != nil {
		return err
	}

	return doit.DisplayOutput(a, out)
}
