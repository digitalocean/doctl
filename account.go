package doit

import (
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func AccountGet(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	err := accountGet(client, c.App.Writer)
	if err != nil {
		log.WithField("err", err).Fatal("could not display account")
	}
}

func accountGet(client *godo.Client, w io.Writer) error {
	a, _, err := client.Account.Get()
	if err != nil {
		return err
	}

	return writeJSON(a, w)
}
