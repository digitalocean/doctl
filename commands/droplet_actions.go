package commands

import (
	"strconv"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/spf13/cobra"
)

type actionFn func(das do.DropletActionsService) (*do.Action, error)

func performAction(c *CmdConfig, fn actionFn) error {
	das := c.DropletActions()

	a, err := fn(das)
	if err != nil {
		return err
	}

	item := &action{actions: do.Actions{*a}}
	return c.Display(item)
}

// DropletAction creates the droplet-action command.
func DropletAction() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "droplet-action",
		Aliases: []string{"da"},
		Short:   "droplet action commands",
		Long:    "droplet-action is used to access droplet action commands",
	}

	cmdDropletActionGet := CmdBuilder(cmd, RunDropletActionGet, "get", "get droplet action", Writer,
		aliasOpt("g"), displayerType(&action{}))
	AddIntFlag(cmdDropletActionGet, doit.ArgActionID, 0, "Action ID", requiredOpt())

	CmdBuilder(cmd, RunDropletActionDisableBackups,
		"disable-backups <droplet-id>", "disable backups", Writer, displayerType(&action{}))

	CmdBuilder(cmd, RunDropletActionReboot,
		"reboot <droplet-id>", "reboot droplet", Writer, displayerType(&action{}))

	CmdBuilder(cmd, RunDropletActionPowerCycle,
		"power-cycle <droplet-id>", "power cycle droplet", Writer, displayerType(&action{}))

	CmdBuilder(cmd, RunDropletActionShutdown,
		"shutdown <droplet-id>", "shutdown droplet", Writer, displayerType(&action{}))

	CmdBuilder(cmd, RunDropletActionPowerOff,
		"power-off <droplet-id>", "power off droplet", Writer, displayerType(&action{}))

	CmdBuilder(cmd, RunDropletActionPowerOn,
		"power-on <droplet-id>", "power on droplet", Writer, displayerType(&action{}))

	CmdBuilder(cmd, RunDropletActionPasswordReset,
		"power-reset <droplet-id>", "power reset droplet", Writer, displayerType(&action{}))

	CmdBuilder(cmd, RunDropletActionEnableIPv6,
		"enable-ipv6 <droplet-id>", "enable ipv6", Writer, displayerType(&action{}))

	CmdBuilder(cmd, RunDropletActionEnablePrivateNetworking,
		"enable-private-networking <droplet-id>", "enable private networking", Writer, displayerType(&action{}))

	CmdBuilder(cmd, RunDropletActionUpgrade,
		"upgrade <droplet-id>", "upgrade droplet", Writer, displayerType(&action{}))

	cmdDropletActionRestore := CmdBuilder(cmd, RunDropletActionRestore,
		"restore <droplet-id>", "restore backup", Writer, displayerType(&action{}))
	AddIntFlag(cmdDropletActionRestore, doit.ArgImageID, 0, "Image ID", requiredOpt())

	cmdDropletActionResize := CmdBuilder(cmd, RunDropletActionResize,
		"resize <droplet-id>", "resize droplet", Writer, displayerType(&action{}))
	AddBoolFlag(cmdDropletActionResize, doit.ArgResizeDisk, false, "Resize disk")
	AddStringFlag(cmdDropletActionResize, doit.ArgSizeSlug, "", "New size")

	cmdDropletActionRebuild := CmdBuilder(cmd, RunDropletActionRebuild,
		"rebuild <droplet-id>", "rebuild droplet", Writer, displayerType(&action{}))
	AddIntFlag(cmdDropletActionRebuild, doit.ArgImageID, 0, "Image ID", requiredOpt())

	cmdDropletActionRename := CmdBuilder(cmd, RunDropletActionRename,
		"rename <droplet-id>", "rename droplet", Writer, displayerType(&action{}))
	AddStringFlag(cmdDropletActionRename, doit.ArgDropletName, "", "Droplet name", requiredOpt())

	cmdDropletActionChangeKernel := CmdBuilder(cmd, RunDropletActionChangeKernel,
		"change-kernel <droplet-id>", "change kernel", Writer)
	AddIntFlag(cmdDropletActionChangeKernel, doit.ArgKernelID, 0, "Kernel ID", requiredOpt())

	cmdDropletActionSnapshot := CmdBuilder(cmd, RunDropletActionSnapshot,
		"snapshot <droplet-id>", "snapshot droplet", Writer, displayerType(&action{}))
	AddIntFlag(cmdDropletActionSnapshot, doit.ArgSnapshotName, 0, "Snapshot name", requiredOpt())

	return cmd
}

