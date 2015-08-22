package commands

import "github.com/Sirupsen/logrus"

func checkErr(err error) {
	if err == nil {
		return
	}

	logrus.WithField("err", err).Fatal("an error occurred")
}
