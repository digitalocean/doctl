package commands

import (
	"errors"
	"io"
	"os"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Actions creates the action commands heirarchy.
func Actions() *cobra.Command {
	cmdActions := &cobra.Command{
		Use:   "action",
		Short: "action commands",
		Long:  "action is used to access action commands",
	}

	cmdActionGet := NewCmdActionGet(os.Stdout)
	addIntFlag(cmdActionGet, doit.ArgActionID, 0, "Action ID")
	cmdActions.AddCommand(cmdActionGet)

	cmdActionList := NewCmdActionList(os.Stdout)
	cmdActions.AddCommand(cmdActionList)

	return cmdActions
}

// NewCmdActionList creates an action list command.
func NewCmdActionList(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "action list",
		Long:  "list actions",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunActionList(out))
		},
	}
}

// RunActionList run action list.
func RunActionList(out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Actions.List(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := doit.PaginateResp(f)
	if err != nil {
		return err
	}

	list := make([]godo.Action, len(si))
	for i := range si {
		list[i] = si[i].(godo.Action)
	}

	return doit.DisplayOutput(list, out)
}

// NewCmdActionGet creates an action get command.
func NewCmdActionGet(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "action get",
		Long:  "get action",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunActionGet(out))
		},
	}
}

// RunActionGet runs action get.
func RunActionGet(out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetInt(doit.ArgActionID)
	if id < 1 {
		return errors.New("invalid action id")
	}

	a, _, err := client.Actions.Get(id)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(a, out)
}
