package images

import "github.com/digitalocean/godo"

func Delete(client *godo.Client, id int) error {
	_, err := client.Images.Delete(id)
	return err
}
