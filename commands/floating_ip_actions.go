package commands

import (
	"fmt"
	"io"
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

	cmdBuilder(cmd, RunFloatingIPActionsGet,
		"get <floating-ip> <action-id>", "get floating-ip action", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunFloatingIPActionsAssign,
		"assign <floating-ip> <droplet-id>", "assign a floating IP to a droplet", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunFloatingIPActionsUnassign,
		"unassign <floating-ip>", "unassign a floating IP to a droplet", writer, displayerType(&action{}))

	return cmd
}

// RunFloatingIPActionsGet retrieves an action for a floating IP.
func RunFloatingIPActionsGet(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	ip := args[0]

	client := config.GetGodoClient()
	fia := do.NewFloatingIPActionsService(client)

	actionID, err := strconv.Atoi(args[1])
	if err != nil {
		return err
	}

	a, err := fia.Get(ip, actionID)
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

// RunFloatingIPActionsAssign assigns a floating IP to a droplet.
func RunFloatingIPActionsAssign(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	ip := args[0]

	client := config.GetGodoClient()
	fia := do.NewFloatingIPActionsService(client)

	dropletID, err := strconv.Atoi(args[1])
	if err != nil {
		return err
	}

	a, err := fia.Assign(ip, dropletID)
	if err != nil {
		checkErr(fmt.Errorf("could not assign IP to droplet: %v", err))
	}

	dc := &displayer{
		ns:     ns,
		config: config,
		item:   &action{actions: do.Actions{*a}},
		out:    out,
	}

	return dc.Display()
}

// RunFloatingIPActionsUnassign unassigns a floating IP to a droplet.
func RunFloatingIPActionsUnassign(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	ip := args[0]

	client := config.GetGodoClient()
	fia := do.NewFloatingIPActionsService(client)

	a, err := fia.Unassign(ip)
	if err != nil {
		checkErr(fmt.Errorf("could not unassign IP to droplet: %v", err))
	}

	dc := &displayer{
		ns:     ns,
		config: config,
		item:   &action{actions: do.Actions{*a}},
		out:    out,
	}
	return dc.Display()
}
