package main

import (
	"github.com/bryanl/docli"
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
		Action: docli.AccountGet,
	}
}
