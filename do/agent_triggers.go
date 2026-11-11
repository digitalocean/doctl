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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	RotateSecret(triggerID string, gracePeriodSeconds *int) (*HostedAgentTriggerRotateSecretResult, error)
	ListExecutions(triggerID string, opt *godo.HostedAgentTriggerExecutionListOptions) ([]HostedAgentTriggerExecution, string, error)
	GetExecution(triggerID, executionID string) (*HostedAgentTriggerExecution, error)
	Cancel(triggerID, executionID string, force bool) (*HostedAgentTriggerExecution, error)
	GetBySession(sessionID string) (*HostedAgentTrigger, error)
	ListReusableSessions(*godo.HostedAgentReusableSessionListOptions) ([]HostedAgentReusableSession, string, error)
	ListWebhookProviders() ([]HostedAgentWebhookProvider, error)
	// Run fires the trigger now (POST .../triggers/{id}/executions) and returns
	// the pending execution to poll. godo has no method for this route yet, so
	// the call goes through the godo client's request plumbing directly.
	Run(triggerID string) (*HostedAgentTriggerExecution, error)
	// The Canvas set: the triggers API fields godo does not carry yet
	// (keep_session_awake, webhook.events), read and written through the godo
	// client's request plumbing. CreateForCanvas is Create with
	// keep_session_awake set (a reuse trigger that runs while its bound
	// session is awake and leaves it awake, for a session open in Agent
	// Canvas) and, for a webhook, the event filter. GetWithDetails and
	// ListWithDetails are Get and List that also return those fields;
	// UpdateWebhookEvents replaces a webhook trigger's event filter.
	CreateForCanvas(create *godo.HostedAgentTriggerCreateRequest, webhookEvents []string) (*HostedAgentTriggerCreateResult, *HostedAgentTriggerDetails, error)
	GetWithDetails(triggerID string) (*HostedAgentTrigger, *HostedAgentTriggerDetails, error)
	ListWithDetails(pageSize int, pageToken string) ([]HostedAgentTrigger, []HostedAgentTriggerDetails, string, error)
	UpdateWebhookEvents(triggerID string, events []string) (*HostedAgentTrigger, *HostedAgentTriggerDetails, error)
}

// HostedAgentTriggerDetails are the trigger fields godo's type does not carry
// yet, read from the same JSON.
type HostedAgentTriggerDetails struct {
	KeepSessionAwake bool
	// WebhookEvents is the webhook trigger's event filter; empty is every
	// delivery.
	WebhookEvents []string
}

type hostedAgentTriggersService struct {
	svc    godo.HostedAgentTriggersService
	client *godo.Client
}

var _ HostedAgentTriggersService = &hostedAgentTriggersService{}

// NewHostedAgentTriggersService builds a HostedAgentTriggersService from a godo client.
func NewHostedAgentTriggersService(client *godo.Client) HostedAgentTriggersService {
	return &hostedAgentTriggersService{svc: client.HostedAgentTriggers, client: client}
}

// hostedAgentTriggerRunPath is the run-now route, on the same prefix godo
// uses for the other trigger routes; hostedAgentTriggersPath is the create route.
const (
	hostedAgentTriggerRunPath = "/v2/harness-runtime/triggers/%s/executions"
	hostedAgentTriggersPath   = "/v2/harness-runtime/triggers"
)

// triggerDetails decodes the Canvas fields from a trigger's JSON.
type triggerDetailsJSON struct {
	KeepSessionAwake bool `json:"keep_session_awake"`
	Webhook          *struct {
		Events []string `json:"events"`
	} `json:"webhook"`
}

func (j triggerDetailsJSON) details() HostedAgentTriggerDetails {
	d := HostedAgentTriggerDetails{KeepSessionAwake: j.KeepSessionAwake}
	if j.Webhook != nil {
		d.WebhookEvents = j.Webhook.Events
	}
	return d
}

// decodeTriggerWithDetails reads one trigger JSON object twice: as godo's
// type and as the Canvas fields.
func decodeTriggerWithDetails(raw json.RawMessage) (*HostedAgentTrigger, *HostedAgentTriggerDetails, error) {
	var t godo.HostedAgentTrigger
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, nil, err
	}
	var j triggerDetailsJSON
	if err := json.Unmarshal(raw, &j); err != nil {
		return nil, nil, err
	}
	d := j.details()
	return &HostedAgentTrigger{HostedAgentTrigger: &t}, &d, nil
}

func (s *hostedAgentTriggersService) CreateForCanvas(create *godo.HostedAgentTriggerCreateRequest, webhookEvents []string) (*HostedAgentTriggerCreateResult, *HostedAgentTriggerDetails, error) {
	if s.client == nil {
		return nil, nil, fmt.Errorf("triggers: no API client")
	}
	// The godo request as JSON, plus the fields godo does not know.
	raw, err := json.Marshal(create)
	if err != nil {
		return nil, nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, nil, err
	}
	body["keep_session_awake"] = true
	if len(webhookEvents) > 0 {
		wh, _ := body["webhook"].(map[string]any)
		if wh == nil {
			wh = map[string]any{}
		}
		wh["events"] = webhookEvents
		body["webhook"] = wh
	}
	ctx := context.TODO()
	req, err := s.client.NewRequest(ctx, http.MethodPost, hostedAgentTriggersPath, body)
	if err != nil {
		return nil, nil, err
	}
	var root struct {
		Trigger       json.RawMessage `json:"trigger"`
		WebhookSecret string          `json:"webhook_secret"`
	}
	if _, err := s.client.Do(ctx, req, &root); err != nil {
		return nil, nil, err
	}
	result := &HostedAgentTriggerCreateResult{WebhookSecret: root.WebhookSecret}
	if len(root.Trigger) == 0 {
		return result, &HostedAgentTriggerDetails{}, nil
	}
	t, d, err := decodeTriggerWithDetails(root.Trigger)
	if err != nil {
		return nil, nil, err
	}
	result.Trigger = t
	return result, d, nil
}

