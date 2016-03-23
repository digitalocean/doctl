package godomock

import "github.com/digitalocean/godo"

// RegionsService is the godo RegionsService interface.
type RegionsService interface {
	List(*godo.ListOptions) ([]godo.Region, *godo.Response, error)
}
