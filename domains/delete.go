package domains

import (
	log "github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
)

func Delete(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	name := c.String("domain-name")
	_, err := client.Domains.Delete(name)
	if err != nil {
		log.WithField("err", err).Fatal("could not delete account")
	}
}