// RunDropletActionGet returns a droplet action by id.
func RunDropletActionGet(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		dropletID, err := strconv.Atoi(c.Args[0])
		if err != nil {
			return nil, err
		}

		actionID, err := c.Doit.GetInt(c.NS, doit.ArgActionID)
		if err != nil {
			return nil, err
		}

		a, err := das.Get(dropletID, actionID)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionDisableBackups disables backups for a droplet.
func RunDropletActionDisableBackups(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])
		if err != nil {
			return nil, err
		}

		a, err := das.DisableBackups(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionReboot reboots a droplet.
func RunDropletActionReboot(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])
		if err != nil {
			return nil, err
		}

		a, err := das.Reboot(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionPowerCycle power cycles a droplet.
func RunDropletActionPowerCycle(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PowerCycle(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionShutdown shuts a droplet down.
func RunDropletActionShutdown(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		a, err := das.Shutdown(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionPowerOff turns droplet power off.
func RunDropletActionPowerOff(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PowerOff(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionPowerOn turns droplet power on.
func RunDropletActionPowerOn(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PowerOn(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionPasswordReset resets the droplet root password.
func RunDropletActionPasswordReset(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PasswordReset(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionEnableIPv6 enables IPv6 for a droplet.
func RunDropletActionEnableIPv6(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.EnableIPv6(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionEnablePrivateNetworking enables private networking for a droplet.
func RunDropletActionEnablePrivateNetworking(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.EnablePrivateNetworking(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionUpgrade upgrades a droplet.
func RunDropletActionUpgrade(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.Upgrade(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionRestore restores a droplet using an image id.
func RunDropletActionRestore(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		image, err := c.Doit.GetInt(c.NS, doit.ArgImageID)
		if err != nil {
			return nil, err
		}

		a, err := das.Restore(id, image)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionResize resizesx a droplet giving a size slug and
// optionally expands the disk.
func RunDropletActionResize(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		size, err := c.Doit.GetString(c.NS, doit.ArgSizeSlug)
		if err != nil {
			return nil, err
		}

		disk, err := c.Doit.GetBool(c.NS, doit.ArgResizeDisk)
		if err != nil {
			return nil, err
		}

		a, err := das.Resize(id, size, disk)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionRebuild rebuilds a droplet using an image id or slug.
func RunDropletActionRebuild(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		image, err := c.Doit.GetString(c.NS, doit.ArgImage)
		if err != nil {
			return nil, err
		}

		var a *do.Action
		if i, aerr := strconv.Atoi(image); aerr == nil {
			a, err = das.RebuildByImageID(id, i)
		} else {
			a, err = das.RebuildByImageSlug(id, image)
		}
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionRename renames a droplet.
func RunDropletActionRename(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		name, err := c.Doit.GetString(c.NS, doit.ArgDropletName)
		if err != nil {
			return nil, err
		}

		a, err := das.Rename(id, name)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionChangeKernel changes the kernel for a droplet.
func RunDropletActionChangeKernel(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		kernel, err := c.Doit.GetInt(c.NS, doit.ArgKernelID)
		if err != nil {
			return nil, err
		}

		a, err := das.ChangeKernel(id, kernel)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionSnapshot creates a snapshot for a droplet.
func RunDropletActionSnapshot(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doit.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		name, err := c.Doit.GetString(c.NS, doit.ArgSnapshotName)
		if err != nil {
			return nil, err
		}

		a, err := das.Snapshot(id, name)
		return a, err
	}

	return performAction(c, fn)
}
