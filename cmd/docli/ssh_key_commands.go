package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/sshkeys"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func sshKeyCommands() cli.Command {
	return cli.Command{
		Name:  "ssh-key",
		Usage: "ssh key commands",
		Subcommands: []cli.Command{
			sshKeyList(),
			sshKeyCreate(),
			sshKeyGet(),
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

func sshKeyCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create ssh key",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "ssh key name",
			},
			cli.StringFlag{
				Name:  "public-key",
				Usage: "ssh public key",
			},
		},
		Action: func(c *cli.Context) {
			token := c.GlobalString("token")
			client := newClient(token)
			cr := &sshkeys.CreateRequest{
				Name:      c.String("name"),
				PublicKey: c.String("public-key"),
			}

			if !cr.IsValid() {
				log.Error("invalid parameters")
				return
			}

			key, err := sshkeys.Create(client, cr)
			if err != nil {
				log.WithField("err", err).Error("unable to create key")
				return
			}

			j, err := toJSON(key)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

func sshKeyGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get ssh key",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "ssh key id",
			},
			cli.StringFlag{
				Name:  "fingerprint",
				Usage: "ssh key fingerprint",
			},
		},
		Before: func(c *cli.Context) error {
			id := c.Int("id")
			fingerprint := c.String("fingerprint")

			return sshkeys.IsValidGetArgs(id, fingerprint)
		},
		Action: func(c *cli.Context) {
			token := c.GlobalString("token")
			client := newClient(token)

			id := c.Int("id")
			fingerprint := c.String("fingerprint")

			var key *godo.Key
			var err error

			switch {
			case id != 0:
				key, err = sshkeys.RetrieveByID(client, id)
			default:
				key, err = sshkeys.RetrieveByFingerprint(client, fingerprint)
			}

			if err != nil {
				panic(err)
			}

			j, err := toJSON(key)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}
