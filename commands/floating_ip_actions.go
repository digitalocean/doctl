package commands

import (
	"io"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/cobra"
)

// FloatingIPAction creates the floating IP action commmand.
func FloatingIPAction() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "floating-ip-action",
		Short: "floating IP action commands",
		Long:  "floating IP action commands",
	}

	cmdFloatingIPActionsGet := cmdBuilder(RunFloatingIPActionsGet,
		"get", "get floating-ip action", writer)
	cmd.AddCommand(cmdFloatingIPActionsGet)
	addStringFlag(cmdFloatingIPActionsGet, doit.ArgIPAddress, "", "floating IP address")
	addIntFlag(cmdFloatingIPActionsGet, doit.ArgActionID, 0, "action id")

	cmdFloatingIPActionsAssign := cmdBuilder(RunFloatingIPActionsAssign,
		"assign", "assign a floating IP to a droplet", writer)
	cmd.AddCommand(cmdFloatingIPActionsAssign)
	addStringFlag(cmdFloatingIPActionsAssign, doit.ArgIPAddress, "", "floating IP address")
	addIntFlag(cmdFloatingIPActionsAssign, doit.ArgDropletID, 0, "ID of the droplet to assign the IP to")

	cmdFloatingIPActionsUnassign := cmdBuilder(RunFloatingIPActionsUnassign,
		"unassign", "unassign a floating IP to a droplet", writer)
	cmd.AddCommand(cmdFloatingIPActionsUnassign)
	addStringFlag(cmdFloatingIPActionsUnassign, doit.ArgIPAddress, "", "floating IP address")

	return cmd
}

// RunFloatingIPActionsGet retrieves an action for a floating IP.
func RunFloatingIPActionsGet(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()
	ip, err := config.GetString(ns, doit.ArgIPAddress)
	if err != nil {
		return err
	}

	actionID, err := config.GetInt(ns, doit.ArgActionID)
	if err != nil {
		return err
	}

	action, _, err := client.FloatingIPActions.Get(ip, actionID)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(action, out)
}

// RunFloatingIPActionsAssign assigns a floating IP to a droplet.
func RunFloatingIPActionsAssign(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()
	ip, err := config.GetString(ns, doit.ArgIPAddress)
	if err != nil {
		return err
	}

	dropletID, err := config.GetInt(ns, doit.ArgDropletID)
	if err != nil {
		return err
	}

	action, _, err := client.FloatingIPActions.Assign(ip, dropletID)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not assign IP to droplet")
	}
	return doit.DisplayOutput(action, out)
}

// RunFloatingIPActionsUnassign unassigns a floating IP to a droplet.
func RunFloatingIPActionsUnassign(ns string, config doit.Config, out io.Writer) error {
	client := config.GetGodoClient()
	ip, err := config.GetString(ns, doit.ArgIPAddress)
	if err != nil {
		return err
	}

	action, _, err := client.FloatingIPActions.Unassign(ip)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not unsassign IP to droplet")
	}
	return doit.DisplayOutput(action, out)
}
