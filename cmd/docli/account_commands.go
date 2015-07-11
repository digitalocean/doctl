package main

import (
	"github.com/bryanl/doit"
	"github.com/codegangsta/cli"
)

func accountCommands() cli.Command {
	return cli.Command{
		Name:  "account",
		Usage: "account commands",
		Subcommands: []cli.Command{
			accountGet(),
		},
	}
}

func accountGet() cli.Command {
	return cli.Command{
		Name:   "get",
		Usage:  "get account",
		Action: doit.AccountGet,
	}
}
