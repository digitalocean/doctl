package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to harness-api REST (grpc-gateway).
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a harness API client.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 0, // streaming has no timeout on client; per-request below
		},
	}
}

func (c *Client) url(path string) string {
	return c.BaseURL + path
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.url(path), rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(b))
		if resp.StatusCode == http.StatusNotFound && strings.Contains(msg, "page not found") {
			return fmt.Errorf("%s %s: %s — check --harness-url points at the REST gateway (port 8080), not the health port (8081)", method, path, msg)
		}
		if strings.Contains(msg, "error reading server preface") {
			return fmt.Errorf("%s %s: harness-api gRPC loopback TLS mismatch — redeploy/restart the pod with a current harness-api image (see harness-api/cmd/harness-api/main.go loopback_insecure)", method, path)
		}
		if strings.Contains(strings.ToLower(msg), "method not allowed") {
			return fmt.Errorf("%s %s: %s — redeploy harness-api (credentials route fix); try: curl -sk %s%s", method, path, msg, c.BaseURL, path)
		}
		return fmt.Errorf("%s %s: %s", method, path, msg)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Ping checks that the harness REST API is reachable (GET /v2/agents/sessions).
func (c *Client) Ping(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodGet, "/v2/agents/sessions", nil, &ListSessionsResponse{})
}

// ListSessionsResponse is GET /v2/agents/sessions.
type ListSessionsResponse struct {
	Sessions []Session `json:"sessions"`
}

// CreateSession provisions a new agent session.
func (c *Client) CreateSession(ctx context.Context, agentKind, repoHint string) (*Session, error) {
	var resp CreateSessionResponse
	err := c.doJSON(ctx, http.MethodPost, "/v2/agents/sessions", CreateSessionRequest{
		AgentKind: agentKind,
		RepoHint:  repoHint,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Session, nil
}

// GetSessionMeta returns PoC session metadata (github_login, etc.).
func (c *Client) GetSessionMeta(ctx context.Context, sessionID string) (*SessionMeta, error) {
	var meta SessionMeta
	path := fmt.Sprintf("/v2/internal/agents/sessions/%s/meta", url.PathEscape(sessionID))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// GetSession returns a session snapshot.
func (c *Client) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	var resp GetSessionResponse
	path := fmt.Sprintf("/v2/agents/sessions/%s", url.PathEscape(sessionID))
	err := c.doJSON(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Session, nil
}

// SendInput sends a user message.
func (c *Client) SendInput(ctx context.Context, sessionID, text string) (string, error) {
	var resp SendInputResponse
	path := fmt.Sprintf("/v2/agents/sessions/%s/input", url.PathEscape(sessionID))
	err := c.doJSON(ctx, http.MethodPost, path, SendInputRequest{Text: text}, &resp)
	if err != nil {
		return "", err
	}
	return resp.RunID, nil
}

// ResolveHITL submits a HITL decision.
func (c *Client) ResolveHITL(ctx context.Context, sessionID, requestID, outcome string) error {
	path := fmt.Sprintf("/v2/agents/sessions/%s/hitl/%s", url.PathEscape(sessionID), url.PathEscape(requestID))
	return c.doJSON(ctx, http.MethodPost, path, ResolveHITLRequest{
		Outcome: outcome,
		Source:  ResolutionSourceInline,
	}, nil)
}

// StartOAuthFlow begins GitHub OAuth for a session.
func (c *Client) StartOAuthFlow(ctx context.Context, sessionID string) (*StartOAuthFlowResponse, error) {
	var resp StartOAuthFlowResponse
	// grpc-gateway maps {provider} via runtime.Enum — use proto enum name.
	path := fmt.Sprintf("/v2/agents/sessions/%s/oauth/OAUTH_PROVIDER_GITHUB", url.PathEscape(sessionID))
	err := c.doJSON(ctx, http.MethodPost, path, map[string]any{}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// StreamSession opens the event stream. Caller must close resp.Body.
func (c *Client) StreamSession(ctx context.Context, sessionID string, replayFrom string, replayOnly bool) (*http.Response, error) {
	q := url.Values{}
	if replayFrom != "" {
		q.Set("replay_from", replayFrom)
	}
	if replayOnly {
		q.Set("replay_only", "true")
	}
	path := fmt.Sprintf("/v2/agents/sessions/%s/stream", url.PathEscape(sessionID))
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 0}
	return client.Do(req)
}

// SetCredentials uploads session secrets to harness-api (PoC HTTP endpoint).
func (c *Client) SetCredentials(ctx context.Context, sessionID string, req CredentialRequest) error {
	path := fmt.Sprintf("/v2/agents/sessions/%s/credentials", url.PathEscape(sessionID))
	return c.doJSON(ctx, http.MethodPost, path, req, nil)
}

// WaitForReady polls until session status is READY.
func (c *Client) WaitForReady(ctx context.Context, sessionID string, interval time.Duration) (*Session, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		sess, err := c.GetSession(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if sess.Status == SessionStatusReady {
			return sess, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// AgentKindFromCLI maps doctl --agent values to proto enum names.
func AgentKindFromCLI(name string) (string, error) {
	switch strings.ToLower(name) {
	case "claude-code", "claude_code", "claude":
		return AgentKindClaudeCode, nil
	case "opencode", "open-code":
		return AgentKindOpenCode, nil
	default:
		return "", fmt.Errorf("unsupported agent %q (try claude-code or opencode)", name)
	}
}