func (s *hostedAgentTriggersService) GetWithDetails(triggerID string) (*HostedAgentTrigger, *HostedAgentTriggerDetails, error) {
	if s.client == nil {
		return nil, nil, fmt.Errorf("triggers: no API client")
	}
	ctx := context.TODO()
	req, err := s.client.NewRequest(ctx, http.MethodGet, fmt.Sprintf(hostedAgentTriggersPath+"/%s", url.PathEscape(triggerID)), nil)
	if err != nil {
		return nil, nil, err
	}
	var root struct {
		Trigger json.RawMessage `json:"trigger"`
	}
	if _, err := s.client.Do(ctx, req, &root); err != nil {
		return nil, nil, err
	}
	if len(root.Trigger) == 0 {
		return nil, nil, fmt.Errorf("triggers: the API returned no trigger")
	}
	return decodeTriggerWithDetails(root.Trigger)
}

func (s *hostedAgentTriggersService) ListWithDetails(pageSize int, pageToken string) ([]HostedAgentTrigger, []HostedAgentTriggerDetails, string, error) {
	if s.client == nil {
		return nil, nil, "", fmt.Errorf("triggers: no API client")
	}
	q := url.Values{}
	if pageSize > 0 {
		q.Set("page_size", fmt.Sprint(pageSize))
	}
	if pageToken != "" {
		q.Set("page_token", pageToken)
	}
	path := hostedAgentTriggersPath
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	ctx := context.TODO()
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, "", err
	}
	var root struct {
		Triggers      []json.RawMessage `json:"triggers"`
		NextPageToken string            `json:"next_page_token"`
	}
	if _, err := s.client.Do(ctx, req, &root); err != nil {
		return nil, nil, "", err
	}
	triggers := make([]HostedAgentTrigger, 0, len(root.Triggers))
	details := make([]HostedAgentTriggerDetails, 0, len(root.Triggers))
	for _, raw := range root.Triggers {
		t, d, err := decodeTriggerWithDetails(raw)
		if err != nil {
			return nil, nil, "", err
		}
		triggers = append(triggers, *t)
		details = append(details, *d)
	}
	return triggers, details, root.NextPageToken, nil
}

func (s *hostedAgentTriggersService) UpdateWebhookEvents(triggerID string, events []string) (*HostedAgentTrigger, *HostedAgentTriggerDetails, error) {
	if s.client == nil {
		return nil, nil, fmt.Errorf("triggers: no API client")
	}
	if events == nil {
		events = []string{}
	}
	ctx := context.TODO()
	req, err := s.client.NewRequest(ctx, http.MethodPatch, fmt.Sprintf(hostedAgentTriggersPath+"/%s", url.PathEscape(triggerID)),
		map[string]any{"webhook": map[string]any{"events": events}})
	if err != nil {
		return nil, nil, err
	}
	var root struct {
		Trigger json.RawMessage `json:"trigger"`
	}
	if _, err := s.client.Do(ctx, req, &root); err != nil {
		return nil, nil, err
	}
	if len(root.Trigger) == 0 {
		return nil, nil, fmt.Errorf("triggers: the API returned no trigger")
	}
	return decodeTriggerWithDetails(root.Trigger)
}

func (s *hostedAgentTriggersService) Run(triggerID string) (*HostedAgentTriggerExecution, error) {
	if s.client == nil {
		return nil, fmt.Errorf("triggers: no API client")
	}
	ctx := context.TODO()
	req, err := s.client.NewRequest(ctx, http.MethodPost, fmt.Sprintf(hostedAgentTriggerRunPath, url.PathEscape(triggerID)), nil)
	if err != nil {
		return nil, err
	}
	var root struct {
		Execution *godo.HostedAgentTriggerExecution `json:"execution"`
	}
	if _, err := s.client.Do(ctx, req, &root); err != nil {
		return nil, err
	}
	if root.Execution == nil {
		return nil, fmt.Errorf("triggers: the API returned no execution")
	}
	return &HostedAgentTriggerExecution{HostedAgentTriggerExecution: root.Execution}, nil
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
// one. gracePeriodSeconds nil selects the server default; 0 revokes now.
func (s *hostedAgentTriggersService) RotateSecret(triggerID string, gracePeriodSeconds *int) (*HostedAgentTriggerRotateSecretResult, error) {
	// Left nil for the default rotation so no grace_period_seconds parameter
	// goes on the wire at all.
	var opt *godo.HostedAgentTriggerRotateSecretOptions
	if gracePeriodSeconds != nil {
		opt = &godo.HostedAgentTriggerRotateSecretOptions{GracePeriodSeconds: gracePeriodSeconds}
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

func (s *hostedAgentTriggersService) Cancel(triggerID, executionID string, force bool) (*HostedAgentTriggerExecution, error) {
	e, _, err := s.svc.Cancel(context.TODO(), triggerID, executionID, &godo.HostedAgentTriggerCancelExecutionOptions{Force: force})
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
