package regions

import (
	"github.com/bryanl/docli/docli"
	"github.com/digitalocean/godo"
)

// List all regions.
func List(client *godo.Client) ([]godo.Region, error) {
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

	si, err := docli.PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]godo.Region, len(si))
	for i := range si {
		list[i] = si[i].(godo.Region)
	}

	return list, nil
}
