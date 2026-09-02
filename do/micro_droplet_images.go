/*
Copyright 2025 The Doctl Authors All rights reserved.
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

// MicroDropletImage wraps a godo.MicroDropletImage.
type MicroDropletImage struct {
	*godo.MicroDropletImage
}

// MicroDropletImages is a slice of MicroDropletImage.
type MicroDropletImages []MicroDropletImage

// MicroDropletImagesService is an interface for interacting with DigitalOcean's
// MicroDroplet image API.
type MicroDropletImagesService interface {
	List() (MicroDropletImages, error)
	Get(id string) (*MicroDropletImage, error)
	Create(req *godo.MicroDropletImageCreateRequest) (*MicroDropletImage, error)
	Delete(id string) error
}

type microDropletImagesService struct {
	client *godo.Client
}

var _ MicroDropletImagesService = &microDropletImagesService{}

// NewMicroDropletImagesService builds a MicroDropletImagesService backed by
// the provided godo client.
func NewMicroDropletImagesService(client *godo.Client) MicroDropletImagesService {
	return &microDropletImagesService{client: client}
}

func (s *microDropletImagesService) List() (MicroDropletImages, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.MicroDropletImages.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]any, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(MicroDropletImages, len(si))
	for i := range si {
		img := si[i].(godo.MicroDropletImage)
		list[i] = MicroDropletImage{MicroDropletImage: &img}
	}
	return list, nil
}

func (s *microDropletImagesService) Get(id string) (*MicroDropletImage, error) {
	img, _, err := s.client.MicroDropletImages.Get(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &MicroDropletImage{MicroDropletImage: img}, nil
}

func (s *microDropletImagesService) Create(req *godo.MicroDropletImageCreateRequest) (*MicroDropletImage, error) {
	img, _, err := s.client.MicroDropletImages.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &MicroDropletImage{MicroDropletImage: img}, nil
}

func (s *microDropletImagesService) Delete(id string) error {
	_, err := s.client.MicroDropletImages.Delete(context.TODO(), id)
	return err
}
