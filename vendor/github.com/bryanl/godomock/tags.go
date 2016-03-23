package godomock

import "github.com/digitalocean/godo"

// TagsService is the godo Tags interface.
type TagsService interface {
	List(*godo.ListOptions) ([]godo.Tag, *godo.Response, error)
	Get(string) (*godo.Tag, *godo.Response, error)
	Create(*godo.TagCreateRequest) (*godo.Tag, *godo.Response, error)
	Update(string, *godo.TagUpdateRequest) (*godo.Response, error)
	Delete(string) (*godo.Response, error)
	TagResources(string, *godo.TagResourcesRequest) (*godo.Response, error)
	UntagResources(string, *godo.UntagResourcesRequest) (*godo.Response, error)
}
