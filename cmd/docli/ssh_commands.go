package main

import (
	"github.com/bryanl/docli"
	"github.com/codegangsta/cli"
)

func sshCommands() cli.Command {
	return cli.Command{
		Name:  "ssh",
		Usage: "SSH to droplet. Provide name or id",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  docli.ArgDropletName,
				Usage: "droplet name",
			},
			cli.IntFlag{
				Name:  docli.ArgDropletID,
				Usage: "droplet id",
			},
		},
		Action: docli.SSH,
	}
}
