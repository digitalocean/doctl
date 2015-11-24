package commands

import (
	"io"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/Sirupsen/logrus"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// ImageAction creates the image action commmand.
func ImageAction() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image-action",
		Short: "image-action commands",
		Long:  "image-action commands",
	}

	cmdImageActionsGet := cmdBuilder(RunImageActionsGet,
		"get <image-id>", "get image action", writer)
	cmd.AddCommand(cmdImageActionsGet)
	addIntFlag(cmdImageActionsGet, doit.ArgActionID, 0, "action id", requiredOpt())

	cmdImageActionsTransfer := cmdBuilder(RunImageActionsTransfer,
		"transfer <image-id>", "transfer imagr", writer)
	cmd.AddCommand(cmdImageActionsTransfer)
	addStringFlag(cmdImageActionsTransfer, doit.ArgRegionSlug, "", "region", requiredOpt())

	return cmd
}

// RunImageActionsGet retrieves an action for an image.
func RunImageActionsGet(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	imageID, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	actionID, err := config.GetInt(ns, doit.ArgActionID)
	if err != nil {
		return err
	}

	action, _, err := client.ImageActions.Get(imageID, actionID)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(action, out)
}

// RunImageActionsTransfer an image.
func RunImageActionsTransfer(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()

	if len(args) != 1 {
		return doit.NewMissingArgsErr(ns)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	region, err := config.GetString(ns, doit.ArgRegionSlug)
	if err != nil {
		return err
	}

	req := &godo.ActionRequest{
		"region": region,
	}

	action, _, err := client.ImageActions.Transfer(id, req)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not transfer image")
	}

	return doit.DisplayOutput(action, out)
}
