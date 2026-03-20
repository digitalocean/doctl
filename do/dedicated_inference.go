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

// DedicatedInference wraps a godo.DedicatedInference.
type DedicatedInference struct {
	*godo.DedicatedInference
}

// DedicatedInferences is a slice of DedicatedInference.
type DedicatedInferences []DedicatedInference

// DedicatedInferenceToken wraps a godo.DedicatedInferenceToken.
type DedicatedInferenceToken struct {
	*godo.DedicatedInferenceToken
}

// DedicatedInferenceAcceleratorInfo wraps a godo.DedicatedInferenceAcceleratorInfo.
type DedicatedInferenceAcceleratorInfo struct {
	*godo.DedicatedInferenceAcceleratorInfo
}

// DedicatedInferenceAcceleratorInfos is a slice of DedicatedInferenceAcceleratorInfo.
type DedicatedInferenceAcceleratorInfos []DedicatedInferenceAcceleratorInfo

// DedicatedInferenceService is an interface for interacting with DigitalOcean's Dedicated Inference API.
type DedicatedInferenceService interface {
	Create(req *godo.DedicatedInferenceCreateRequest) (*DedicatedInference, *DedicatedInferenceToken, error)
	Get(id string) (*DedicatedInference, error)
	Delete(id string) error
	ListAccelerators(diID string, slug string) (DedicatedInferenceAcceleratorInfos, error)
}

var _ DedicatedInferenceService = &dedicatedInferenceService{}

type dedicatedInferenceService struct {
	client *godo.Client
}

// NewDedicatedInferenceService builds an instance of DedicatedInferenceService.
func NewDedicatedInferenceService(client *godo.Client) DedicatedInferenceService {
	return &dedicatedInferenceService{
		client: client,
	}
}

// Create creates a new dedicated inference endpoint.
func (s *dedicatedInferenceService) Create(req *godo.DedicatedInferenceCreateRequest) (*DedicatedInference, *DedicatedInferenceToken, error) {
	d, t, _, err := s.client.DedicatedInference.Create(context.TODO(), req)
	if err != nil {
		return nil, nil, err
	}
	var token *DedicatedInferenceToken
	if t != nil {
		token = &DedicatedInferenceToken{DedicatedInferenceToken: t}
	}
	return &DedicatedInference{DedicatedInference: d}, token, nil
}

// Get retrieves a dedicated inference endpoint by ID.
func (s *dedicatedInferenceService) Get(id string) (*DedicatedInference, error) {
	d, _, err := s.client.DedicatedInference.Get(context.TODO(), id)
	if err != nil {
		return nil, err
	}
	return &DedicatedInference{DedicatedInference: d}, nil
}

// Delete deletes a dedicated inference endpoint by ID.
func (s *dedicatedInferenceService) Delete(id string) error {
	_, err := s.client.DedicatedInference.Delete(context.TODO(), id)
	return err
}

// ListAccelerators lists accelerators for a dedicated inference endpoint.
func (s *dedicatedInferenceService) ListAccelerators(diID string, slug string) (DedicatedInferenceAcceleratorInfos, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.DedicatedInference.ListAccelerators(context.TODO(), diID, &godo.DedicatedInferenceListAcceleratorsOptions{Slug: slug, ListOptions: *opt})
		if err != nil {
			return nil, nil, err
		}

		items := make([]any, len(list))
		for i := range list {
			items[i] = list[i]
		}
		return items, resp, nil
	}

	si, err := PaginateResp(f)
	if err != nil {
		return nil, err
	}

	list := make(DedicatedInferenceAcceleratorInfos, len(si))
	for i := range si {
		a := si[i].(godo.DedicatedInferenceAcceleratorInfo)
		list[i] = DedicatedInferenceAcceleratorInfo{DedicatedInferenceAcceleratorInfo: &a}
	}
	return list, nil
}
