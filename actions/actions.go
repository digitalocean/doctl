package actions

import (
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/bryanl/docli"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func Action(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	opts := docli.LoadOpts(c)
	err := actionsList(client, opts, c.App.Writer)
	if err != nil {
		log.WithField("err", err).Fatal("could not list actions")
	}
}

func Get(c *cli.Context) {
	client := docli.NewClient(c, docli.DefaultClientSource)
	id := c.Int("action-id")
	a, _, err := client.Actions.Get(id)
	if err != nil {
		log.WithField("err", err).Fatal("could not retrieve action")
	}
	docli.WriteJSON(a, c.App.Writer)
}

func actionsList(client *godo.Client, opts *docli.Opts, w io.Writer) error {
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

	si, err := docli.PaginateResp(f, opts)
	if err != nil {
		return err
	}

	list := make([]godo.Action, len(si))
	for i := range si {
		list[i] = si[i].(godo.Action)
	}

	return docli.WriteJSON(list, w)
}
