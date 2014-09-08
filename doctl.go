package main

import (
	"os"

	"github.com/codegangsta/cli"
)

const AppVersion = "0.0.2"

func main() {
	app := cli.NewApp()
	app.Name = "doctl"
	app.Version = AppVersion
	app.Usage = "Digital Ocean Control TooL."
	app.Commands = []cli.Command{
		DropletCommand,
		SSHCommand,
		RegionCommand,
		SizeCommand,
	}

	app.Run(os.Args)
}
