package domains

import (
	"github.com/bryanl/docli/docli"
	"github.com/digitalocean/godo"
)

// List lists all domains.
func List(client *godo.Client, opts *docli.Opts) ([]godo.Domain, error) {
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
		return nil, err
	}

	list := make([]godo.Domain, len(si))
	for i := range si {
		list[i] = si[i].(godo.Domain)
	}

	return list, nil
}
