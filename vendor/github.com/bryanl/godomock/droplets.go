package godomock

import "github.com/digitalocean/godo"

// DropletsService is the godo DropletsService interface.
type DropletsService interface {
	List(*godo.ListOptions) ([]godo.Droplet, *godo.Response, error)
	ListByTag(string, *godo.ListOptions) ([]godo.Droplet, *godo.Response, error)
	Get(int) (*godo.Droplet, *godo.Response, error)
	Create(*godo.DropletCreateRequest) (*godo.Droplet, *godo.Response, error)
	CreateMultiple(*godo.DropletMultiCreateRequest) ([]godo.Droplet, *godo.Response, error)
	Delete(int) (*godo.Response, error)
	DeleteByTag(string) (*godo.Response, error)
	Kernels(int, *godo.ListOptions) ([]godo.Kernel, *godo.Response, error)
	Snapshots(int, *godo.ListOptions) ([]godo.Image, *godo.Response, error)
	Backups(int, *godo.ListOptions) ([]godo.Image, *godo.Response, error)
	Actions(int, *godo.ListOptions) ([]godo.Action, *godo.Response, error)
	Neighbors(int) ([]godo.Droplet, *godo.Response, error)
}
