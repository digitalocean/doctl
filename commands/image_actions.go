package commands

import (
	"fmt"
	"io"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
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

	cmdImageActionsGet := cmdBuilder(cmd, RunImageActionsGet,
		"get <image-id>", "get image action", writer, displayerType(&action{}))
	addIntFlag(cmdImageActionsGet, doit.ArgActionID, 0, "action id", requiredOpt())

	cmdImageActionsTransfer := cmdBuilder(cmd, RunImageActionsTransfer,
		"transfer <image-id>", "transfer imagr", writer, displayerType(&action{}))
	addStringFlag(cmdImageActionsTransfer, doit.ArgRegionSlug, "", "region", requiredOpt())

	return cmd
}

// RunImageActionsGet retrieves an action for an image.
func RunImageActionsGet(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ias := do.NewImageActionsService(client)

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

	a, err := ias.Get(imageID, actionID)
	if err != nil {
		return err
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &action{actions: do.Actions{*a}},
		out:    out,
	}

	return displayOutput(dc)
}

// RunImageActionsTransfer an image.
func RunImageActionsTransfer(ns string, config doit.Config, out io.Writer, args []string) error {
	client := config.GetGodoClient()
	ias := do.NewImageActionsService(client)

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

	a, err := ias.Transfer(id, req)
	if err != nil {
		checkErr(fmt.Errorf("could not transfer image: %v", err))
	}

	dc := &outputConfig{
		ns:     ns,
		config: config,
		item:   &action{actions: do.Actions{*a}},
		out:    out,
	}

	return displayOutput(dc)
}
