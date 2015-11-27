package commands

import (
	"io"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
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
	addIntFlag(cmdDropletActionGet, doit.ArgActionID, 0, "Action ID", requiredOpt())

	cmdDropletActionDisableBackups := cmdBuilder(RunDropletActionDisableBackups,
		"disable-backups <droplet-id>", "disable backups", writer)
	cmd.AddCommand(cmdDropletActionDisableBackups)

	cmdDropletActionReboot := cmdBuilder(RunDropletActionReboot,
		"reboot <droplet-id>", "reboot droplet", writer)
	cmd.AddCommand(cmdDropletActionReboot)

	cmdDropletActionPowerCycle := cmdBuilder(RunDropletActionPowerCycle,
		"power-cycle <droplet-id>", "power cycle droplet", writer)
	cmd.AddCommand(cmdDropletActionPowerCycle)

	cmdDropletActionShutdown := cmdBuilder(RunDropletActionShutdown,
		"shutdown <droplet-id>", "shutdown droplet", writer)
	cmd.AddCommand(cmdDropletActionShutdown)

	cmdDropletActionPowerOff := cmdBuilder(RunDropletActionPowerOff,
		"power-off <droplet-id>", "power off droplet", writer)
	cmd.AddCommand(cmdDropletActionPowerOff)

	cmdDropletActionPowerOn := cmdBuilder(RunDropletActionPowerOn,
		"power-on <droplet-id>", "power on droplet", writer)
	cmd.AddCommand(cmdDropletActionPowerOn)

	cmdDropletActionPasswordReset := cmdBuilder(RunDropletActionPasswordReset,
		"power-reset <droplet-id>", "power reset droplet", writer)
	cmd.AddCommand(cmdDropletActionPasswordReset)

	cmdDropletActionEnableIPv6 := cmdBuilder(RunDropletActionEnableIPv6,
		"enable-ipv6 <droplet-id>", "enable ipv6", writer)
	cmd.AddCommand(cmdDropletActionEnableIPv6)

	cmdDropletActionEnablePrivateNetworking := cmdBuilder(RunDropletActionEnablePrivateNetworking,
		"enable-private-networking <droplet-id>", "enable private networking", writer)
	cmd.AddCommand(cmdDropletActionEnablePrivateNetworking)

	cmdDropletActionUpgrade := cmdBuilder(RunDropletActionUpgrade,
		"upgrade <droplet-id>", "upgrade droplet", writer)
	cmd.AddCommand(cmdDropletActionUpgrade)

	cmdDropletActionRestore := cmdBuilder(RunDropletActionRestore,
		"restore <droplet-id>", "restore backup", writer)
	cmd.AddCommand(cmdDropletActionRestore)
	addIntFlag(cmdDropletActionRestore, doit.ArgImageID, 0, "Image ID", requiredOpt())

	cmdDropletActionResize := cmdBuilder(RunDropletActionResize,
		"resize <droplet-id>", "resize droplet", writer)
	cmd.AddCommand(cmdDropletActionResize)
	addIntFlag(cmdDropletActionResize, doit.ArgImageID, 0, "Image ID", requiredOpt())
	addBoolFlag(cmdDropletActionResize, doit.ArgResizeDisk, false, "Resize disk")

	cmdDropletActionRebuild := cmdBuilder(RunDropletActionRebuild,
		"rebuild <droplet-id>", "rebuild droplet", writer)
	cmd.AddCommand(cmdDropletActionRebuild)
	addIntFlag(cmdDropletActionRebuild, doit.ArgImageID, 0, "Image ID", requiredOpt())

	cmdDropletActionRename := cmdBuilder(RunDropletActionRename,
		"rename <droplet-id>", "rename droplet", writer)
	cmd.AddCommand(cmdDropletActionRename)
	addStringFlag(cmdDropletActionRename, doit.ArgDropletName, "", "Droplet name", requiredOpt())

	cmdDropletActionChangeKernel := cmdBuilder(RunDropletActionChangeKernel,
		"change-kernel <droplet-id>", "change kernel", writer)
	cmd.AddCommand(cmdDropletActionChangeKernel)
	addIntFlag(cmdDropletActionChangeKernel, doit.ArgKernelID, 0, "Kernel ID", requiredOpt())

	cmdDropletActionSnapshot := cmdBuilder(RunDropletActionSnapshot,
		"snapshot <droplet-id>", "snapshot droplet", writer)
	cmd.AddCommand(cmdDropletActionSnapshot)
	addIntFlag(cmdDropletActionSnapshot, doit.ArgSnapshotName, 0, "Snapshot name", requiredOpt())

	return cmd
}

// RunDropletActionGet returns a droplet action by id.
func RunDropletActionGet(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
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

		a, _, err := client.DropletActions.Get(dropletID, actionID)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionDisableBackups disables backups for a droplet.
func RunDropletActionDisableBackups(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.DisableBackups(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionReboot reboots a droplet.
func RunDropletActionReboot(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.Reboot(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionPowerCycle power cycles a droplet.
func RunDropletActionPowerCycle(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.PowerCycle(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionShutdown shuts a droplet down.
func RunDropletActionShutdown(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		a, _, err := client.DropletActions.Shutdown(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionPowerOff turns droplet power off.
func RunDropletActionPowerOff(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.PowerOff(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionPowerOn turns droplet power on.
func RunDropletActionPowerOn(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.PowerOn(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionPasswordReset resets the droplet root password.
func RunDropletActionPasswordReset(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.PasswordReset(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionEnableIPv6 enables IPv6 for a droplet.
func RunDropletActionEnableIPv6(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.EnableIPv6(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionEnablePrivateNetworking enables private networking for a droplet.
func RunDropletActionEnablePrivateNetworking(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.EnablePrivateNetworking(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionUpgrade upgrades a droplet.
func RunDropletActionUpgrade(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

		if err != nil {
			return nil, err
		}

		a, _, err := client.DropletActions.Upgrade(id)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionRestore restores a droplet using an image id.
func RunDropletActionRestore(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
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

		a, _, err := client.DropletActions.Restore(id, image)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionResize resizesx a droplet giving a size slug and
// optionally expands the disk.
func RunDropletActionResize(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		if len(args) != 1 {
			return nil, doit.NewMissingArgsErr(ns)
		}
		id, err := strconv.Atoi(args[0])

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
func RunDropletActionRebuild(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
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
func RunDropletActionRename(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
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

		a, _, err := client.DropletActions.Rename(id, name)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionChangeKernel changes the kernel for a droplet.
func RunDropletActionChangeKernel(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
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

		a, _, err := client.DropletActions.ChangeKernel(id, kernel)
		return a, err
	}

	return performAction(out, config, fn)
}

// RunDropletActionSnapshot creates a snapshot for a droplet.
func RunDropletActionSnapshot(ns string, config doit.Config, out io.Writer, args []string) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
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

		a, _, err := client.DropletActions.Snapshot(id, name)
		return a, err
	}

	return performAction(out, config, fn)
}
