package commands

import (
	"io"
	"os"

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

	cmdImageActionsGet := NewCmdImageActionsGet(os.Stdout)
	cmd.AddCommand(cmdImageActionsGet)
	addIntFlag(cmdImageActionsGet, doit.ArgImageID, 0, "image id")
	addIntFlag(cmdImageActionsGet, doit.ArgActionID, 0, "action id")

	cmdImageActionsTransfer := NewCmdImageActionsTransfer(os.Stdout)
	cmd.AddCommand(cmdImageActionsTransfer)
	addIntFlag(cmdImageActionsTransfer, doit.ArgImageID, 0, "image id")
	addStringFlag(cmdImageActionsTransfer, doit.ArgRegionSlug, "", "region")

	return cmd
}

// NewCmdDropletActions creates a droplet action get command.
func NewCmdImageActionsGet(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "get image action",
		Long:  "get image action",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunImageActionsGet(cmdNS(cmd), out), cmd)
		},
	}
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

// NewCmdDropletActions creates a droplet action get command.
func NewCmdImageActionsTransfer(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "transfer",
		Short: "transfer image",
		Long:  "transfer image",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunImageActionsTransfer(cmdNS(cmd), out), cmd)
		},
	}
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
