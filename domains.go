package doit

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func DomainCreate(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	req := &godo.DomainCreateRequest{
		Name:      c.String("domain-name"),
		IPAddress: c.String("ip-address"),
	}

	d, _, err := client.Domains.Create(req)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not create domain")
	}
	writeJSON(d, c.App.Writer)
}

func DomainDelete(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	name := c.String("domain-name")
	_, err := client.Domains.Delete(name)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not delete account")
	}
}

// List lists all domains.
func DomainList(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	opts := LoadOpts(c)

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

	si, err := PaginateResp(f, opts)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list domains")
	}

	list := make([]godo.Domain, len(si))
	for i := range si {
		list[i] = si[i].(godo.Domain)
	}

	writeJSON(list, c.App.Writer)
}

func DomainGet(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.String("domain-name")
	d, _, err := client.Domains.Get(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not retrieve domain")
	}

	err = displayOutput(c, d)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}

}
