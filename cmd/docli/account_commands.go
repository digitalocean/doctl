package main

import (
	"fmt"

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
		Name:  "get",
		Usage: "get account",
		Action: func(c *cli.Context) {
			client := newClient(c)

			a, err := account.Get(client)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(a)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}
