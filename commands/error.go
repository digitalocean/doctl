package commands

import (
	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

func checkErr(err error, cmd ...*cobra.Command) {
	if err == nil {
		return
	}

	if len(cmd) > 0 {
		cmd[0].Help()
	}

	logrus.WithField("err", err).Fatal("an error occurred")
}
