package do

import "github.com/digitalocean/godo"

// Image is a werapper for godo.Image
type Image struct {
	*godo.Image
}

// Images is a slice of Droplet.
type Images []Image
