package commands

import (
	"io"
	"strconv"

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

	cmdActionGet := cmdBuilder(RunCmdActionGet, "get ACTIONID", "get action", writer, aliasOpt("g"))
	cmdActions.AddCommand(cmdActionGet)

	cmdActionList := cmdBuilder(RunCmdActionList, "list", "list actions", writer, aliasOpt("ls"))
	cmdActions.AddCommand(cmdActionList)

	return cmdActions
}

// RunCmdActionList run action list.
func RunCmdActionList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
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

	return displayOutput(&action{actions: list}, out)
}

// RunCmdActionGet runs action get.
func RunCmdActionGet(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	client := config.GetGodoClient()

	a, _, err := client.Actions.Get(id)
	if err != nil {
		return err
	}

	return displayOutput(&action{actions: actions{*a}}, out)
}
