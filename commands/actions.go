package commands

import (
	"io"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/spf13/cobra"
)

// Actions creates the action commands heirarchy.
func Actions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "action",
		Short: "action commands",
		Long:  "action is used to access action commands",
	}

	cmdBuilder(cmd, RunCmdActionGet, "get ACTIONID", "get action", writer,
		aliasOpt("g"), displayerType(&action{}))

	cmdBuilder(cmd, RunCmdActionList, "list", "list actions", writer,
		aliasOpt("ls"), displayerType(&action{}))

	return cmd
}

// RunCmdActionList run action list.
func RunCmdActionList(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	as := do.NewActionsService(client)

	newActions, err := as.List()
	if err != nil {
		return err
	}

	// FIXME remove once all actions are fixed
	oldActions := actions{}
	for _, a := range newActions {
		oldActions = append(oldActions, *a.Action)
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &action{actions: oldActions},
		out:    out,
	}

	return displayOutput(dc)
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
	as := do.NewActionsService(client)
	a, err := as.Get(id)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &action{actions: actions{*a.Action}},
		out:    out,
	}

	return displayOutput(dc)
}
