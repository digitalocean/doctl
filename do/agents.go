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
	"fmt"
	"net/http"
	"net/url"

	"github.com/digitalocean/godo"
)

// HostedAgentSession wraps godo.HostedAgentSession.
type HostedAgentSession struct {
	*godo.HostedAgentSession
}

// hostedAgentProviderAuthBasePath is the harness-api team-scoped provider
// connect surface (Barbican connectlinks). Routes:
//
//	POST /v2/agents/auth/{provider}        -> start (or resume) the connect flow
//	GET  /v2/agents/auth/{provider}/poll   -> poll a pending connect link
//
// The godo HostedAgents service does not expose these yet, so the calls are
// issued directly against the shared godo client here.
const hostedAgentProviderAuthBasePath = "/v2/agents/auth/%s"

// HostedAgentProviderAuthStart is the response to POST /v2/agents/auth/{provider}.
// Status is "pending" when the user must still authorize in a browser, or
// "success" when the team is already connected (ConnectURL/PollURL empty). The
// authorization handle is never exposed: tokens are exchanged server-side at
// session time.
type HostedAgentProviderAuthStart struct {
	Provider         string `json:"provider"`
	Status           string `json:"status"`
	ConnectURL       string `json:"connect_url,omitempty"`
	PollURL          string `json:"poll_url,omitempty"`
	VerificationCode string `json:"verification_code,omitempty"`
	ExpiresAt        string `json:"expires_at,omitempty"`
}

// HostedAgentProviderAuthPoll is the response to GET
// /v2/agents/auth/{provider}/poll. It reports only whether authorization has
// completed; no secret is returned.
type HostedAgentProviderAuthPoll struct {
	Provider  string `json:"provider"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// HostedAgentsService is the doctl-facing wrapper around godo.HostedAgentsService.
// It folds (response, err) into err and uses context.TODO() so command runners
// stay terse, matching the pattern used by every other do/* service.
type HostedAgentsService interface {
	// CreateSessionFromManifest POSTs the manifest bytes verbatim with
	// Content-Type: application/x-yaml. Server owns schema validation.
	// For OpenAI sandbox-provider sessions (adapter codex-agentapi), pass
	// OpenAISessionID in opt so harness-api persists openai_session_id
	// (?openai_session_id=). opt may be nil.
	CreateSessionFromManifest(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*HostedAgentSession, error)
	ListSessions(*godo.HostedAgentSessionListOptions) ([]HostedAgentSession, string, error)
	GetSession(sessionID string) (*HostedAgentSession, error)
	DestroySession(sessionID string) error
	PauseSession(sessionID string) error
	ResumeSession(sessionID string) error
	SendInput(sessionID string, input *godo.HostedAgentSendInputRequest) (*godo.HostedAgentSendInputResponse, error)
	ResolveHITL(sessionID, requestID string, body *godo.HostedAgentResolveHITLRequest) error
	// StartProviderAuth begins (or resumes) the team-scoped connect flow for an
	// external provider (e.g. "github"). The team is taken from the
	// authenticated principal server-side; there is no request body.
	StartProviderAuth(provider string) (*HostedAgentProviderAuthStart, error)
	// PollProviderAuth checks whether a pending connect link has been authorized.
	// pollURL is the poll_url returned by StartProviderAuth.
	PollProviderAuth(provider, pollURL string) (*HostedAgentProviderAuthPoll, error)
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

func (s *hostedAgentsService) CreateSessionFromManifest(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*HostedAgentSession, error) {
	sess, _, err := s.client.HostedAgents.CreateSessionFromManifest(context.TODO(), manifest, opt)
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

func (s *hostedAgentsService) ResolveHITL(sessionID, requestID string, body *godo.HostedAgentResolveHITLRequest) error {
	_, err := s.client.HostedAgents.ResolveHITL(context.TODO(), sessionID, requestID, body)
	return err
}

func (s *hostedAgentsService) StartProviderAuth(provider string) (*HostedAgentProviderAuthStart, error) {
	path := fmt.Sprintf(hostedAgentProviderAuthBasePath, provider)
	// The server derives the team from the authenticated principal; an empty
	// JSON object keeps a well-formed body on the POST.
	req, err := s.client.NewRequest(context.TODO(), http.MethodPost, path, struct{}{})
	if err != nil {
		return nil, err
	}
	root := new(HostedAgentProviderAuthStart)
	if _, err := s.client.Do(context.TODO(), req, root); err != nil {
		return nil, err
	}
	return root, nil
}

func (s *hostedAgentsService) PollProviderAuth(provider, pollURL string) (*HostedAgentProviderAuthPoll, error) {
	path := fmt.Sprintf(hostedAgentProviderAuthBasePath+"/poll", provider)
	q := url.Values{}
	q.Set("poll_url", pollURL)
	path += "?" + q.Encode()
	req, err := s.client.NewRequest(context.TODO(), http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	root := new(HostedAgentProviderAuthPoll)
	if _, err := s.client.Do(context.TODO(), req, root); err != nil {
		return nil, err
	}
	return root, nil
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
