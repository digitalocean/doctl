package main

import (
	"github.com/bryanl/doit"
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
		Action: doit.ActionList,
	}
}

func actionGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get action by id",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgActionID,
				Usage: "Action id",
			},
		},
		Action: doit.ActionGet,
	}
}
