/*
Copyright 2024 The Doctl Authors All rights reserved.
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

// BYOIPPrefix wraps a godo BYOIPPrefix.
type BYOIPPrefix struct {
	BYOIPPrefix *godo.BYOIPPrefix
}

type BYOIPPrefixCreate struct {
	*godo.BYOIPPrefixCreateResp
}

// BYOIPPrefixResource wraps a godo BYOIPPrefixResource.
type BYOIPPrefixResource struct {
	BYOIPPrefixResource *godo.BYOIPPrefixResource
}

// BYOIPPrefixes is a slice of BYOIPPrefix.
type BYOIPPrefixes []BYOIPPrefix

// BYOIPPrefixResources is a slice of BYOIPPrefixResources.
type BYOIPPrefixResources []BYOIPPrefixResource

// BYOIPPrefixesService is the godo BYOIPPrefixesService interface.
type BYOIPPrefixsService interface {
	List() (BYOIPPrefixes, error)
	Get(prefixUUID string) (*BYOIPPrefix, error)
	Create(ficr *godo.BYOIPPrefixCreateReq) (*godo.BYOIPPrefixCreateResp, error)
	Delete(prefixUUID string) error

	GetPrefixResources(prefixUUID string) (BYOIPPrefixResources, error)
}

type byoipPrefixService struct {
	client *godo.Client
}

var _ BYOIPPrefixsService = &byoipPrefixService{}

// NewBYOIPPrefixService builds an instance of BYOIPPrefixsService.
func NewBYOIPPrefixService(client *godo.Client) BYOIPPrefixsService {
	return &byoipPrefixService{
		client: client,
	}
}

func (bps *byoipPrefixService) List() (BYOIPPrefixes, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := bps.client.BYOIPPrefixes.List(context.TODO(), opt)
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

	list := make(BYOIPPrefixes, 0, len(si))
	for _, x := range si {
		bp := x.(*godo.BYOIPPrefix)
		list = append(list, BYOIPPrefix{BYOIPPrefix: bp})
	}

	return list, nil
}

func (bps *byoipPrefixService) Get(prefixUUID string) (*BYOIPPrefix, error) {
	byoipPrefix, _, err := bps.client.BYOIPPrefixes.Get(context.TODO(), prefixUUID)
	if err != nil {
		return nil, err
	}

	return &BYOIPPrefix{BYOIPPrefix: byoipPrefix}, nil
}

func (bps *byoipPrefixService) Create(bpcr *godo.BYOIPPrefixCreateReq) (*godo.BYOIPPrefixCreateResp, error) {
	prefixCreateResp, _, err := bps.client.BYOIPPrefixes.Create(context.TODO(), bpcr)
	if err != nil {
		return nil, err
	}

	return prefixCreateResp, nil
}

func (fis *byoipPrefixService) Delete(prefixUUID string) error {
	_, err := fis.client.BYOIPPrefixes.Delete(context.TODO(), prefixUUID)
	return err
}

func (bps *byoipPrefixService) GetPrefixResources(prefixUUID string) (BYOIPPrefixResources, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := bps.client.BYOIPPrefixes.GetResources(context.TODO(), prefixUUID, opt)
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

	list := make(BYOIPPrefixResources, 0, len(si))
	for _, x := range si {
		byoipPrefixResource := x.(godo.BYOIPPrefixResource)
		list = append(list, BYOIPPrefixResource{BYOIPPrefixResource: &byoipPrefixResource})
	}

	return list, nil
}
