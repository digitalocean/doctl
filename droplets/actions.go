package droplets

import (
	"github.com/bryanl/docli/docli"
	"github.com/digitalocean/godo"
)

// Actions returns a list of actions for a droplet.
func Actions(client *godo.Client, opts *docli.Opts, id int) ([]godo.Action, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Actions(id, opt)
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
		return nil, err
	}

	list := make([]godo.Action, len(si))
	for i := range si {
		list[i] = si[i].(godo.Action)
	}

	return list, nil

}
