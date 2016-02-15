package commands

import (
	"fmt"
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

	cmdImageActionsGet := cmdBuilder2(cmd, RunImageActionsGet,
		"get <image-id>", "get image action", writer, displayerType(&action{}))
	addIntFlag(cmdImageActionsGet, doit.ArgActionID, 0, "action id", requiredOpt())

	cmdImageActionsTransfer := cmdBuilder2(cmd, RunImageActionsTransfer,
		"transfer <image-id>", "transfer imagr", writer, displayerType(&action{}))
	addStringFlag(cmdImageActionsTransfer, doit.ArgRegionSlug, "", "region", requiredOpt())

	return cmd
}

// RunImageActionsGet retrieves an action for an image.
func RunImageActionsGet(c *cmdConfig) error {
	ias := c.imageActionsService()

	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	imageID, err := strconv.Atoi(c.args[0])
	if err != nil {
		return err
	}

	actionID, err := c.doitConfig.GetInt(c.ns, doit.ArgActionID)
	if err != nil {
		return err
	}

	a, err := ias.Get(imageID, actionID)
	if err != nil {
		return err
	}

	item := &action{actions: do.Actions{*a}}
	return c.display(item)
}

// RunImageActionsTransfer an image.
func RunImageActionsTransfer(c *cmdConfig) error {
	ias := c.imageActionsService()

	if len(c.args) != 1 {
		return doit.NewMissingArgsErr(c.ns)
	}

	id, err := strconv.Atoi(c.args[0])
	if err != nil {
		return err
	}

	region, err := c.doitConfig.GetString(c.ns, doit.ArgRegionSlug)
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

	item := &action{actions: do.Actions{*a}}
	return c.display(item)
}
