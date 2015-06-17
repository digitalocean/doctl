package main

import (
	"github.com/bryanl/docli"
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
				Name:  docli.ArgImageID,
				Usage: "image id",
			},
			cli.IntFlag{
				Name:  docli.ArgActionID,
				Usage: "action id",
			},
			jsonFlag(),
			textFlag(),
		},
		Action: docli.ImageActionsGet,
	}
}

func imageActionTransfer() cli.Command {
	return cli.Command{
		Name:  "transfer",
		Usage: "tranfser image (not implemented)",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  docli.ArgImageID,
				Usage: "image id",
			},
			cli.StringFlag{
				Name:  docli.ArgRegionSlug,
				Usage: "region",
			},
			jsonFlag(),
			textFlag(),
		},
		Action: docli.ImageActionsTransfer,
	}
}
