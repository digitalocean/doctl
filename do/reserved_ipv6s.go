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

// ReservedIP wraps a godo ReservedIP.
type ReservedIPv6 struct {
	*godo.ReservedIPV6
}

// ReservedIPv6s is a slice of ReservedIPv6.
type ReservedIPv6s []ReservedIPv6

// ReservedIPv6sService is the godo ReservedIPv6sService interface.
type ReservedIPv6sService interface {
	List() (ReservedIPv6s, error)
	Get(ip string) (*ReservedIPv6, error)
	Create(ficr *godo.ReservedIPV6CreateRequest) (*ReservedIPv6, error)
	Delete(ip string) error

	Assign(ip string, dropletID int) (*Action, error)
	Unassign(ip string) (*Action, error)
}

type reservedIPv6sService struct {
	client *godo.Client
}

var _ ReservedIPv6sService = &reservedIPv6sService{}

// NewReservedIPsService builds an instance of ReservedIPsService.
func NewReservedIPv6sService(client *godo.Client) ReservedIPv6sService {
	return &reservedIPv6sService{
		client: client,
	}
}

func (fis *reservedIPv6sService) List() (ReservedIPv6s, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := fis.client.ReservedIPV6s.List(context.TODO(), opt)
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

	list := make(ReservedIPv6s, 0, len(si))
	for _, x := range si {
		fip := x.(godo.ReservedIPV6)
		list = append(list, ReservedIPv6{ReservedIPV6: &fip})
	}

	return list, nil
}

func (fis *reservedIPv6sService) Get(ip string) (*ReservedIPv6, error) {
	fip, _, err := fis.client.ReservedIPV6s.Get(context.TODO(), ip)
	if err != nil {
		return nil, err
	}

	return &ReservedIPv6{ReservedIPV6: fip}, nil
}

func (fis *reservedIPv6sService) Create(ficr *godo.ReservedIPV6CreateRequest) (*ReservedIPv6, error) {
	fip, _, err := fis.client.ReservedIPV6s.Create(context.TODO(), ficr)
	if err != nil {
		return nil, err
	}

	return &ReservedIPv6{ReservedIPV6: fip}, nil
}

func (fis *reservedIPv6sService) Delete(ip string) error {
	_, err := fis.client.ReservedIPV6s.Delete(context.TODO(), ip)
	return err
}

func (fia *reservedIPv6sService) Assign(ip string, dropletID int) (*Action, error) {
	a, _, err := fia.client.ReservedIPV6Actions.Assign(context.TODO(), ip, dropletID)
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}

func (fia *reservedIPv6sService) Unassign(ip string) (*Action, error) {
	a, _, err := fia.client.ReservedIPV6Actions.Unassign(context.TODO(), ip)
	if err != nil {
		return nil, err
	}

	return &Action{Action: a}, nil
}
