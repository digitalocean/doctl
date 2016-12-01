package commands

import "github.com/spf13/cobra"

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

	CmdBuilder(cmd, RunSnapshotListVolume, "list volume", "list volume", Writer,
		aliasOpt("lsv"), displayerType(&snapshot{}), docCategories("droplet"))

	CmdBuilder(cmd, RunSnapshotListDroplet, "list droplet", "list droplet", Writer,
		aliasOpt("lsd"), displayerType(&snapshot{}), docCategories("droplet"))

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
