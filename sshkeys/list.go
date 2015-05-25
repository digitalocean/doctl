package sshkeys

import "github.com/digitalocean/godo"

func List(client *godo.Client) ([]godo.Key, error) {
	list := []godo.Key{}
	opt := &godo.ListOptions{}
	for {
		keys, resp, err := client.Keys.List(opt)
		if err != nil {
			return nil, err
		}

		// append the current page's droplets to our list
		for _, d := range keys {
			list = append(list, d)

		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	return list, nil
}
