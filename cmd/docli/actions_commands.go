package main

import (
	"github.com/bryanl/docli"
	"github.com/codegangsta/cli"
)

func actionCommands() cli.Command {
	return cli.Command{
		Name:  "action",
		Usage: "action commands",
		Subcommands: []cli.Command{
			actionList(),
			actionGet(),
		},
	}
}

func actionList() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list actions",
		Flags: []cli.Flag{
			jsonFlag(),
			textFlag(),
		},
		Action: docli.ActionList,
	}
}

func actionGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get action by id",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  docli.ArgActionID,
				Usage: "Action id",
			},
			jsonFlag(),
			textFlag(),
		},
		Action: docli.ActionGet,
	}
}
