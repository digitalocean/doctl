package main

import (
	"github.com/bryanl/docli"
	"github.com/codegangsta/cli"
)

func sizeCommands() cli.Command {
	return cli.Command{
		Name:  "size",
		Usage: "size commands",
		Subcommands: []cli.Command{
			sizeList(),
		},
	}
}

func sizeList() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list sizes",
		Flags: []cli.Flag{
			jsonFlag(),
			textFlag(),
		},
		Action: docli.SizeList,
	}
}
