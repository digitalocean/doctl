package domains

import (
	"github.com/Sirupsen/logrus"
	"github.com/bryanl/docli"
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
		logrus.WithField("err", err).Fatal("could not create domain")
	}
	docli.WriteJSON(d, c.App.Writer)
}

func Delete(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	name := c.String("domain-name")
	_, err := client.Domains.Delete(name)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not delete account")
	}
}

// List lists all domains.
func List(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	opts := docli.LoadOpts(c)

	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Domains.List(opt)
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
		logrus.WithField("err", err).Fatal("could not list domains")
	}

	list := make([]godo.Domain, len(si))
	for i := range si {
		list[i] = si[i].(godo.Domain)
	}

	docli.WriteJSON(list, c.App.Writer)
}

func Get(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.String("domain-name")
	a, _, err := client.Domains.Get(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not retrieve domain")
	}
	docli.WriteJSON(a, c.App.Writer)
}
