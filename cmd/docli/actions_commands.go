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
		Name:   "list",
		Usage:  "list actions",
		Action: docli.ActionList,
	}
}

func actionGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get action by id",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "action-id",
				Usage: "Action id",
			},
		},
		Action: docli.ActionGet,
	}
}
