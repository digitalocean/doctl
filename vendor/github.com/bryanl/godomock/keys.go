package godomock

import "github.com/digitalocean/godo"

// KeysService is the godo KeysService interface.
type KeysService interface {
	List(*godo.ListOptions) ([]godo.Key, *godo.Response, error)
	GetByID(int) (*godo.Key, *godo.Response, error)
	GetByFingerprint(string) (*godo.Key, *godo.Response, error)
	Create(*godo.KeyCreateRequest) (*godo.Key, *godo.Response, error)
	UpdateByID(int, *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error)
	UpdateByFingerprint(string, *godo.KeyUpdateRequest) (*godo.Key, *godo.Response, error)
	DeleteByID(int) (*godo.Response, error)
	DeleteByFingerprint(string) (*godo.Response, error)
}
