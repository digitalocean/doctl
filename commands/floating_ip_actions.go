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

	CmdBuilder(cmd, RunFloatingIPActionsGet,
		"get <floating-ip> <action-id>", "get floating-ip action", Writer,
		displayerType(&action{}), docCategories("floatingip"))

	CmdBuilder(cmd, RunFloatingIPActionsAssign,
		"assign <floating-ip> <droplet-id>", "assign a floating IP to a droplet", Writer,
		displayerType(&action{}), docCategories("floatingip"))

	CmdBuilder(cmd, RunFloatingIPActionsUnassign,
		"unassign <floating-ip>", "unassign a floating IP to a droplet", Writer,
		displayerType(&action{}), docCategories("floatingip"))

	return cmd
}

// RunFloatingIPActionsGet retrieves an action for a floating IP.
func RunFloatingIPActionsGet(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doit.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	fia := c.FloatingIPActions()

	actionID, err := strconv.Atoi(c.Args[1])
	if err != nil {
		return err
	}

	a, err := fia.Get(ip, actionID)
	if err != nil {
		return err
	}

	item := &action{actions: do.Actions{*a}}
	return c.Display(item)
}

// RunFloatingIPActionsAssign assigns a floating IP to a droplet.
func RunFloatingIPActionsAssign(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doit.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	fia := c.FloatingIPActions()

	dropletID, err := strconv.Atoi(c.Args[1])
	if err != nil {
		return err
	}

	a, err := fia.Assign(ip, dropletID)
	if err != nil {
		checkErr(fmt.Errorf("could not assign IP to droplet: %v", err))
	}

	item := &action{actions: do.Actions{*a}}
	return c.Display(item)
}

// RunFloatingIPActionsUnassign unassigns a floating IP to a droplet.
func RunFloatingIPActionsUnassign(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	fia := c.FloatingIPActions()

	a, err := fia.Unassign(ip)
	if err != nil {
		checkErr(fmt.Errorf("could not unassign IP to droplet: %v", err))
	}

	item := &action{actions: do.Actions{*a}}
	return c.Display(item)
}
