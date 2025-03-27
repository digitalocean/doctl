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

// PartnerAttachment wraps a godo PartnerAttachment.
type PartnerAttachment struct {
	PartnerAttachment *godo.PartnerAttachment
}

// PartnerAttachments is a slice of PartnerAttachment.
type PartnerAttachments []PartnerAttachment

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

// PartnerAttachmentsService is an interface for interacting with
// DigitalOcean's partner attachment api.
type PartnerAttachmentsService interface {
	Create(*godo.PartnerAttachmentCreateRequest) (*PartnerAttachment, error)
	GetPartnerAttachment(paID string) (*PartnerAttachment, error)
	ListPartnerAttachments() (PartnerAttachments, error)
	DeletePartnerAttachment(paID string) error
	UpdatePartnerAttachment(paID string, req *godo.PartnerAttachmentUpdateRequest) (*PartnerAttachment, error)
	ListPartnerAttachmentRoutes(paID string) (PartnerAttachmentRoutes, error)
	GetBGPAuthKey(paID string) (*PartnerAttachmentBGPAuthKey, error)
	RegenerateServiceKey(paID string) (*PartnerAttachmentRegenerateServiceKey, error)
	GetServiceKey(paID string) (*PartnerAttachmentServiceKey, error)
}

var _ PartnerAttachmentsService = &partnerAttachmentsService{}

type partnerAttachmentsService struct {
	client *godo.Client
}

// NewPartnerAttachmentsService builds an instance of PartnerAttachmentsService.
func NewPartnerAttachmentsService(client *godo.Client) PartnerAttachmentsService {
	return &partnerAttachmentsService{
		client: client,
	}
}

// Create creates a partner connect attachment.
func (p *partnerAttachmentsService) Create(req *godo.PartnerAttachmentCreateRequest) (*PartnerAttachment, error) {
	pia, _, err := p.client.PartnerAttachment.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &PartnerAttachment{PartnerAttachment: pia}, nil
}

// GetPartnerAttachment retrieves a partner connect attachment.
func (p *partnerAttachmentsService) GetPartnerAttachment(iaID string) (*PartnerAttachment, error) {
	partnerIA, _, err := p.client.PartnerAttachment.Get(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerAttachment{PartnerAttachment: partnerIA}, nil
}

// ListPartnerAttachments lists all partner connect attachments.
func (p *partnerAttachmentsService) ListPartnerAttachments() (PartnerAttachments, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := p.client.PartnerAttachment.List(context.TODO(), opt)
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

	list := make([]PartnerAttachment, len(si))
	for i := range si {
		a := si[i].(*godo.PartnerAttachment)
		list[i] = PartnerAttachment{PartnerAttachment: a}
	}

	return list, nil
}

// DeletePartnerAttachment deletes a partner connect attachment.
func (p *partnerAttachmentsService) DeletePartnerAttachment(iaID string) error {
	_, err := p.client.PartnerAttachment.Delete(context.TODO(), iaID)
	return err
}

// UpdatePartnerAttachment updates a partner attachment.
func (p *partnerAttachmentsService) UpdatePartnerAttachment(iaID string, req *godo.PartnerAttachmentUpdateRequest) (*PartnerAttachment, error) {
	partnerIA, _, err := p.client.PartnerAttachment.Update(context.TODO(), iaID, req)
	if err != nil {
		return nil, err
	}

	return &PartnerAttachment{PartnerAttachment: partnerIA}, nil
}

// ListPartnerAttachmentRoutes lists all partner attachment routes.
func (p *partnerAttachmentsService) ListPartnerAttachmentRoutes(iaID string) (PartnerAttachmentRoutes, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := p.client.PartnerAttachment.ListRoutes(context.TODO(), iaID, opt)
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
func (p *partnerAttachmentsService) GetBGPAuthKey(iaID string) (*PartnerAttachmentBGPAuthKey, error) {
	bgpAuthKey, _, err := p.client.PartnerAttachment.GetBGPAuthKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerAttachmentBGPAuthKey{BgpAuthKey: bgpAuthKey}, nil
}

// RegenerateServiceKey regenerates a service key of a partner connect attachment.
func (p *partnerAttachmentsService) RegenerateServiceKey(iaID string) (*PartnerAttachmentRegenerateServiceKey, error) {
	regenerateServiceKey, _, err := p.client.PartnerAttachment.RegenerateServiceKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerAttachmentRegenerateServiceKey{RegenerateServiceKey: regenerateServiceKey}, nil
}

// GetServiceKey retrieves a service key of a partner connect attachment.
func (p *partnerAttachmentsService) GetServiceKey(iaID string) (*PartnerAttachmentServiceKey, error) {
	serviceKey, _, err := p.client.PartnerAttachment.GetServiceKey(context.TODO(), iaID)
	if err != nil {
		return nil, err
	}
	return &PartnerAttachmentServiceKey{ServiceKey: serviceKey}, nil
}
