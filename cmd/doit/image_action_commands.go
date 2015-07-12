package main

import (
	"github.com/bryanl/doit"
	"github.com/codegangsta/cli"
)

func imageActionCommands() cli.Command {
	return cli.Command{
		Name:  "image-action",
		Usage: "image action commands",
		Subcommands: []cli.Command{
			imageActionGet(),
			imageActionTransfer(),
		},
	}
}

func imageActionGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get image action",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgImageID,
				Usage: "image id",
			},
			cli.IntFlag{
				Name:  doit.ArgActionID,
				Usage: "action id",
			},
			jsonFlag(),
			textFlag(),
		},
		Action: doit.ImageActionsGet,
	}
}

func imageActionTransfer() cli.Command {
	return cli.Command{
		Name:  "transfer",
		Usage: "tranfser image (not implemented)",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgImageID,
				Usage: "image id",
			},
			cli.StringFlag{
				Name:  doit.ArgRegionSlug,
				Usage: "region",
			},
			jsonFlag(),
			textFlag(),
		},
		Action: doit.ImageActionsTransfer,
	}
}
