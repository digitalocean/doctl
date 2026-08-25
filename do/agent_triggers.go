/*
Copyright 2026 The Doctl Authors All rights reserved.
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
	"time"

	"github.com/digitalocean/godo"
)

// HostedAgentTrigger wraps godo.HostedAgentTrigger.
type HostedAgentTrigger struct {
	*godo.HostedAgentTrigger
}

// HostedAgentTriggerCreateResult is returned by Create. WebhookSecret is present
// for webhook triggers only and shown exactly once.
type HostedAgentTriggerCreateResult struct {
	Trigger       *HostedAgentTrigger
	WebhookSecret string
}

// HostedAgentTriggerExecution wraps godo.HostedAgentTriggerExecution.
type HostedAgentTriggerExecution struct {
	*godo.HostedAgentTriggerExecution
}

// HostedAgentWebhookProvider wraps godo.HostedAgentWebhookProvider.
type HostedAgentWebhookProvider struct {
	*godo.HostedAgentWebhookProvider
}

// HostedAgentReusableSession wraps godo.HostedAgentReusableSession.
type HostedAgentReusableSession struct {
	*godo.HostedAgentReusableSession
}

// HostedAgentTriggerRotateSecretResult is the outcome of a secret rotation.
// PreviousExpiresAt and PreviousRevoked are read straight from the API rather
// than derived from each other: the server sets exactly one, and inferring the
// other from a missing field would turn any unrelated omission into a false
// "the old secret is dead".
type HostedAgentTriggerRotateSecretResult struct {
	Secret            string
	PreviousExpiresAt string
	PreviousRevoked   bool
}

// HostedAgentTriggersService is the doctl-facing wrapper around
// godo.HostedAgentTriggersService.
type HostedAgentTriggersService interface {
	List(*godo.HostedAgentTriggerListOptions) ([]HostedAgentTrigger, string, error)
	Create(*godo.HostedAgentTriggerCreateRequest) (*HostedAgentTriggerCreateResult, error)
	Get(triggerID string) (*HostedAgentTrigger, error)
	Update(triggerID string, update *godo.HostedAgentTriggerUpdateRequest) (*HostedAgentTrigger, error)
	Delete(triggerID string) error
	RotateSecret(triggerID string, revokePrevious bool) (*HostedAgentTriggerRotateSecretResult, error)
	ListExecutions(triggerID string, opt *godo.HostedAgentTriggerExecutionListOptions) ([]HostedAgentTriggerExecution, string, error)
	GetExecution(triggerID, executionID string) (*HostedAgentTriggerExecution, error)
	GetBySession(sessionID string) (*HostedAgentTrigger, error)
	ListReusableSessions(*godo.HostedAgentReusableSessionListOptions) ([]HostedAgentReusableSession, string, error)
	ListWebhookProviders() ([]HostedAgentWebhookProvider, error)
}

type hostedAgentTriggersService struct {
	svc godo.HostedAgentTriggersService
}

var _ HostedAgentTriggersService = &hostedAgentTriggersService{}

// NewHostedAgentTriggersService builds a HostedAgentTriggersService from a godo client.
func NewHostedAgentTriggersService(client *godo.Client) HostedAgentTriggersService {
	return &hostedAgentTriggersService{svc: client.HostedAgentTriggers}
}

func (s *hostedAgentTriggersService) List(opt *godo.HostedAgentTriggerListOptions) ([]HostedAgentTrigger, string, error) {
	resp, _, err := s.svc.List(context.TODO(), opt)
	if err != nil {
		return nil, "", err
	}
	out := make([]HostedAgentTrigger, len(resp.Triggers))
	for i := range resp.Triggers {
		t := resp.Triggers[i]
		out[i] = HostedAgentTrigger{HostedAgentTrigger: &t}
	}
	return out, resp.NextPageToken, nil
}

func (s *hostedAgentTriggersService) Create(create *godo.HostedAgentTriggerCreateRequest) (*HostedAgentTriggerCreateResult, error) {
	resp, _, err := s.svc.Create(context.TODO(), create)
	if err != nil {
		return nil, err
	}
	result := &HostedAgentTriggerCreateResult{WebhookSecret: resp.WebhookSecret}
	if resp.Trigger != nil {
		result.Trigger = &HostedAgentTrigger{HostedAgentTrigger: resp.Trigger}
	}
	return result, nil
}

func (s *hostedAgentTriggersService) Get(triggerID string) (*HostedAgentTrigger, error) {
	t, _, err := s.svc.Get(context.TODO(), triggerID)
	if err != nil {
		return nil, err
	}
	return &HostedAgentTrigger{HostedAgentTrigger: t}, nil
}

func (s *hostedAgentTriggersService) Update(triggerID string, update *godo.HostedAgentTriggerUpdateRequest) (*HostedAgentTrigger, error) {
	t, _, err := s.svc.Update(context.TODO(), triggerID, update)
	if err != nil {
		return nil, err
	}
	return &HostedAgentTrigger{HostedAgentTrigger: t}, nil
}

func (s *hostedAgentTriggersService) Delete(triggerID string) error {
	_, err := s.svc.Delete(context.TODO(), triggerID)
	return err
}

// RotateSecret issues a new webhook secret and reports what became of the old
// one.
func (s *hostedAgentTriggersService) RotateSecret(triggerID string, revokePrevious bool) (*HostedAgentTriggerRotateSecretResult, error) {
	// Left nil for the default rotation so no revoke_previous parameter goes on
	// the wire at all, rather than an explicit =false.
	var opt *godo.HostedAgentTriggerRotateSecretOptions
	if revokePrevious {
		opt = &godo.HostedAgentTriggerRotateSecretOptions{RevokePrevious: true}
	}
	resp, _, err := s.svc.RotateSecret(context.TODO(), triggerID, opt)
	if err != nil {
		return nil, err
	}
	out := &HostedAgentTriggerRotateSecretResult{
		Secret: resp.WebhookSecret,
		// Read, never inferred from a missing expiry. The API guarantees exactly
		// one of the two fields, so an absent expiry means "revoked" only if the
		// response really said so — any other cause of a missing field (an older
		// server, a proxy dropping it) would otherwise be reported to the
		// operator as a dead secret while it is still live.
		PreviousRevoked: resp.PreviousSecretRevoked,
	}
	if resp.PreviousSecretExpiresAt != nil {
		out.PreviousExpiresAt = resp.PreviousSecretExpiresAt.UTC().Format(time.RFC3339)
	}
	return out, nil
}

func (s *hostedAgentTriggersService) ListExecutions(triggerID string, opt *godo.HostedAgentTriggerExecutionListOptions) ([]HostedAgentTriggerExecution, string, error) {
	resp, _, err := s.svc.ListExecutions(context.TODO(), triggerID, opt)
	if err != nil {
		return nil, "", err
	}
	out := make([]HostedAgentTriggerExecution, len(resp.Executions))
	for i := range resp.Executions {
		e := resp.Executions[i]
		out[i] = HostedAgentTriggerExecution{HostedAgentTriggerExecution: &e}
	}
	return out, resp.NextPageToken, nil
}

func (s *hostedAgentTriggersService) GetExecution(triggerID, executionID string) (*HostedAgentTriggerExecution, error) {
	e, _, err := s.svc.GetExecution(context.TODO(), triggerID, executionID)
	if err != nil {
		return nil, err
	}
	return &HostedAgentTriggerExecution{HostedAgentTriggerExecution: e}, nil
}

func (s *hostedAgentTriggersService) GetBySession(sessionID string) (*HostedAgentTrigger, error) {
	t, _, err := s.svc.GetBySession(context.TODO(), sessionID)
	if err != nil {
		return nil, err
	}
	return &HostedAgentTrigger{HostedAgentTrigger: t}, nil
}

func (s *hostedAgentTriggersService) ListReusableSessions(opt *godo.HostedAgentReusableSessionListOptions) ([]HostedAgentReusableSession, string, error) {
	resp, _, err := s.svc.ListReusableSessions(context.TODO(), opt)
	if err != nil {
		return nil, "", err
	}
	out := make([]HostedAgentReusableSession, len(resp.Sessions))
	for i := range resp.Sessions {
		sess := resp.Sessions[i]
		out[i] = HostedAgentReusableSession{HostedAgentReusableSession: &sess}
	}
	return out, resp.NextPageToken, nil
}

func (s *hostedAgentTriggersService) ListWebhookProviders() ([]HostedAgentWebhookProvider, error) {
	resp, _, err := s.svc.ListWebhookProviders(context.TODO())
	if err != nil {
		return nil, err
	}
	out := make([]HostedAgentWebhookProvider, len(resp.Providers))
	for i := range resp.Providers {
		p := resp.Providers[i]
		out[i] = HostedAgentWebhookProvider{HostedAgentWebhookProvider: &p}
	}
	return out, nil
}
