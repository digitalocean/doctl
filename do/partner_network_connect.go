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
	*godo.PartnerNetworkConnect
}

// PartnerNetworkConnects is a slice of PartnerNetworkConnect.
type PartnerNetworkConnects []PartnerNetworkConnect

// PartnerAttachmentRoute wraps a godo RemoteRoute.
type PartnerAttachmentRoute struct {
	*godo.RemoteRoute
}

// PartnerAttachmentRoutes is a slice of PartnerAttachmentRoute.
type PartnerAttachmentRoutes []PartnerAttachmentRoute

// PartnerAttachmentBGPAuthKey wrap a godo BgpAuthKey.
type PartnerAttachmentBGPAuthKey struct {
	*godo.BgpAuthKey
}

// PartnerAttachmentRegenerateServiceKey wraps a godo RegenerateServiceKey.
type PartnerAttachmentRegenerateServiceKey struct {
	*godo.RegenerateServiceKey
}

// PartnerAttachmentServiceKey wraps a godo ServiceKey.
type PartnerAttachmentServiceKey struct {
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
	ListPartnerAttachmentRoutes(pncID string) (PartnerAttachmentRoutes, error)
	GetBGPAuthKey(pncID string) (*PartnerAttachmentBGPAuthKey, error)
	RegenerateServiceKey(pncID string) (*PartnerAttachmentRegenerateServiceKey, error)
	GetServiceKey(pncID string) (*PartnerAttachmentServiceKey, error)
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
		a := si[i].(*godo.PartnerNetworkConnect)
		list[i] = PartnerNetworkConnect{PartnerNetworkConnect: a}
	}

	return list, nil
}

// DeletePartnerNetworkConnect deletes a partner connect attachment.
func (p *partnerNetworkConnectsService) DeletePartnerNetworkConnect(iaID string) error {
	_, err := p.client.PartnerNetworkConnect.Delete(context.TODO(), iaID)
	return err
}

// UpdatePartnerNetworkConnect updates a partner connect attachment.
func (p *partnerNetworkConnectsService) UpdatePartnerNetworkConnect(iaID string, req *godo.PartnerNetworkConnectUpdateRequest) (*PartnerNetworkConnect, error) {
	partnerIA, _, err := p.client.PartnerNetworkConnect.Update(context.TODO(), iaID, req)
	if err != nil {
		return nil, err
	}

	return &PartnerNetworkConnect{PartnerNetworkConnect: partnerIA}, nil
}

// ListPartnerAttachmentRoutes lists all partner attachment routes.
func (p *partnerNetworkConnectsService) ListPartnerAttachmentRoutes(iaID string) (PartnerAttachmentRoutes, error) {
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

	list := make([]PartnerAttachmentRoute, len(si))
	for i := range si {
		a := si[i].(*godo.RemoteRoute)
		list[i] = PartnerAttachmentRoute{RemoteRoute: a}
	}

	return list, nil
}

// GetBGPAuthKey retrieves a BGP auth key of a partner connect attachment.
func (p *partnerNetworkConnectsService) GetBGPAuthKey(iaID string) (*PartnerAttachmentBGPAuthKey, error) {
	bgpAuthKey, _, err := p.client.PartnerNetworkConnect.GetBGPAuthKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerAttachmentBGPAuthKey{BgpAuthKey: bgpAuthKey}, nil
}

// RegenerateServiceKey regenerates a service key of a partner connect attachment.
func (p *partnerNetworkConnectsService) RegenerateServiceKey(iaID string) (*PartnerAttachmentRegenerateServiceKey, error) {
	regenerateServiceKey, _, err := p.client.PartnerNetworkConnect.RegenerateServiceKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerAttachmentRegenerateServiceKey{RegenerateServiceKey: regenerateServiceKey}, nil
}

// GetServiceKey retrieves a service key of a partner connect attachment.
func (p *partnerNetworkConnectsService) GetServiceKey(iaID string) (*PartnerAttachmentServiceKey, error) {
	serviceKey, _, err := p.client.PartnerNetworkConnect.GetServiceKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerAttachmentServiceKey{ServiceKey: serviceKey}, nil
}
