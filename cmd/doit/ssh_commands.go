package main

import (
	"github.com/bryanl/doit"
	"github.com/codegangsta/cli"
)

func sshCommands() cli.Command {
	return cli.Command{
		Name:  "ssh",
		Usage: "SSH to droplet. Provide name or id",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  doit.ArgDropletName,
				Usage: "droplet name",
			},
			cli.IntFlag{
				Name:  doit.ArgDropletID,
				Usage: "droplet id",
			},
			cli.StringFlag{
				Name:  doit.ArgSSHUser,
				Usage: "ssh user",
			},
			cli.StringSliceFlag{
				Name:  doit.ArgSSHOption,
				Usage: "ssh options",
			},
		},
		Action: doit.SSH,
	}
}
