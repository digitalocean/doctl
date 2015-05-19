package main

import (
	"fmt"
	"os"

	"code.google.com/p/goauth2/oauth"
	"github.com/bryanl/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func main() {
	app := cli.NewApp()
	app.Name = "docli"
	app.Usage = "DigitalOcean API CLI"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "token",
			Usage:  "DigitalOcean API V2 Token",
			EnvVar: "DO_TOKEN",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "droplet",
			Usage: "droplet commands",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "list droplets",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:   "token",
							Usage:  "DigitalOcean API V2 Token",
							EnvVar: "DO_TOKEN",
						},
						cli.BoolFlag{
							Name:  "json",
							Usage: "return list of droplets as JSON array",
						},
					},
					Action: func(c *cli.Context) {
						token := c.String("token")
						t := &oauth.Transport{
							Token: &oauth.Token{AccessToken: token},
						}

						client := godo.NewClient(t.Client())
						list, err := droplets.List(client)
						if err != nil {
							panic(err)
						}
						if c.Bool("json") {
							j, err := droplets.ToJSON(list)
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
				},
			},
		},
	}

	app.Run(os.Args)
}
