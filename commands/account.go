package commands

import (
	"io"

	"github.com/bryanl/doit"
	"github.com/spf13/cobra"
)

// Account creates the account commands heirarchy.
func Account() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "account commands",
		Long:  "account is used to access account commands",
	}

	cmdBuilder(cmd, RunAccountGet, "get", "get account", writer,
		aliasOpt("g"), displayerType(&account{}))

	return cmd
}

// RunAccountGet runs account get.
func RunAccountGet(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	a, _, err := client.Account.Get()
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &account{Account: a},
		out:    out,
	}

	return displayOutput(dc)
}
