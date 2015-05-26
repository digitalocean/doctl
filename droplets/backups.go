package droplets

import (
	"github.com/bryanl/docli/docli"
	"github.com/digitalocean/godo"
)

// Backups returns a list of backup images for a droplet.
func Backups(client *godo.Client, id int) ([]godo.Image, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Backups(id, opt)
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

	list := make([]godo.Image, len(si))
	for i := range si {
		list[i] = si[i].(godo.Image)
	}

	return list, nil

}
