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
	// CreateSessionFromManifest POSTs the manifest bytes verbatim with
	// Content-Type: application/x-yaml. Server owns schema validation.
	CreateSessionFromManifest(manifest []byte) (*HostedAgentSession, error)
	ListSessions(*godo.HostedAgentSessionListOptions) ([]HostedAgentSession, string, error)
	GetSession(sessionID string) (*HostedAgentSession, error)
	DestroySession(sessionID string) error
	PauseSession(sessionID string) error
	ResumeSession(sessionID string) error
	SendInput(sessionID string, input *godo.HostedAgentSendInputRequest) (*godo.HostedAgentSendInputResponse, error)
	// RelayRequest forwards one native agent-protocol request frame to the
	// session's agent and returns its reply verbatim. Blocks on the agent, so
	// it takes a context of its own rather than the package-wide
	// context.TODO(): the caller is a proxy answering a client that is itself
	// blocked, and it needs to be able to give up.
	RelayRequest(ctx context.Context, sessionID string, body *godo.HostedAgentRelayRequest) (*godo.HostedAgentRelayResponse, error)
	ResolveHITL(sessionID, requestID string, body *godo.HostedAgentResolveHITLRequest) error
	StreamSession(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error)
	// Workspace file transfer APIs (/workspace/transfers). Used for all upload/download sizes.
	CreateWorkspaceTransfer(sessionID string, create *godo.HostedAgentWorkspaceTransferCreateRequest) (*godo.HostedAgentWorkspaceTransfer, error)
	CreateWorkspaceTransferPartUploadURLs(sessionID, transferID string, input *godo.HostedAgentWorkspaceTransferPartUploadURLsRequest) (*godo.HostedAgentWorkspaceTransferPartUploadURLs, error)
	CommitWorkspaceTransfer(sessionID, transferID string, input *godo.HostedAgentWorkspaceTransferCommitRequest) (*godo.HostedAgentWorkspaceTransfer, error)
	GetWorkspaceTransfer(sessionID, transferID string) (*godo.HostedAgentWorkspaceTransfer, error)
	CancelWorkspaceTransfer(sessionID, transferID string, input *godo.HostedAgentWorkspaceTransferCancelRequest) (*godo.HostedAgentWorkspaceTransferCancelResponse, error)
}

type hostedAgentsService struct {
	client *godo.Client
}

var _ HostedAgentsService = &hostedAgentsService{}

// NewHostedAgentsService builds a HostedAgentsService bound to the given godo client.
func NewHostedAgentsService(client *godo.Client) HostedAgentsService {
	return &hostedAgentsService{client: client}
}

func (s *hostedAgentsService) CreateSessionFromManifest(manifest []byte) (*HostedAgentSession, error) {
	sess, _, err := s.client.HostedAgents.CreateSessionFromManifest(context.TODO(), manifest)
	if err != nil {
		return nil, err
	}
	return &HostedAgentSession{HostedAgentSession: sess}, nil
}

func (s *hostedAgentsService) ListSessions(opt *godo.HostedAgentSessionListOptions) ([]HostedAgentSession, string, error) {
	resp, _, err := s.client.HostedAgents.ListSessions(context.TODO(), opt)
	if err != nil {
		return nil, "", err
	}
	out := make([]HostedAgentSession, len(resp.Sessions))
	for i := range resp.Sessions {
		sess := resp.Sessions[i]
		out[i] = HostedAgentSession{HostedAgentSession: &sess}
	}
	return out, resp.NextPageToken, nil
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

func (s *hostedAgentsService) PauseSession(sessionID string) error {
	_, err := s.client.HostedAgents.PauseSession(context.TODO(), sessionID)
	return err
}

func (s *hostedAgentsService) ResumeSession(sessionID string) error {
	_, err := s.client.HostedAgents.ResumeSession(context.TODO(), sessionID)
	return err
}

func (s *hostedAgentsService) SendInput(sessionID string, input *godo.HostedAgentSendInputRequest) (*godo.HostedAgentSendInputResponse, error) {
	resp, _, err := s.client.HostedAgents.SendInput(context.TODO(), sessionID, input)
	return resp, err
}

func (s *hostedAgentsService) RelayRequest(ctx context.Context, sessionID string, body *godo.HostedAgentRelayRequest) (*godo.HostedAgentRelayResponse, error) {
	resp, _, err := s.client.HostedAgents.RelayRequest(ctx, sessionID, body)
	return resp, err
}

func (s *hostedAgentsService) ResolveHITL(sessionID, requestID string, body *godo.HostedAgentResolveHITLRequest) error {
	_, err := s.client.HostedAgents.ResolveHITL(context.TODO(), sessionID, requestID, body)
	return err
}

// StreamSession opens the SSE stream and returns the typed godo iterator. The
// caller MUST Close the returned stream. ctx is passed straight through so
// cancellation terminates the stream.
func (s *hostedAgentsService) StreamSession(ctx context.Context, sessionID string, opt *godo.HostedAgentSessionStreamOptions) (*godo.HostedAgentSessionStream, error) {
	stream, _, err := s.client.HostedAgents.StreamSession(ctx, sessionID, opt)
	return stream, err
}

func (s *hostedAgentsService) CreateWorkspaceTransfer(sessionID string, create *godo.HostedAgentWorkspaceTransferCreateRequest) (*godo.HostedAgentWorkspaceTransfer, error) {
	xfer, _, err := s.client.HostedAgents.CreateWorkspaceTransfer(context.TODO(), sessionID, create)
	return xfer, err
}

func (s *hostedAgentsService) CreateWorkspaceTransferPartUploadURLs(sessionID, transferID string, input *godo.HostedAgentWorkspaceTransferPartUploadURLsRequest) (*godo.HostedAgentWorkspaceTransferPartUploadURLs, error) {
	part, _, err := s.client.HostedAgents.CreateWorkspaceTransferPartUploadURLs(context.TODO(), sessionID, transferID, input)
	return part, err
}

func (s *hostedAgentsService) CommitWorkspaceTransfer(sessionID, transferID string, input *godo.HostedAgentWorkspaceTransferCommitRequest) (*godo.HostedAgentWorkspaceTransfer, error) {
	xfer, _, err := s.client.HostedAgents.CommitWorkspaceTransfer(context.TODO(), sessionID, transferID, input)
	return xfer, err
}

func (s *hostedAgentsService) GetWorkspaceTransfer(sessionID, transferID string) (*godo.HostedAgentWorkspaceTransfer, error) {
	xfer, _, err := s.client.HostedAgents.GetWorkspaceTransfer(context.TODO(), sessionID, transferID)
	return xfer, err
}

func (s *hostedAgentsService) CancelWorkspaceTransfer(sessionID, transferID string, input *godo.HostedAgentWorkspaceTransferCancelRequest) (*godo.HostedAgentWorkspaceTransferCancelResponse, error) {
	resp, _, err := s.client.HostedAgents.CancelWorkspaceTransfer(context.TODO(), sessionID, transferID, input)
	return resp, err
}
