package doit

import (
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

type actionFn func(client *godo.Client, c *cli.Context) (*godo.Action, error)

func performAction(c *cli.Context, fn actionFn) {
	client := NewClient(c, DefaultConfig)

	a, err := fn(client, c)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not disable backups for droplet")
	}

	err = DisplayOutput(a, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}

}

// Get returns a droplet action by id.
func DropletActionGet(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		dropletID := c.Int(ArgDropletID)
		actionID := c.Int(ArgActionID)

		a, _, err := client.DropletActions.Get(dropletID, actionID)
		return a, err
	}

	performAction(c, fn)
}

// DisableBackups disables backups for a droplet.
func DropletActionDisableBackups(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)

		a, _, err := client.DropletActions.DisableBackups(id)
		return a, err
	}

	performAction(c, fn)
}

// Reboot reboots a droplet.
func DropletActionReboot(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)

		a, _, err := client.DropletActions.Reboot(id)
		return a, err
	}

	performAction(c, fn)
}

// PowerCycle power cycles a droplet.
func DropletActionPowerCycle(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)
		a, _, err := client.DropletActions.PowerCycle(id)
		return a, err
	}

	performAction(c, fn)
}

// Shutdown shuts a droplet down.
func DropletActionShutdown(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)

		a, _, err := client.DropletActions.Shutdown(id)
		return a, err
	}

	performAction(c, fn)
}

// PowerOff turns droplet power off.
func DropletActionPowerOff(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)

		a, _, err := client.DropletActions.PowerOff(id)
		return a, err
	}

	performAction(c, fn)
}

// PowerOn turns droplet power on.
func DropletActionPowerOn(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)

		a, _, err := client.DropletActions.PowerOn(id)
		return a, err
	}

	performAction(c, fn)
}

// PasswordReset resets the droplet root password.
func DropletActionPasswordReset(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)

		a, _, err := client.DropletActions.PasswordReset(id)
		return a, err
	}

	performAction(c, fn)
}

// EnableIPv6 enables IPv6 for a droplet.
func DropletActionEnableIPv6(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)

		a, _, err := client.DropletActions.EnableIPv6(id)
		return a, err
	}

	performAction(c, fn)
}

// EnablePrivateNetworking enables private networking for a droplet.
func DropletActionEnablePrivateNetworking(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)

		a, _, err := client.DropletActions.EnablePrivateNetworking(id)
		return a, err
	}

	performAction(c, fn)
}

// Upgrade upgrades a droplet.
func DropletActionUpgrade(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)

		a, _, err := client.DropletActions.Upgrade(id)
		return a, err
	}

	performAction(c, fn)
}

// Restore restores a droplet using an image id.
func DropletActionRestore(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)
		image := c.Int(ArgImageID)

		a, _, err := client.DropletActions.Restore(id, image)
		return a, err
	}

	performAction(c, fn)
}

// Resize resizesx a droplet giving a size slug and optionally expands the disk.
func DropletActionResize(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)
		size := c.String(ArgImageSlug)
		disk := c.Bool(ArgResizeDisk)

		a, _, err := client.DropletActions.Resize(id, size, disk)
		return a, err
	}

	performAction(c, fn)
}

// Rebuild rebuilds a droplet using an image id or slug.
func DropletActionRebuild(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)
		image := c.String(ArgImage)

		var a *godo.Action
		var err error
		if i, aerr := strconv.Atoi(image); aerr == nil {
			a, _, err = client.DropletActions.RebuildByImageID(id, i)
		} else {
			a, _, err = client.DropletActions.RebuildByImageSlug(id, image)
		}
		return a, err
	}

	performAction(c, fn)
}

// Rename renames a droplet.
func DropletActionRename(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)
		name := c.String(ArgDropletName)

		a, _, err := client.DropletActions.Rename(id, name)
		return a, err
	}

	performAction(c, fn)
}

// ChangeKernel changes the kernel for a droplet.
func DropletActionChangeKernel(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)
		kernel := c.Int(ArgKernelID)

		a, _, err := client.DropletActions.ChangeKernel(id, kernel)
		return a, err
	}

	performAction(c, fn)
}

// Snapshot creates a snapshot for a droplet.
func DropletActionSnapshot(c *cli.Context) {
	fn := func(client *godo.Client, c *cli.Context) (*godo.Action, error) {
		id := c.Int(ArgDropletID)
		name := c.String(ArgSnapshotName)

		a, _, err := client.DropletActions.Snapshot(id, name)
		return a, err
	}

	performAction(c, fn)
}
