package main

import (
	"fmt"

	"github.com/bryanl/docli/images"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func imageCommands() cli.Command {
	return cli.Command{
		Name:  "image",
		Usage: "image commands",
		Subcommands: []cli.Command{
			imageList(),
			imageListDistributions(),
			imageListApplication(),
			imageListUser(),
			imageGet(),
			imageActions(),
			imageUpdate(),
			imageDelete(),
		},
	}
}

func imageList() cli.Command {
	return cli.Command{
		Name:  "list",
		Usage: "list images",
		Action: func(c *cli.Context) {
			opts := loadOpts(c)
			client := newClient(c)
			list, err := images.List(client, opts)
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

func imageListDistributions() cli.Command {
	return cli.Command{
		Name:  "list-distribution",
		Usage: "list distribution images",
		Action: func(c *cli.Context) {
			opts := loadOpts(c)
			client := newClient(c)
			list, err := images.ListDistribution(client, opts)
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

func imageListApplication() cli.Command {
	return cli.Command{
		Name:  "list-application",
		Usage: "list application images",
		Action: func(c *cli.Context) {
			opts := loadOpts(c)
			client := newClient(c)
			list, err := images.ListApplication(client, opts)
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

func imageListUser() cli.Command {
	return cli.Command{
		Name:  "list-user",
		Usage: "list user images",
		Action: func(c *cli.Context) {
			opts := loadOpts(c)
			client := newClient(c)
			list, err := images.ListUser(client, opts)
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

func imageGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get image by id",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "image id",
			},
			cli.StringFlag{
				Name:  "slug",
				Usage: "image slug",
			},
		},
		Before: func(c *cli.Context) error {
			id := c.Int("id")
			slug := c.String("slug")

			if id > 0 && len(slug) > 0 {
				return fmt.Errorf("id and slug are mutually exclusive")
			}

			if id < 1 && len(slug) < 1 {
				return fmt.Errorf("either id or slug are required")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			client := newClient(c)

			var image *godo.Image
			var err error
			if id := c.Int("id"); id > 0 {
				image, err = images.GetByID(client, id)
			} else {
				slug := c.String("slug")
				image, err = images.GetBySlug(client, slug)
			}

			if err != nil {
				panic(err)
			}

			j, err := toJSON(image)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

func imageActions() cli.Command {
	return cli.Command{
		Name:  "actions",
		Usage: "image actions",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "image id",
			},
		},
		Before: func(c *cli.Context) error {
			id := c.Int("id")
			slug := c.String("slug")

			if id > 0 && len(slug) > 0 {
				return fmt.Errorf("id and slug are mutually exclusive")
			}

			if id < 1 && len(slug) < 1 {
				return fmt.Errorf("either id or slug are required")
			}

			return fmt.Errorf("not yet implemented in godo")

		},
		Action: func(c *cli.Context) {
		},
	}
}

func imageUpdate() cli.Command {
	return cli.Command{
		Name:  "update",
		Usage: "update image (not implemented)",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "image id (required)",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid image id")
			}

			return fmt.Errorf("not implemented")
			//return nil
		},
		Action: func(c *cli.Context) {
		},
	}
}

func imageDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete image",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "id",
				Usage: "image id (required)",
			},
		},
		Before: func(c *cli.Context) error {
			if !c.IsSet("id") {
				return fmt.Errorf("invalid image id")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			client := newClient(c)
			id := c.Int("id")
			err := images.Delete(client, id)
			if err != nil {
				panic(err)
			}
		},
	}
}
