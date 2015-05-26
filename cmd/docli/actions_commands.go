package main

import (
	"fmt"

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
		Name:  "list",
		Usage: "list actions",
		Action: func(c *cli.Context) {
			token := c.GlobalString("token")
			client := newClient(token)

			list, err := actions.List(client)
			if err != nil {
				// TODO this needs to be json
				panic(err)
			}

			j, err := toJSON(list)
			if err != nil {
				panic(err)
			}
			fmt.Println(j)
		},
	}
}
