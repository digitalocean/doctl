package droplets

import (
	"fmt"

	"github.com/bryanl/docli/docli"
	"github.com/digitalocean/godo"
	"github.com/fatih/color"
)

// List returns a list of all droplets.
func List(client *godo.Client) ([]godo.Droplet, error) {
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

	si, err := docli.PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make([]godo.Droplet, len(si))
	for i := range si {
		list[i] = si[i].(godo.Droplet)
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
