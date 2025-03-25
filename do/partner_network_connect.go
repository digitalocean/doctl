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

// PartnerNetworkConnect wraps a godo PartnerNetworkConnect.
type PartnerNetworkConnect struct {
	PartnerNetworkConnect *godo.PartnerAttachment
}

// PartnerNetworkConnects is a slice of PartnerNetworkConnect.
type PartnerNetworkConnects []PartnerNetworkConnect

// PartnerNetworkConnectRoute wraps a godo RemoteRoute.
type PartnerNetworkConnectRoute struct {
	*godo.RemoteRoute
}

// PartnerNetworkConnectRoutes is a slice of PartnerNetworkConnectRoute.
type PartnerNetworkConnectRoutes []PartnerNetworkConnectRoute

// PartnerNetworkConnectBGPAuthKey wrap a godo BgpAuthKey.
type PartnerNetworkConnectBGPAuthKey struct {
	*godo.BgpAuthKey
}

// PartnerNetworkConnectRegenerateServiceKey wraps a godo RegenerateServiceKey.
type PartnerNetworkConnectRegenerateServiceKey struct {
	*godo.RegenerateServiceKey
}

// PartnerNetworkConnectServiceKey wraps a godo ServiceKey.
type PartnerNetworkConnectServiceKey struct {
	*godo.ServiceKey
}

// PartnerNetworkConnectsService is an interface for interacting with
// DigitalOcean's partner network connect api.
type PartnerNetworkConnectsService interface {
	Create(*godo.PartnerNetworkConnectCreateRequest) (*PartnerNetworkConnect, error)
	GetPartnerNetworkConnect(pncID string) (*PartnerNetworkConnect, error)
	ListPartnerNetworkConnects() (PartnerNetworkConnects, error)
	DeletePartnerNetworkConnect(pncID string) error
	UpdatePartnerNetworkConnect(pncID string, req *godo.PartnerNetworkConnectUpdateRequest) (*PartnerNetworkConnect, error)
	ListPartnerNetworkConnectRoutes(pncID string) (PartnerNetworkConnectRoutes, error)
	GetBGPAuthKey(pncID string) (*PartnerNetworkConnectBGPAuthKey, error)
	RegenerateServiceKey(pncID string) (*PartnerNetworkConnectRegenerateServiceKey, error)
	GetServiceKey(pncID string) (*PartnerNetworkConnectServiceKey, error)
}

var _ PartnerNetworkConnectsService = &partnerNetworkConnectsService{}

type partnerNetworkConnectsService struct {
	client *godo.Client
}

// NewPartnerNetworkConnectsService builds an instance of PartnerNetworkConnectsService.
func NewPartnerNetworkConnectsService(client *godo.Client) PartnerNetworkConnectsService {
	return &partnerNetworkConnectsService{
		client: client,
	}
}

// Create creates a partner connect attachment.
func (p *partnerNetworkConnectsService) Create(req *godo.PartnerNetworkConnectCreateRequest) (*PartnerNetworkConnect, error) {
	pia, _, err := p.client.PartnerNetworkConnect.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &PartnerNetworkConnect{PartnerNetworkConnect: pia}, nil
}

// GetPartnerNetworkConnect retrieves a partner connect attachment.
func (p *partnerNetworkConnectsService) GetPartnerNetworkConnect(iaID string) (*PartnerNetworkConnect, error) {
	partnerIA, _, err := p.client.PartnerNetworkConnect.Get(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerNetworkConnect{PartnerNetworkConnect: partnerIA}, nil
}

// ListPartnerNetworkConnects lists all partner connect attachments.
func (p *partnerNetworkConnectsService) ListPartnerNetworkConnects() (PartnerNetworkConnects, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := p.client.PartnerNetworkConnect.List(context.TODO(), opt)
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

	list := make([]PartnerNetworkConnect, len(si))
	for i := range si {
		a := si[i].(*godo.PartnerAttachment)
		list[i] = PartnerNetworkConnect{PartnerNetworkConnect: a}
	}

	return list, nil
}

// DeletePartnerNetworkConnect deletes a partner connect attachment.
func (p *partnerNetworkConnectsService) DeletePartnerNetworkConnect(iaID string) error {
	_, err := p.client.PartnerNetworkConnect.Delete(context.TODO(), iaID)
	return err
}

// UpdatePartnerNetworkConnect updates a partner network connect.
func (p *partnerNetworkConnectsService) UpdatePartnerNetworkConnect(iaID string, req *godo.PartnerNetworkConnectUpdateRequest) (*PartnerNetworkConnect, error) {
	partnerIA, _, err := p.client.PartnerNetworkConnect.Update(context.TODO(), iaID, req)
	if err != nil {
		return nil, err
	}

	return &PartnerNetworkConnect{PartnerNetworkConnect: partnerIA}, nil
}

// ListPartnerNetworkConnectRoutes lists all partner network connect routes.
func (p *partnerNetworkConnectsService) ListPartnerNetworkConnectRoutes(iaID string) (PartnerNetworkConnectRoutes, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := p.client.PartnerNetworkConnect.ListRoutes(context.TODO(), iaID, opt)
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

	list := make([]PartnerNetworkConnectRoute, len(si))
	for i := range si {
		a := si[i].(*godo.RemoteRoute)
		list[i] = PartnerNetworkConnectRoute{RemoteRoute: a}
	}

	return list, nil
}

// GetBGPAuthKey retrieves a BGP auth key of a partner connect attachment.
func (p *partnerNetworkConnectsService) GetBGPAuthKey(iaID string) (*PartnerNetworkConnectBGPAuthKey, error) {
	bgpAuthKey, _, err := p.client.PartnerNetworkConnect.GetBGPAuthKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerNetworkConnectBGPAuthKey{BgpAuthKey: bgpAuthKey}, nil
}

// RegenerateServiceKey regenerates a service key of a partner connect attachment.
func (p *partnerNetworkConnectsService) RegenerateServiceKey(iaID string) (*PartnerNetworkConnectRegenerateServiceKey, error) {
	regenerateServiceKey, _, err := p.client.PartnerNetworkConnect.RegenerateServiceKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerNetworkConnectRegenerateServiceKey{RegenerateServiceKey: regenerateServiceKey}, nil
}

// GetServiceKey retrieves a service key of a partner connect attachment.
func (p *partnerNetworkConnectsService) GetServiceKey(iaID string) (*PartnerNetworkConnectServiceKey, error) {
	serviceKey, _, err := p.client.PartnerNetworkConnect.GetServiceKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerNetworkConnectServiceKey{ServiceKey: serviceKey}, nil
}
