package main

import (
	"github.com/bryanl/doit"
	"github.com/codegangsta/cli"
)

func dropletCommands() cli.Command {
	return cli.Command{
		Name:  "droplet",
		Usage: "droplet commands",
		Subcommands: []cli.Command{
			dropletList(),
			dropletCreate(),
			dropletGet(),
			dropletKernels(),
			dropletSnapshots(),
			dropletBackups(),
			dropletActions(),
			dropletDelete(),
			dropletNeighbors(),
		},
	}
}

func dropletList() cli.Command {
	return cli.Command{
		Name:   "list",
		Usage:  "list droplets",
		Action: doit.DropletList,
	}
}

func dropletCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create droplet",
		Flags: []cli.Flag{

			cli.StringFlag{
				Name:  doit.ArgDropletName,
				Usage: "droplet name",
			},
			cli.StringFlag{
				Name:  doit.ArgRegionSlug,
				Usage: "droplet region",
			},
			cli.StringFlag{
				Name:  doit.ArgSizeSlug,
				Usage: "droplet size",
			},
			cli.StringFlag{
				Name:  doit.ArgImage,
				Usage: "droplet image",
			},
			cli.StringSliceFlag{
				Name:  doit.ArgSSHKeys,
				Value: &cli.StringSlice{},
				Usage: "droplet public SSH keys",
			},
			cli.BoolFlag{
				Name:  doit.ArgBackups,
				Usage: "enable droplet backups",
			},
			cli.BoolFlag{
				Name:  doit.ArgIPv6,
				Usage: "enable droplet IPv6",
			},
			cli.BoolFlag{
				Name:  doit.ArgPrivateNetworking,
				Usage: "enable droplet private networking",
			},
			cli.StringFlag{
				Name:  doit.ArgUserData,
				Usage: "droplet user data",
			},
			cli.StringFlag{
				Name:  doit.ArgUserDataFile,
				Usage: "reads droplet user data from a file",
			},
			cli.BoolFlag{
				Name:  doit.ArgDropletWait,
				Usage: "wait for droplet to become active",
			},
		},
		Action: doit.DropletCreate,
	}
}

func dropletDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: doit.DropletDelete,
	}
}

func dropletGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: doit.DropletGet,
	}
}

func dropletKernels() cli.Command {
	return cli.Command{
		Name:  "kernels",
		Usage: "get kernels for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: doit.DropletKernels,
	}
}

func dropletSnapshots() cli.Command {
	return cli.Command{
		Name:  "snapshots",
		Usage: "get snapshots for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: doit.DropletSnapshots,
	}
}

func dropletBackups() cli.Command {
	return cli.Command{
		Name:  "backups",
		Usage: "get backups for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: doit.DropletBackups,
	}
}

func dropletActions() cli.Command {
	return cli.Command{
		Name:  "actions",
		Usage: "get actions for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: doit.DropletActions,
	}
}

func dropletNeighbors() cli.Command {
	return cli.Command{
		Name:  "neighbors",
		Usage: "get neighbors for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: doit.DropletNeighbors,
	}
}
