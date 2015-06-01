package domainrecs

import (
	"github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func Create(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	domainName := c.String("domain-name")

	drcr := &godo.DomainRecordEditRequest{
		Type:     c.String("record-type"),
		Name:     c.String("record-name"),
		Data:     c.String("record-data"),
		Priority: c.Int("record-priority"),
		Port:     c.Int("record-port"),
		Weight:   c.Int("record-weight"),
	}

	r, _, err := client.Domains.CreateRecord(domainName, drcr)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create record")
	}

	docli.WriteJSON(r, c.App.Writer)
}
