package main

import (
	"fmt"

	"github.com/digitalocean/doctl/Godeps/_workspace/src/github.com/digitalocean/godo"
)

func FindDropletByName(c *godo.Client, name string) (*godo.Droplet, error) {
	opt := &godo.ListOptions{}
	for {
		dropletPage, resp, err := c.Droplets.List(opt)
		if err != nil {
			return nil, err
		}

		// append the current page's droplets to our list
		for _, d := range dropletPage {
			if d.Name == name {
				return &d, nil
			}
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			return nil, fmt.Errorf("%q not found.", name)
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}
}

func FindKeyByName(c *godo.Client, name string) (*godo.Key, error) {
	opt := &godo.ListOptions{}
	for {
		keyPage, resp, err := c.Keys.List(opt)
		if err != nil {
			return nil, err
		}

		// append the current page's keys to our list
		for _, k := range keyPage {
			if k.Name == name {
				return &k, nil
			}
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			return nil, fmt.Errorf("%q not found.", name)
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}
}

func PublicIPForDroplet(d *godo.Droplet) string {
	var publicIP string
	for _, network := range d.Networks.V4 {
		if network.Type == "public" {
			publicIP = network.IPAddress
		}
	}
	return publicIP
}
