package main

import (
	"github.com/bryanl/doit"
	"github.com/codegangsta/cli"
)

func regionCommands() cli.Command {
	return cli.Command{
		Name:  "region",
		Usage: "region commands",
		Subcommands: []cli.Command{
			regionList(),
		},
	}
}

func regionList() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list regions",
		Flags: []cli.Flag{
			jsonFlag(),
			textFlag(),
		},
		Action: doit.RegionList,
	}
}
