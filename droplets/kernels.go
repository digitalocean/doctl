package droplets

import (
	"github.com/bryanl/docli/docli"
	"github.com/digitalocean/godo"
)

func Kernels(client *godo.Client, id int) ([]godo.Kernel, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := client.Droplets.Kernels(id, opt)
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

	list := make([]godo.Kernel, len(si))
	for i := range si {
		list[i] = si[i].(godo.Kernel)
	}

	return list, nil

}
