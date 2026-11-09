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
	"errors"
	"fmt"
	"net/http"

	"github.com/digitalocean/godo"
)

// ErrNoBillingPermission is returned by Get when the balance request is
// refused with a 403. A missing billing:read scope is one cause, but a team
// role or account state can also produce it, so the text names no cause. The
// returned error also wraps the original API error, so callers can read the
// status and the server's message with errors.As.
var ErrNoBillingPermission = errors.New("the prepayment balance could not be read: the request was refused")

// PrepaymentConfig aliases godo.PrepaymentConfig so harness-runtime callers can
// use do.* types without depending on godo directly.
type PrepaymentConfig = godo.PrepaymentConfig

// PrepaymentStatus aliases godo.PrepaymentStatus.
type PrepaymentStatus = godo.PrepaymentStatus

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
	// Get returns config and status together (same wire response as GetConfig)
	// and accepts a context so callers enriching a 402 card can bound the
	// lookup. A 403 matches ErrNoBillingPermission and still wraps the API
	// error; any other failure is returned as is.
	Get(ctx context.Context) (*PrepaymentConfig, *PrepaymentStatus, error)
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

func (ps *prepaymentService) Get(ctx context.Context) (*PrepaymentConfig, *PrepaymentStatus, error) {
	resp, _, err := ps.client.Prepayment.GetConfig(ctx)
	if err != nil {
		var er *godo.ErrorResponse
		if errors.As(err, &er) && er.Response != nil && er.Response.StatusCode == http.StatusForbidden {
			// Wrap both so callers keep matching the sentinel while the balance
			// command can still read the status and server message.
			return nil, nil, fmt.Errorf("%w: %w", ErrNoBillingPermission, err)
		}
		return nil, nil, err
	}
	if resp == nil {
		return nil, nil, nil
	}
	return resp.Config, resp.Status, nil
}
