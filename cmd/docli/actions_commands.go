package main

import (
	"github.com/bryanl/docli/actions"
	"github.com/codegangsta/cli"
)

func actionCommands() cli.Command {
	return cli.Command{
		Name:  "action",
		Usage: "action commands",
		Subcommands: []cli.Command{
			actionList(),
		},
	}
}

func actionList() cli.Command {
	return cli.Command{
		Name:   "list",
		Usage:  "list actions",
		Action: actions.Action,
	}
}
