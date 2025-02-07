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
	*godo.PartnerInterconnectAttachment
}

// PartnerInterconnectAttachments is a slice of PartnerInterconnectAttachment.
type PartnerInterconnectAttachments []PartnerInterconnectAttachment

// PartnerInterconnectAttachmentsService is an interface for interacting with
// DigitalOcean's partner interconnect attachments api.
type PartnerInterconnectAttachmentsService interface {
	Create(*godo.PartnerInterconnectAttachmentCreateRequest) (*PartnerInterconnectAttachment, error)
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
func (p *partnerInterconnectAttachmentsService) Create(req *godo.PartnerInterconnectAttachmentCreateRequest) (*PartnerInterconnectAttachment, error) {
	pia, _, err := p.client.PartnerInterconnectAttachments.Create(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &PartnerInterconnectAttachment{PartnerInterconnectAttachment: pia}, nil
}
