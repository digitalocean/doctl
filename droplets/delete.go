package droplets

import "github.com/digitalocean/godo"

// Delete destroy a droplet by id.
func Delete(client *godo.Client, id int) error {
	_, err := client.Droplets.Delete(id)
	if err != nil {
		return err
	}

	return nil
}
