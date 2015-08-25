package commands

import (
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Droplet creates the droplet command.
func ImageAction() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image-action",
		Short: "image-action commands",
		Long:  "image-action commands",
	}

	cmdImageActionsGet := cmdBuilder(RunImageActionsGet,
		"get", "get image action", writer)
	cmd.AddCommand(cmdImageActionsGet)
	addIntFlag(cmdImageActionsGet, doit.ArgImageID, 0, "image id")
	addIntFlag(cmdImageActionsGet, doit.ArgActionID, 0, "action id")

	cmdImageActionsTransfer := cmdBuilder(RunImageActionsTransfer,
		"transfer", "transfer imagr", writer)
	cmd.AddCommand(cmdImageActionsTransfer)
	addIntFlag(cmdImageActionsTransfer, doit.ArgImageID, 0, "image id")
	addStringFlag(cmdImageActionsTransfer, doit.ArgRegionSlug, "", "region")

	return cmd
}

// Get retrieves an action for an image.
func RunImageActionsGet(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	imageID := doit.VConfig.GetInt(ns, doit.ArgImageID)
	actionID := doit.VConfig.GetInt(ns, doit.ArgActionID)

	action, _, err := client.ImageActions.Get(imageID, actionID)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(action, out)
}

// Tranfer an image.
func RunImageActionsTransfer(ns string, out io.Writer) error {
	client := doit.VConfig.GetGodoClient()
	id := doit.VConfig.GetInt(ns, doit.ArgImageID)
	req := &godo.ActionRequest{
		"region": doit.VConfig.GetString(ns, doit.ArgRegionSlug),
	}

	action, _, err := client.ImageActions.Transfer(id, req)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not transfer image")
	}

	return doit.DisplayOutput(action, out)
}
