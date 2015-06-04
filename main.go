package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/codegangsta/cli"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/golang.org/x/oauth2"
)

const AppVersion = "0.0.16"

var APIKey string
var OutputFormat string

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func init() {
	log.SetFlags(0)
	log.SetPrefix("doctl> ")
}

func main() {
	app := buildApp()
	app.RunAndExitOnError()
}

func buildApp() *cli.App {
	app := cli.NewApp()
	app.Name = "doctl"
	app.Version = AppVersion
	app.Usage = "Digital Ocean Control TooL."
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "api-key, k",
			Value:  "",
			Usage:  "API Key for DO APIv2.",
			EnvVar: "DIGITALOCEAN_API_KEY, DIGITAL_OCEAN_API_KEY",
		},
		cli.StringFlag{Name: "format,f", Value: "yaml", Usage: "Format for output."},
		cli.BoolFlag{Name: "debug,d", Usage: "Turn on debug output."},
	}
	app.Before = func(ctx *cli.Context) error {
		if ctx.String("api-key") != "" {
			APIKey = ctx.String("api-key")
		}

		if APIKey == "" && !ctx.Bool("help") && !ctx.Bool("version") {
			return errors.New("must provide API Key via DIGITALOCEAN_API_KEY environment variable or via CLI argument.")
		}

		switch ctx.String("format") {
		case "json":
			OutputFormat = ctx.String("format")
		case "yaml":
			OutputFormat = ctx.String("format")
		default:
			return fmt.Errorf("invalid output format: %q, available output options: json, yaml.", ctx.String("format"))
		}

		return nil
	}
	app.Commands = []cli.Command{
		AccountCommand,
		ActionCommand,
		DomainCommand,
		DropletCommand,
		RegionCommand,
		SizeCommand,
		SSHCommand,
	}

	return app
}
