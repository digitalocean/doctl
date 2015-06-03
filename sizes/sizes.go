package sizes

import (
	"github.com/Sirupsen/logrus"
	"github.com/bryanl/docli/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

// List all sizes.
func List(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	opts := docli.LoadOpts(c)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Sizes.List(opt)
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
		logrus.WithField("err", err).Fatal("could not list sizes")
	}

	list := make([]godo.Size, len(si))
	for i := range si {
		list[i] = si[i].(godo.Size)
	}

	err = docli.WriteJSON(list, c.App.Writer)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write JSON")
	}

}
