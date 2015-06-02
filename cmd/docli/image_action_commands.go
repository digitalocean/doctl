package main

import (
	"github.com/bryanl/docli/imagesactions"
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
				Name:  "image-id",
				Usage: "image id",
			},
			cli.IntFlag{
				Name:  "action-id",
				Usage: "action id",
			},
		},
		Action: imageactions.Get,
	}
}

func imageActionTransfer() cli.Command {
	return cli.Command{
		Name:  "transfer",
		Usage: "tranfser image (not implemented)",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "image-id",
				Usage: "image id",
			},
			cli.StringFlag{
				Name:  "region",
				Usage: "region",
			},
		},
		Action: imageactions.Transfer,
	}
}
