package main

import (
	"github.com/bryanl/docli"
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
		Name:   "list",
		Usage:  "list regions",
		Action: docli.RegionList,
	}
}
