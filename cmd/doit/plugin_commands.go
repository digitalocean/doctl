package main

import (
	"github.com/bryanl/doit"
	"github.com/codegangsta/cli"
)

func pluginCommands() cli.Command {
	return cli.Command{
		Name:  "plugin",
		Usage: "plugin commands",
		Subcommands: []cli.Command{
			pluginList(),
		},
	}
}

func pluginList() cli.Command {
	return cli.Command{
		Name:   "list",
		Usage:  "list plugins",
		Action: doit.PluginList,
	}
}
