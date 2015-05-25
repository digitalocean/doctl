package droplets

import (
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/fatih/color"
)

// List returns a list of all droplets.
func List(client *godo.Client) ([]godo.Droplet, error) {
	list := []godo.Droplet{}
	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(opt)
		if err != nil {
			return nil, err
		}

		// append the current page's droplets to our list
		for _, d := range droplets {
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

// PublicIP extracts the public IP from a droplet.
func PublicIP(droplet *godo.Droplet) string {
	var ip string
	for _, v4 := range droplet.Networks.V4 {
		if v4.Type == "public" {
			ip = v4.IPAddress
		}
	}

	return ip
}

// ToText converts a droplet to text.
func ToText(d *godo.Droplet) string {
	return fmt.Sprintf("%s (ip: %s, status: %s, region: %s, id: %d)",
		d.Name, PublicIP(d), status(d), d.Region.Slug, d.ID)
}

func status(d *godo.Droplet) string {
	if d.Status == "active" {
		return color.New(color.FgGreen).SprintFunc()(d.Status)
	}

	return color.New(color.FgRed).SprintFunc()(d.Status)

}
