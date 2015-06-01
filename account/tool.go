package account

import (
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func Action(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	err := AccountGet(client, c.App.Writer)
	if err != nil {
		log.WithField("err", err).Fatal("could not display account")
	}
}

func AccountGet(client *godo.Client, w io.Writer) error {
	a, _, err := client.Account.Get()
	if err != nil {
		return err
	}

	return docli.WriteJSON(a, w)
}
