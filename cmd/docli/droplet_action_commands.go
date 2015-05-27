package main

import (
	"fmt"

	"github.com/bryanl/docli/dropletactions"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func dropletActionCommands() cli.Command {
	return cli.Command{
		Name:  "droplet-action",
		Usage: "droplet action commands",
		Subcommands: []cli.Command{
			dropletChangeKernel(),
			dropletDisableBackups(),
			dropletEnableIPv6(),
			dropletEnablePrivateNetworking(),
			dropletPasswordReset(),
			dropletPowerCycle(),
			dropletPowerOff(),
			dropletPowerOn(),
			dropletReboot(),
			dropletRebuild(),
			dropletRename(),
			dropletRestore(),
			dropletShutdown(),
			dropletUpgrade(),
			dropletActionGet(),
		},
	}
}

func dropletDisableBackups() cli.Command {
	fn := dropletactions.DisableBackups
	return noArgDropletCommand("disable-backups", "disables backups for droplet", fn)
}

func dropletReboot() cli.Command {
	fn := dropletactions.Reboot
	return noArgDropletCommand("reboot", "reboot droplet", fn)
}

func dropletPowerCycle() cli.Command {
	fn := dropletactions.PowerCycle
	return noArgDropletCommand("power-cycle", "power cyle droplet", fn)
}

func dropletShutdown() cli.Command {
	fn := dropletactions.Shutdown
	return noArgDropletCommand("shutdown", "shutdown droplet", fn)
}

func dropletPowerOff() cli.Command {
	fn := dropletactions.PowerOff
	return noArgDropletCommand("power-off", "power off droplet", fn)
}

func dropletPowerOn() cli.Command {
	fn := dropletactions.PowerOn
	return noArgDropletCommand("power-on", "power on droplet", fn)
}

func dropletPasswordReset() cli.Command {
	fn := dropletactions.PasswordReset
	return noArgDropletCommand("password-reset", "reset password for droplet", fn)
}

func dropletEnableIPv6() cli.Command {
	fn := dropletactions.EnableIPv6
	return noArgDropletCommand("power-on", "enable ipv6 for droplet", fn)
}

func dropletEnablePrivateNetworking() cli.Command {
	fn := dropletactions.PasswordReset
	return noArgDropletCommand("private-networking", "enable private networking for droplet", fn)
}

func dropletUpgrade() cli.Command {
	fn := dropletactions.Upgrade
	return noArgDropletCommand("upgrade", "upgrade droplet", fn)
}

func dropletRestore() cli.Command {
	return cli.Command{
		Name:  "restore",
		Usage: "restore droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id (required)",
			},
			// TODO make this a string, so it can handle slugs
			cli.IntFlag{
				Name:  "image",
				Usage: "image slug or id (required)",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			if !c.IsSet("image") {
				return fmt.Errorf("invalid image")
			}

			return nil
		},
		Action: func(c *cli.Context) {

			client := newClient(c)

			id := c.Int("id")
			image := c.Int("image")

			a, err := dropletactions.Restore(client, id, image)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(a)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

func dropletResize() cli.Command {
	return cli.Command{
		Name:  "resize",
		Usage: "resize droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id (required)",
			},
			cli.StringFlag{
				Name:  "size",
				Usage: "size slug to resize to (required)",
			},
			cli.BoolFlag{
				Name:  "disk",
				Usage: "increase disk size",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			if !c.IsSet("size") {
				return fmt.Errorf("invalid size slug")
			}

			return nil
		},
		Action: func(c *cli.Context) {

			client := newClient(c)

			id := c.Int("id")
			size := c.String("size")
			disk := c.Bool("disk")

			a, err := dropletactions.Resize(client, id, size, disk)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(a)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

func dropletRebuild() cli.Command {
	return cli.Command{
		Name:  "rebuild",
		Usage: "rebuild droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id (required)",
			},
			cli.StringFlag{
				Name:  "image",
				Usage: "image slug or image id (required)",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			if !c.IsSet("image") {
				return fmt.Errorf("invalid image")
			}

			return nil
		},
		Action: func(c *cli.Context) {

			client := newClient(c)

			id := c.Int("id")
			image := c.String("image")

			a, err := dropletactions.Rebuild(client, id, image)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(a)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

func dropletRename() cli.Command {
	return cli.Command{
		Name:  "rename",
		Usage: "rename droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id (required)",
			},
			cli.StringFlag{
				Name:  "name",
				Usage: "new name for droplet (required)",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			if !c.IsSet("name") {
				return fmt.Errorf("invalid name")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			client := newClient(c)

			id := c.Int("id")
			name := c.String("name")

			a, err := dropletactions.Rename(client, id, name)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(a)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

func dropletChangeKernel() cli.Command {
	return cli.Command{
		Name:  "change-kernel",
		Usage: "change kernel for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id (required)",
			},
			cli.IntFlag{
				Name:  "kernel",
				Usage: "new kernel for droplet (required)",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			if !c.IsSet("kernel") {
				return fmt.Errorf("invalid kernel")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			client := newClient(c)

			id := c.Int("id")
			kernel := c.Int("kernel")

			a, err := dropletactions.ChangeKernel(client, id, kernel)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(a)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

func dropletSnapshot() cli.Command {
	return cli.Command{
		Name:  "snapshot",
		Usage: "snapshot droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id (required)",
			},
			cli.StringFlag{
				Name:  "name",
				Usage: "name for snapshot",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			client := newClient(c)

			id := c.Int("id")
			name := c.String("name")

			a, err := dropletactions.Snapshot(client, id, name)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(a)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

type noArgDropletFn func(client *godo.Client, id int) (*godo.Action, error)

func noArgDropletCommand(name, usage string, fn noArgDropletFn) cli.Command {
	return cli.Command{
		Name:  name,
		Usage: usage,
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id (required)",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			client := newClient(c)

			id := c.Int("id")

			a, err := fn(client, id)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(a)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

func dropletActionGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get droplet action",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
			cli.StringFlag{
				Name:  "action-id",
				Usage: "action id",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			if !c.IsSet("action-id") {
				return fmt.Errorf("invalid action id")
			}

			return nil
		},
		Action: func(c *cli.Context) {

			client := newClient(c)

			id := c.Int("id")
			actionID := c.Int("action-id")

			a, err := dropletactions.Get(client, id, actionID)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(a)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}
