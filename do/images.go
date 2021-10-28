/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// Image is a wrapper for godo.Image
type Image struct {
	*godo.Image
}

// Images is a slice of Droplet.
type Images []Image

// ImagesService is the godo ImagesService interface.
type ImagesService interface {
	List(public bool) (Images, error)
	ListDistribution(public bool) (Images, error)
	ListApplication(public bool) (Images, error)
	ListUser(public bool) (Images, error)
	GetByID(id int) (*Image, error)
	GetBySlug(slug string) (*Image, error)
	Update(id int, iur *godo.ImageUpdateRequest) (*Image, error)
	Delete(id int) error
	Create(icr *godo.CustomImageCreateRequest) (*Image, error)
}

type imagesService struct {
	client *godo.Client
}

var _ ImagesService = &imagesService{}

// NewImagesService builds an instance of ImagesService.
func NewImagesService(client *godo.Client) ImagesService {
	return &imagesService{
		client: client,
	}
}

func (is *imagesService) List(public bool) (Images, error) {
	return is.listImages(is.client.Images.List, public)
}

func (is *imagesService) ListDistribution(public bool) (Images, error) {
	return is.listImages(is.client.Images.ListDistribution, public)
}

func (is *imagesService) ListApplication(public bool) (Images, error) {
	return is.listImages(is.client.Images.ListApplication, public)
}

func (is *imagesService) ListUser(public bool) (Images, error) {
	return is.listImages(is.client.Images.ListUser, public)
}

func (is *imagesService) GetByID(id int) (*Image, error) {
	i, _, err := is.client.Images.GetByID(context.TODO(), id)
	if err != nil {
		return nil, err
	}

	return &Image{Image: i}, nil
}

func (is *imagesService) GetBySlug(slug string) (*Image, error) {
	i, _, err := is.client.Images.GetBySlug(context.TODO(), slug)
	if err != nil {
		return nil, err
	}

	return &Image{Image: i}, nil
}

func (is *imagesService) Update(id int, iur *godo.ImageUpdateRequest) (*Image, error) {
	i, _, err := is.client.Images.Update(context.TODO(), id, iur)
	if err != nil {
		return nil, err
	}

	return &Image{Image: i}, nil
}

func (is *imagesService) Delete(id int) error {
	_, err := is.client.Images.Delete(context.TODO(), id)
	return err
}

func (is *imagesService) Create(icr *godo.CustomImageCreateRequest) (*Image, error) {
	i, _, err := is.client.Images.Create(context.TODO(), icr)
	if err != nil {
		return nil, err
	}

	return &Image{Image: i}, nil
}

type listFn func(context.Context, *godo.ListOptions) ([]godo.Image, *godo.Response, error)

func (is *imagesService) listImages(lFn listFn, public bool) (Images, error) {
	fn := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := lFn(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := []interface{}{}
		for _, i := range list {
			if (public && i.Public) || !i.Public {
				si = append(si, i)
			}
		}

		return si, resp, err
	}

	si, err := PaginateResp(fn)
	if err != nil {
		return nil, err
	}

	list := make(Images, 0, len(si))
	for i := range si {
		image := si[i].(godo.Image)
		list = append(list, Image{Image: &image})
	}

	return list, nil
}
