package commands

import (
	"strconv"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

type driveActionFn func(das do.DriveActionsService) (*do.Action, error)

func performDriveAction(c *CmdConfig, fn driveActionFn) error {
	das := c.DriveActions()

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

// DriveAction creates the drive command
// NOTE: This command will only work for those accepted
// into the block storage private beta on DigitalOcean
func DriveAction() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "drive-action",
			Short: "drive action commands",
			Long:  "drive-action is used to access drive action commands",
		},
	}
	defer betaCmd()(cmd) // TODO(antoine): remove once out of beta

	CmdBuilder(cmd, RunDriveAttach, "attach <drive-id> <droplet-id>", "attach a drive", Writer,
		aliasOpt("a"))

	CmdBuilder(cmd, RunDriveDetach, "detach <drive-id>", "detach a drive", Writer,
		aliasOpt("d"))

	return cmd

}

// RunDriveAttach attaches a drive to a droplet.
func RunDriveAttach(c *CmdConfig) error {
	fn := func(das do.DriveActionsService) (*do.Action, error) {
		if len(c.Args) != 2 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		driveID := c.Args[0]
		dropletID, err := strconv.Atoi(c.Args[1])
		if err != nil {
			return nil, err

		}
		a, err := das.Attach(driveID, dropletID)
		return a, err
	}
	return performDriveAction(c, fn)
}

// RunDriveDetach detaches a drive from the droplet it's attached to.
func RunDriveDetach(c *CmdConfig) error {
	fn := func(das do.DriveActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		driveID := c.Args[0]
		a, err := das.Detach(driveID)
		return a, err
	}
	return performDriveAction(c, fn)
}
