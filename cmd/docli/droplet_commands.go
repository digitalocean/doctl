package main

import (
	"fmt"

	"github.com/bryanl/docli/droplets"
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
		Action: droplets.List,
	}
}

func dropletCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create droplet",
		Flags: []cli.Flag{

			cli.StringFlag{
				Name:  "name",
				Usage: "droplet name",
			},
			cli.StringFlag{
				Name:  "region",
				Usage: "droplet region",
			},
			cli.StringFlag{
				Name:  "size",
				Usage: "droplet size",
			},
			cli.StringFlag{
				Name:  "image",
				Usage: "droplet image",
			},
			cli.StringSliceFlag{
				Name:  "ssh-keys",
				Value: &cli.StringSlice{},
				Usage: "droplet public SSH keys",
			},
			cli.BoolFlag{
				Name:  "backups",
				Usage: "enable droplet backups",
			},
			cli.BoolFlag{
				Name:  "ipv6",
				Usage: "enable droplet IPv6",
			},
			cli.BoolFlag{
				Name:  "private-networking",
				Usage: "enable droplet private networking",
			},
			cli.StringFlag{
				Name:  "user-data",
				Usage: "droplet name",
			},
		},
		Action: droplets.Create,
	}
}

func dropletDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Action: droplets.Delete,
	}
}

func dropletGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Action: droplets.Get,
	}
}

func dropletKernels() cli.Command {
	return cli.Command{
		Name:  "kernels",
		Usage: "get kernels for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Action: droplets.Kernels,
	}
}

func dropletSnapshots() cli.Command {
	return cli.Command{
		Name:  "snapshots",
		Usage: "get snapshots for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Action: droplets.Snapshots,
	}
}

func dropletBackups() cli.Command {
	return cli.Command{
		Name:  "backups",
		Usage: "get backups for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			return nil
		},
		Action: droplets.Backups,
	}
}

func dropletActions() cli.Command {
	return cli.Command{
		Name:  "actions",
		Usage: "get actions for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Action: droplets.Actions,
	}
}

func dropletNeighbors() cli.Command {
	return cli.Command{
		Name:  "neighbors",
		Usage: "get neighbors for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Action: droplets.Neighbors,
	}
}
