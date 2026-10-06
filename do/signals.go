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
	"fmt"
	"time"

	"github.com/digitalocean/godo"
)

// --- Consent types ---

// SignalsConsent represents a Signals collection consent for one (team, agent).
type SignalsConsent struct {
	ID        uint64    `json:"id"`
	TeamID    uint64    `json:"team_id"`
	AgentID   string    `json:"agent_id"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SignalsConsents is a slice of SignalsConsent.
type SignalsConsents []SignalsConsent

// --- Export types ---

// SignalsExport represents a Signals bulk export job.
type SignalsExport struct {
	ID           string    `json:"id"`
	TeamID       int64     `json:"team_id"`
	AgentID      string    `json:"agent_id"`
	Status       string    `json:"status"`
	SignalTypes  []string  `json:"signal_types"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CompletedAt  *string   `json:"completed_at,omitempty"`
	DownloadURL  *string   `json:"download_url,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
}

// SignalsExports is a slice of SignalsExport.
type SignalsExports []SignalsExport

// SignalsExportListOptions holds filters for listing exports.
type SignalsExportListOptions struct {
	AgentID string
	Limit   int
	After   string
}

// SignalsCreateExportRequest is the request body for creating an export.
type SignalsCreateExportRequest struct {
	AgentID     string   `json:"agent_id"`
	SignalTypes []string `json:"signal_types,omitempty"`
	StartTime   *int64   `json:"start_time,omitempty"`
	EndTime     *int64   `json:"end_time,omitempty"`
}

// --- Service interface ---

// SignalsService is an interface for interacting with DigitalOcean's Signals APIs.
type SignalsService interface {
	// Consent endpoints (consent-gateway /v1/consent)
	ListConsents() (SignalsConsents, error)
	GetConsent(agentID string) (*SignalsConsent, error)
	SetConsent(agentID string, enabled bool) (*SignalsConsent, error)

	// Export endpoints (signals-api /v1/signals/exports)
	ListExports(opts *SignalsExportListOptions) (SignalsExports, error)
	CreateExport(req *SignalsCreateExportRequest) (*SignalsExport, error)
	GetExport(exportID string) (*SignalsExport, error)
}

var _ SignalsService = &signalsService{}

type signalsService struct {
	client *godo.Client
}

// NewSignalsService builds an instance of SignalsService.
func NewSignalsService(client *godo.Client) SignalsService {
	return &signalsService{client: client}
}

// --- Consent implementation ---

type listConsentsResponse struct {
	TeamID   uint64           `json:"team_id"`
	Consents []SignalsConsent `json:"consents"`
}

func (s *signalsService) ListConsents() (SignalsConsents, error) {
	req, err := s.client.NewRequest(context.TODO(), "GET", "/v1/consent", nil)
	if err != nil {
		return nil, err
	}

	var resp listConsentsResponse
	_, err = s.client.Do(context.TODO(), req, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Consents, nil
}

func (s *signalsService) GetConsent(agentID string) (*SignalsConsent, error) {
	path := fmt.Sprintf("/v1/consent/%s", agentID)
	req, err := s.client.NewRequest(context.TODO(), "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var consent SignalsConsent
	_, err = s.client.Do(context.TODO(), req, &consent)
	if err != nil {
		return nil, err
	}

	return &consent, nil
}

type setConsentRequest struct {
	Enabled bool `json:"enabled"`
}

type setConsentResponse struct {
	Consent SignalsConsent `json:"consent"`
}

func (s *signalsService) SetConsent(agentID string, enabled bool) (*SignalsConsent, error) {
	path := fmt.Sprintf("/v1/consent/%s", agentID)
	req, err := s.client.NewRequest(context.TODO(), "PUT", path, &setConsentRequest{Enabled: enabled})
	if err != nil {
		return nil, err
	}

	var resp setConsentResponse
	_, err = s.client.Do(context.TODO(), req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp.Consent, nil
}

// --- Export implementation ---

type listExportsResponse struct {
	Exports  []SignalsExport `json:"exports"`
	PageInfo *struct {
		HasNextPage bool    `json:"has_next_page"`
		EndCursor   *string `json:"end_cursor,omitempty"`
	} `json:"page_info,omitempty"`
}

func (s *signalsService) ListExports(opts *SignalsExportListOptions) (SignalsExports, error) {
	path := "/v1/signals/exports"

	params := ""
	if opts != nil {
		sep := "?"
		if opts.AgentID != "" {
			params += sep + "agent_id=" + opts.AgentID
			sep = "&"
		}
		if opts.Limit > 0 {
			params += sep + fmt.Sprintf("limit=%d", opts.Limit)
			sep = "&"
		}
		if opts.After != "" {
			params += sep + "after=" + opts.After
			sep = "&"
		}
	}

	req, err := s.client.NewRequest(context.TODO(), "GET", path+params, nil)
	if err != nil {
		return nil, err
	}

	var resp listExportsResponse
	_, err = s.client.Do(context.TODO(), req, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Exports, nil
}

func (s *signalsService) CreateExport(createReq *SignalsCreateExportRequest) (*SignalsExport, error) {
	req, err := s.client.NewRequest(context.TODO(), "POST", "/v1/signals/exports", createReq)
	if err != nil {
		return nil, err
	}

	var export SignalsExport
	_, err = s.client.Do(context.TODO(), req, &export)
	if err != nil {
		return nil, err
	}

	return &export, nil
}

func (s *signalsService) GetExport(exportID string) (*SignalsExport, error) {
	path := fmt.Sprintf("/v1/signals/exports/%s", exportID)
	req, err := s.client.NewRequest(context.TODO(), "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var export SignalsExport
	_, err = s.client.Do(context.TODO(), req, &export)
	if err != nil {
		return nil, err
	}

	return &export, nil
}
