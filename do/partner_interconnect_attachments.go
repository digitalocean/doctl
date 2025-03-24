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

// PartnerInterconnectAttachment wraps a godo PartnerInterconnectAttachment.
type PartnerInterconnectAttachment struct {
	*godo.PartnerNetworkConnect
}

// PartnerInterconnectAttachments is a slice of PartnerInterconnectAttachment.
type PartnerInterconnectAttachments []PartnerInterconnectAttachment

// PartnerInterconnectAttachmentRoute wraps a godo RemoteRoute.
type PartnerInterconnectAttachmentRoute struct {
	*godo.RemoteRoute
}

// PartnerInterconnectAttachmentRoutes is a slice of PartnerInterconnectAttachmentRoute.
type PartnerInterconnectAttachmentRoutes []PartnerInterconnectAttachmentRoute

// PartnerInterconnectAttachmentBGPAuthKey wrap a godo BgpAuthKey.
type PartnerInterconnectAttachmentBGPAuthKey struct {
	*godo.BgpAuthKey
}

// PartnerInterconnectAttachmentRegenerateServiceKey wraps a godo ServiceKey.
type PartnerInterconnectAttachmentRegenerateServiceKey struct {
	*godo.RegenerateServiceKey
}

// PartnerInterconnectAttachmentServiceKey wraps a godo ServiceKey.
type PartnerInterconnectAttachmentServiceKey struct {
	*godo.ServiceKey
}

// PartnerInterconnectAttachmentsService is an interface for interacting with
// DigitalOcean's partner interconnect attachments api.
type PartnerInterconnectAttachmentsService interface {
	Create(request *godo.PartnerNetworkConnectCreateRequest) (*PartnerInterconnectAttachment, error)
	GetPartnerInterconnectAttachment(iaID string) (*PartnerInterconnectAttachment, error)
	ListPartnerInterconnectAttachments() (PartnerInterconnectAttachments, error)
	DeletePartnerInterconnectAttachment(iaID string) error
	UpdatePartnerInterconnectAttachment(iaID string, req *godo.PartnerNetworkConnectUpdateRequest) (*PartnerInterconnectAttachment, error)
	ListPartnerInterconnectAttachmentRoutes(iaID string) (PartnerInterconnectAttachmentRoutes, error)
	GetBGPAuthKey(iaID string) (*PartnerInterconnectAttachmentBGPAuthKey, error)
	RegenerateServiceKey(iaID string) (*PartnerInterconnectAttachmentRegenerateServiceKey, error)
	GetServiceKey(iaID string) (*PartnerInterconnectAttachmentServiceKey, error)
}

var _ PartnerInterconnectAttachmentsService = &partnerInterconnectAttachmentsService{}

type partnerInterconnectAttachmentsService struct {
	client *godo.Client
}

// NewPartnerInterconnectAttachmentsService builds an instance of PartnerInterconnectAttachmentsService.
func NewPartnerInterconnectAttachmentsService(client *godo.Client) PartnerInterconnectAttachmentsService {
	return &partnerInterconnectAttachmentsService{
		client: client,
	}
}

// Create creates a partner interconnect attachment.
func (p *partnerInterconnectAttachmentsService) Create(req *godo.PartnerNetworkConnectCreateRequest) (*PartnerInterconnectAttachment, error) {
	pia, _, err := p.client.PartnerNetworkConnect.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &PartnerInterconnectAttachment{PartnerNetworkConnect: pia}, nil
}

// GetPartnerInterconnectAttachment retrieves a partner interconnect attachment.
func (p *partnerInterconnectAttachmentsService) GetPartnerInterconnectAttachment(iaID string) (*PartnerInterconnectAttachment, error) {
	partnerIA, _, err := p.client.PartnerNetworkConnect.Get(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerInterconnectAttachment{PartnerNetworkConnect: partnerIA}, nil
}

// ListPartnerInterconnectAttachments lists all partner interconnect attachments.
func (p *partnerInterconnectAttachmentsService) ListPartnerInterconnectAttachments() (PartnerInterconnectAttachments, error) {
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

	list := make([]PartnerInterconnectAttachment, len(si))
	for i := range si {
		a := si[i].(*godo.PartnerNetworkConnect)
		list[i] = PartnerInterconnectAttachment{PartnerNetworkConnect: a}
	}

	return list, nil
}

func (p *partnerInterconnectAttachmentsService) DeletePartnerInterconnectAttachment(iaID string) error {
	_, err := p.client.PartnerNetworkConnect.Delete(context.TODO(), iaID)
	return err
}

func (p *partnerInterconnectAttachmentsService) UpdatePartnerInterconnectAttachment(iaID string, req *godo.PartnerNetworkConnectUpdateRequest) (*PartnerInterconnectAttachment, error) {
	partnerIA, _, err := p.client.PartnerNetworkConnect.Update(context.TODO(), iaID, req)
	if err != nil {
		return nil, err
	}

	return &PartnerInterconnectAttachment{PartnerNetworkConnect: partnerIA}, nil
}

// ListPartnerInterconnectAttachmentRoutes lists all partner interconnect attachment routes.
func (p *partnerInterconnectAttachmentsService) ListPartnerInterconnectAttachmentRoutes(iaID string) (PartnerInterconnectAttachmentRoutes, error) {
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

	list := make([]PartnerInterconnectAttachmentRoute, len(si))
	for i := range si {
		a := si[i].(*godo.RemoteRoute)
		list[i] = PartnerInterconnectAttachmentRoute{RemoteRoute: a}
	}

	return list, nil
}

func (p *partnerInterconnectAttachmentsService) GetBGPAuthKey(iaID string) (*PartnerInterconnectAttachmentBGPAuthKey, error) {
	bgpAuthKey, _, err := p.client.PartnerNetworkConnect.GetBGPAuthKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerInterconnectAttachmentBGPAuthKey{BgpAuthKey: bgpAuthKey}, nil
}

func (p *partnerInterconnectAttachmentsService) RegenerateServiceKey(iaID string) (*PartnerInterconnectAttachmentRegenerateServiceKey, error) {
	regenerateServiceKey, _, err := p.client.PartnerNetworkConnect.RegenerateServiceKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerInterconnectAttachmentRegenerateServiceKey{RegenerateServiceKey: regenerateServiceKey}, nil
}

// GetServiceKey retrieves a service key of a partner interconnect attachment.
func (p *partnerInterconnectAttachmentsService) GetServiceKey(iaID string) (*PartnerInterconnectAttachmentServiceKey, error) {
	serviceKey, _, err := p.client.PartnerNetworkConnect.GetServiceKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerInterconnectAttachmentServiceKey{ServiceKey: serviceKey}, nil
}
