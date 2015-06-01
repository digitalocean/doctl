package domainrecs

import (
	"github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
)

func Delete(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	domainName := c.String("domain-name")
	recordID := c.Int("record-id")

	_, err := client.Domains.DeleteRecord(domainName, recordID)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not delete record")
	}
}
