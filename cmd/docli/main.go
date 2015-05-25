package main

import (
	"encoding/json"
	"os"

	"code.google.com/p/goauth2/oauth"

	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func main() {
	app := cli.NewApp()
	app.Name = "docli"
	app.Usage = "DigitalOcean API CLI"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		tokenFlag(),
	}

	app.Commands = []cli.Command{
		dropletCommands(),
		sshKeyCommands(),
	}

	app.Run(os.Args)
}

func tokenFlag() cli.Flag {
	return cli.StringFlag{
		Name:   "token",
		Usage:  "DigitalOcean API V2 Token",
		EnvVar: "DO_TOKEN",
	}
}

func newClient(token string) *godo.Client {
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: token},
	}

	return godo.NewClient(t.Client())
}

func toJSON(item interface{}) (string, error) {
	b, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}
