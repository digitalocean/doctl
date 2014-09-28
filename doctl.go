package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/slantview/doctl/commands"
)

const AppVersion = "0.0.9"

func init() {
	commands.APIKey = os.Getenv("DIGITAL_OCEAN_API_KEY")
}

func main() {
	app := cli.NewApp()
	app.Name = "doctl"
	app.Version = AppVersion
	app.Usage = "Digital Ocean Control TooL."
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "api-key,k", Value: "", Usage: "API Key for DO APIv2."},
		cli.StringFlag{Name: "format,f", Value: "yaml", Usage: "Format for output."},
	}
	app.Before = func(ctx *cli.Context) error {
		if ctx.String("api-key") != "" {
			commands.APIKey = ctx.String("api-key")
		}

		switch ctx.String("format") {
		case "json":
			commands.OutputFormat = ctx.String("format")
		case "yaml":
			commands.OutputFormat = ctx.String("format")
		default:
			fmt.Printf("Invalid output format: %s. Available output options: json, yaml.\n", ctx.String("format"))
			os.Exit(64)
		}

		if commands.APIKey == "" {
			cli.ShowAppHelp(ctx)
			fmt.Println("Must provide API Key via DIGITAL_OCEAN_API_KEY environment variable or via CLI argument.")
			os.Exit(1)
		}

		return nil
	}
	app.Commands = []cli.Command{
		commands.ActionCommand,
		commands.DomainCommand,
		commands.DropletCommand,
		commands.RegionCommand,
		commands.SizeCommand,
		commands.SSHCommand,
	}

	app.Run(os.Args)
}
