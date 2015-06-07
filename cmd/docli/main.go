package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/docli"
	"github.com/codegangsta/cli"
	"golang.org/x/oauth2"
)

type tokenSource struct {
	AccessToken string
}

func (t *tokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: t.AccessToken,
	}, nil
}

func init() {
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.WarnLevel)

	docli.Bail = func(err error, msg string) {
		logrus.WithField("err", err).Fatal(msg)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "docli"
	app.Usage = "DigitalOcean API CLI"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		tokenFlag(),
		debugFlag(),
	}

	app.Commands = []cli.Command{
		accountCommands(),
		actionCommands(),
		domainCommands(),
		dropletCommands(),
		dropletActionCommands(),
		imageActionCommands(),
		imageCommands(),
		regionCommands(),
		sizeCommands(),
		sshKeyCommands(),
	}

	app.RunAndExitOnError()
}

func tokenFlag() cli.Flag {
	return cli.StringFlag{
		Name:   "token",
		Usage:  "DigitalOcean API V2 Token",
		EnvVar: "DIGITAL_OCEAN_TOKEN",
	}
}

func debugFlag() cli.Flag {
	return cli.BoolFlag{
		Name:  "debug",
		Usage: "Debug",
	}
}
