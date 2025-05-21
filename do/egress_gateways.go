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

type EgressGatewaysService interface {
	Create(*godo.EgressGatewayRequest) (*godo.EgressGateway, error)
	Get(string) (*godo.EgressGateway, error)
	List() ([]*godo.EgressGateway, error)
	Update(string, *godo.EgressGatewayRequest) (*godo.EgressGateway, error)
	Delete(string) error
}

var _ EgressGatewaysService = &egressGatewayService{}

type egressGatewayService struct {
	client *godo.Client
}

func NewEgressGatewaysService(client *godo.Client) EgressGatewaysService {
	return &egressGatewayService{
		client: client,
	}
}

// Create creates a new Egress Gateway
func (e *egressGatewayService) Create(createReq *godo.EgressGatewayRequest) (*godo.EgressGateway, error) {
	gateway, _, err := e.client.EgressGateways.Create(context.Background(), createReq)
	if err != nil {
		return nil, err
	}
	return gateway, nil
}

// Get retrieves an existing Egress Gateway
func (e *egressGatewayService) Get(gatewayID string) (*godo.EgressGateway, error) {
	gateway, _, err := e.client.EgressGateways.Get(context.Background(), gatewayID)
	if err != nil {
		return nil, err
	}
	return gateway, nil
}

// List lists all existing Egress Gateways
func (e *egressGatewayService) List() ([]*godo.EgressGateway, error) {
	listResp, err := PaginateResp(func(opts *godo.ListOptions) ([]any, *godo.Response, error) {
		gateways, resp, err := e.client.EgressGateways.List(context.Background(), &godo.EgressGatewaysListOptions{
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
	gateways := make([]*godo.EgressGateway, 0, len(listResp))
	for _, pool := range listResp {
		gateways = append(gateways, pool.(*godo.EgressGateway))
	}
	return gateways, nil
}

// Update updates an existing Egress Gateway
func (e *egressGatewayService) Update(gatewayID string, updateReq *godo.EgressGatewayRequest) (*godo.EgressGateway, error) {
	gateway, _, err := e.client.EgressGateways.Update(context.Background(), gatewayID, updateReq)
	if err != nil {
		return nil, err
	}
	return gateway, nil
}

// Delete deletes an existing Egress Gateway
func (e *egressGatewayService) Delete(gatewayID string) error {
	_, err := e.client.EgressGateways.Delete(context.Background(), gatewayID)
	return err
}
