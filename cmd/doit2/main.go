package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/bryanl/doit"
	"github.com/bryanl/doit/commands"
)

func init() {
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.InfoLevel)

	doit.Bail = func(err error, msg string) {
		logrus.WithField("err", err).Fatal(msg)
	}
}

func main() {
	err := commands.LoadConfig()
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to load config")
	}

	commands.Root().Execute()
}
