package images

import "github.com/digitalocean/godo"

func GetByID(client *godo.Client, id int) (*godo.Image, error) {
	image, _, err := client.Images.GetByID(id)
	if err != nil {
		return nil, err
	}

	return image, err
}

func GetBySlug(client *godo.Client, slug string) (*godo.Image, error) {
	image, _, err := client.Images.GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	return image, err
}
