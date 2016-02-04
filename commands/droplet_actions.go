package commands

import (
	"io"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/do"
	"github.com/spf13/cobra"
)

type actionFn func(das do.DropletActionsService) (*do.Action, error)

func performAction(out io.Writer, ns string, config doit.Config, fn actionFn) error {
	client := config.GetGodoClient()
	das := do.NewDropletActionsService(client)

	a, err := fn(das)
	if err != nil {
		return err
	}

	dc := &displayer{
		ns:     ns,
		config: config,
		item:   &action{actions: do.Actions{*a}},
		out:    out,
	}

	return dc.Display()
}

// DropletAction creates the droplet-action command.
func DropletAction() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "droplet-action",
		Aliases: []string{"da"},
		Short:   "droplet action commands",
		Long:    "droplet-action is used to access droplet action commands",
	}

	cmdDropletActionGet := cmdBuilder(cmd, RunDropletActionGet, "get", "get droplet action", writer,
		aliasOpt("g"), displayerType(&action{}))
	addIntFlag(cmdDropletActionGet, doit.ArgActionID, 0, "Action ID", requiredOpt())

	cmdBuilder(cmd, RunDropletActionDisableBackups,
		"disable-backups <droplet-id>", "disable backups", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunDropletActionReboot,
		"reboot <droplet-id>", "reboot droplet", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunDropletActionPowerCycle,
		"power-cycle <droplet-id>", "power cycle droplet", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunDropletActionShutdown,
		"shutdown <droplet-id>", "shutdown droplet", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunDropletActionPowerOff,
		"power-off <droplet-id>", "power off droplet", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunDropletActionPowerOn,
		"power-on <droplet-id>", "power on droplet", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunDropletActionPasswordReset,
		"power-reset <droplet-id>", "power reset droplet", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunDropletActionEnableIPv6,
		"enable-ipv6 <droplet-id>", "enable ipv6", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunDropletActionEnablePrivateNetworking,
		"enable-private-networking <droplet-id>", "enable private networking", writer, displayerType(&action{}))

	cmdBuilder(cmd, RunDropletActionUpgrade,
		"upgrade <droplet-id>", "upgrade droplet", writer, displayerType(&action{}))

	cmdDropletActionRestore := cmdBuilder(cmd, RunDropletActionRestore,
		"restore <droplet-id>", "restore backup", writer, displayerType(&action{}))
	addIntFlag(cmdDropletActionRestore, doit.ArgImageID, 0, "Image ID", requiredOpt())

	cmdDropletActionResize := cmdBuilder(cmd, RunDropletActionResize,
		"resize <droplet-id>", "resize droplet", writer, displayerType(&action{}))
	addBoolFlag(cmdDropletActionResize, doit.ArgResizeDisk, false, "Resize disk")
	addStringFlag(cmdDropletActionResize, doit.ArgSizeSlug, "", "New size")

	cmdDropletActionRebuild := cmdBuilder(cmd, RunDropletActionRebuild,
		"rebuild <droplet-id>", "rebuild droplet", writer, displayerType(&action{}))
	addIntFlag(cmdDropletActionRebuild, doit.ArgImageID, 0, "Image ID", requiredOpt())

	cmdDropletActionRename := cmdBuilder(cmd, RunDropletActionRename,
		"rename <droplet-id>", "rename droplet", writer, displayerType(&action{}))
	addStringFlag(cmdDropletActionRename, doit.ArgDropletName, "", "Droplet name", requiredOpt())

	cmdDropletActionChangeKernel := cmdBuilder(cmd, RunDropletActionChangeKernel,
		"change-kernel <droplet-id>", "change kernel", writer)
	addIntFlag(cmdDropletActionChangeKernel, doit.ArgKernelID, 0, "Kernel ID", requiredOpt())

	cmdDropletActionSnapshot := cmdBuilder(cmd, RunDropletActionSnapshot,
		"snapshot <droplet-id>", "snapshot droplet", writer, displayerType(&action{}))
	addIntFlag(cmdDropletActionSnapshot, doit.ArgSnapshotName, 0, "Snapshot name", requiredOpt())

	return cmd
}

// RunDropletActionGet returns a droplet action by id.
func RunDropletActionGet(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		dropletID, err := strconv.Atoi(args[0])
		if err != nil {
			return nil, err
		}

		actionID, err := config.GetInt(ns, doit.ArgActionID)
		if err != nil {
			return nil, err
		}

		a, err := das.Get(dropletID, actionID)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionDisableBackups disables backups for a droplet.
func RunDropletActionDisableBackups(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return nil, err
		}

		a, err := das.DisableBackups(id)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionReboot reboots a droplet.
func RunDropletActionReboot(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return nil, err
		}

		a, err := das.Reboot(id)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionPowerCycle power cycles a droplet.
func RunDropletActionPowerCycle(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PowerCycle(id)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionShutdown shuts a droplet down.
func RunDropletActionShutdown(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		a, err := das.Shutdown(id)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionPowerOff turns droplet power off.
func RunDropletActionPowerOff(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PowerOff(id)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionPowerOn turns droplet power on.
func RunDropletActionPowerOn(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PowerOn(id)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionPasswordReset resets the droplet root password.
func RunDropletActionPasswordReset(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PasswordReset(id)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionEnableIPv6 enables IPv6 for a droplet.
func RunDropletActionEnableIPv6(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.EnableIPv6(id)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionEnablePrivateNetworking enables private networking for a droplet.
func RunDropletActionEnablePrivateNetworking(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.EnablePrivateNetworking(id)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionUpgrade upgrades a droplet.
func RunDropletActionUpgrade(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.Upgrade(id)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionRestore restores a droplet using an image id.
func RunDropletActionRestore(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		image, err := config.GetInt(ns, doit.ArgImageID)
		if err != nil {
			return nil, err
		}

		a, err := das.Restore(id, image)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionResize resizesx a droplet giving a size slug and
// optionally expands the disk.
func RunDropletActionResize(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		size, err := config.GetString(ns, doit.ArgSizeSlug)
		if err != nil {
			return nil, err
		}

		disk, err := config.GetBool(ns, doit.ArgResizeDisk)
		if err != nil {
			return nil, err
		}

		a, err := das.Resize(id, size, disk)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionRebuild rebuilds a droplet using an image id or slug.
func RunDropletActionRebuild(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		image, err := config.GetString(ns, doit.ArgImage)
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

	return performAction(out, ns, config, fn)
}

// RunDropletActionRename renames a droplet.
func RunDropletActionRename(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		name, err := config.GetString(ns, doit.ArgDropletName)
		if err != nil {
			return nil, err
		}

		a, err := das.Rename(id, name)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionChangeKernel changes the kernel for a droplet.
func RunDropletActionChangeKernel(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		kernel, err := config.GetInt(ns, doit.ArgKernelID)
		if err != nil {
			return nil, err
		}

		a, err := das.ChangeKernel(id, kernel)
		return a, err
	}

	return performAction(out, ns, config, fn)
}

// RunDropletActionSnapshot creates a snapshot for a droplet.
func RunDropletActionSnapshot(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		name, err := config.GetString(ns, doit.ArgSnapshotName)
		if err != nil {
			return nil, err
		}

		a, err := das.Snapshot(id, name)
		return a, err
	}

	return performAction(out, ns, config, fn)
}
