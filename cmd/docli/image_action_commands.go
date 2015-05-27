package main

import (
	"fmt"

	"github.com/bryanl/docli/imagesactions"
	"github.com/codegangsta/cli"
)

func imageActionCommands() cli.Command {
	return cli.Command{
		Name:  "image-action",
		Usage: "image action commands",
		Subcommands: []cli.Command{
			imageActionGet(),
			imageActionTransfer(),
		},
	}
}

func imageActionGet() cli.Command {
	return cli.Command{
		Name:  "get",
		Usage: "get image action",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:  "image-id",
				Usage: "image id",
			},
			cli.IntFlag{
				Name:  "action-id",
				Usage: "action id",
			},
		},
		Before: func(c *cli.Context) error {
			imageID := c.Int("image-id")
			if imageID < 1 {
				return fmt.Errorf("image id required")
			}

			actionID := c.Int("action-id")
			if actionID < 1 {
				return fmt.Errorf("action id required")
			}

			return nil
		},
		Action: func(c *cli.Context) {
			client := newClient(c)

			imageID := c.Int("image-id")
			actionID := c.Int("action-id")

			action, err := imageactions.Get(client, imageID, actionID)
			if err != nil {
				panic(err)
			}

			j, err := toJSON(action)
			if err != nil {
				panic(err)
			}

			fmt.Println(j)
		},
	}
}

func imageActionTransfer() cli.Command {
	return cli.Command{
		Name:  "transfer",
		Usage: "tranfser image (not implemented)",
	}
}
