package commands

import (
	"io"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/digitalocean/godo"
	"github.com/bryanl/doit/Godeps/_workspace/src/github.com/spf13/cobra"
)

type actionFn func(client *godo.Client) (*godo.Action, error)

func performAction(out io.Writer, config doit.Config, fn actionFn) error {
	client := config.GetGodoClient()

	a, err := fn(client)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(a, out)
}

// DropletAction creates the droplet-action command.
func DropletAction() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "droplet-action",
		Aliases: []string{"da"},
		Short:   "droplet action commands",
		Long:    "droplet-action is used to access droplet action commands",
	}

	cmdDropletActionGet := cmdBuilder(RunDropletActionGet, "get", "get droplet action", writer, aliasOpt("g"))
	cmd.AddCommand(cmdDropletActionGet)
	addIntFlag(cmdDropletActionGet, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())
	addIntFlag(cmdDropletActionGet, doit.ArgActionID, 0, "Action ID", requiredOpt())

	cmdDropletActionDisableBackups := cmdBuilder(RunDropletActionDisableBackups,
		"disable-backups", "disable backups", writer)
	cmd.AddCommand(cmdDropletActionDisableBackups)
	addIntFlag(cmdDropletActionDisableBackups, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletActionReboot := cmdBuilder(RunDropletActionReboot,
		"reboot", "reboot droplet", writer)
	cmd.AddCommand(cmdDropletActionReboot)
	addIntFlag(cmdDropletActionReboot, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletActionPowerCycle := cmdBuilder(RunDropletActionPowerCycle,
		"power-cycle", "power cycle droplet", writer)
	cmd.AddCommand(cmdDropletActionPowerCycle)
	addIntFlag(cmdDropletActionPowerCycle, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletActionShutdown := cmdBuilder(RunDropletActionShutdown,
		"shutdown", "shutdown droplet", writer)
	cmd.AddCommand(cmdDropletActionShutdown)
	addIntFlag(cmdDropletActionShutdown, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletActionPowerOff := cmdBuilder(RunDropletActionPowerOff,
		"power-off", "power off droplet", writer)
	cmd.AddCommand(cmdDropletActionPowerOff)
	addIntFlag(cmdDropletActionPowerOff, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletActionPowerOn := cmdBuilder(RunDropletActionPowerOn,
		"power-on", "power on droplet", writer)
	cmd.AddCommand(cmdDropletActionPowerOn)
	addIntFlag(cmdDropletActionPowerOn, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletActionPasswordReset := cmdBuilder(RunDropletActionPasswordReset,
		"power-reset", "power reset droplet", writer)
	cmd.AddCommand(cmdDropletActionPasswordReset)
	addIntFlag(cmdDropletActionPasswordReset, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletActionEnableIPv6 := cmdBuilder(RunDropletActionEnableIPv6,
		"enable-ipv6", "enable ipv6", writer)
	cmd.AddCommand(cmdDropletActionEnableIPv6)
	addIntFlag(cmdDropletActionEnableIPv6, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletActionEnablePrivateNetworking := cmdBuilder(RunDropletActionEnablePrivateNetworking,
		"enable-private-networking", "enable private networking", writer)
	cmd.AddCommand(cmdDropletActionEnablePrivateNetworking)
	addIntFlag(cmdDropletActionEnablePrivateNetworking, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletActionUpgrade := cmdBuilder(RunDropletActionUpgrade,
		"upgrade", "upgrade droplet", writer)
	cmd.AddCommand(cmdDropletActionUpgrade)
	addIntFlag(cmdDropletActionUpgrade, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())

	cmdDropletActionRestore := cmdBuilder(RunDropletActionRestore,
		"restore", "restore backup", writer)
	cmd.AddCommand(cmdDropletActionRestore)
	addIntFlag(cmdDropletActionRestore, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())
	addIntFlag(cmdDropletActionRestore, doit.ArgImageID, 0, "Image ID", requiredOpt())

	cmdDropletActionResize := cmdBuilder(RunDropletActionResize,
		"resize", "resize droplet", writer)
	cmd.AddCommand(cmdDropletActionResize)
	addIntFlag(cmdDropletActionResize, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())
	addIntFlag(cmdDropletActionResize, doit.ArgImageID, 0, "Image ID", requiredOpt())
	addBoolFlag(cmdDropletActionResize, doit.ArgResizeDisk, false, "Resize disk")

	cmdDropletActionRebuild := cmdBuilder(RunDropletActionRebuild,
		"rebuild", "rebuild droplet", writer)
	cmd.AddCommand(cmdDropletActionRebuild)
	addIntFlag(cmdDropletActionRebuild, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())
	addIntFlag(cmdDropletActionRebuild, doit.ArgImageID, 0, "Image ID", requiredOpt())

	cmdDropletActionRename := cmdBuilder(RunDropletActionRename,
		"rename", "rename droplet", writer)
	cmd.AddCommand(cmdDropletActionRename)
	addIntFlag(cmdDropletActionRename, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())
	addStringFlag(cmdDropletActionRename, doit.ArgDropletName, "", "Droplet name", requiredOpt())

	cmdDropletActionChangeKernel := cmdBuilder(RunDropletActionChangeKernel,
		"change-kernel", "change kernel", writer)
	cmd.AddCommand(cmdDropletActionChangeKernel)
	addIntFlag(cmdDropletActionChangeKernel, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())
	addIntFlag(cmdDropletActionChangeKernel, doit.ArgKernelID, 0, "Kernel ID", requiredOpt())

	cmdDropletActionSnapshot := cmdBuilder(RunDropletActionSnapshot,
		"snapshot", "snapshot droplet", writer)
	cmd.AddCommand(cmdDropletActionSnapshot)
	addIntFlag(cmdDropletActionSnapshot, doit.ArgDropletID, 0, "Droplet ID", requiredOpt())
	addIntFlag(cmdDropletActionSnapshot, doit.ArgSnapshotName, 0, "Snapshot name", requiredOpt())

	return cmd
}

// RunDropletActionGet returns a droplet action by id.
func RunDropletActionGet(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		dropletID, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		actionID, err := config.GetInt(ns, doit.ArgActionID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.Get(dropletID, actionID)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionDisableBackups disables backups for a droplet.
func RunDropletActionDisableBackups(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.DisableBackups(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionReboot reboots a droplet.
func RunDropletActionReboot(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.Reboot(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionPowerCycle power cycles a droplet.
func RunDropletActionPowerCycle(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.PowerCycle(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionShutdown shuts a droplet down.
func RunDropletActionShutdown(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.Shutdown(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionPowerOff turns droplet power off.
func RunDropletActionPowerOff(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.PowerOff(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionPowerOn turns droplet power on.
func RunDropletActionPowerOn(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.PowerOn(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionPasswordReset resets the droplet root password.
func RunDropletActionPasswordReset(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.PasswordReset(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionEnableIPv6 enables IPv6 for a droplet.
func RunDropletActionEnableIPv6(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.EnableIPv6(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionEnablePrivateNetworking enables private networking for a droplet.
func RunDropletActionEnablePrivateNetworking(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.EnablePrivateNetworking(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionUpgrade upgrades a droplet.
func RunDropletActionUpgrade(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.Upgrade(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionRestore restores a droplet using an image id.
func RunDropletActionRestore(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		image, err := config.GetInt(ns, doit.ArgImageID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.Restore(id, image)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionResize resizesx a droplet giving a size slug and
// optionally expands the disk.
func RunDropletActionResize(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		size, err := config.GetString(ns, doit.ArgImageSlug)
		if err != nil {
			return nil, err
		}

		disk, err := config.GetBool(ns, doit.ArgResizeDisk)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.Resize(id, size, disk)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionRebuild rebuilds a droplet using an image id or slug.
func RunDropletActionRebuild(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		image, err := config.GetString(ns, doit.ArgImage)
		if err != nil {
			return nil, err
		}

		var a *godo.Action
		if i, aerr := strconv.Atoi(image); aerr == nil {
			a, _, err = client.DropletActions.RebuildByImageID(id, i)
		} else {
			a, _, err = client.DropletActions.RebuildByImageSlug(id, image)
		}
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionRename renames a droplet.
func RunDropletActionRename(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		name, err := config.GetString(ns, doit.ArgDropletName)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.Rename(id, name)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionChangeKernel changes the kernel for a droplet.
func RunDropletActionChangeKernel(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		kernel, err := config.GetInt(ns, doit.ArgKernelID)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.ChangeKernel(id, kernel)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionSnapshot creates a snapshot for a droplet.
func RunDropletActionSnapshot(ns string, config doit.Config, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id, err := config.GetInt(ns, doit.ArgDropletID)
		if err != nil {
			return nil, err
		}

		name, err := config.GetString(ns, doit.ArgSnapshotName)
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.Snapshot(id, name)
		return a, err
	}

	return performAction(out, config, fn)
}
