package main

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
	"github.com/codegangsta/cli"
	"golang.org/x/oauth2"
)

const (
	configFile = ".doitcfg"
)

var (
	argMap = doit.ConfigArgMap{
		"token":  "token",
		"output": "output",
	}
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
	logrus.SetLevel(logrus.InfoLevel)

	doit.Bail = func(err error, msg string) {
		logrus.WithField("err", err).Fatal(msg)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "doit"
	app.Usage = "DigitalOcean Interactive Tool"
	app.Version = "0.4.0"
	app.Flags = []cli.Flag{
		tokenFlag(),
		debugFlag(),
		outputFlag(),
	}

	app.Commands = []cli.Command{
		accountCommands(),
		actionCommands(),
		domainCommands(),
		dropletCommands(),
		dropletActionCommands(),
		imageActionCommands(),
		imageCommands(),
		pluginCommands(),
		sshKeyCommands(),
		regionCommands(),
		sizeCommands(),
		sshCommands(),
	}

	fp, _ := configFilePath()
	if _, err := os.Stat(fp); err == nil {
		if guts, err := ioutil.ReadFile(fp); err == nil {
			cf := doit.NewConfigFile(argMap, guts)
			if newArgs, err := cf.Args(); err == nil {
				os.Args = doit.GlobalArgs(os.Args, newArgs)
			}
		}
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

func outputFlag() cli.Flag {
	return cli.StringFlag{
		Name:  doit.ArgOutput,
		Usage: "output format (json or text)",
	}
}

func configFilePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(usr.HomeDir, configFile)
	return dir, nil
}
