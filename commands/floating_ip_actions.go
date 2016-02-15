package commands

import (
	"fmt"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/spf13/cobra"
)

// FloatingIPAction creates the floating IP action commmand.
func FloatingIPAction() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "floating-ip-action",
		Short:   "floating IP action commands",
		Long:    "floating IP action commands",
		Aliases: []string{"fipa"},
	}

	cmdBuilder2(cmd, RunFloatingIPActionsGet,
		"get <floating-ip> <action-id>", "get floating-ip action", writer, displayerType(&action{}))

	cmdBuilder2(cmd, RunFloatingIPActionsAssign,
		"assign <floating-ip> <droplet-id>", "assign a floating IP to a droplet", writer, displayerType(&action{}))

	cmdBuilder2(cmd, RunFloatingIPActionsUnassign,
		"unassign <floating-ip>", "unassign a floating IP to a droplet", writer, displayerType(&action{}))

	return cmd
}

// RunFloatingIPActionsGet retrieves an action for a floating IP.
func RunFloatingIPActionsGet(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	ip := c.args[0]

	fia := c.floatingIPActionsService()

	actionID, err := strconv.Atoi(c.args[1])
	if err != nil {
		return err
	}

	a, err := fia.Get(ip, actionID)
	if err != nil {
		return err
	}

	item := &action{actions: do.Actions{*a}}
	return c.display(item)
}

// RunFloatingIPActionsAssign assigns a floating IP to a droplet.
func RunFloatingIPActionsAssign(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	ip := c.args[0]

	fia := c.floatingIPActionsService()

	dropletID, err := strconv.Atoi(c.args[1])
	if err != nil {
		return err
	}

	a, err := fia.Assign(ip, dropletID)
	if err != nil {
		checkErr(fmt.Errorf("could not assign IP to droplet: %v", err))
	}

	item := &action{actions: do.Actions{*a}}
	return c.display(item)
}

// RunFloatingIPActionsUnassign unassigns a floating IP to a droplet.
func RunFloatingIPActionsUnassign(c *cmdConfig) error {
	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	ip := c.args[0]

	fia := c.floatingIPActionsService()

	a, err := fia.Unassign(ip)
	if err != nil {
		checkErr(fmt.Errorf("could not unassign IP to droplet: %v", err))
	}

	item := &action{actions: do.Actions{*a}}
	return c.display(item)
}
