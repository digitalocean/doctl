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

// ReservedIP wraps a godo ReservedIP.
type ReservedIP struct {
	*godo.ReservedIP
}

// ReservedIPs is a slice of ReservedIP.
type ReservedIPs []ReservedIP

// ReservedIPsService is the godo ReservedIPsService interface.
type ReservedIPsService interface {
	List() (ReservedIPs, error)
	Get(ip string) (*ReservedIP, error)
	Create(ficr *godo.ReservedIPCreateRequest) (*ReservedIP, error)
	Delete(ip string) error
}

type reservedIPsService struct {
	client *godo.Client
}

var _ ReservedIPsService = &reservedIPsService{}

// NewReservedIPsService builds an instance of ReservedIPsService.
func NewReservedIPsService(client *godo.Client) ReservedIPsService {
	return &reservedIPsService{
		client: client,
	}
}

func (fis *reservedIPsService) List() (ReservedIPs, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := fis.client.ReservedIPs.List(context.TODO(), opt)
		if err != nil {
			return nil, nil, err
		}

		si := make([]interface{}, len(list))
		for i := range list {
			si[i] = list[i]
		}

		return si, resp, err
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(ReservedIPs, 0, len(si))
	for _, x := range si {
		fip := x.(godo.ReservedIP)
		list = append(list, ReservedIP{ReservedIP: &fip})
	}

	return list, nil
}

func (fis *reservedIPsService) Get(ip string) (*ReservedIP, error) {
	fip, _, err := fis.client.ReservedIPs.Get(context.TODO(), ip)
	if err != nil {
		return nil, err
	}

	return &ReservedIP{ReservedIP: fip}, nil
}

func (fis *reservedIPsService) Create(ficr *godo.ReservedIPCreateRequest) (*ReservedIP, error) {
	fip, _, err := fis.client.ReservedIPs.Create(context.TODO(), ficr)
	if err != nil {
		return nil, err
	}

	return &ReservedIP{ReservedIP: fip}, nil
}

func (fis *reservedIPsService) Delete(ip string) error {
	_, err := fis.client.ReservedIPs.Delete(context.TODO(), ip)
	return err
}
