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

	CmdBuilder(cmd, RunCmdActionGet, "get ACTIONID", "get action", Writer,
		aliasOpt("g"), displayerType(&action{}))

	CmdBuilder(cmd, RunCmdActionList, "list", "list actions", Writer,
		aliasOpt("ls"), displayerType(&action{}))

	cmdActionWait := CmdBuilder(cmd, RunCmdActionWait, "wait ACTIONID", "wait for action to complete", Writer,
		aliasOpt("w"), displayerType(&action{}))
	AddIntFlag(cmdActionWait, doit.ArgPollTime, 5, "Re-poll time in seconds",
		shortFlag("p"))

	return cmd
}

// RunCmdActionList run action list.
func RunCmdActionList(c *CmdConfig) error {
	as := c.Actions()

	newActions, err := as.List()
	if err != nil {
		return err
	}

	item := &action{actions: newActions}
	return c.Display(item)
}

// RunCmdActionGet runs action get.
func RunCmdActionGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	id, err := strconv.Atoi(c.Args[0])
	if err != nil {
		return err
	}

	as := c.Actions()
	a, err := as.Get(id)
	if err != nil {
		return err
	}

	return c.Display(&action{actions: do.Actions{*a}})
}

// RunCmdActionWait waits for an action to complete or error.
func RunCmdActionWait(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	id, err := strconv.Atoi(c.Args[0])
	if err != nil {
		return err
	}

	pollTime, err := c.Doit.GetInt(c.NS, doit.ArgPollTime)
	if err != nil {
		return err
	}

	as := c.Actions()

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

	return c.Display(&action{actions: do.Actions{*a}})
}
