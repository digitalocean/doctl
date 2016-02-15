package commands

import (
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
func RunCmdActionList(c *cmdConfig) error {
	as := c.actionsService()

	newActions, err := as.List()
	if err != nil {
		return err
	}

	item := &action{actions: newActions}
	return c.display(item)
}

// RunCmdActionGet runs action get.
func RunCmdActionGet(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	id, err := strconv.Atoi(c.args[0])
	if err != nil {
		return err
	}

	as := c.actionsService()
	a, err := as.Get(id)
	if err != nil {
		return err
	}

	return c.display(&action{actions: do.Actions{*a}})
}

// RunCmdActionWait waits for an action to complete or error.
func RunCmdActionWait(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	id, err := strconv.Atoi(c.args[0])
	if err != nil {
		return err
	}

	pollTime, err := c.doitConfig.GetInt(c.ns, doit.ArgPollTime)
	if err != nil {
		return err
	}

	as := c.actionsService()

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

	return c.display(&action{actions: do.Actions{*a}})
}
