package main

import (
	"fmt"

	"github.com/bryanl/docli/sizes"
	"github.com/codegangsta/cli"
)

func sizeCommands() cli.Command {
	return cli.Command{
		Name:  "size",
		Usage: "size commands",
		Subcommands: []cli.Command{
			sizeList(),
		},
	}
}

func sizeList() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list sizes",
		Action: func(c *cli.Context) {
			opts := loadOpts(c)
			client := newClient(c)

			list, err := sizes.List(client, opts)
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
