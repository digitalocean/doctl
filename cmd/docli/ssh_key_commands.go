package main

import (
	"fmt"

	"github.com/bryanl/docli/sshkeys"
	"github.com/codegangsta/cli"
)

func sshKeyCommands() cli.Command {
	return cli.Command{
		Name:  "ssh-key",
		Usage: "ssh key commands",
		Subcommands: []cli.Command{
			sshKeyList(),
		},
	}
}

func sshKeyList() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list ssh keys",
		Action: func(c *cli.Context) {
			token := c.GlobalString("token")
			client := newClient(token)

			list, err := sshkeys.List(client)
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
