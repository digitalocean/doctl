package main

import (
	"github.com/bryanl/doit"
	"github.com/codegangsta/cli"
)

func pluginCommands() cli.Command {
	return cli.Command{
		Name:   "plugin",
		Usage:  "plugin commands",
		Action: doit.Plugin,
	}
}
