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
	GetPartnerInterconnectAttachment(iaID string) (*PartnerNetworkConnect, error)
	ListPartnerInterconnectAttachments() (PartnerNetworkConnects, error)
	DeletePartnerInterconnectAttachment(iaID string) error
	UpdatePartnerInterconnectAttachment(iaID string, req *godo.PartnerNetworkConnectUpdateRequest) (*PartnerNetworkConnect, error)
	ListPartnerAttachmentRoutes(iaID string) (PartnerAttachmentRoutes, error)
	GetBGPAuthKey(iaID string) (*PartnerAttachmentBGPAuthKey, error)
	RegenerateServiceKey(iaID string) (*PartnerAttachmentRegenerateServiceKey, error)
	GetServiceKey(iaID string) (*PartnerAttachmentServiceKey, error)
}

var _ PartnerNetworkConnectsService = &partnerNetworkConnectsService{}

type partnerNetworkConnectsService struct {
	client *godo.Client
}

// NewPartnerInterconnectAttachmentsService builds an instance of PartnerNetworkConnectsService.
func NewPartnerInterconnectAttachmentsService(client *godo.Client) PartnerNetworkConnectsService {
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

// GetPartnerInterconnectAttachment retrieves a partner connect attachment.
func (p *partnerNetworkConnectsService) GetPartnerInterconnectAttachment(iaID string) (*PartnerNetworkConnect, error) {
	partnerIA, _, err := p.client.PartnerNetworkConnect.Get(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerNetworkConnect{PartnerNetworkConnect: partnerIA}, nil
}

// ListPartnerInterconnectAttachments lists all partner connect attachments.
func (p *partnerNetworkConnectsService) ListPartnerInterconnectAttachments() (PartnerNetworkConnects, error) {
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

func (p *partnerNetworkConnectsService) DeletePartnerInterconnectAttachment(iaID string) error {
	_, err := p.client.PartnerNetworkConnect.Delete(context.TODO(), iaID)
	return err
}

func (p *partnerNetworkConnectsService) UpdatePartnerInterconnectAttachment(iaID string, req *godo.PartnerNetworkConnectUpdateRequest) (*PartnerNetworkConnect, error) {
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

func (p *partnerNetworkConnectsService) GetBGPAuthKey(iaID string) (*PartnerAttachmentBGPAuthKey, error) {
	bgpAuthKey, _, err := p.client.PartnerNetworkConnect.GetBGPAuthKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerAttachmentBGPAuthKey{BgpAuthKey: bgpAuthKey}, nil
}

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
