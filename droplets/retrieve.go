package droplets

import "github.com/digitalocean/godo"

// Get retrieve a droplet by int.
func Get(client *godo.Client, id int) (*godo.Droplet, error) {
	root, _, err := client.Droplets.Get(id)
	if err != nil {
		return nil, err
	}

	return root.Droplet, nil
}
