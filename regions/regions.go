package regions

import (
	"github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

// List all regions.
func List(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	opts := docli.LoadOpts(c)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Regions.List(opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := docli.PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list regions")
	}

	list := make([]godo.Region, len(si))
	for i := range si {
		list[i] = si[i].(godo.Region)
	}

	err = docli.WriteJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}

}
