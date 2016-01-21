package commands

import (
	"fmt"
	"io"

	"github.com/bryanl/doit"
	"github.com/spf13/cobra"
)

// FloatingIPAction creates the floating IP action commmand.
func FloatingIPAction() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "floating-ip-action",
		Short: "floating IP action commands",
		Long:  "floating IP action commands",
	}

	cmdFloatingIPActionsGet := cmdBuilder(cmd, RunFloatingIPActionsGet,
		"get <floating-ip>", "get floating-ip action", writer)
	addIntFlag(cmdFloatingIPActionsGet, doit.ArgActionID, 0, "action id", requiredOpt())

	cmdFloatingIPActionsAssign := cmdBuilder(cmd, RunFloatingIPActionsAssign,
		"assign <floating-ip>", "assign a floating IP to a droplet", writer)
	addIntFlag(cmdFloatingIPActionsAssign, doit.ArgDropletID, 0, "ID of the droplet to assign the IP to", requiredOpt())

	cmdBuilder(cmd, RunFloatingIPActionsUnassign,
		"unassign <floating-ip>", "unassign a floating IP to a droplet", writer)

	return cmd
}

// RunFloatingIPActionsGet retrieves an action for a floating IP.
func RunFloatingIPActionsGet(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	ip := args[0]

	client := config.GetGodoClient()

	actionID, err := config.GetInt(ns, doit.ArgActionID)
	if err != nil {
		return err
	}

	a, _, err := client.FloatingIPActions.Get(ip, actionID)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &action{actions: actions{*a}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunFloatingIPActionsAssign assigns a floating IP to a droplet.
func RunFloatingIPActionsAssign(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	ip := args[0]

	client := config.GetGodoClient()

	dropletID, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	a, _, err := client.FloatingIPActions.Assign(ip, dropletID)
	if err != nil {
		checkErr(fmt.Errorf("could not assign IP to droplet: %v", err))
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &action{actions: actions{*a}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunFloatingIPActionsUnassign unassigns a floating IP to a droplet.
func RunFloatingIPActionsUnassign(ns string, config doit.Config, out io.Writer, args []string) error {
	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	ip := args[0]

	client := config.GetGodoClient()

	a, _, err := client.FloatingIPActions.Unassign(ip)
	if err != nil {
		checkErr(fmt.Errorf("could not unassign IP to droplet: %v", err))
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &action{actions: actions{*a}},
		out:    out,
	}
	return displayOutput(dc)
}
