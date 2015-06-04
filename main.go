package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/codegangsta/cli"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/golang.org/x/oauth2"
)

const AppVersion = "0.0.16"

var defaultConfigPath string

var APIKey string
var ConfigPath string
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
	defaultConfigPath = filepath.Join(getHomeDir(), "/.docfg")
}

func main() {
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
		cli.StringFlag{Name: "config, c", Value: defaultConfigPath, Usage: "Path to configuration file"},
	}
	app.Before = func(ctx *cli.Context) error {
		config, err := getConfig(ctx.String("config"))
		if err != nil {
			fmt.Printf("Unable to load config file: %s\n", err)
			os.Exit(1)
		}

		if ctx.String("api-key") != "" {
			APIKey = ctx.String("api-key")
		}

		if APIKey == "" {
			APIKey = config.APIKey
		}

		if APIKey == "" && ctx.BoolT("help") != false {
			cli.ShowAppHelp(ctx)
			fmt.Println("Must provide API Key via DIGITALOCEAN_API_KEY environment variable or via CLI argument.")
			os.Exit(1)
		}

		switch ctx.String("format") {
		case "json":
			OutputFormat = ctx.String("format")
		case "yaml":
			OutputFormat = ctx.String("format")
		default:
			fmt.Printf("Invalid output format: %s. Available output options: json, yaml.\n", ctx.String("format"))
			os.Exit(64)
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

	app.Run(os.Args)
}
