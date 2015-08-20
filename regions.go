package doit

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

// List all regions.
func RegionList(c *cli.Context) {
	client := NewClient(c, DefaultConfig)

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

	si, err := PaginateResp(f)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list regions")
	}

	list := make([]godo.Region, len(si))
	for i := range si {
		list[i] = si[i].(godo.Region)
	}

	err = DisplayOutput(list)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}

}
