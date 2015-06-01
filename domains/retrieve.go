package domains

import (
	log "github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
)

func Get(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.String("domain-name")
	a, _, err := client.Domains.Get(id)
	if err != nil {
		log.WithField("err", err).Fatal("could not retrieve domain")
	}
	docli.WriteJSON(a, c.App.Writer)
}
