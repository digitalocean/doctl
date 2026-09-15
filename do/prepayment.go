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

// PrepaymentConfigResponse is a wrapper for godo.PrepaymentConfigResponse.
type PrepaymentConfigResponse struct {
	*godo.PrepaymentConfigResponse
}

// PrepaymentStatusResponse is a wrapper for godo.PrepaymentStatusResponse.
type PrepaymentStatusResponse struct {
	*godo.PrepaymentStatusResponse
}

// PrepaymentService is an interface for interacting with DigitalOcean's prepayment api.
type PrepaymentService interface {
	GetConfig() (*PrepaymentConfigResponse, error)
	GetStatus() (*PrepaymentStatusResponse, error)
}

type prepaymentService struct {
	client *godo.Client
}

var _ PrepaymentService = &prepaymentService{}

// NewPrepaymentService builds a PrepaymentService instance.
func NewPrepaymentService(godoClient *godo.Client) PrepaymentService {
	return &prepaymentService{
		client: godoClient,
	}
}

func (ps *prepaymentService) GetConfig() (*PrepaymentConfigResponse, error) {
	resp, _, err := ps.client.Prepayment.GetConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	return &PrepaymentConfigResponse{PrepaymentConfigResponse: resp}, nil
}

func (ps *prepaymentService) GetStatus() (*PrepaymentStatusResponse, error) {
	resp, _, err := ps.client.Prepayment.GetStatus(context.TODO())
	if err != nil {
		return nil, err
	}

	return &PrepaymentStatusResponse{PrepaymentStatusResponse: resp}, nil
}
