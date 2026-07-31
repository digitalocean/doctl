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

// MicroDroplet wraps a godo.MicroDroplet.
type MicroDroplet struct {
	*godo.MicroDroplet
}

// MicroDroplets is a slice of MicroDroplet.
type MicroDroplets []MicroDroplet

// MicroDropletCheckpoint wraps a godo.MicroDropletCheckpoint.
type MicroDropletCheckpoint struct {
	*godo.MicroDropletCheckpoint
}

// MicroDropletCheckpoints is a slice of MicroDropletCheckpoint.
type MicroDropletCheckpoints []MicroDropletCheckpoint

// MicroDropletsService is an interface for interacting with DigitalOcean's
// MicroDroplet API.
type MicroDropletsService interface {
	List() (MicroDroplets, error)
	ListByRegion(region string) (MicroDroplets, error)
	ListByName(name string) (MicroDroplets, error)
	Get(id string) (*MicroDroplet, error)
	Create(req *godo.MicroDropletCreateRequest) (*MicroDroplet, error)
	Pause(id string) (*MicroDroplet, error)
	Resume(id string) (*MicroDroplet, error)
	Delete(id string) error
	ListCheckpoints(id string) (MicroDropletCheckpoints, error)
}

type microDropletsService struct {
	client *godo.Client
}

var _ MicroDropletsService = &microDropletsService{}

// NewMicroDropletsService builds a MicroDropletsService backed by the provided
// godo client.
func NewMicroDropletsService(client *godo.Client) MicroDropletsService {
	return &microDropletsService{client: client}
}

func (s *microDropletsService) List() (MicroDroplets, error) {
	return s.paginate(func(opt *godo.ListOptions) ([]godo.MicroDroplet, *godo.Response, error) {
		return s.client.MicroDroplets.List(context.TODO(), opt)
	})
}

func (s *microDropletsService) ListByRegion(region string) (MicroDroplets, error) {
	return s.paginate(func(opt *godo.ListOptions) ([]godo.MicroDroplet, *godo.Response, error) {
		return s.client.MicroDroplets.ListByRegion(context.TODO(), region, opt)
	})
}

func (s *microDropletsService) ListByName(name string) (MicroDroplets, error) {
	return s.paginate(func(opt *godo.ListOptions) ([]godo.MicroDroplet, *godo.Response, error) {
		return s.client.MicroDroplets.ListByName(context.TODO(), name, opt)
	})
}

func (s *microDropletsService) paginate(lister func(*godo.ListOptions) ([]godo.MicroDroplet, *godo.Response, error)) (MicroDroplets, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := lister(opt)
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

	list := make(MicroDroplets, len(si))
	for i := range si {
		md := si[i].(godo.MicroDroplet)
		list[i] = MicroDroplet{MicroDroplet: &md}
	}
	return list, nil
}

func (s *microDropletsService) Get(id string) (*MicroDroplet, error) {
	md, _, err := s.client.MicroDroplets.Get(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &MicroDroplet{MicroDroplet: md}, nil
}

func (s *microDropletsService) Create(req *godo.MicroDropletCreateRequest) (*MicroDroplet, error) {
	md, _, err := s.client.MicroDroplets.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &MicroDroplet{MicroDroplet: md}, nil
}

func (s *microDropletsService) Pause(id string) (*MicroDroplet, error) {
	md, _, err := s.client.MicroDroplets.Pause(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &MicroDroplet{MicroDroplet: md}, nil
}

func (s *microDropletsService) Resume(id string) (*MicroDroplet, error) {
	md, _, err := s.client.MicroDroplets.Resume(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &MicroDroplet{MicroDroplet: md}, nil
}

func (s *microDropletsService) Delete(id string) error {
	_, err := s.client.MicroDroplets.Delete(context.TODO(), id)
	return err
}

func (s *microDropletsService) ListCheckpoints(id string) (MicroDropletCheckpoints, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.MicroDroplets.ListCheckpoints(context.TODO(), id, opt)
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

	list := make(MicroDropletCheckpoints, len(si))
	for i := range si {
		cp := si[i].(godo.MicroDropletCheckpoint)
		list[i] = MicroDropletCheckpoint{MicroDropletCheckpoint: &cp}
	}
	return list, nil
}
