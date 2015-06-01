package domainrecs

import (
	log "github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
)

// Retrieve a domain record.
func Get(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	domainName := c.String("domain-name")
	recordID := c.Int("record-id")

	r, _, err := client.Domains.Record(domainName, recordID)
	if err != nil {
		log.WithField("err", err).Fatal("could not display record")
	}

	docli.WriteJSON(r, c.App.Writer)
}
