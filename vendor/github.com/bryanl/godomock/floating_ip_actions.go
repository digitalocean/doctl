package godomock

import "github.com/digitalocean/godo"

// FloatingIPActionsService is the godo FloatingIPsAction interface.
type FloatingIPActionsService interface {
	Assign(ip string, dropletID int) (*godo.Action, *godo.Response, error)
	Unassign(ip string) (*godo.Action, *godo.Response, error)
	Get(ip string, actionID int) (*godo.Action, *godo.Response, error)
	List(ip string, opt *godo.ListOptions) ([]godo.Action, *godo.Response, error)
}
