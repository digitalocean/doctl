package docli

import (
	"fmt"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

// Get returns a droplet action by id.
func DropletActionGet(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	dropletID := c.Int(argDropletID)
	actionID := c.Int(argActionID)

	a, _, err := client.DropletActions.Get(dropletID, actionID)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not disable backups for droplet")
	}

	err = WriteJSON(a, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// DisableBackups disables backups for a droplet.
func DropletActionDisableBackups(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)

	r, _, err := client.DropletActions.DisableBackups(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not disable backups for droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Reboot reboots a droplet.
func DropletActionReboot(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)

	r, _, err := client.DropletActions.Reboot(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not reboot droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// PowerCycle power cycles a droplet.
func DropletActionPowerCycle(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)
	r, _, err := client.DropletActions.PowerCycle(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not power cycle droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Shutdown shuts a droplet down.
func DropletActionShutdown(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)

	r, _, err := client.DropletActions.Shutdown(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not shutdown droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// PowerOff turns droplet power off.
func DropletActionPowerOff(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)

	r, _, err := client.DropletActions.PowerOff(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not power off droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// PowerOn turns droplet power on.
func DropletActionPowerOn(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)

	r, _, err := client.DropletActions.PowerOn(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not power on droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// PasswordReset resets the droplet root password.
func DropletActionPasswordReset(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)

	r, _, err := client.DropletActions.PasswordReset(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not reset password for droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// EnableIPv6 enables IPv6 for a droplet.
func DropletActionEnableIPv6(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)

	r, _, err := client.DropletActions.EnableIPv6(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not enable IPv6 for droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// EnablePrivateNetworking enables private networking for a droplet.
func DropletActionEnablePrivateNetworking(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)

	r, _, err := client.DropletActions.EnablePrivateNetworking(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not enable private networking for droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Upgrade upgrades a droplet.
func DropletActionUpgrade(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)

	r, _, err := client.DropletActions.Upgrade(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not upgrade droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Restore restores a droplet using an image id.
func DropletActionRestore(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)
	image := c.Int(argImageID)

	r, _, err := client.DropletActions.Restore(id, image)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not restore droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Resize resizesx a droplet giving a size slug and optionally expands the disk.
func DropletActionResize(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)
	size := c.String(argImageSlug)
	disk := c.Bool(argResizeDisk)

	r, _, err := client.DropletActions.Resize(id, size, disk)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not resize droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Rebuild rebuilds a droplet using an image id or slug.
func DropletActionRebuild(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)
	image := c.String(argImage)

	var r *godo.Action
	var err error
	if i, aerr := strconv.Atoi(image); aerr == nil {
		fmt.Println("rebuilding by id")
		r, _, err = client.DropletActions.RebuildByImageID(id, i)
	} else {
		r, _, err = client.DropletActions.RebuildByImageSlug(id, image)
	}
	if err != nil {
		logrus.WithField("err", err).Fatal("could not rebuild droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Rename renames a droplet.
func DropletActionRename(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)
	name := c.String(argDropletName)

	r, _, err := client.DropletActions.Rename(id, name)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not rename droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// ChangeKernel changes the kernel for a droplet.
func DropletActionChangeKernel(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)
	kernel := c.Int(argKernelID)

	r, _, err := client.DropletActions.ChangeKernel(id, kernel)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not change droplet kernel")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}

// Snapshot creates a snapshot for a droplet.
func DropletActionSnapshot(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int(argDropletID)
	name := c.String(argSnapshotName)

	r, _, err := client.DropletActions.Snapshot(id, name)
	if err != nil {
		logrus.WithField("err", err).Fatal("could create snapshot for droplet")
	}

	err = WriteJSON(r, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}
}
