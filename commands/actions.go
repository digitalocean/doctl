package commands

import (
	"io"
	"strconv"
	"time"

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

	cmdActionWait := cmdBuilder(cmd, RunCmdActionWait, "wait ACTIONID", "wait for action to complete", writer,
		aliasOpt("w"), displayerType(&action{}))
	addIntFlag(cmdActionWait, doit.ArgPollTime, 5, "Re-poll time in seconds",
		shortFlag("p"))

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

	item := &action{actions: newActions}
	dc := &displayer{
		ns:     ns,
		config: config,
		item:   item,
		out:    out,
	}

	return dc.Display()
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

	dc := &displayer{
		ns:     ns,
		config: config,
		item:   &action{actions: do.Actions{*a}},
		out:    out,
	}

	return dc.Display()
}

// RunCmdActionWait waits for an action to complete or error.
func RunCmdActionWait(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	pollTime, err := config.GetInt(ns, doit.ArgPollTime)
	if err != nil {
		return err
	}

	client := config.GetGodoClient()
	as := do.NewActionsService(client)

	var a *do.Action

	for {
		a, err = as.Get(id)
		if err != nil {
			return err
		}

		if a.Status != "in-progress" {
			break
		}

		time.Sleep(time.Duration(pollTime) * time.Second)
	}

	dc := &displayer{
		ns:     ns,
		config: config,
		item:   &action{actions: do.Actions{*a}},
		out:    out,
	}

	return dc.Display()
}
