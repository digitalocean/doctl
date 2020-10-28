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

// VPC wraps a godo VPC.
type VPC struct {
	*godo.VPC
}

// VPCs is a slice of VPC.
type VPCs []VPC

// VPCsService is the godo VPCsService interface.
type VPCsService interface {
	Get(vpcUUID string) (*VPC, error)
	List() (VPCs, error)
	Create(vpcr *godo.VPCCreateRequest) (*VPC, error)
	Update(vpcUUID string, vpcr *godo.VPCUpdateRequest) (*VPC, error)
	PartialUpdate(vpcUUID string, options ...godo.VPCSetField) (*VPC, error)
	Delete(vpcUUID string) error
}

var _ VPCsService = &vpcsService{}

type vpcsService struct {
	client *godo.Client
}

// NewVPCsService builds an instance of VPCsService.
func NewVPCsService(client *godo.Client) VPCsService {
	return &vpcsService{
		client: client,
	}
}

func (v *vpcsService) Get(vpcUUID string) (*VPC, error) {
	vpc, _, err := v.client.VPCs.Get(context.TODO(), vpcUUID)
	if err != nil {
		return nil, err
	}

	return &VPC{VPC: vpc}, nil
}

func (v *vpcsService) List() (VPCs, error) {
	f := func(opt *godo.ListOptions) ([]interface{}, *godo.Response, error) {
		list, resp, err := v.client.VPCs.List(context.TODO(), opt)
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

	list := make([]VPC, len(si))
	for i := range si {
		a := si[i].(*godo.VPC)
		list[i] = VPC{VPC: a}
	}

	return list, nil
}

func (v *vpcsService) Create(vpcr *godo.VPCCreateRequest) (*VPC, error) {
	vpc, _, err := v.client.VPCs.Create(context.TODO(), vpcr)
	if err != nil {
		return nil, err
	}

	return &VPC{VPC: vpc}, nil
}

func (v *vpcsService) Update(vpcUUID string, vpcr *godo.VPCUpdateRequest) (*VPC, error) {
	vpc, _, err := v.client.VPCs.Update(context.TODO(), vpcUUID, vpcr)
	if err != nil {
		return nil, err
	}

	return &VPC{VPC: vpc}, nil
}

func (v *vpcsService) PartialUpdate(vpcUUID string, options ...godo.VPCSetField) (*VPC, error) {
	vpc, _, err := v.client.VPCs.Set(context.TODO(), vpcUUID, options...)
	if err != nil {
		return nil, err
	}

	return &VPC{VPC: vpc}, nil
}

func (v *vpcsService) Delete(vpcUUID string) error {
	_, err := v.client.VPCs.Delete(context.TODO(), vpcUUID)
	return err
}
