package doit

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

// List all sizes.
func SizeList(c *cli.Context) {
	client := NewClient(c, DefaultConfig)

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

	si, err := PaginateResp(f)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list sizes")
	}

	list := make([]godo.Size, len(si))
	for i := range si {
		list[i] = si[i].(godo.Size)
	}

	err = DisplayOutput(c, list)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}
