package doit

import (
	"fmt"

	"github.com/digitalocean/godo"
)

func getDropletByID(client *godo.Client, id int) (*godo.Droplet, error) {
	if id < 1 {
		return nil, fmt.Errorf("missing droplet id")
	}

	droplet, _, err := client.Droplets.Get(id)
	return droplet, err
}

func listDroplets(client *godo.Client) ([]godo.Droplet, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.List(opt)
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
		return nil, err
	}

	list := make([]godo.Droplet, len(si))
	for i := range si {
		list[i] = si[i].(godo.Droplet)
	}

	return list, nil
}
