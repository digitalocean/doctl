package godomock

import "github.com/digitalocean/godo"

// ImagesService is the godo ImagesService interface.
type ImagesService interface {
	List(*godo.ListOptions) ([]godo.Image, *godo.Response, error)
	ListDistribution(opt *godo.ListOptions) ([]godo.Image, *godo.Response, error)
	ListApplication(opt *godo.ListOptions) ([]godo.Image, *godo.Response, error)
	ListUser(opt *godo.ListOptions) ([]godo.Image, *godo.Response, error)
	GetByID(int) (*godo.Image, *godo.Response, error)
	GetBySlug(string) (*godo.Image, *godo.Response, error)
	Update(int, *godo.ImageUpdateRequest) (*godo.Image, *godo.Response, error)
	Delete(int) (*godo.Response, error)
}
