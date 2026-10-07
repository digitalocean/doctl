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

// SignalsConsent wraps a godo.SignalsConsentRecord.
type SignalsConsent struct {
	*godo.SignalsConsentRecord
}

// SignalsConsents is a slice of SignalsConsent.
type SignalsConsents []SignalsConsent

// SignalsAgentConsent wraps a godo.SignalsAgentConsent.
type SignalsAgentConsent struct {
	*godo.SignalsAgentConsent
}

// SignalsSession wraps a godo.SignalsSession.
type SignalsSession struct {
	*godo.SignalsSession
}

// SignalsSessions is a slice of SignalsSession.
type SignalsSessions []SignalsSession

// SignalsSessionDialogue wraps a godo.SignalsSessionDialogue.
type SignalsSessionDialogue struct {
	*godo.SignalsSessionDialogue
}

// SignalsSessionDialogues is a slice of SignalsSessionDialogue.
type SignalsSessionDialogues []SignalsSessionDialogue

// SignalsExport wraps a godo.SignalsExportJob.
type SignalsExport struct {
	*godo.SignalsExportJob
}

// SignalsExports is a slice of SignalsExport.
type SignalsExports []SignalsExport

// SignalsExportDownload wraps a godo.SignalsExportDownload.
type SignalsExportDownload struct {
	*godo.SignalsExportDownload
}

// SignalsExportOptions wraps a godo.SignalsExportOptions.
type SignalsExportOptions struct {
	*godo.SignalsExportOptions
}

// SignalsService talks to consent-gateway and signals-api via godo.
type SignalsService interface {
	ListConsents() (SignalsConsents, error)
	GetConsent(agentID string) (*SignalsAgentConsent, error)
	SetConsent(agentID string, enabled bool) (*SignalsConsent, error)

	ListAgentSessions(agentID string, opts *godo.SignalsListAgentSessionsOptions) (SignalsSessions, error)
	ListSessionDialogues(sessionID string, opts *godo.SignalsListDialoguesOptions) (SignalsSessionDialogues, error)

	ListExports(opts *godo.SignalsListExportsOptions) (SignalsExports, error)
	CreateExport(req *godo.SignalsCreateExportRequest) (*SignalsExport, error)
	GetExport(exportID string) (*SignalsExport, error)
	GetExportDownload(exportID string) (*SignalsExportDownload, error)
	GetExportOptions() (*SignalsExportOptions, error)
}

var _ SignalsService = &signalsService{}

type signalsService struct {
	client *godo.Client
}

// NewSignalsService builds an instance of SignalsService.
func NewSignalsService(client *godo.Client) SignalsService {
	return &signalsService{client: client}
}

func (s *signalsService) ListConsents() (SignalsConsents, error) {
	resp, _, err := s.client.Signals.ListConsents(context.TODO())
	if err != nil {
		return nil, err
	}
	out := make(SignalsConsents, len(resp.Consents))
	for i := range resp.Consents {
		out[i] = SignalsConsent{SignalsConsentRecord: &resp.Consents[i]}
	}
	return out, nil
}

func (s *signalsService) GetConsent(agentID string) (*SignalsAgentConsent, error) {
	consent, _, err := s.client.Signals.GetAgentConsent(context.TODO(), agentID)
	if err != nil {
		return nil, err
	}
	return &SignalsAgentConsent{SignalsAgentConsent: consent}, nil
}

func (s *signalsService) SetConsent(agentID string, enabled bool) (*SignalsConsent, error) {
	record, _, err := s.client.Signals.SetAgentConsent(context.TODO(), agentID, enabled)
	if err != nil {
		return nil, err
	}
	return &SignalsConsent{SignalsConsentRecord: record}, nil
}

func (s *signalsService) ListAgentSessions(agentID string, opts *godo.SignalsListAgentSessionsOptions) (SignalsSessions, error) {
	resp, _, err := s.client.Signals.ListAgentSessions(context.TODO(), agentID, opts)
	if err != nil {
		return nil, err
	}
	out := make(SignalsSessions, len(resp.Edges))
	for i, e := range resp.Edges {
		node := e.Node
		out[i] = SignalsSession{SignalsSession: &node}
	}
	return out, nil
}

func (s *signalsService) ListSessionDialogues(sessionID string, opts *godo.SignalsListDialoguesOptions) (SignalsSessionDialogues, error) {
	resp, _, err := s.client.Signals.ListSessionDialogues(context.TODO(), sessionID, opts)
	if err != nil {
		return nil, err
	}
	out := make(SignalsSessionDialogues, len(resp.Edges))
	for i, e := range resp.Edges {
		node := e.Node
		out[i] = SignalsSessionDialogue{SignalsSessionDialogue: &node}
	}
	return out, nil
}

func (s *signalsService) ListExports(opts *godo.SignalsListExportsOptions) (SignalsExports, error) {
	resp, _, err := s.client.Signals.ListExports(context.TODO(), opts)
	if err != nil {
		return nil, err
	}
	out := make(SignalsExports, len(resp.Edges))
	for i, e := range resp.Edges {
		node := e.Node
		out[i] = SignalsExport{SignalsExportJob: &node}
	}
	return out, nil
}

func (s *signalsService) CreateExport(req *godo.SignalsCreateExportRequest) (*SignalsExport, error) {
	job, _, err := s.client.Signals.CreateExport(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &SignalsExport{SignalsExportJob: job}, nil
}

func (s *signalsService) GetExport(exportID string) (*SignalsExport, error) {
	job, _, err := s.client.Signals.GetExport(context.TODO(), exportID)
	if err != nil {
		return nil, err
	}
	return &SignalsExport{SignalsExportJob: job}, nil
}

func (s *signalsService) GetExportDownload(exportID string) (*SignalsExportDownload, error) {
	dl, _, err := s.client.Signals.GetExportDownload(context.TODO(), exportID)
	if err != nil {
		return nil, err
	}
	return &SignalsExportDownload{SignalsExportDownload: dl}, nil
}

func (s *signalsService) GetExportOptions() (*SignalsExportOptions, error) {
	opts, _, err := s.client.Signals.GetExportOptions(context.TODO())
	if err != nil {
		return nil, err
	}
	return &SignalsExportOptions{SignalsExportOptions: opts}, nil
}
