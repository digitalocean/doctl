package godomock

import "github.com/digitalocean/godo"

// SizesService is the godo SizesService interface.
type SizesService interface {
	List(*godo.ListOptions) ([]godo.Size, *godo.Response, error)
}
