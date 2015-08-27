package commands

import (
	"io"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

type actionFn func(client *godo.Client) (*godo.Action, error)

func performAction(out io.Writer, fn actionFn) error {
	client := doit.DoitConfig.GetGodoClient()

	a, err := fn(client)
	if err != nil {
		return err
	}

	return doit.DisplayOutput(a, out)
}

// DropletAction creates the droplet-action command.
func DropletAction() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "droplet-action",
		Short: "droplet action commands",
		Long:  "droplet-action is used to access droplet action commands",
	}

	cmdDropletActionGet := cmdBuilder(RunDropletActionGet, "get", "get droplet action", writer)
	cmd.AddCommand(cmdDropletActionGet)
	addIntFlag(cmdDropletActionGet, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionGet, doit.ArgActionID, 0, "Action ID")

	cmdDropletActionDisableBackups := cmdBuilder(RunDropletActionDisableBackups,
		"disable-backups", "disable backups", writer)
	cmd.AddCommand(cmdDropletActionDisableBackups)
	addIntFlag(cmdDropletActionDisableBackups, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionReboot := cmdBuilder(RunDropletActionReboot,
		"reboot", "reboot droplet", writer)
	cmd.AddCommand(cmdDropletActionReboot)
	addIntFlag(cmdDropletActionReboot, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionPowerCycle := cmdBuilder(RunDropletActionPowerCycle,
		"power-cycle", "power cycle droplet", writer)
	cmd.AddCommand(cmdDropletActionPowerCycle)
	addIntFlag(cmdDropletActionPowerCycle, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionShutdown := cmdBuilder(RunDropletActionShutdown,
		"shutdown", "shutdown droplet", writer)
	cmd.AddCommand(cmdDropletActionShutdown)
	addIntFlag(cmdDropletActionShutdown, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionPowerOff := cmdBuilder(RunDropletActionPowerOff,
		"power-off", "power off droplet", writer)
	cmd.AddCommand(cmdDropletActionPowerOff)
	addIntFlag(cmdDropletActionPowerOff, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionPowerOn := cmdBuilder(RunDropletActionPowerOn,
		"power-on", "power on droplet", writer)
	cmd.AddCommand(cmdDropletActionPowerOn)
	addIntFlag(cmdDropletActionPowerOn, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionPasswordReset := cmdBuilder(RunDropletActionPasswordReset,
		"power-reset", "power reset droplet", writer)
	cmd.AddCommand(cmdDropletActionPasswordReset)
	addIntFlag(cmdDropletActionPasswordReset, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionEnableIPv6 := cmdBuilder(RunDropletActionEnableIPv6,
		"enable-ipv6", "enable ipv6", writer)
	cmd.AddCommand(cmdDropletActionEnableIPv6)
	addIntFlag(cmdDropletActionEnableIPv6, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionEnablePrivateNetworking := cmdBuilder(RunDropletActionEnablePrivateNetworking,
		"enable-private-networking", "enable private networking", writer)
	cmd.AddCommand(cmdDropletActionEnablePrivateNetworking)
	addIntFlag(cmdDropletActionEnablePrivateNetworking, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionUpgrade := cmdBuilder(RunDropletActionUpgrade,
		"upgrade", "upgrade droplet", writer)
	cmd.AddCommand(cmdDropletActionUpgrade)
	addIntFlag(cmdDropletActionUpgrade, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionRestore := cmdBuilder(RunDropletActionRestore,
		"restore", "restore backup", writer)
	cmd.AddCommand(cmdDropletActionRestore)
	addIntFlag(cmdDropletActionRestore, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionRestore, doit.ArgImageID, 0, "Image ID")

	cmdDropletActionResize := cmdBuilder(RunDropletActionResize,
		"resize", "resize droplet", writer)
	cmd.AddCommand(cmdDropletActionResize)
	addIntFlag(cmdDropletActionResize, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionResize, doit.ArgImageID, 0, "Image ID")
	addBoolFlag(cmdDropletActionResize, doit.ArgResizeDisk, false, "Resize disk")

	cmdDropletActionRebuild := cmdBuilder(RunDropletActionRebuild,
		"rebuild", "rebuild droplet", writer)
	cmd.AddCommand(cmdDropletActionRebuild)
	addIntFlag(cmdDropletActionRebuild, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionRebuild, doit.ArgImageID, 0, "Image ID")

	cmdDropletActionRename := cmdBuilder(RunDropletActionRename,
		"rename", "rename droplet", writer)
	cmd.AddCommand(cmdDropletActionRename)
	addIntFlag(cmdDropletActionRename, doit.ArgDropletID, 0, "Droplet ID")
	addStringFlag(cmdDropletActionRename, doit.ArgDropletName, "", "Droplet name")

	cmdDropletActionChangeKernel := cmdBuilder(RunDropletActionChangeKernel,
		"change-kernel", "change kernel", writer)
	cmd.AddCommand(cmdDropletActionChangeKernel)
	addIntFlag(cmdDropletActionChangeKernel, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionChangeKernel, doit.ArgKernelID, 0, "Kernel ID")

	cmdDropletActionSnapshot := cmdBuilder(RunDropletActionSnapshot,
		"snapshot", "snapshot droplet", writer)
	cmd.AddCommand(cmdDropletActionSnapshot)
	addIntFlag(cmdDropletActionSnapshot, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionSnapshot, doit.ArgSnapshotName, 0, "Snapshot name")

	return cmd
}

// RunDropletActionGet returns a droplet action by id.
func RunDropletActionGet(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		dropletID := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)
		actionID := doit.DoitConfig.GetInt(ns, doit.ArgActionID)

		a, _, err := client.DropletActions.Get(dropletID, actionID)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionDisableBackups disables backups for a droplet.
func RunDropletActionDisableBackups(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.DisableBackups(id)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionReboot reboots a droplet.
func RunDropletActionReboot(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.Reboot(id)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionPowerCycle power cycles a droplet.
func RunDropletActionPowerCycle(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)
		a, _, err := client.DropletActions.PowerCycle(id)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionShutdown shuts a droplet down.
func RunDropletActionShutdown(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.Shutdown(id)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionPowerOff turns droplet power off.
func RunDropletActionPowerOff(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.PowerOff(id)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionPowerOn turns droplet power on.
func RunDropletActionPowerOn(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.PowerOn(id)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionPasswordReset resets the droplet root password.
func RunDropletActionPasswordReset(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.PasswordReset(id)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionEnableIPv6 enables IPv6 for a droplet.
func RunDropletActionEnableIPv6(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.EnableIPv6(id)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionEnablePrivateNetworking enables private networking for a droplet.
func RunDropletActionEnablePrivateNetworking(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.EnablePrivateNetworking(id)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionUpgrade upgrades a droplet.
func RunDropletActionUpgrade(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.Upgrade(id)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionRestore restores a droplet using an image id.
func RunDropletActionRestore(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)
		image := doit.DoitConfig.GetInt(ns, doit.ArgImageID)

		a, _, err := client.DropletActions.Restore(id, image)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionResize resizesx a droplet giving a size slug and
// optionally expands the disk.
func RunDropletActionResize(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)
		size := doit.DoitConfig.GetString(ns, doit.ArgImageSlug)
		disk := doit.DoitConfig.GetBool(ns, doit.ArgResizeDisk)

		a, _, err := client.DropletActions.Resize(id, size, disk)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionRebuild rebuilds a droplet using an image id or slug.
func RunDropletActionRebuild(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)
		image := doit.DoitConfig.GetString(ns, doit.ArgImage)

		var a *godo.Action
		var err error
		if i, aerr := strconv.Atoi(image); aerr == nil {
			a, _, err = client.DropletActions.RebuildByImageID(id, i)
		} else {
			a, _, err = client.DropletActions.RebuildByImageSlug(id, image)
		}
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionRename renames a droplet.
func RunDropletActionRename(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)
		name := doit.DoitConfig.GetString(ns, doit.ArgDropletName)

		a, _, err := client.DropletActions.Rename(id, name)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionChangeKernel changes the kernel for a droplet.
func RunDropletActionChangeKernel(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)
		kernel := doit.DoitConfig.GetInt(ns, doit.ArgKernelID)

		a, _, err := client.DropletActions.ChangeKernel(id, kernel)
		return a, err
	}

	return performAction(out, fn)
}

// RunDropletActionSnapshot creates a snapshot for a droplet.
func RunDropletActionSnapshot(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.DoitConfig.GetInt(ns, doit.ArgDropletID)
		name := doit.DoitConfig.GetString(ns, doit.ArgSnapshotName)

		a, _, err := client.DropletActions.Snapshot(id, name)
		return a, err
	}

	return performAction(out, fn)
}
