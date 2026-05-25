package agents

// Session mirrors harness-api Session JSON (snake_case proto JSON).
type Session struct {
	SessionID    string                       `json:"session_id"`
	TeamID       string                       `json:"team_id"`
	AgentKind    string                       `json:"agent_kind"`
	Status       string                       `json:"status"`
	SandboxID    string                       `json:"sandbox_id"`
	RepoHint     string                       `json:"repo_hint"`
	ProviderAuth map[string]string            `json:"provider_auth"`
}

// CredentialRequest is POST /v2/agents/sessions/{id}/credentials (grpc-gateway).
type CredentialRequest struct {
	GitHubToken             string `json:"github_token,omitempty"`
	GitHubOwner             string `json:"github_owner,omitempty"`
	OpenCodeAPIKey          string `json:"opencode_api_key,omitempty"`
	OpenRouterAPIKey        string `json:"openrouter_api_key,omitempty"`
	OpenCodeModel           string `json:"opencode_model,omitempty"`
	DigitalOceanAccessToken string `json:"digitalocean_access_token,omitempty"`
	AnthropicAPIKey         string `json:"anthropic_api_key,omitempty"`
}

// HasAny reports whether any secret field is set (owner alone is not enough).
func (r CredentialRequest) HasAny() bool {
	return r.GitHubToken != "" || r.OpenCodeAPIKey != "" || r.OpenRouterAPIKey != "" ||
		r.OpenCodeModel != "" || r.DigitalOceanAccessToken != "" || r.AnthropicAPIKey != ""
}

// SessionMeta is GET /v2/agents/sessions/{id}/meta.
type SessionMeta struct {
	GitHubLogin string `json:"github_login,omitempty"`
	GitHubOwner string `json:"github_owner,omitempty"`
}

// CreateSessionRequest is the POST /v2/agents/sessions body.
type CreateSessionRequest struct {
	AgentKind string `json:"agent_kind"`
	RepoHint  string `json:"repo_hint,omitempty"`
}

// CreateSessionResponse wraps the created session.
type CreateSessionResponse struct {
	Session Session `json:"session"`
}

// GetSessionResponse wraps a session snapshot.
type GetSessionResponse struct {
	Session Session `json:"session"`
}

// SendInputRequest is the POST input body.
type SendInputRequest struct {
	Text string `json:"text"`
}

// SendInputResponse is returned from SendInput.
type SendInputResponse struct {
	RunID string `json:"run_id"`
}

// ResolveHITLRequest is the POST hitl body.
type ResolveHITLRequest struct {
	Outcome string `json:"outcome"`
	Reason  string `json:"reason,omitempty"`
	Source  string `json:"source,omitempty"`
}

// StartOAuthFlowResponse is returned from StartOAuthFlow.
type StartOAuthFlowResponse struct {
	AuthorizeURL   string `json:"authorize_url"`
	FlowKind       string `json:"flow_kind"`
	DeviceUserCode string `json:"device_user_code,omitempty"`
}

// Event is the canonical event envelope from StreamSession.
type Event struct {
	EventID   string `json:"event_id"`
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`

	RunStarted        *RunStarted        `json:"run_started,omitempty"`
	TokenChunk        *TokenChunk        `json:"token_chunk,omitempty"`
	ToolCallStarted   *ToolCallStarted   `json:"tool_call_started,omitempty"`
	ToolCallCompleted *ToolCallCompleted `json:"tool_call_completed,omitempty"`
	HITLRequested     *HITLRequested     `json:"hitl_requested,omitempty"`
	HITLResolved      *HITLResolved      `json:"hitl_resolved,omitempty"`
	RunCompleted      *RunCompleted      `json:"run_completed,omitempty"`
	RunFailed         *RunFailed         `json:"run_failed,omitempty"`
	SessionUpdated    *SessionUpdated    `json:"session_updated,omitempty"`
}

type RunStarted struct {
	Run              map[string]any `json:"run,omitempty"`
	UserInputPreview string         `json:"user_input_preview,omitempty"`
}

type TokenChunk struct {
	Text string `json:"text"`
}

type ToolCallStarted struct {
	ToolCallID string         `json:"tool_call_id"`
	ToolName   string         `json:"tool_name"`
	Args       map[string]any `json:"args,omitempty"`
}

type ToolCallCompleted struct {
	ToolCallID string    `json:"tool_call_id"`
	Ok         bool      `json:"ok"`
	DurationMs flexInt64 `json:"duration_ms"`
	Summary    string    `json:"summary,omitempty"`
}

type HITLRequested struct {
	Request HITLRequest `json:"request"`
}

type HITLRequest struct {
	RequestID string         `json:"request_id"`
	SessionID string         `json:"session_id"`
	RunID     string         `json:"run_id"`
	Action    string         `json:"action"`
	Details   map[string]any `json:"details,omitempty"`
	Workdir   string         `json:"workdir,omitempty"`
}

type HITLResolved struct {
	Decision map[string]any `json:"decision,omitempty"`
}

type RunCompleted struct{}

type RunFailed struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type SessionUpdated struct {
	SessionDelta  Session  `json:"session_delta,omitempty"`
	ChangedFields []string `json:"changed_fields,omitempty"`
}

// Agent kind constants (proto enum names).
const (
	AgentKindClaudeCode = "AGENT_KIND_CLAUDE_CODE"
	AgentKindOpenCode   = "AGENT_KIND_OPENCODE"
)

// Session status constants.
const (
	SessionStatusProvisioning = "SESSION_STATUS_PROVISIONING"
	SessionStatusReady        = "SESSION_STATUS_READY"
)

// HITL outcomes.
const (
	HITLOutcomeApprove = "HITL_OUTCOME_APPROVE"
	HITLOutcomeReject  = "HITL_OUTCOME_REJECT"
	HITLOutcomeDefer   = "HITL_OUTCOME_DEFER"
)

// Provider auth states.
const (
	ProviderAuthAuthorized = "PROVIDER_AUTH_STATE_AUTHORIZED"
	ProviderAuthPending      = "PROVIDER_AUTH_STATE_PENDING"
)

// ResolutionSourceInline is sent for TUI keystrokes.
const ResolutionSourceInline = "RESOLUTION_SOURCE_INLINE_KEYSTROKE"
