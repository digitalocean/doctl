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

type VPCNATGatewaysService interface {
	Create(*godo.VPCNATGatewayRequest) (*godo.VPCNATGateway, error)
	Get(string) (*godo.VPCNATGateway, error)
	List() ([]*godo.VPCNATGateway, error)
	Update(string, *godo.VPCNATGatewayRequest) (*godo.VPCNATGateway, error)
	Delete(string) error
}

var _ VPCNATGatewaysService = &VPCNATGatewayService{}

type VPCNATGatewayService struct {
	client *godo.Client
}

func NewVPCNATGatewaysService(client *godo.Client) VPCNATGatewaysService {
	return &VPCNATGatewayService{
		client: client,
	}
}

// Create creates a new VPC NAT Gateway
func (e *VPCNATGatewayService) Create(createReq *godo.VPCNATGatewayRequest) (*godo.VPCNATGateway, error) {
	gateway, _, err := e.client.VPCNATGateways.Create(context.Background(), createReq)
	if err != nil {
		return nil, err
	}
	return gateway, nil
}

// Get retrieves an existing VPC NAT Gateway
func (e *VPCNATGatewayService) Get(gatewayID string) (*godo.VPCNATGateway, error) {
	gateway, _, err := e.client.VPCNATGateways.Get(context.Background(), gatewayID)
	if err != nil {
		return nil, err
	}
	return gateway, nil
}

// List lists all existing VPC NAT Gateways
func (e *VPCNATGatewayService) List() ([]*godo.VPCNATGateway, error) {
	listResp, err := PaginateResp(func(opts *godo.ListOptions) ([]any, *godo.Response, error) {
		gateways, resp, err := e.client.VPCNATGateways.List(context.Background(), &godo.VPCNATGatewaysListOptions{
			ListOptions: *opts,
		})
		if err != nil {
			return nil, nil, err
		}
		anyResp := make([]any, 0, len(gateways))
		for _, gateway := range gateways {
			anyResp = append(anyResp, gateway)
		}
		return anyResp, resp, err
	})
	if err != nil {
		return nil, err
	}
	gateways := make([]*godo.VPCNATGateway, 0, len(listResp))
	for _, pool := range listResp {
		gateways = append(gateways, pool.(*godo.VPCNATGateway))
	}
	return gateways, nil
}

// Update updates an existing VPC NAT Gateway
func (e *VPCNATGatewayService) Update(gatewayID string, updateReq *godo.VPCNATGatewayRequest) (*godo.VPCNATGateway, error) {
	gateway, _, err := e.client.VPCNATGateways.Update(context.Background(), gatewayID, updateReq)
	if err != nil {
		return nil, err
	}
	return gateway, nil
}

// Delete deletes an existing VPC NAT Gateway
func (e *VPCNATGatewayService) Delete(gatewayID string) error {
	_, err := e.client.VPCNATGateways.Delete(context.Background(), gatewayID)
	return err
}
