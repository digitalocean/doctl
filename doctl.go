package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
)

const AppVersion = "0.0.8"

var APIKey = ""

func init() {
	if os.Getenv("DIGITAL_OCEAN_API_KEY") != "" {
		APIKey = os.Getenv("DIGITAL_OCEAN_API_KEY")
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "doctl"
	app.Version = AppVersion
	app.Usage = "Digital Ocean Control TooL."
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "api-key,k", Value: "", Usage: "API Key for DO APIv2."},
	}
	app.Before = func(ctx *cli.Context) error {
		if ctx.String("api-key") != "" {
			APIKey = ctx.String("api-key")
		}

		if APIKey == "" {
			cli.ShowAppHelp(ctx)
			fmt.Println("Must provide API Key via DIGITAL_OCEAN_API_KEY environment variable or via CLI argument.")
			os.Exit(1)
		}
		return nil
	}
	app.Commands = []cli.Command{
		ActionCommand,
		DomainCommand,
		DropletCommand,
		RegionCommand,
		SizeCommand,
		SSHCommand,
	}

	app.Run(os.Args)
}
