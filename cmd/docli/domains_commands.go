package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/domains"
	"github.com/codegangsta/cli"
)

func domainCommands() cli.Command {
	return cli.Command{
		Name:  "domain",
		Usage: "domain commands",
		Subcommands: []cli.Command{
			domainList(),
			domainCreate(),
			domainGet(),
		},
	}
}

func domainList() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list domains",
		Action: func(c *cli.Context) {
			token := c.GlobalString("token")
			client := newClient(token)

			list, err := domains.List(client)
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

func domainCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "domain name",
			},
			cli.StringFlag{
				Name:  "ip-address",
				Usage: "domain ip address",
			},
		},
		Before: func(c *cli.Context) error {
			cr := &domains.CreateRequest{
				Name:      c.String("name"),
				IPAddress: c.String("ip-address"),
			}

			if !cr.IsValid() {
				return fmt.Errorf("invalid arguments")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			token := c.GlobalString("token")
			client := newClient(token)
			cr := &domains.CreateRequest{
				Name:      c.String("name"),
				IPAddress: c.String("ip-address"),
			}

			key, err := domains.Create(client, cr)
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

func domainGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "domain name",
			},
		},
		Before: func(c *cli.Context) error {
			name := c.String("name")
			if len(name) < 1 {
				return fmt.Errorf("invalid domain name")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			token := c.GlobalString("token")
			client := newClient(token)

			name := c.String("name")

			domain, err := domains.Retrieve(client, name)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(domain)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}
