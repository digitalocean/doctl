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

// DedicatedInferenceService is an interface for interacting with DigitalOcean's Dedicated Inference API.
type DedicatedInferenceService interface {
	Create(req *godo.DedicatedInferenceCreateRequest) (*DedicatedInference, *DedicatedInferenceToken, error)
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
