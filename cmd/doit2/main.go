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
	commands.Execute()
}
