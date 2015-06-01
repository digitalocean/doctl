package actions

import (
	"github.com/bryanl/docli/docli"
	"github.com/digitalocean/godo"
)

// List lists all actions.
func List3(client *godo.Client, opts *docli.Opts) ([]godo.Action, error) {
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
		return nil, err
	}

	list := make([]godo.Action, len(si))
	for i := range si {
		list[i] = si[i].(godo.Action)
	}

	return list, nil
}

func List(client *godo.Client, opts *docli.Opts) ([]godo.Action, error) {
	fn := func(opt *godo.ListOptions) ([]godo.Action, *godo.Response, error) {
		return client.Actions.List(opt)
	}

	actions, err := docli.PaginageResp2(fn, opts)
	if err != nil {
		return nil, err
	}
	return actions.([]godo.Action), nil
}
