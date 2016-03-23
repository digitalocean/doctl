package godomock

import "github.com/digitalocean/godo"

// ImageActionsService is the ImageActionsService interface.
type ImageActionsService interface {
	Get(int, int) (*godo.Action, *godo.Response, error)
	Transfer(int, *godo.ActionRequest) (*godo.Action, *godo.Response, error)
}
