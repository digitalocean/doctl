package main

import (
	"github.com/bryanl/docli"
	"github.com/codegangsta/cli"
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
	fn := docli.DropletActionDisableBackups
	return noArgDropletCommand("disable-backups", "disables backups for droplet", fn)
}

func dropletReboot() cli.Command {
	fn := docli.DropletActionReboot
	return noArgDropletCommand("reboot", "reboot droplet", fn)
}

func dropletPowerCycle() cli.Command {
	fn := docli.DropletActionPowerCycle
	return noArgDropletCommand("power-cycle", "power cyle droplet", fn)
}

func dropletShutdown() cli.Command {
	fn := docli.DropletActionShutdown
	return noArgDropletCommand("shutdown", "shutdown droplet", fn)
}

func dropletPowerOff() cli.Command {
	fn := docli.DropletActionPowerOff
	return noArgDropletCommand("power-off", "power off droplet", fn)
}

func dropletPowerOn() cli.Command {
	fn := docli.DropletActionPowerOn
	return noArgDropletCommand("power-on", "power on droplet", fn)
}

func dropletPasswordReset() cli.Command {
	fn := docli.DropletActionPasswordReset
	return noArgDropletCommand("password-reset", "reset password for droplet", fn)
}

func dropletEnableIPv6() cli.Command {
	fn := docli.DropletActionEnableIPv6
	return noArgDropletCommand("power-on", "enable ipv6 for droplet", fn)
}

func dropletEnablePrivateNetworking() cli.Command {
	fn := docli.DropletActionPasswordReset
	return noArgDropletCommand("private-networking", "enable private networking for droplet", fn)
}

func dropletUpgrade() cli.Command {
	fn := docli.DropletActionUpgrade
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
		Action: docli.DropletActionRestore,
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
		Action: docli.DropletActionResize,
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
		Action: docli.DropletActionRebuild,
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
		Action: docli.DropletActionRename,
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
		Action: docli.DropletActionChangeKernel,
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
		Action: docli.DropletActionSnapshot,
	}
}

type noArgDropletFn func(c *cli.Context)

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
		Action: fn,
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
		Action: docli.DropletActionGet,
	}
}
