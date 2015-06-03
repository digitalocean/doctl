package main

import (
	"github.com/bryanl/docli"
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
		Name:  "list",
		Usage: "list droplets",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "json",
				Usage: "return list of droplets as JSON array",
			},
		},
		Action: docli.DropletList,
	}
}

func dropletCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create droplet",
		Flags: []cli.Flag{

			cli.StringFlag{
				Name:  docli.ArgDropletName,
				Usage: "droplet name",
			},
			cli.StringFlag{
				Name:  docli.ArgRegionSlug,
				Usage: "droplet region",
			},
			cli.StringFlag{
				Name:  docli.ArgSizeSlug,
				Usage: "droplet size",
			},
			cli.StringFlag{
				Name:  docli.ArgImage,
				Usage: "droplet image",
			},
			cli.StringSliceFlag{
				Name:  docli.ArgSSHKeys,
				Value: &cli.StringSlice{},
				Usage: "droplet public SSH keys",
			},
			cli.BoolFlag{
				Name:  docli.ArgBackups,
				Usage: "enable droplet backups",
			},
			cli.BoolFlag{
				Name:  docli.ArgIPv6,
				Usage: "enable droplet IPv6",
			},
			cli.BoolFlag{
				Name:  docli.ArgPrivateNetworking,
				Usage: "enable droplet private networking",
			},
			cli.StringFlag{
				Name:  docli.ArgUserData,
				Usage: "droplet name",
			},
		},
		Action: docli.DropletCreate,
	}
}

func dropletDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  docli.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: docli.DropletDelete,
	}
}

func dropletGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  docli.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: docli.DropletGet,
	}
}

func dropletKernels() cli.Command {
	return cli.Command{
		Name:  "kernels",
		Usage: "get kernels for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  docli.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: docli.DropletKernels,
	}
}

func dropletSnapshots() cli.Command {
	return cli.Command{
		Name:  "snapshots",
		Usage: "get snapshots for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  docli.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: docli.DropletSnapshots,
	}
}

func dropletBackups() cli.Command {
	return cli.Command{
		Name:  "backups",
		Usage: "get backups for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  docli.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: docli.DropletBackups,
	}
}

func dropletActions() cli.Command {
	return cli.Command{
		Name:  "actions",
		Usage: "get actions for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  docli.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: docli.DropletActions,
	}
}

func dropletNeighbors() cli.Command {
	return cli.Command{
		Name:  "neighbors",
		Usage: "get neighbors for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  docli.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: docli.DropletNeighbors,
	}
}
