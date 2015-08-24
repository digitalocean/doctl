package commands

import (
	"io"
	"os"
	"strconv"

	"github.com/bryanl/doit"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

type actionFn func(client *godo.Client) (*godo.Action, error)

func performAction(out io.Writer, fn actionFn) error {
	client := doit.VConfig.GetGodoClient()

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
		Long:  "droplet-action us used to access droplet action commands",
	}

	cmdDropletActionGet := NewCmdDropletActionGet(os.Stdout)
	cmd.AddCommand(cmdDropletActionGet)
	addIntFlag(cmdDropletActionGet, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionGet, doit.ArgActionID, 0, "Action ID")

	cmdDropletActionDisableBackups := NewCmdDropletActionDisableBackups(os.Stdout)
	cmd.AddCommand(cmdDropletActionDisableBackups)
	addIntFlag(cmdDropletActionDisableBackups, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionReboot := NewCmdDropletActionReboot(os.Stdout)
	cmd.AddCommand(cmdDropletActionReboot)
	addIntFlag(cmdDropletActionReboot, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionPowerCycle := NewCmdDropletActionPowerCycle(os.Stdout)
	cmd.AddCommand(cmdDropletActionPowerCycle)
	addIntFlag(cmdDropletActionPowerCycle, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionShutdown := NewCmdDropletActionShutdown(os.Stdout)
	cmd.AddCommand(cmdDropletActionShutdown)
	addIntFlag(cmdDropletActionShutdown, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionPowerOff := NewCmdDropletActionPowerOff(os.Stdout)
	cmd.AddCommand(cmdDropletActionPowerOff)
	addIntFlag(cmdDropletActionPowerOff, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionPowerOn := NewCmdDropletActionPowerOn(os.Stdout)
	cmd.AddCommand(cmdDropletActionPowerOn)
	addIntFlag(cmdDropletActionPowerOn, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionPasswordReset := NewCmdDropletActionPasswordReset(os.Stdout)
	cmd.AddCommand(cmdDropletActionPasswordReset)
	addIntFlag(cmdDropletActionPasswordReset, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionEnableIPv6 := NewCmdDropletActionEnableIPv6(os.Stdout)
	cmd.AddCommand(cmdDropletActionEnableIPv6)
	addIntFlag(cmdDropletActionEnableIPv6, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionEnablePrivateNetworking := NewCmdDropletActionEnablePrivateNetworking(os.Stdout)
	cmd.AddCommand(cmdDropletActionEnablePrivateNetworking)
	addIntFlag(cmdDropletActionEnablePrivateNetworking, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionUpgrade := NewCmdDropletActionUpgrade(os.Stdout)
	cmd.AddCommand(cmdDropletActionUpgrade)
	addIntFlag(cmdDropletActionUpgrade, doit.ArgDropletID, 0, "Droplet ID")

	cmdDropletActionRestore := NewCmdDropletActionRestore(os.Stdout)
	cmd.AddCommand(cmdDropletActionRestore)
	addIntFlag(cmdDropletActionRestore, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionRestore, doit.ArgImageID, 0, "Image ID")

	cmdDropletActionResize := NewCmdDropletActionResize(os.Stdout)
	cmd.AddCommand(cmdDropletActionResize)
	addIntFlag(cmdDropletActionResize, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionResize, doit.ArgImageID, 0, "Image ID")
	addBoolFlag(cmdDropletActionResize, doit.ArgResizeDisk, false, "Resize disk")

	cmdDropletActionRebuild := NewCmdDropletActionRebuild(os.Stdout)
	cmd.AddCommand(cmdDropletActionRebuild)
	addIntFlag(cmdDropletActionRebuild, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionRebuild, doit.ArgImageID, 0, "Image ID")

	cmdDropletActionRename := NewCmdDropletActionRename(os.Stdout)
	cmd.AddCommand(cmdDropletActionRename)
	addIntFlag(cmdDropletActionRename, doit.ArgDropletID, 0, "Droplet ID")
	addStringFlag(cmdDropletActionRename, doit.ArgDropletName, "", "Droplet name")

	cmdDropletActionChangeKernel := NewCmdDropletActionChangeKernel(os.Stdout)
	cmd.AddCommand(cmdDropletActionChangeKernel)
	addIntFlag(cmdDropletActionChangeKernel, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionChangeKernel, doit.ArgKernelID, 0, "Kernel ID")

	cmdDropletActionSnapshot := NewCmdDropletActionSnapshot(os.Stdout)
	cmd.AddCommand(cmdDropletActionSnapshot)
	addIntFlag(cmdDropletActionSnapshot, doit.ArgDropletID, 0, "Droplet ID")
	addIntFlag(cmdDropletActionSnapshot, doit.ArgSnapshotName, 0, "Snapshot name")

	return cmd
}

// NewCmdDropletActionGet creates a droplet action get command.
func NewCmdDropletActionGet(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "get droplet action",
		Long:  "get droplet action",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionGet(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionGet returns a droplet action by id.
func RunDropletActionGet(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		dropletID := doit.VConfig.GetInt(ns, doit.ArgDropletID)
		actionID := doit.VConfig.GetInt(ns, doit.ArgActionID)

		a, _, err := client.DropletActions.Get(dropletID, actionID)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionDisableBackups creates a droplet action disable backups
// command.
func NewCmdDropletActionDisableBackups(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "disable-backups",
		Short: "disable backups",
		Long:  "disable backups for droplet",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionDisableBackups(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionDisableBackups disables backups for a droplet.
func RunDropletActionDisableBackups(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.DisableBackups(id)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionReboot creates a droplet action reboot command.
func NewCmdDropletActionReboot(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "reboot",
		Short: "reboot droplet",
		Long:  "reboot droplet",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionReboot(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionReboot reboots a droplet.
func RunDropletActionReboot(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.Reboot(id)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionPowerCycle creates a droplet action power cycle command.
func NewCmdDropletActionPowerCycle(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "power-cycle",
		Short: "power cycle",
		Long:  "power cycle",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionGet(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionPowerCycle power cycles a droplet.
func RunDropletActionPowerCycle(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)
		a, _, err := client.DropletActions.PowerCycle(id)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionShutdown creates a droplet action shutdown command.
func NewCmdDropletActionShutdown(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "shutdown",
		Short: "shutdown",
		Long:  "shutdown",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionShutdown(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionShutdown shuts a droplet down.
func RunDropletActionShutdown(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.Shutdown(id)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionPowerOff creates a droplet action power off command.
func NewCmdDropletActionPowerOff(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "power-off",
		Short: "power off",
		Long:  "power off",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionPowerOff(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionPowerOff turns droplet power off.
func RunDropletActionPowerOff(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.PowerOff(id)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionPowerOn creates a droplet action power on command.
func NewCmdDropletActionPowerOn(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "power-on",
		Short: "power on",
		Long:  "power on",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionPowerOn(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionPowerOn turns droplet power on.
func RunDropletActionPowerOn(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.PowerOn(id)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionPasswordReset creates a droplet action password reset
// command.
func NewCmdDropletActionPasswordReset(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "password-reset",
		Short: "password reset",
		Long:  "password reset",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionPasswordReset(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionPasswordReset resets the droplet root password.
func RunDropletActionPasswordReset(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.PasswordReset(id)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionEnableIPv6 creates a droplet action enable ipv6 command.
func NewCmdDropletActionEnableIPv6(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "enable-ipv6",
		Short: "enable ipv6",
		Long:  "enable ipv6",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionEnableIPv6(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionEnableIPv6 enables IPv6 for a droplet.
func RunDropletActionEnableIPv6(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.EnableIPv6(id)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionEnablePrivateNetworking creates a droplet action enable
// private netowrking command.
func NewCmdDropletActionEnablePrivateNetworking(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "enable-private-networking",
		Short: "enable private networking",
		Long:  "enable private networking",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionEnablePrivateNetworking(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionEnablePrivateNetworking enables private networking for a droplet.
func RunDropletActionEnablePrivateNetworking(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.EnablePrivateNetworking(id)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionUpgrade creates a droplet action upgrade command.
func NewCmdDropletActionUpgrade(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "upgrade droplet",
		Long:  "upgrade droplet",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionUpgrade(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionUpgrade upgrades a droplet.
func RunDropletActionUpgrade(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)

		a, _, err := client.DropletActions.Upgrade(id)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionRestore creates a droplet action restore command.
func NewCmdDropletActionRestore(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "restore",
		Short: "restore droplet",
		Long:  "restore droplet",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionRestore(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionRestore restores a droplet using an image id.
func RunDropletActionRestore(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)
		image := doit.VConfig.GetInt(ns, doit.ArgImageID)

		a, _, err := client.DropletActions.Restore(id, image)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionResize creates a droplet action resize command.
func NewCmdDropletActionResize(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "resize",
		Short: "resize droplet",
		Long:  "resize droplet",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionResize(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionResize resizesx a droplet giving a size slug and
// optionally expands the disk.
func RunDropletActionResize(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)
		size := doit.VConfig.GetString(ns, doit.ArgImageSlug)
		disk := doit.VConfig.GetBool(ns, doit.ArgResizeDisk)

		a, _, err := client.DropletActions.Resize(id, size, disk)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionRebuild creates a droplet action rebuild command.
func NewCmdDropletActionRebuild(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "rebuild",
		Short: "rebuild",
		Long:  "rebuild",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionRebuild(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionRebuild rebuilds a droplet using an image id or slug.
func RunDropletActionRebuild(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)
		image := doit.VConfig.GetString(ns, doit.ArgImage)

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

// NewCmdDropletActionRename creates a droplet action rename command.
func NewCmdDropletActionRename(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "rename",
		Short: "rename",
		Long:  "rename",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionRename(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionRename renames a droplet.
func RunDropletActionRename(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)
		name := doit.VConfig.GetString(ns, doit.ArgDropletName)

		a, _, err := client.DropletActions.Rename(id, name)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionChangeKernel creates a droplet action change kernel command.
func NewCmdDropletActionChangeKernel(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "change-kernel",
		Short: "change kernel",
		Long:  "change kernel",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionChangeKernel(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionChangeKernel changes the kernel for a droplet.
func RunDropletActionChangeKernel(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)
		kernel := doit.VConfig.GetInt(ns, doit.ArgKernelID)

		a, _, err := client.DropletActions.ChangeKernel(id, kernel)
		return a, err
	}

	return performAction(out, fn)
}

// NewCmdDropletActionSnapshot creates a droplet action snapshot command.
func NewCmdDropletActionSnapshot(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "snapshot",
		Short: "snapshot",
		Long:  "perform Droplet snapshot",
		Run: func(cmd *cobra.Command, args []string) {
			checkErr(RunDropletActionSnapshot(cmdNS(cmd), out), cmd)
		},
	}
}

// RunDropletActionSnapshot creates a snapshot for a droplet.
func RunDropletActionSnapshot(ns string, out io.Writer) error {
	fn := func(client *godo.Client) (*godo.Action, error) {
		id := doit.VConfig.GetInt(ns, doit.ArgDropletID)
		name := doit.VConfig.GetString(ns, doit.ArgSnapshotName)

		a, _, err := client.DropletActions.Snapshot(id, name)
		return a, err
	}

	return performAction(out, fn)
}
