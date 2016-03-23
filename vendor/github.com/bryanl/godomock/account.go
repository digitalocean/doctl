package godomock

import "github.com/digitalocean/godo"

// AccountService is the godo AccountService interface.
type AccountService interface {
	Get() (*godo.Account, *godo.Response, error)
}
