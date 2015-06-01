package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/droplets"
	"github.com/codegangsta/cli"
)

func dropletCommands() cli.Command {
	return cli.Command{
		Name:  "droplet",
		Usage: "droplet commands",
		Subcommands: []cli.Command{
			dropletList(),
			dropletCreate(),
			dropletGet(),
			dropletKernels(),
			dropletSnapshots(),
			dropletBackups(),
			dropletActions(),
			dropletDelete(),
			dropletNeighbors(),
		},
	}
}

func dropletList() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list droplets",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "json",
				Usage: "return list of droplets as JSON array",
			},
		},
		Action: func(c *cli.Context) {
			opts := loadOpts(c)
			client := newClient(c)

			list, err := droplets.List(client, opts)
			if err != nil {
				panic(err)
			}
			if c.Bool("json") {
				j, err := toJSON(list)
				if err != nil {
					panic(err)
				}
				fmt.Println(j)
			} else {
				for _, d := range list {
					fmt.Printf("%s\n", droplets.ToText(&d))
				}
			}

		},
	}
}

func dropletCreate() cli.Command {
	return cli.Command{
		Name:  "create",
		Usage: "create droplet",
		Flags: []cli.Flag{

			cli.StringFlag{
				Name:  "name",
				Usage: "droplet name",
			},
			cli.StringFlag{
				Name:  "region",
				Usage: "droplet region",
			},
			cli.StringFlag{
				Name:  "size",
				Usage: "droplet size",
			},
			cli.StringFlag{
				Name:  "image",
				Usage: "droplet image",
			},
			cli.StringSliceFlag{
				Name:  "ssh-keys",
				Value: &cli.StringSlice{},
				Usage: "droplet public SSH keys",
			},
			cli.BoolFlag{
				Name:  "backups",
				Usage: "enable droplet backups",
			},
			cli.BoolFlag{
				Name:  "ipv6",
				Usage: "enable droplet IPv6",
			},
			cli.BoolFlag{
				Name:  "private-networking",
				Usage: "enable droplet private networking",
			},
			cli.StringFlag{
				Name:  "user-data",
				Usage: "droplet name",
			},
		},
		Action: func(c *cli.Context) {
			client := newClient(c)
			cr := &droplets.CreateRequest{
				Name:              c.String("name"),
				Region:            c.String("region"),
				Size:              c.String("size"),
				Image:             c.String("image"),
				SSHKeys:           c.StringSlice("ssh-keys"),
				Backups:           c.Bool("backups"),
				IPv6:              c.Bool("ipv6"),
				PrivateNetworking: c.Bool("private-networking"),
				UserData:          c.String("user-data"),
			}

			droplet, err := droplets.Create(client, cr)
			if err != nil {
				log.WithField("err", err).Error("unable to create droplet")
				return
			}

			fmt.Printf("created droplet %d\n", droplet.ID)
		},
	}
}

func dropletDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			client := newClient(c)

			id := c.Int("id")

			err := droplets.Delete(client, id)
			if err != nil {
				panic(err)
			}
		},
	}
}

func dropletGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			client := newClient(c)

			id := c.Int("id")

			droplet, err := droplets.Get(client, id)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(droplet)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

func dropletKernels() cli.Command {
	return cli.Command{
		Name:  "kernels",
		Usage: "get kernels for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			opts := loadOpts(c)
			client := newClient(c)

			id := c.Int("id")

			list, err := droplets.Kernels(client, opts, id)
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

func dropletSnapshots() cli.Command {
	return cli.Command{
		Name:  "snapshots",
		Usage: "get snapshots for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			opts := loadOpts(c)
			client := newClient(c)

			id := c.Int("id")

			list, err := droplets.Snapshots(client, opts, id)
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

func dropletBackups() cli.Command {
	return cli.Command{
		Name:  "backups",
		Usage: "get backups for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			opts := loadOpts(c)
			client := newClient(c)

			id := c.Int("id")

			list, err := droplets.Backups(client, opts, id)
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

func dropletActions() cli.Command {
	return cli.Command{
		Name:  "actions",
		Usage: "get actions for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			opts := loadOpts(c)
			client := newClient(c)

			id := c.Int("id")

			list, err := droplets.Actions(client, opts, id)
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

func dropletNeighbors() cli.Command {
	return cli.Command{
		Name:  "neighbors",
		Usage: "get neighbors for droplet",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "droplet id",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid droplet id")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			client := newClient(c)

			id := c.Int("id")

			list, err := droplets.Neighbors(client, id)
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
