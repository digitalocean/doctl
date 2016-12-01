package commands

import (
	"github.com/digitalocean/doctl"
	"github.com/spf13/cobra"
)

func Snapshot() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "snapshot",
			Aliases: []string{"d"},
			Short:   "snapshot commands",
			Long:    "snapshot is used to access snapshot commands",
		},
		IsIndex: true,
	}

	CmdBuilder(cmd, RunSnapshotList, "list", "list", Writer,
		aliasOpt("ls"), displayerType(&snapshot{}), docCategories("droplet"))

	CmdBuilder(cmd, RunSnapshotListVolume, "lvolume", "list volume", Writer,
		aliasOpt("lsv"), displayerType(&snapshot{}), docCategories("droplet"))

	CmdBuilder(cmd, RunSnapshotListDroplet, "ldroplet", "list droplet", Writer,
		aliasOpt("lsd"), displayerType(&snapshot{}), docCategories("droplet"))

	/*	cmdRunDropletGet := CmdBuilder(cmd, RunSnapshotGet, "get", "get snapshot", Writer,
		aliasOpt("g"), displayerType(&droplet{}), docCategories("droplet"))*/

	CmdBuilder(cmd, RunSnapshotDelete, "delete", "delete snapshot", Writer,
		aliasOpt("d"), displayerType(&droplet{}), docCategories("droplet"))

	return cmd
}

func RunSnapshotList(c *CmdConfig) error {
	ss := c.Snapshots()

	list, err := ss.List()
	if err != nil {
		return err
	}
	item := &snapshot{snapshots: list}
	return c.Display(item)
}

func RunSnapshotListVolume(c *CmdConfig) error {
	ss := c.Snapshots()

	list, err := ss.ListVolume()
	if err != nil {
		return err
	}
	item := &snapshot{snapshots: list}
	return c.Display(item)
}

func RunSnapshotListDroplet(c *CmdConfig) error {
	ss := c.Snapshots()

	list, err := ss.ListDroplet()
	if err != nil {
		return err
	}
	item := &snapshot{snapshots: list}
	return c.Display(item)
}

/*func RunSnapshotGet(c *CmdConfig) error {
	snapshotId, err := getSnapshotIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	ss := c.Snapshots()

	s, err := ss.Get(snapshotId)
	if err != nil {
		return err
	}

	//item := &snapshot{snapshots: do.Snapshot{*s}}

	return nil
}*/

func RunSnapshotDelete(c *CmdConfig) error {
	snapshotId, err := getSnapshotIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	ss := c.Snapshots()

	derr := ss.Delete(snapshotId)
	if derr != nil {
		return derr
	}

	return nil
}

func getSnapshotIDArg(ns string, args []string) (string, error) {
	if len(args) != 1 {
		return "", doctl.NewMissingArgsErr(ns)
	}

	return args[0], nil
}
