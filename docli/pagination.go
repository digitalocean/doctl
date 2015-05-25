package docli

import "github.com/digitalocean/godo"

type Generator func(*godo.ListOptions) ([]interface{}, *godo.Response, error)

func PaginateResp(gen Generator) ([]interface{}, error) {
	opt := &godo.ListOptions{}
	list := []interface{}{}

	for {
		items, resp, err := gen(opt)
		if err != nil {
			return nil, err
		}

		for _, i := range items {
			list = append(list, i)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		opt.Page = page + 1
	}

	return list, nil

}
