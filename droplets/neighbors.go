package droplets

import "github.com/digitalocean/godo"

// Neighbors returns a list of neighbors for a droplet.
func Neighbors(client *godo.Client, id int) ([]godo.Droplet, error) {
	list, _, err := client.Droplets.Neighbors(id)
	return list, err
}
