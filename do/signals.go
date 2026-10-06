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
	"net/url"
	"time"

	"github.com/digitalocean/godo"
)

const (
	signalsConsentBasePath       = "/v1/consent"
	signalsExportsBasePath       = "/v1/signals/exports"
	signalsExportTriggerBasePath = "/v1/signals/export-trigger"
)

// SignalsConsent is one (team, agent) collection-consent row from
// consent-gateway GET/PUT /v1/consent.
type SignalsConsent struct {
	ID        uint64    `json:"id,omitempty"`
	TeamID    uint64    `json:"team_id"`
	AgentID   string    `json:"agent_id"`
	Enabled   bool      `json:"enabled"`
	Allowed   *bool     `json:"allowed,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// SignalsConsents is a slice of SignalsConsent.
type SignalsConsents []SignalsConsent

// SignalsExport is signals-api ExportJob. Timestamps are Unix seconds.
// Status: queued, running, complete, failed, expired.
type SignalsExport struct {
	ExportID     string               `json:"export_id"`
	AgentID      *string              `json:"agent_id,omitempty"`
	Status       string               `json:"status"`
	Filters      SignalsExportFilters `json:"filters"`
	CreatedAt    int64                `json:"created_at"`
	CompletedAt  *int64               `json:"completed_at"`
	ExpiresAt    *int64               `json:"expires_at"`
	ErrorMessage *string              `json:"error_message"`
}

// SignalsExportFilters is the create-time filter snapshot. JSON key is
// signal_type (singular), matching POST /v1/signals/exports.
type SignalsExportFilters struct {
	SessionIDs     []string `json:"session_ids,omitempty"`
	SignalType     []string `json:"signal_type,omitempty"`
	SignalCategory *string  `json:"signal_category,omitempty"`
	SignalLayer    *string  `json:"signal_layer,omitempty"`
	Concerning     *bool    `json:"concerning,omitempty"`
	StartTime      *int64   `json:"start_time,omitempty"`
	EndTime        *int64   `json:"end_time,omitempty"`
}

// SignalsExports is a slice of SignalsExport.
type SignalsExports []SignalsExport

// SignalsExportListOptions holds filters for listing exports.
type SignalsExportListOptions struct {
	AgentID string
	Limit   int
	After   string
}

// SignalsCreateExportRequest is POST /v1/signals/exports.
// Do not send signal_types — the API rejects unknown fields.
type SignalsCreateExportRequest struct {
	AgentID        string   `json:"agent_id"`
	SessionIDs     []string `json:"session_ids,omitempty"`
	SignalType     []string `json:"signal_type,omitempty"`
	SignalCategory *string  `json:"signal_category,omitempty"`
	SignalLayer    *string  `json:"signal_layer,omitempty"`
	Concerning     *bool    `json:"concerning,omitempty"`
	StartTime      *int64   `json:"start_time,omitempty"`
	EndTime        *int64   `json:"end_time,omitempty"`
}

// SignalsExportDownload is GET /v1/signals/exports/{id}/download.
type SignalsExportDownload struct {
	DownloadURL string `json:"download_url"`
	ExpiresAt   int64  `json:"expires_at"`
}

// SignalsExportOptions is GET /v1/signals/exports/options.
type SignalsExportOptions struct {
	Filters struct {
		SignalType []string `json:"signal_type"`
	} `json:"filters"`
}

// SignalsExportTrigger is GET/PUT /v1/signals/export-trigger.
// This is the only export API that uses signal_types (plural).
type SignalsExportTrigger struct {
	Enabled     bool     `json:"enabled"`
	Cadence     string   `json:"cadence"`
	SignalTypes []string `json:"signal_types"`
	UpdatedAt   *int64   `json:"updated_at,omitempty"`
}

// SignalsExportTriggerUpsertRequest is PUT /v1/signals/export-trigger.
type SignalsExportTriggerUpsertRequest struct {
	Enabled     bool     `json:"enabled"`
	Cadence     string   `json:"cadence,omitempty"`
	SignalTypes []string `json:"signal_types,omitempty"`
}

// SignalsService talks to consent-gateway and signals-api over the public
// Oceanus paths (same as Cloud UI).
type SignalsService interface {
	ListConsents() (SignalsConsents, error)
	GetConsent(agentID string) (*SignalsConsent, error)
	SetConsent(agentID string, enabled bool) (*SignalsConsent, error)

	ListExports(opts *SignalsExportListOptions) (SignalsExports, error)
	CreateExport(req *SignalsCreateExportRequest) (*SignalsExport, error)
	GetExport(exportID string) (*SignalsExport, error)
	GetExportDownload(exportID string) (*SignalsExportDownload, error)
	GetExportOptions() (*SignalsExportOptions, error)

	GetExportTrigger() (*SignalsExportTrigger, error)
	UpsertExportTrigger(req *SignalsExportTriggerUpsertRequest) (*SignalsExportTrigger, error)
}

var _ SignalsService = &signalsService{}

type signalsService struct {
	client *godo.Client
}

// NewSignalsService builds an instance of SignalsService.
func NewSignalsService(client *godo.Client) SignalsService {
	return &signalsService{client: client}
}

type listConsentsResponse struct {
	TeamID   uint64           `json:"team_id"`
	Consents []SignalsConsent `json:"consents"`
}

func (s *signalsService) ListConsents() (SignalsConsents, error) {
	req, err := s.client.NewRequest(context.TODO(), "GET", signalsConsentBasePath, nil)
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
	if agentID == "" {
		return nil, fmt.Errorf("agent-id is required")
	}
	path := fmt.Sprintf("%s/%s", signalsConsentBasePath, url.PathEscape(agentID))
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
	if agentID == "" {
		return nil, fmt.Errorf("agent-id is required")
	}
	path := fmt.Sprintf("%s/%s", signalsConsentBasePath, url.PathEscape(agentID))
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

type listExportsResponse struct {
	Edges []struct {
		Cursor string        `json:"cursor"`
		Node   SignalsExport `json:"node"`
	} `json:"edges"`
	PageInfo struct {
		HasNextPage bool    `json:"has_next_page"`
		EndCursor   *string `json:"end_cursor,omitempty"`
	} `json:"page_info"`
}

func (s *signalsService) ListExports(opts *SignalsExportListOptions) (SignalsExports, error) {
	path := signalsExportsBasePath
	if opts != nil {
		q := url.Values{}
		if opts.AgentID != "" {
			q.Set("agent_id", opts.AgentID)
		}
		if opts.Limit > 0 {
			q.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.After != "" {
			q.Set("after", opts.After)
		}
		if encoded := q.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}
	req, err := s.client.NewRequest(context.TODO(), "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp listExportsResponse
	_, err = s.client.Do(context.TODO(), req, &resp)
	if err != nil {
		return nil, err
	}
	out := make(SignalsExports, 0, len(resp.Edges))
	for _, e := range resp.Edges {
		out = append(out, e.Node)
	}
	return out, nil
}

func (s *signalsService) CreateExport(createReq *SignalsCreateExportRequest) (*SignalsExport, error) {
	if createReq == nil || createReq.AgentID == "" {
		return nil, fmt.Errorf("agent-id is required")
	}
	req, err := s.client.NewRequest(context.TODO(), "POST", signalsExportsBasePath, createReq)
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
	if exportID == "" {
		return nil, fmt.Errorf("export-id is required")
	}
	path := fmt.Sprintf("%s/%s", signalsExportsBasePath, url.PathEscape(exportID))
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

func (s *signalsService) GetExportDownload(exportID string) (*SignalsExportDownload, error) {
	if exportID == "" {
		return nil, fmt.Errorf("export-id is required")
	}
	path := fmt.Sprintf("%s/%s/download", signalsExportsBasePath, url.PathEscape(exportID))
	req, err := s.client.NewRequest(context.TODO(), "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var dl SignalsExportDownload
	_, err = s.client.Do(context.TODO(), req, &dl)
	if err != nil {
		return nil, err
	}
	return &dl, nil
}

func (s *signalsService) GetExportOptions() (*SignalsExportOptions, error) {
	req, err := s.client.NewRequest(context.TODO(), "GET", signalsExportsBasePath+"/options", nil)
	if err != nil {
		return nil, err
	}
	var opts SignalsExportOptions
	_, err = s.client.Do(context.TODO(), req, &opts)
	if err != nil {
		return nil, err
	}
	return &opts, nil
}

func (s *signalsService) GetExportTrigger() (*SignalsExportTrigger, error) {
	req, err := s.client.NewRequest(context.TODO(), "GET", signalsExportTriggerBasePath, nil)
	if err != nil {
		return nil, err
	}
	var trig SignalsExportTrigger
	_, err = s.client.Do(context.TODO(), req, &trig)
	if err != nil {
		return nil, err
	}
	return &trig, nil
}

func (s *signalsService) UpsertExportTrigger(in *SignalsExportTriggerUpsertRequest) (*SignalsExportTrigger, error) {
	if in == nil {
		return nil, fmt.Errorf("export trigger request is required")
	}
	req, err := s.client.NewRequest(context.TODO(), "PUT", signalsExportTriggerBasePath, in)
	if err != nil {
		return nil, err
	}
	var trig SignalsExportTrigger
	_, err = s.client.Do(context.TODO(), req, &trig)
	if err != nil {
		return nil, err
	}
	return &trig, nil
}
