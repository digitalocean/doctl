package main

import (
	"fmt"

	"github.com/bryanl/docli/regions"
	"github.com/codegangsta/cli"
)

func regionCommands() cli.Command {
	return cli.Command{
		Name:  "region",
		Usage: "region commands",
		Subcommands: []cli.Command{
			regionList(),
		},
	}
}

func regionList() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list regions",
		Action: func(c *cli.Context) {
			client := newClient(c)
			list, err := regions.List(client)
			if err != nil {
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
