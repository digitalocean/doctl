package main

import (
	"log"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/codegangsta/cli"
)

var AccountCommand = cli.Command{
	Name:    "account",
	Aliases: []string{"whoami"},
	Usage:   "Account commands.",
	Action:  accountShow,
	Subcommands: []cli.Command{
		{
			Name:    "show",
			Aliases: []string{"s"},
			Usage:   "Show an account.",
			Action:  accountShow,
		},
	},
}

func accountShow(ctx *cli.Context) {
	account, _, err := client.Account.Get()
	if err != nil {
		log.Fatal(err)
	}

	WriteOutput(account)
}
