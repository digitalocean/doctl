package doit

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func ActionList(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	opts := LoadOpts(c)
	err := actionsList(client, opts, c)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not list actions")
	}
}

func ActionGet(c *cli.Context) {
	client := NewClient(c, DefaultConfig)
	id := c.Int("action-id")

	if id < 1 {
		Bail(fmt.Errorf("missing action id"), "could not retrieve action")
	}

	a, _, err := client.Actions.Get(id)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not retrieve action")
	}

	err = displayOutput(c, a)
	if err != nil {
		logrus.WithField("err", err).Fatal("could not write output")
	}
}

func actionsList(client *godo.Client, opts *Opts, c *cli.Context) error {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Actions.List(opt)
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
		return err
	}

	list := make([]godo.Action, len(si))
	for i := range si {
		list[i] = si[i].(godo.Action)
	}

	return displayOutput(c, list)
}
