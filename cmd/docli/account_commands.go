package main

import (
	"github.com/bryanl/docli/account"
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
		Action: account.Action,
	}
}
