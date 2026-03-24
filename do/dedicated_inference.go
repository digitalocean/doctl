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

// DedicatedInferenceListItem wraps a godo.DedicatedInferenceListItem.
type DedicatedInferenceListItem struct {
	*godo.DedicatedInferenceListItem
}

// DedicatedInferenceListItems is a slice of DedicatedInferenceListItem.
type DedicatedInferenceListItems []DedicatedInferenceListItem

// DedicatedInferenceAcceleratorInfo wraps a godo.DedicatedInferenceAcceleratorInfo.
type DedicatedInferenceAcceleratorInfo struct {
	*godo.DedicatedInferenceAcceleratorInfo
}

// DedicatedInferenceAcceleratorInfos is a slice of DedicatedInferenceAcceleratorInfo.
type DedicatedInferenceAcceleratorInfos []DedicatedInferenceAcceleratorInfo

// DedicatedInferenceTokens is a slice of DedicatedInferenceToken.
type DedicatedInferenceTokens []DedicatedInferenceToken

// DedicatedInferenceSize wraps a godo.DedicatedInferenceSize.
type DedicatedInferenceSize struct {
	*godo.DedicatedInferenceSize
}

// DedicatedInferenceSizes is a slice of DedicatedInferenceSize.
type DedicatedInferenceSizes []DedicatedInferenceSize

// DedicatedInferenceGPUModelConfig wraps a godo.DedicatedInferenceGPUModelConfig.
type DedicatedInferenceGPUModelConfig struct {
	*godo.DedicatedInferenceGPUModelConfig
}

// DedicatedInferenceGPUModelConfigs is a slice of DedicatedInferenceGPUModelConfig.
type DedicatedInferenceGPUModelConfigs []DedicatedInferenceGPUModelConfig

// DedicatedInferenceService is an interface for interacting with DigitalOcean's Dedicated Inference API.
type DedicatedInferenceService interface {
	Create(req *godo.DedicatedInferenceCreateRequest) (*DedicatedInference, *DedicatedInferenceToken, error)
	Get(id string) (*DedicatedInference, error)
	Update(id string, req *godo.DedicatedInferenceUpdateRequest) (*DedicatedInference, error)
	List(region string, name string) (DedicatedInferenceListItems, error)
	Delete(id string) error
	ListAccelerators(diID string, slug string) (DedicatedInferenceAcceleratorInfos, error)
	CreateToken(diID string, req *godo.DedicatedInferenceTokenCreateRequest) (*DedicatedInferenceToken, error)
	ListTokens(diID string) (DedicatedInferenceTokens, error)
	RevokeToken(diID string, tokenID string) error
	GetSizes() ([]string, DedicatedInferenceSizes, error)
	GetGPUModelConfig() (DedicatedInferenceGPUModelConfigs, error)
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

// Update updates an existing dedicated inference endpoint.
func (s *dedicatedInferenceService) Update(id string, req *godo.DedicatedInferenceUpdateRequest) (*DedicatedInference, error) {
	d, _, err := s.client.DedicatedInference.Update(context.TODO(), id, req)
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

// List lists all dedicated inference endpoints.
func (s *dedicatedInferenceService) List(region string, name string) (DedicatedInferenceListItems, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.DedicatedInference.List(context.TODO(), &godo.DedicatedInferenceListOptions{Region: region, Name: name, ListOptions: *opt})
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

	result := make(DedicatedInferenceListItems, len(si))
	for i := range si {
		d := si[i].(godo.DedicatedInferenceListItem)
		result[i] = DedicatedInferenceListItem{DedicatedInferenceListItem: &d}
	}
	return result, nil
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

// CreateToken creates a new auth token for a dedicated inference endpoint.
func (s *dedicatedInferenceService) CreateToken(diID string, req *godo.DedicatedInferenceTokenCreateRequest) (*DedicatedInferenceToken, error) {
	t, _, err := s.client.DedicatedInference.CreateToken(context.TODO(), diID, req)
	if err != nil {
		return nil, err
	}
	return &DedicatedInferenceToken{DedicatedInferenceToken: t}, nil
}

// ListTokens lists all auth tokens for a dedicated inference endpoint.
func (s *dedicatedInferenceService) ListTokens(diID string) (DedicatedInferenceTokens, error) {
	f := func(opt *godo.ListOptions) ([]any, *godo.Response, error) {
		list, resp, err := s.client.DedicatedInference.ListTokens(context.TODO(), diID, opt)
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

	result := make(DedicatedInferenceTokens, len(si))
	for i := range si {
		t := si[i].(godo.DedicatedInferenceToken)
		result[i] = DedicatedInferenceToken{DedicatedInferenceToken: &t}
	}
	return result, nil
}

// RevokeToken revokes an auth token for a dedicated inference endpoint.
func (s *dedicatedInferenceService) RevokeToken(diID string, tokenID string) error {
	_, err := s.client.DedicatedInference.RevokeToken(context.TODO(), diID, tokenID)
	return err
}

// GetSizes returns available dedicated inference sizes and pricing.
func (s *dedicatedInferenceService) GetSizes() ([]string, DedicatedInferenceSizes, error) {
	resp, _, err := s.client.DedicatedInference.GetSizes(context.TODO())
	if err != nil {
		return nil, nil, err
	}

	sizes := make(DedicatedInferenceSizes, len(resp.Sizes))
	for i, sz := range resp.Sizes {
		sizes[i] = DedicatedInferenceSize{DedicatedInferenceSize: sz}
	}
	return resp.EnabledRegions, sizes, nil
}

// GetGPUModelConfig returns supported GPU model configurations.
func (s *dedicatedInferenceService) GetGPUModelConfig() (DedicatedInferenceGPUModelConfigs, error) {
	resp, _, err := s.client.DedicatedInference.GetGPUModelConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	configs := make(DedicatedInferenceGPUModelConfigs, len(resp.GPUModelConfigs))
	for i, cfg := range resp.GPUModelConfigs {
		configs[i] = DedicatedInferenceGPUModelConfig{DedicatedInferenceGPUModelConfig: cfg}
	}
	return configs, nil
}
