package docli

import (
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/digitalocean/godo"
)

func ActionList(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	opts := LoadOpts(c)
	err := actionsList(client, opts, c.App.Writer)
	if err != nil {
		log.WithField("err", err).Fatal("could not list actions")
	}
}

func ActionGet(c *cli.Context) {
	client := NewClient(c, DefaultClientSource)
	id := c.Int("action-id")
	a, _, err := client.Actions.Get(id)
	if err != nil {
		log.WithField("err", err).Fatal("could not retrieve action")
	}
	WriteJSON(a, c.App.Writer)
}

func actionsList(client *godo.Client, opts *Opts, w io.Writer) error {
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

	return WriteJSON(list, w)
}
