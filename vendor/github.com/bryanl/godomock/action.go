package godomock

import "github.com/digitalocean/godo"

// ActionsService is the godo ActionsService interface.
type ActionsService interface {
	List(*godo.ListOptions) ([]godo.Action, *godo.Response, error)
	Get(int) (*godo.Action, *godo.Response, error)
}
