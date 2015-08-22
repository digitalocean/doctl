package commands

import (
	"io"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
	"github.com/spf13/cobra"
)

// Account creates the account commands heirarchy.
func Account() *cobra.Command {
	cmdAccount := &cobra.Command{
		Use:   "account",
		Short: "account commands",
		Long:  "account is used to access account commands",
	}

	cmdAccount.AddCommand(NewCmdAccountGet(os.Stdout))
	return cmdAccount
}

// NewCmdAccountGet creates an Account get command.
func NewCmdAccountGet(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "account info",
		Long:  "get account details",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunAccountGet(out))
		},
	}
}

// RunAccountGet runs account get.
func RunAccountGet(out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	a, _, err := client.Account.Get()
	if err != nil {
		logrus.WithField("err", err).Error("unable to retrieve account")
	}

	return doit.WriteJSON(a, out)
}
