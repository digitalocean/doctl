package domains

import (
	log "github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func Create(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	req := &godo.DomainCreateRequest{
		Name:      c.String("domain-name"),
		IPAddress: c.String("ip-address"),
	}

	d, _, err := client.Domains.Create(req)
	if err != nil {
		log.WithField("err", err).Fatal("could not create domain")
	}
	docli.WriteJSON(d, c.App.Writer)
}
