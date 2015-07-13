package main

import (
	"github.com/bryanl/doit"
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
		Flags:  []cli.Flag{},
		Action: doit.ImagesList,
	}
}

func imageListDistributions() cli.Command {
	return cli.Command{
		Name:   "list-distribution",
		Usage:  "list distribution images",
		Flags:  []cli.Flag{},
		Action: doit.ImagesListDistribution,
	}
}

func imageListApplication() cli.Command {
	return cli.Command{
		Name:   "list-application",
		Usage:  "list application images",
		Flags:  []cli.Flag{},
		Action: doit.ImagesListApplication,
	}
}

func imageListUser() cli.Command {
	return cli.Command{
		Name:   "list-user",
		Usage:  "list user images",
		Flags:  []cli.Flag{},
		Action: doit.ImagesListUser,
	}
}

func imageGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get image by id",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  doit.ArgImage,
				Usage: "image id or slug",
			},
		},
		Action: doit.ImagesGet,
	}
}

func imageActions() cli.Command {
	return cli.Command{
		Name:  "actions",
		Usage: "image actions",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgImageID,
				Usage: "image id",
			},
		},
	}
}

func imageUpdate() cli.Command {
	return cli.Command{
		Name:  "update",
		Usage: "update image (not implemented)",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgImageID,
				Usage: "image id (required)",
			},
			cli.IntFlag{
				Name:  doit.ArgImageName,
				Usage: "image name (required)",
			},
		},
		Action: doit.ImagesUpdate,
	}
}

func imageDelete() cli.Command {
	return cli.Command{
		Name:  "delete",
		Usage: "delete image",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  doit.ArgImageID,
				Usage: "image id (required)",
			},
		},
		Action: doit.ImagesDelete,
	}
}
