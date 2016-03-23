package godomock

import "github.com/digitalocean/godo"

// FloatingIPsService is the godo FloatingIPsService interface.
type FloatingIPsService interface {
	List(*godo.ListOptions) ([]godo.FloatingIP, *godo.Response, error)
	Get(string) (*godo.FloatingIP, *godo.Response, error)
	Create(*godo.FloatingIPCreateRequest) (*godo.FloatingIP, *godo.Response, error)
	Delete(string) (*godo.Response, error)
}
