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

	"github.com/digitalocean/godo"
)

// HostedAgentSession wraps godo.HostedAgentSession.
type HostedAgentSession struct {
	*godo.HostedAgentSession
}

// HostedAgentsService is the doctl-facing wrapper around godo.HostedAgentsService.
// It folds (response, err) into err and uses context.TODO() so command runners
// stay terse, matching the pattern used by every other do/* service.
type HostedAgentsService interface {
	CreateSession(*godo.HostedAgentSessionCreateRequest) (*HostedAgentSession, error)
	ListSessions(*godo.HostedAgentSessionListOptions) ([]HostedAgentSession, error)
	GetSession(sessionID string) (*HostedAgentSession, error)
	DestroySession(sessionID string) error
	SendInput(sessionID string, input *godo.HostedAgentSendInputRequest) (*godo.HostedAgentSendInputResponse, error)
	ResolveHITL(sessionID, requestID string, body *godo.HostedAgentResolveHITLRequest) error
	StartOAuthFlow(sessionID, provider string, body *godo.HostedAgentStartOAuthFlowRequest) (*godo.HostedAgentStartOAuthFlowResponse, error)
	StreamSession(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error)
}

type hostedAgentsService struct {
	client *godo.Client
}

var _ HostedAgentsService = &hostedAgentsService{}

// NewHostedAgentsService builds a HostedAgentsService bound to the given godo client.
func NewHostedAgentsService(client *godo.Client) HostedAgentsService {
	return &hostedAgentsService{client: client}
}

func (s *hostedAgentsService) CreateSession(r *godo.HostedAgentSessionCreateRequest) (*HostedAgentSession, error) {
	sess, _, err := s.client.HostedAgents.CreateSession(context.TODO(), r)
	if err != nil {
		return nil, err
	}
	return &HostedAgentSession{HostedAgentSession: sess}, nil
}

func (s *hostedAgentsService) ListSessions(opt *godo.HostedAgentSessionListOptions) ([]HostedAgentSession, error) {
	resp, _, err := s.client.HostedAgents.ListSessions(context.TODO(), opt)
	if err != nil {
		return nil, err
	}
	out := make([]HostedAgentSession, len(resp.Sessions))
	for i := range resp.Sessions {
		sess := resp.Sessions[i]
		out[i] = HostedAgentSession{HostedAgentSession: &sess}
	}
	return out, nil
}

func (s *hostedAgentsService) GetSession(sessionID string) (*HostedAgentSession, error) {
	sess, _, err := s.client.HostedAgents.GetSession(context.TODO(), sessionID)
	if err != nil {
		return nil, err
	}
	return &HostedAgentSession{HostedAgentSession: sess}, nil
}

func (s *hostedAgentsService) DestroySession(sessionID string) error {
	_, err := s.client.HostedAgents.DestroySession(context.TODO(), sessionID)
	return err
}

func (s *hostedAgentsService) SendInput(sessionID string, input *godo.HostedAgentSendInputRequest) (*godo.HostedAgentSendInputResponse, error) {
	resp, _, err := s.client.HostedAgents.SendInput(context.TODO(), sessionID, input)
	return resp, err
}

func (s *hostedAgentsService) ResolveHITL(sessionID, requestID string, body *godo.HostedAgentResolveHITLRequest) error {
	_, err := s.client.HostedAgents.ResolveHITL(context.TODO(), sessionID, requestID, body)
	return err
}

func (s *hostedAgentsService) StartOAuthFlow(sessionID, provider string, body *godo.HostedAgentStartOAuthFlowRequest) (*godo.HostedAgentStartOAuthFlowResponse, error) {
	resp, _, err := s.client.HostedAgents.StartOAuthFlow(context.TODO(), sessionID, provider, body)
	return resp, err
}

// StreamSession opens the SSE stream and returns the typed godo iterator. The
// caller MUST Close the returned stream. ctx is passed straight through so
// cancellation terminates the stream.
func (s *hostedAgentsService) StreamSession(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
	stream, _, err := s.client.HostedAgents.StreamSession(ctx, sessionID, opt)
	return stream, err
}
