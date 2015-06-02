package main

import (
	"fmt"

	"github.com/bryanl/docli/images"
	"github.com/codegangsta/cli"
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
		Name:   "list",
		Usage:  "list images",
		Action: images.List,
	}
}

func imageListDistributions() cli.Command {
	return cli.Command{
		Name:   "list-distribution",
		Usage:  "list distribution images",
		Action: images.ListDistribution,
	}
}

func imageListApplication() cli.Command {
	return cli.Command{
		Name:   "list-application",
		Usage:  "list application images",
		Action: images.ListApplication,
	}
}

func imageListUser() cli.Command {
	return cli.Command{
		Name:   "list-user",
		Usage:  "list user images",
		Action: images.ListUser,
	}
}

func imageGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get image by id",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "image",
				Usage: "image id or slug",
			},
		},
		Action: images.Get,
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
				Name:  "image-id",
				Usage: "image id (required)",
			},
			cli.IntFlag{
				Name:  "image-name",
				Usage: "image name (required)",
			},
		},
		Action: images.Update,
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
		Action: images.Delete,
	}
}
