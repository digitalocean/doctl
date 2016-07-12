package commands

import (
	"strconv"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

type volumeActionFn func(das do.VolumeActionsService) (*do.Action, error)

func performVolumeAction(c *CmdConfig, fn volumeActionFn) error {
	das := c.VolumeActions()

	a, err := fn(das)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		a, err = actionWait(c, a.ID, 5)
		if err != nil {
			return err
		}

	}

	item := &action{actions: do.Actions{*a}}
	return c.Display(item)
}

// VolumeAction creates the volume command
func VolumeAction() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "volume-action",
			Short: "volume action commands",
			Long:  "volume-action is used to access volume action commands",
		},
	}

	CmdBuilder(cmd, RunVolumeAttach, "attach <volume-id> <droplet-id>", "attach a volume", Writer,
		aliasOpt("a"))

	CmdBuilder(cmd, RunVolumeDetach, "detach <volume-id>", "detach a volume", Writer,
		aliasOpt("d"))

	return cmd

}

// RunVolumeAttach attaches a volume to a droplet.
func RunVolumeAttach(c *CmdConfig) error {
	fn := func(das do.VolumeActionsService) (*do.Action, error) {
		if len(c.Args) != 2 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		volumeID := c.Args[0]
		dropletID, err := strconv.Atoi(c.Args[1])
		if err != nil {
			return nil, err

		}
		a, err := das.Attach(volumeID, dropletID)
		return a, err
	}
	return performVolumeAction(c, fn)
}

// RunVolumeDetach detaches a volume from the droplet it's attached to.
func RunVolumeDetach(c *CmdConfig) error {
	fn := func(das do.VolumeActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		volumeID := c.Args[0]
		a, err := das.Detach(volumeID)
		return a, err
	}
	return performVolumeAction(c, fn)
}
