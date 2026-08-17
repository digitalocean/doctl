package godo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	hostedAgentManifestMediaType  = "application/x-yaml"
	hostedAgentWorkspaceMediaType = "application/octet-stream"

	hostedAgentsSessionsBasePath                    = "/v2/agents/sessions"
	hostedAgentSessionByIDPath                      = hostedAgentsSessionsBasePath + "/%s"
	hostedAgentSessionStreamPath                    = hostedAgentSessionByIDPath + "/stream"
	hostedAgentSessionInputPath                     = hostedAgentSessionByIDPath + "/input"
	hostedAgentSessionRequestPath                   = hostedAgentSessionByIDPath + "/request"
	hostedAgentSessionHITLPath                      = hostedAgentSessionByIDPath + "/hitl/%s"
	hostedAgentSessionSandboxExecPath               = hostedAgentSessionByIDPath + "/sandbox/exec"
	hostedAgentSessionPausePath                     = hostedAgentSessionByIDPath + "/pause"
	hostedAgentSessionResumePath                    = hostedAgentSessionByIDPath + "/resume"
	hostedAgentSessionWorkspaceUploadPath           = hostedAgentSessionByIDPath + "/workspace/upload"
	hostedAgentSessionWorkspaceDownloadPath         = hostedAgentSessionByIDPath + "/workspace/download"
	hostedAgentSessionWorkspaceTransfersPath        = hostedAgentSessionByIDPath + "/workspace/transfers"
	hostedAgentSessionWorkspaceTransferByIDPath     = hostedAgentSessionWorkspaceTransfersPath + "/%s"
	hostedAgentSessionWorkspaceTransferPartURLsPath = hostedAgentSessionWorkspaceTransferByIDPath + "/part-upload-urls"
	hostedAgentSessionWorkspaceTransferCommitPath   = hostedAgentSessionWorkspaceTransferByIDPath + "/commit"
	hostedAgentSessionWorkspaceTransferCancelPath   = hostedAgentSessionWorkspaceTransferByIDPath + "/cancel"

	// Team-scoped external-provider (e.g. GitHub) OAuth connect flow.
	hostedAgentProviderAuthPath     = "/v2/agents/auth/%s"
	hostedAgentProviderAuthPollPath = hostedAgentProviderAuthPath + "/poll"

	hostedAgentSessionCheckpointsPath        = hostedAgentSessionByIDPath + "/checkpoints"
	hostedAgentSessionCheckpointByIDPath     = hostedAgentSessionCheckpointsPath + "/%s"
	hostedAgentSessionCheckpointRollbackPath = hostedAgentSessionCheckpointByIDPath + "/rollback"
	hostedAgentSessionForkPath               = hostedAgentSessionByIDPath + "/fork"

	// HostedAgentForkMaxCount is the v1 cap on children created by one fork call.
	HostedAgentForkMaxCount = 4

	workspaceContentSHA256Header = "X-Content-Sha256"
	workspaceIsArchiveHeader     = "X-Workspace-Is-Archive"
	workspaceSizeBytesHeader     = "X-Workspace-Size-Bytes"

	// workspaceDownloadFooter is appended by OHS after a successful download
	// payload so integrity survives intermediaries that strip HTTP trailers
	// (e.g. Cloudflare). Format: DOWSSHA1 + 64 lowercase hex + '\n' = 73 bytes.
	workspaceDownloadFooterPrefix = "DOWSSHA1"
	workspaceDownloadFooterLen    = len(workspaceDownloadFooterPrefix) + 64 + 1
)

// HostedAgentsService exposes the DigitalOcean Hosted Agents session API
// (HarnessAPI from harness.proto). Routes live under /v2/agents/sessions.
type HostedAgentsService interface {
	// CreateSession provisions a session using the legacy JSON body (agent_kind,
	// repo_hint, idle_timeout_seconds). Prefer CreateSessionFromManifest for
	// agents.digitalocean.com/v1alpha1 Agent manifests.
	CreateSession(context.Context, *HostedAgentSessionCreateRequest) (*HostedAgentSession, *Response, error)
	// CreateSessionFromManifest uploads a customer Agent manifest YAML document
	// (Content-Type: application/x-yaml). For OpenAI sandbox-provider sessions
	// (adapter codex-agentapi), pass OpenAISessionID in opt so harness-api can
	// persist openai_session_id for attach correlation (?openai_session_id=).
	// doctl resolves ${...} placeholders client-side before calling this API;
	// there is no server-side variables map.
	CreateSessionFromManifest(context.Context, []byte, *HostedAgentManifestCreateOptions) (*HostedAgentSession, *Response, error)
	ListSessions(context.Context, *HostedAgentSessionListOptions) (*HostedAgentSessionsListResponse, *Response, error)
	GetSession(context.Context, string) (*HostedAgentSession, *Response, error)
	DestroySession(context.Context, string) (*Response, error)
	PauseSession(context.Context, string) (*Response, error)
	ResumeSession(context.Context, string) (*Response, error)
	StreamSession(context.Context, string, *HostedAgentSessionStreamOptions) (*HostedAgentSessionStream, *Response, error)
	SendInput(context.Context, string, *HostedAgentSendInputRequest) (*HostedAgentSendInputResponse, *Response, error)
	RelayRequest(context.Context, string, *HostedAgentRelayRequest) (*HostedAgentRelayResponse, *Response, error)
	ResolveHITL(context.Context, string, string, *HostedAgentResolveHITLRequest) (*Response, error)

	// StartProviderAuth begins (or resumes) the team-scoped connect flow for an
	// external provider (e.g. "github"). The team is derived from the
	// authenticated principal; there is no request body.
	StartProviderAuth(context.Context, string) (*HostedAgentProviderAuthStart, *Response, error)
	// PollProviderAuth reports whether a pending connect link has been
	// authorized. pollURL is the PollURL returned by StartProviderAuth.
	PollProviderAuth(context.Context, string, string) (*HostedAgentProviderAuthPoll, *Response, error)
	ExecInSandbox(context.Context, string, *HostedAgentSandboxExecRequest) (*HostedAgentSandboxExecResponse, *Response, error)
	UploadWorkspace(context.Context, string, *HostedAgentWorkspaceUploadRequest) (*HostedAgentWorkspaceUploadResponse, *Response, error)
	DownloadWorkspace(context.Context, string, *HostedAgentWorkspaceDownloadRequest) (*HostedAgentWorkspaceDownload, *Response, error)

	// Large-file (>~50 MiB) staged workspace transfer APIs. Streaming
	// UploadWorkspace / DownloadWorkspace remain for smaller payloads.
	CreateWorkspaceTransfer(context.Context, string, *HostedAgentWorkspaceTransferCreateRequest) (*HostedAgentWorkspaceTransfer, *Response, error)
	CreateWorkspaceTransferPartUploadURLs(context.Context, string, string, *HostedAgentWorkspaceTransferPartUploadURLsRequest) (*HostedAgentWorkspaceTransferPartUploadURLs, *Response, error)
	CommitWorkspaceTransfer(context.Context, string, string, *HostedAgentWorkspaceTransferCommitRequest) (*HostedAgentWorkspaceTransfer, *Response, error)
	GetWorkspaceTransfer(context.Context, string, string) (*HostedAgentWorkspaceTransfer, *Response, error)
	CancelWorkspaceTransfer(context.Context, string, string, *HostedAgentWorkspaceTransferCancelRequest) (*HostedAgentWorkspaceTransferCancelResponse, *Response, error)

	// Checkpoint / fork / rollback (session save points).
	CreateCheckpoint(context.Context, string, *HostedAgentCheckpointCreateRequest) (*HostedAgentCheckpoint, *Response, error)
	ListCheckpoints(context.Context, string, *HostedAgentCheckpointListOptions) (*HostedAgentCheckpointsListResponse, *Response, error)
	GetCheckpoint(context.Context, string, string) (*HostedAgentCheckpoint, *Response, error)
	DeleteCheckpoint(context.Context, string, string) (*HostedAgentCheckpointDeleteResponse, *Response, error)
	ForkSession(context.Context, string, *HostedAgentForkSessionRequest) (*HostedAgentForkSessionResponse, *Response, error)
	RollbackToCheckpoint(context.Context, string, string) (*HostedAgentSession, *Response, error)
}

// HostedAgentsServiceOp handles communication with Hosted Agents session methods.
type HostedAgentsServiceOp struct {
	client *Client
}

var _ HostedAgentsService = &HostedAgentsServiceOp{}

// HostedAgentKind identifies the agent runtime provisioned for a session.
type HostedAgentKind string

const (
	HostedAgentKindUnspecified HostedAgentKind = "AGENT_KIND_UNSPECIFIED"
	HostedAgentKindClaudeCode  HostedAgentKind = "AGENT_KIND_CLAUDE_CODE"
	HostedAgentKindOpenCode    HostedAgentKind = "AGENT_KIND_OPENCODE"
	HostedAgentKindCodexCLI    HostedAgentKind = "AGENT_KIND_CODEX_CLI"
	HostedAgentKindCursorCLI   HostedAgentKind = "AGENT_KIND_CURSOR_CLI"
	// HostedAgentKindOpenAICodex is the OpenAI Agents API sandbox-provider kind
	// (adapter codex-agentapi). Attach bridges to OpenAI, not DO's SSE stream.
	HostedAgentKindOpenAICodex HostedAgentKind = "AGENT_KIND_OPENAI_CODEX"
	HostedAgentKindNone        HostedAgentKind = "AGENT_KIND_NONE"
	HostedAgentKindCustom      HostedAgentKind = "AGENT_KIND_CUSTOM"
)

// HostedAgentSessionStatus is the lifecycle status of a hosted agent session.
type HostedAgentSessionStatus string

const (
	HostedAgentSessionStatusUnspecified  HostedAgentSessionStatus = "SESSION_STATUS_UNSPECIFIED"
	HostedAgentSessionStatusProvisioning HostedAgentSessionStatus = "SESSION_STATUS_PROVISIONING"
	HostedAgentSessionStatusReady        HostedAgentSessionStatus = "SESSION_STATUS_READY"
	HostedAgentSessionStatusDetached     HostedAgentSessionStatus = "SESSION_STATUS_DETACHED"
	HostedAgentSessionStatusDestroying   HostedAgentSessionStatus = "SESSION_STATUS_DESTROYING"
	HostedAgentSessionStatusDestroyed    HostedAgentSessionStatus = "SESSION_STATUS_DESTROYED"
	HostedAgentSessionStatusFailed       HostedAgentSessionStatus = "SESSION_STATUS_FAILED"
	HostedAgentSessionStatusPaused       HostedAgentSessionStatus = "SESSION_STATUS_PAUSED"
)

// HostedAgentProviderAuthState tracks OAuth authorization for an external provider.
type HostedAgentProviderAuthState string

const (
	HostedAgentProviderAuthStateUnspecified HostedAgentProviderAuthState = "PROVIDER_AUTH_STATE_UNSPECIFIED"
	HostedAgentProviderAuthStateNone        HostedAgentProviderAuthState = "PROVIDER_AUTH_STATE_NONE"
	HostedAgentProviderAuthStatePending     HostedAgentProviderAuthState = "PROVIDER_AUTH_STATE_PENDING"
	HostedAgentProviderAuthStateAuthorized  HostedAgentProviderAuthState = "PROVIDER_AUTH_STATE_AUTHORIZED"
	HostedAgentProviderAuthStateExpired     HostedAgentProviderAuthState = "PROVIDER_AUTH_STATE_EXPIRED"
)

// HostedAgentRunState is the lifecycle state of a single agent run.
type HostedAgentRunState string

const (
	HostedAgentRunStateUnspecified  HostedAgentRunState = "RUN_STATE_UNSPECIFIED"
	HostedAgentRunStateQueued       HostedAgentRunState = "RUN_STATE_QUEUED"
	HostedAgentRunStateRunning      HostedAgentRunState = "RUN_STATE_RUNNING"
	HostedAgentRunStateAwaitingHITL HostedAgentRunState = "RUN_STATE_AWAITING_HITL"
	HostedAgentRunStatePaused       HostedAgentRunState = "RUN_STATE_PAUSED"
	HostedAgentRunStateCompleted    HostedAgentRunState = "RUN_STATE_COMPLETED"
	HostedAgentRunStateFailed       HostedAgentRunState = "RUN_STATE_FAILED"
)

// HostedAgentHITLActionKind classifies a human-in-the-loop approval request.
type HostedAgentHITLActionKind string

const (
	HostedAgentHITLActionUnspecified               HostedAgentHITLActionKind = "HITL_ACTION_KIND_UNSPECIFIED"
	HostedAgentHITLActionBash                      HostedAgentHITLActionKind = "HITL_ACTION_BASH"
	HostedAgentHITLActionFileWriteOutsideWorkspace HostedAgentHITLActionKind = "HITL_ACTION_FILE_WRITE_OUTSIDE_WORKSPACE"
	HostedAgentHITLActionGitHubCommitPush          HostedAgentHITLActionKind = "HITL_ACTION_GITHUB_COMMIT_PUSH"
	HostedAgentHITLActionGitHubCreatePR            HostedAgentHITLActionKind = "HITL_ACTION_GITHUB_CREATE_PR"
	HostedAgentHITLActionGitHubBranchDelete        HostedAgentHITLActionKind = "HITL_ACTION_GITHUB_BRANCH_DELETE"
	HostedAgentHITLActionGitHubForcePush           HostedAgentHITLActionKind = "HITL_ACTION_GITHUB_FORCE_PUSH"
)

// HostedAgentHITLOutcome is the user's decision on a HITL request.
type HostedAgentHITLOutcome string

const (
	HostedAgentHITLOutcomeUnspecified HostedAgentHITLOutcome = "HITL_OUTCOME_UNSPECIFIED"
	HostedAgentHITLOutcomeApprove     HostedAgentHITLOutcome = "HITL_OUTCOME_APPROVE"
	HostedAgentHITLOutcomeReject      HostedAgentHITLOutcome = "HITL_OUTCOME_REJECT"
	HostedAgentHITLOutcomeDefer       HostedAgentHITLOutcome = "HITL_OUTCOME_DEFER"
)

// HostedAgentResolutionSource records where a HITL decision originated.
type HostedAgentResolutionSource string

const (
	HostedAgentResolutionSourceUnspecified     HostedAgentResolutionSource = "RESOLUTION_SOURCE_UNSPECIFIED"
	HostedAgentResolutionSourceInlineKeystroke HostedAgentResolutionSource = "RESOLUTION_SOURCE_INLINE_KEYSTROKE"
	HostedAgentResolutionSourceOutOfBand       HostedAgentResolutionSource = "RESOLUTION_SOURCE_OUT_OF_BAND"
)

// HostedAgentRunFailureCode classifies a failed run.
type HostedAgentRunFailureCode string

const (
	HostedAgentRunFailureCodeUnspecified    HostedAgentRunFailureCode = "RUN_FAILURE_CODE_UNSPECIFIED"
	HostedAgentRunFailureCodeModelError     HostedAgentRunFailureCode = "RUN_FAILURE_CODE_MODEL_ERROR"
	HostedAgentRunFailureCodeModelTimeout   HostedAgentRunFailureCode = "RUN_FAILURE_CODE_MODEL_TIMEOUT"
	HostedAgentRunFailureCodeToolError      HostedAgentRunFailureCode = "RUN_FAILURE_CODE_TOOL_ERROR"
	HostedAgentRunFailureCodeSandboxLost    HostedAgentRunFailureCode = "RUN_FAILURE_CODE_SANDBOX_LOST"
	HostedAgentRunFailureCodeHITLRejected   HostedAgentRunFailureCode = "RUN_FAILURE_CODE_HITL_REJECTED"
	HostedAgentRunFailureCodeBudgetExceeded HostedAgentRunFailureCode = "RUN_FAILURE_CODE_BUDGET_EXCEEDED"
	HostedAgentRunFailureCodeInternal       HostedAgentRunFailureCode = "RUN_FAILURE_CODE_INTERNAL"
)

// HostedAgentEventKind is the SSE event discriminator for session stream
// payloads. The values are the canonical SPI event type names (dot-separated)
// emitted on the wire's `type` field — NOT proto enum names. They mirror the
// spi.EventType constants owned by the hosted-agents stack.
type HostedAgentEventKind string

const (
	HostedAgentEventKindUnspecified          HostedAgentEventKind = ""
	HostedAgentEventKindRunStarted           HostedAgentEventKind = "run.started"
	HostedAgentEventKindTokenChunk           HostedAgentEventKind = "run.token_delta"
	HostedAgentEventKindToolCallStarted      HostedAgentEventKind = "run.tool_call_started"
	HostedAgentEventKindToolCallCompleted    HostedAgentEventKind = "run.tool_call_completed"
	HostedAgentEventKindHITLRequested        HostedAgentEventKind = "run.human_input_requested"
	HostedAgentEventKindHITLResolved         HostedAgentEventKind = "run.human_input_received"
	HostedAgentEventKindRunCompleted         HostedAgentEventKind = "run.completed"
	HostedAgentEventKindRunFailed            HostedAgentEventKind = "run.failed"
	HostedAgentEventKindRunPaused            HostedAgentEventKind = "run.paused"
	HostedAgentEventKindRunResumed           HostedAgentEventKind = "run.resumed"
	HostedAgentEventKindSessionUpdated       HostedAgentEventKind = "session.updated"
	HostedAgentEventKindRunStateCheckpointed HostedAgentEventKind = "run.state_checkpointed"
	HostedAgentEventKindRunHandoff           HostedAgentEventKind = "run.handoff"
	HostedAgentEventKindRunUsageRecorded     HostedAgentEventKind = "run.usage_recorded"
	HostedAgentEventKindRunSandboxAllocated  HostedAgentEventKind = "run.sandbox_allocated"
	HostedAgentEventKindRunSandboxReleased   HostedAgentEventKind = "run.sandbox_released"
	HostedAgentEventKindRunCostAccrued       HostedAgentEventKind = "run.cost_accrued"
	HostedAgentEventKindRunLog               HostedAgentEventKind = "run.log"
)

// HostedAgentSessionOriginProduct identifies the product workflow that created
// a session. Wire values match harness SessionOrigin.product.
type HostedAgentSessionOriginProduct string

const (
	HostedAgentSessionOriginProductDirect     HostedAgentSessionOriginProduct = "direct"
	HostedAgentSessionOriginProductSimulation HostedAgentSessionOriginProduct = "simulation"
	HostedAgentSessionOriginProductEvaluation HostedAgentSessionOriginProduct = "evaluation"
)

// HostedAgentSessionOriginRequest is the product-provenance claim on create.
// Omission creates a verified direct session. Simulation and evaluation require
// resource_id; direct forbids it. verified is server-assigned and must not be
// sent on create (see HostedAgentSessionOrigin).
type HostedAgentSessionOriginRequest struct {
	Product    HostedAgentSessionOriginProduct `json:"product"`
	ResourceID string                          `json:"resource_id,omitempty"`
}

// HostedAgentSessionOrigin is server-returned product-workflow provenance.
// Verified is true for direct sessions and for product claims established
// through a trusted adapter.
type HostedAgentSessionOrigin struct {
	Product    HostedAgentSessionOriginProduct `json:"product"`
	ResourceID string                          `json:"resource_id,omitempty"`
	Verified   bool                            `json:"verified"`
}

// HostedAgentSession is a provisioned hosted-agent sandbox session.
type HostedAgentSession struct {
	SessionID    string                                  `json:"session_id"`
	Name         string                                  `json:"name,omitempty"`
	TeamID       uint64                                  `json:"team_id"`
	AgentKind    HostedAgentKind                         `json:"agent_kind"`
	Status       HostedAgentSessionStatus                `json:"status"`
	CreatedAt    Timestamp                               `json:"created_at"`
	LastEventAt  Timestamp                               `json:"last_event_at"`
	RepoHint     string                                  `json:"repo_hint,omitempty"`
	ProviderAuth map[string]HostedAgentProviderAuthState `json:"provider_auth,omitempty"`
	// Origin is present for newly created sessions (including direct). Older
	// sessions may omit it.
	Origin *HostedAgentSessionOrigin `json:"origin,omitempty"`
	// OpenAISessionID is the OpenAI Agents session id (sess_…) linked to this DO
	// sandbox for AGENT_KIND_OPENAI_CODEX. Used by attach to bridge to OpenAI;
	// omitempty for other agent kinds.
	OpenAISessionID string `json:"openai_session_id,omitempty"`
	// OpenAIEnvironmentID is captured from the resolved CODEX_ENVIRONMENT_ID
	// guest env value at create. Non-secret correlation metadata.
	OpenAIEnvironmentID string `json:"openai_environment_id,omitempty"`
	// ParentSessionID is set on forked child sessions; empty/omitted for roots.
	ParentSessionID string `json:"parent_session_id,omitempty"`
	// ForkID is a branch label on forked sessions; empty/omitted for roots.
	ForkID string `json:"fork_id,omitempty"`
}

// HostedAgentRun represents a single execution within a session.
type HostedAgentRun struct {
	RunID                string              `json:"run_id"`
	SessionID            string              `json:"session_id"`
	State                HostedAgentRunState `json:"state"`
	StartedAt            Timestamp           `json:"started_at"`
	EndedAt              *Timestamp          `json:"ended_at,omitempty"`
	CumulativeCostMicros int64               `json:"cumulative_cost_micros"`
}

// HostedAgentHITLRequest is a pending human-in-the-loop approval.
type HostedAgentHITLRequest struct {
	RequestID string                    `json:"request_id"`
	SessionID string                    `json:"session_id"`
	RunID     string                    `json:"run_id"`
	Action    HostedAgentHITLActionKind `json:"action"`
	Details   map[string]interface{}    `json:"details,omitempty"`
	Workdir   string                    `json:"workdir,omitempty"`
	Deadline  *Timestamp                `json:"deadline,omitempty"`
}

// HostedAgentHITLDecision records a resolved HITL request.
type HostedAgentHITLDecision struct {
	RequestID string                 `json:"request_id"`
	Outcome   HostedAgentHITLOutcome `json:"outcome"`
	Actor     string                 `json:"actor,omitempty"`
	At        Timestamp              `json:"at"`
	Reason    string                 `json:"reason,omitempty"`
}

// HostedAgentEvent is one SSE payload from GET /v2/agents/sessions/{id}/stream.
//
// The server serializes the SPI canonical event envelope, whose JSON shape
// differs from this struct's field names: the discriminator is `type` (not
// `kind`), the per-kind body is `data` (not `payload`), the timestamp is
// `timestamp` (not `at`), and the team id rides as a decimal string in
// `tenant_id`. UnmarshalJSON maps that wire shape onto these fields, so callers
// read Kind/Payload/At/TeamID directly.
type HostedAgentEvent struct {
	EventID   string
	SessionID string
	RunID     string
	TeamID    uint64
	Seq       uint64
	At        Timestamp
	Kind      HostedAgentEventKind
	Payload   json.RawMessage

	// SourceEventID is the native event id from the agent runtime before
	// canonical mapping (Event.source_event_id). Empty when the runtime does
	// not supply stable ids or the server does not forward it.
	SourceEventID string
	// SourceEventType is the native event type label from the agent runtime
	// (e.g. codex's "item/agentMessage/delta"). Empty when not forwarded.
	SourceEventType string
	// SourceRaw is the exact native event bytes the in-sandbox adapter
	// captured before canonical mapping (Event.source_raw) — for codex, one
	// JSON-RPC frame as read off the app-server transport. Only present when
	// the stream was opened with HostedAgentSessionStreamOptions.IncludeRaw
	// and the server retained the bytes; base64 on the wire (JSON []byte).
	SourceRaw []byte
}

// hostedAgentEventWire is the on-the-wire SPI canonical event envelope.
type hostedAgentEventWire struct {
	EventID         string               `json:"event_id"`
	RunID           string               `json:"run_id"`
	TenantID        string               `json:"tenant_id"`
	SessionID       string               `json:"session_id"`
	Timestamp       Timestamp            `json:"timestamp"`
	Seq             uint64               `json:"seq"`
	SourceEventID   string               `json:"source_event_id,omitempty"`
	SourceEventType string               `json:"source_event_type,omitempty"`
	SourceRaw       []byte               `json:"source_raw,omitempty"`
	Type            HostedAgentEventKind `json:"type"`
	Data            json.RawMessage      `json:"data"`
}

// UnmarshalJSON decodes the SPI canonical event wire shape.
func (e *HostedAgentEvent) UnmarshalJSON(b []byte) error {
	var w hostedAgentEventWire
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	e.EventID = w.EventID
	e.RunID = w.RunID
	e.SessionID = w.SessionID
	e.Seq = w.Seq
	e.At = w.Timestamp
	e.Kind = w.Type
	e.Payload = w.Data
	e.SourceEventID = w.SourceEventID
	e.SourceEventType = w.SourceEventType
	e.SourceRaw = w.SourceRaw
	if w.TenantID != "" {
		id, err := strconv.ParseUint(w.TenantID, 10, 64)
		if err != nil {
			return fmt.Errorf("hosted agents: tenant_id %q: %w", w.TenantID, err)
		}
		e.TeamID = id
	}
	return nil
}

// HostedAgentSessionCreateRequest is the body for POST /v2/agents/sessions.
type HostedAgentSessionCreateRequest struct {
	AgentKind          HostedAgentKind `json:"agent_kind"`
	RepoHint           string          `json:"repo_hint,omitempty"`
	IdleTimeoutSeconds int64           `json:"idle_timeout_seconds,omitempty"`
	// Origin claims product-workflow provenance. Omit for a verified direct
	// session. Simulation/evaluation require resource_id.
	Origin *HostedAgentSessionOriginRequest `json:"origin,omitempty"`
}

// HostedAgentManifestCreateOptions configures CreateSessionFromManifest.
// OpenAISessionID is sent as the openai_session_id query parameter (not in the
// YAML body). harness-api persists it for AGENT_KIND_OPENAI_CODEX attach
// correlation. See docs/design/openai-sandbox-provider.md.
type HostedAgentManifestCreateOptions struct {
	OpenAISessionID string `url:"openai_session_id,omitempty"`
}

// HostedAgentSessionListOptions specifies optional list filters.
type HostedAgentSessionListOptions struct {
	PageToken string                   `url:"page_token,omitempty"`
	PageSize  int                      `url:"page_size,omitempty"`
	Status    HostedAgentSessionStatus `url:"status,omitempty"`
	Name      string                   `url:"name,omitempty"`
	// ParentSessionID lists child (forked) sessions of the given parent.
	ParentSessionID string `url:"parent_session_id,omitempty"`
}

// HostedAgentSessionsListResponse is returned by GET /v2/agents/sessions.
type HostedAgentSessionsListResponse struct {
	Sessions      []HostedAgentSession `json:"sessions"`
	NextPageToken string               `json:"next_page_token"`
}

// HostedAgentSessionStreamOptions configures the session SSE stream.
type HostedAgentSessionStreamOptions struct {
	ReplayFrom string
	ReplayOnly bool

	// Before turns the request into a single backward page of durable
	// history: the events strictly older than this event id, oldest-first,
	// after which the stream closes without live events. Set it to the
	// oldest event id already held. Requires ReplayOnly, since a live
	// attach cannot start in the past.
	//
	// A cursorless replay only covers the newest window of a session's
	// history (the server's replay budget), so walking backwards from its
	// oldest event is how older history is read. HasMore reports whether
	// the walk can continue.
	Before string

	// Limit caps the events in one history page. Only meaningful with
	// Before. Zero leaves the server's default (200); the server also caps
	// any request at its replay budget.
	Limit int

	// IncludeRaw asks the server to include each event's native source bytes
	// (HostedAgentEvent.SourceRaw) alongside the canonical payload. Raw
	// payloads meaningfully fatten every event, so this is opt-in; consumers
	// that don't translate native protocols should leave it off.
	IncludeRaw bool
}

// HostedAgentSendInputRequest is the body for POST .../input.
type HostedAgentSendInputRequest struct {
	Text string `json:"text"`

	// SourceRaw optionally carries the client's exact native protocol frame
	// this input was extracted from — for codex, the TUI's turn/start
	// JSON-RPC message with its full params. The in-sandbox adapter uses it
	// as the template for the turn it drives, so client intent beyond plain
	// text (input items, model, effort, approval policy, ...) survives the
	// text reduction. Only meaningful when the caller speaks the session's
	// own agent protocol; Text stays required either way. Inbound
	// counterpart of HostedAgentEvent.SourceRaw. Base64 on the wire per the
	// proto bytes JSON mapping — encoding/json does that for []byte.
	SourceRaw []byte `json:"source_raw,omitempty"`
}

// HostedAgentSendInputResponse is returned by POST .../input.
type HostedAgentSendInputResponse struct {
	RunID string `json:"run_id"`
}

// HostedAgentRelayRequest is the body for POST .../request: one native
// agent-protocol request frame, forwarded to the session's agent verbatim.
//
// Where SendInput carries the one message with a canonical meaning ("the user
// said something"), this carries everything else a client that speaks the
// session's own protocol needs to ask — for codex, the requests behind
// interrupts, slash commands, and model pickers. The control plane never
// parses the frame; only the in-sandbox adapter decides what is safe to
// forward.
type HostedAgentRelayRequest struct {
	// SourceRaw is the caller's native protocol request frame — for codex, a
	// single JSON-RPC request object carrying the caller's own id. Named to
	// match SendInput's field: both mean "my own frame, verbatim". Base64 on
	// the wire per the proto bytes JSON mapping.
	SourceRaw []byte `json:"source_raw"`
}

// HostedAgentRelayResponse is the reply to POST .../request.
type HostedAgentRelayResponse struct {
	// SourceRaw is the agent's reply frame, addressed to the caller's own
	// request id but otherwise verbatim. A protocol-level failure (a JSON-RPC
	// error object) is a normal reply and arrives here rather than as an HTTP
	// error.
	//
	// Empty means the in-sandbox adapter declined to forward the method.
	// Callers must answer their own caller on that case rather than waiting
	// for something that will not come.
	SourceRaw []byte `json:"source_raw,omitempty"`
}

// HostedAgentResolveHITLRequest is the body for POST .../hitl/{requestID}.
type HostedAgentResolveHITLRequest struct {
	Outcome HostedAgentHITLOutcome      `json:"outcome"`
	Reason  string                      `json:"reason,omitempty"`
	Source  HostedAgentResolutionSource `json:"source,omitempty"`

	// SourceRaw is the client's reply in the agent's own protocol, forwarded
	// to the in-sandbox agent untouched. It carries what Outcome cannot: an
	// elicitation's content, a tool's requested input, a scope beyond this one
	// call. Outcome stays required alongside it — it is what the audit trail
	// records, and what the agent falls back to when this is absent.
	SourceRaw []byte `json:"source_raw,omitempty"`
}

// HostedAgentProviderAuthStart is returned by POST /v2/agents/auth/{provider}.
// Status is "pending" when the user must still authorize in a browser
// (ConnectURL/PollURL/VerificationCode are set), or "success" when the team is
// already connected (those fields are empty). It is a free-form connect-flow
// status, distinct from the HostedAgentProviderAuthState session field. The
// authorization handle is never exposed: tokens are exchanged server-side at
// session time.
type HostedAgentProviderAuthStart struct {
	Provider         string     `json:"provider"`
	Status           string     `json:"status"`
	ConnectURL       string     `json:"connect_url,omitempty"`
	PollURL          string     `json:"poll_url,omitempty"`
	VerificationCode string     `json:"verification_code,omitempty"`
	ExpiresAt        *Timestamp `json:"expires_at,omitempty"`
}

// HostedAgentProviderAuthPoll is returned by GET
// /v2/agents/auth/{provider}/poll. It reports only whether authorization has
// completed ("pending" or "success"); no secret is returned.
type HostedAgentProviderAuthPoll struct {
	Provider  string     `json:"provider"`
	Status    string     `json:"status"`
	ExpiresAt *Timestamp `json:"expires_at,omitempty"`
}

// HostedAgentSandboxExecRequest is the body for POST .../sandbox/exec.
type HostedAgentSandboxExecRequest struct {
	Argv           []string `json:"argv"`
	Workdir        string   `json:"workdir,omitempty"`
	TimeoutSeconds int64    `json:"timeout_seconds,omitempty"`
}

// HostedAgentSandboxExecResponse is returned by POST .../sandbox/exec.
type HostedAgentSandboxExecResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
}

// HostedAgentWorkspaceUploadRequest is the input for UploadWorkspace.
type HostedAgentWorkspaceUploadRequest struct {
	// Path is the destination resolved inside the workspace root (/workspace). Required.
	Path string
	// IsArchive indicates Body is a tar archive to extract at Path.
	IsArchive bool
	// ContentSHA256 is an optional hex SHA-256 digest of the payload, forwarded to the guest for verification.
	ContentSHA256 string
	// Body is the raw file or tar bytes to upload. Required.
	Body io.Reader
}

// HostedAgentWorkspaceUploadResponse is returned by UploadWorkspace.
type HostedAgentWorkspaceUploadResponse struct {
	Path         string `json:"path"`
	BytesWritten int64  `json:"bytes_written"`
}

// HostedAgentWorkspaceDownloadRequest is the input for DownloadWorkspace.
type HostedAgentWorkspaceDownloadRequest struct {
	// Path is the source resolved inside the workspace root (/workspace). Required.
	Path string
	// AsArchive tar-streams the directory at Path when true.
	AsArchive bool
}

// HostedAgentWorkspaceDownload is the streaming result of DownloadWorkspace.
// Body strips the trailing integrity footer and verifies SHA-256 of the
// payload: read it to EOF and then Close it. A missing, invalid, or mismatched
// footer is an error. An HTTP X-Content-Sha256 trailer may still be present
// but is best-effort only; the body footer is the source of truth.
type HostedAgentWorkspaceDownload struct {
	Body io.ReadCloser
	// IsArchive is true when the payload is a tar stream.
	IsArchive bool
	// SizeBytes is the X-Workspace-Size-Bytes hint (0 when unknown); payload
	// size only (excludes the integrity footer). Not a Content-Length.
	SizeBytes int64
}

// HostedAgentWorkspaceTransferDirection is the direction of a staged transfer.
type HostedAgentWorkspaceTransferDirection string

const (
	HostedAgentWorkspaceTransferDirectionUpload   HostedAgentWorkspaceTransferDirection = "upload"
	HostedAgentWorkspaceTransferDirectionDownload HostedAgentWorkspaceTransferDirection = "download"
)

// HostedAgentWorkspaceTransferStatus is the status of a staged transfer.
type HostedAgentWorkspaceTransferStatus string

const (
	HostedAgentWorkspaceTransferStatusPending    HostedAgentWorkspaceTransferStatus = "pending"
	HostedAgentWorkspaceTransferStatusInProgress HostedAgentWorkspaceTransferStatus = "in_progress"
	HostedAgentWorkspaceTransferStatusCompleted  HostedAgentWorkspaceTransferStatus = "completed"
	HostedAgentWorkspaceTransferStatusFailed     HostedAgentWorkspaceTransferStatus = "failed"
)

// HostedAgentWorkspaceTransferCreateRequest starts a large upload or download.
// Upload uses IsArchive; download uses AsArchive (per OHS HTTP contract).
type HostedAgentWorkspaceTransferCreateRequest struct {
	Direction HostedAgentWorkspaceTransferDirection `json:"direction"`
	Path      string                                `json:"path"`
	IsArchive bool                                  `json:"is_archive,omitempty"`
	AsArchive bool                                  `json:"as_archive,omitempty"`
	SizeBytes int64                                 `json:"size_bytes,omitempty"`
	SHA256    string                                `json:"sha256,omitempty"`
}

// HostedAgentWorkspaceTransfer is returned by create/commit/get transfer calls.
type HostedAgentWorkspaceTransfer struct {
	TransferID   string                                `json:"transfer_id"`
	Direction    HostedAgentWorkspaceTransferDirection `json:"direction,omitempty"`
	Status       HostedAgentWorkspaceTransferStatus    `json:"status"`
	UploadID     string                                `json:"upload_id,omitempty"`
	PartSize     int64                                 `json:"part_size,omitempty"`
	ExpiresAt    *Timestamp                            `json:"expires_at,omitempty"`
	SizeBytes    int64                                 `json:"size_bytes,omitempty"`
	BytesWritten int64                                 `json:"bytes_written,omitempty"`
	SHA256       string                                `json:"sha256,omitempty"`
	DownloadURL  string                                `json:"download_url,omitempty"`
	ErrorMessage string                                `json:"error_message,omitempty"`
}

// HostedAgentWorkspaceTransferPartUploadURLsRequest requests presigned URLs for
// one or more upload parts. PartNumbers are 1-based.
type HostedAgentWorkspaceTransferPartUploadURLsRequest struct {
	PartNumbers []int `json:"part_numbers"`
}

// HostedAgentWorkspaceTransferPartUploadURL is one entry in a part-upload-urls response.
// PUT the part bytes directly to UploadURL (not through OHS).
type HostedAgentWorkspaceTransferPartUploadURL struct {
	PartNumber int    `json:"part_number"`
	UploadURL  string `json:"upload_url"`
}

// HostedAgentWorkspaceTransferPartUploadURLs is returned by CreateWorkspaceTransferPartUploadURLs.
type HostedAgentWorkspaceTransferPartUploadURLs struct {
	PartURLs []HostedAgentWorkspaceTransferPartUploadURL `json:"part_urls"`
}

// HostedAgentWorkspaceTransferCommitRequest finalizes an upload after all parts are PUT.
type HostedAgentWorkspaceTransferCommitRequest struct {
	SHA256 string `json:"sha256,omitempty"`
}

// HostedAgentWorkspaceTransferCancelRequest aborts an in-flight transfer.
type HostedAgentWorkspaceTransferCancelRequest struct {
	Reason string `json:"reason,omitempty"`
}

// HostedAgentWorkspaceTransferCancelResponse is returned by CancelWorkspaceTransfer.
type HostedAgentWorkspaceTransferCancelResponse struct {
	TransferID string                             `json:"transfer_id"`
	Aborted    bool                               `json:"aborted"`
	Status     HostedAgentWorkspaceTransferStatus `json:"status"`
}

type hostedAgentSessionRoot struct {
	Session *HostedAgentSession `json:"session"`
}

// CreateSession provisions a new hosted agent session.
func (s *HostedAgentsServiceOp) CreateSession(ctx context.Context, create *HostedAgentSessionCreateRequest) (*HostedAgentSession, *Response, error) {
	if create == nil {
		return nil, nil, errors.New("hosted agents: create request is required")
	}
	if create.AgentKind == "" || create.AgentKind == HostedAgentKindUnspecified {
		return nil, nil, errors.New("hosted agents: agent_kind is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, hostedAgentsSessionsBasePath, create)
	if err != nil {
		return nil, nil, err
	}
	root := new(hostedAgentSessionRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Session == nil {
		return nil, resp, errors.New("hosted agents: create session returned no session")
	}
	return root.Session, resp, nil
}

func (s *HostedAgentsServiceOp) CreateSessionFromManifest(ctx context.Context, manifest []byte, opt *HostedAgentManifestCreateOptions) (*HostedAgentSession, *Response, error) {
	if len(bytes.TrimSpace(manifest)) == 0 {
		return nil, nil, errors.New("hosted agents: manifest is required")
	}
	path, err := addOptions(hostedAgentsSessionsBasePath, opt)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.newCreateSessionPostRequest(ctx, path, bytes.NewReader(manifest), hostedAgentManifestMediaType)
	if err != nil {
		return nil, nil, err
	}
	return s.doCreateSession(ctx, req)
}

func (s *HostedAgentsServiceOp) doCreateSession(ctx context.Context, req *http.Request) (*HostedAgentSession, *Response, error) {
	root := new(hostedAgentSessionRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Session == nil {
		return nil, resp, errors.New("hosted agents: create session returned no session")
	}
	return root.Session, resp, nil
}

func (s *HostedAgentsServiceOp) newCreateSessionPostRequest(ctx context.Context, path string, body io.Reader, contentType string) (*http.Request, error) {
	u, err := s.client.BaseURL.Parse(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", mediaType)
	req.Header.Set("User-Agent", s.client.UserAgent)
	return req, nil
}

// ListSessions returns sessions visible to the caller's team.
//
// The server omits simulation and evaluation sessions from the list so customer
// surfaces stay free of product-owned internal runs. GetSession by session_id
// still returns those sessions for the owning product workflow.
func (s *HostedAgentsServiceOp) ListSessions(ctx context.Context, opt *HostedAgentSessionListOptions) (*HostedAgentSessionsListResponse, *Response, error) {
	path, err := addOptions(hostedAgentsSessionsBasePath, opt)
	if err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentSessionsListResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetSession returns a single session by ID.
func (s *HostedAgentsServiceOp) GetSession(ctx context.Context, sessionID string) (*HostedAgentSession, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	path := fmt.Sprintf(hostedAgentSessionByIDPath, sessionID)
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(hostedAgentSessionRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if root.Session == nil {
		return nil, resp, errors.New("hosted agents: get session returned no session")
	}
	return root.Session, resp, nil
}

// DestroySession tears down a session. The API returns HTTP 204 on success.
func (s *HostedAgentsServiceOp) DestroySession(ctx context.Context, sessionID string) (*Response, error) {
	if sessionID == "" {
		return nil, errors.New("hosted agents: session id is required")
	}
	path := fmt.Sprintf(hostedAgentSessionByIDPath, sessionID)
	req, err := s.client.NewRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

func (s *HostedAgentsServiceOp) PauseSession(ctx context.Context, sessionID string) (*Response, error) {
	if sessionID == "" {
		return nil, errors.New("hosted agents: session id is required")
	}
	path := fmt.Sprintf(hostedAgentSessionPausePath, sessionID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, struct{}{})
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

func (s *HostedAgentsServiceOp) ResumeSession(ctx context.Context, sessionID string) (*Response, error) {
	if sessionID == "" {
		return nil, errors.New("hosted agents: session id is required")
	}
	path := fmt.Sprintf(hostedAgentSessionResumePath, sessionID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, struct{}{})
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// StreamSession opens the SSE stream for a session. Callers MUST Close the stream.
func (s *HostedAgentsServiceOp) StreamSession(ctx context.Context, sessionID string, opt *HostedAgentSessionStreamOptions) (*HostedAgentSessionStream, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	path := fmt.Sprintf(hostedAgentSessionStreamPath, sessionID)
	if opt != nil {
		// The server answers this combination with a 400; rejecting it here
		// spends no round trip to learn the same thing.
		if opt.Before != "" && !opt.ReplayOnly {
			return nil, nil, errors.New("hosted agents: before requires replay only")
		}
		q := url.Values{}
		if opt.ReplayFrom != "" {
			q.Set("replay_from", opt.ReplayFrom)
		}
		if opt.ReplayOnly {
			q.Set("replay_only", "true")
		}
		if opt.Before != "" {
			q.Set("before", opt.Before)
		}
		if opt.Limit > 0 {
			q.Set("limit", strconv.Itoa(opt.Limit))
		}
		if opt.IncludeRaw {
			q.Set("include_raw", "true")
		}
		if encoded := q.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := s.client.DoStream(ctx, req)
	if err != nil {
		return nil, resp, err
	}
	stream := &HostedAgentSessionStream{
		raw:  NewSSEReader(resp.Body),
		body: resp.Body,
	}
	stream.raw.OnComment = stream.observeComment
	return stream, resp, nil
}

// SendInput enqueues user text for the in-sandbox agent runtime.
func (s *HostedAgentsServiceOp) SendInput(ctx context.Context, sessionID string, input *HostedAgentSendInputRequest) (*HostedAgentSendInputResponse, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if input == nil {
		return nil, nil, errors.New("hosted agents: input is required")
	}
	if strings.TrimSpace(input.Text) == "" {
		return nil, nil, errors.New("hosted agents: text is required")
	}
	path := fmt.Sprintf(hostedAgentSessionInputPath, sessionID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, input)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentSendInputResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// RelayRequest forwards one native agent-protocol request frame to the
// session's agent and returns its reply verbatim. Blocks on the agent, so it
// is slower than the other session calls; an empty reply means the in-sandbox
// adapter declined the method.
func (s *HostedAgentsServiceOp) RelayRequest(ctx context.Context, sessionID string, body *HostedAgentRelayRequest) (*HostedAgentRelayResponse, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if body == nil || len(body.SourceRaw) == 0 {
		return nil, nil, errors.New("hosted agents: source_raw is required")
	}
	path := fmt.Sprintf(hostedAgentSessionRequestPath, sessionID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentRelayResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// ResolveHITL submits a human-in-the-loop decision. The API returns HTTP 204 on success.
func (s *HostedAgentsServiceOp) ResolveHITL(ctx context.Context, sessionID, requestID string, body *HostedAgentResolveHITLRequest) (*Response, error) {
	if sessionID == "" {
		return nil, errors.New("hosted agents: session id is required")
	}
	if requestID == "" {
		return nil, errors.New("hosted agents: request id is required")
	}
	if body == nil {
		return nil, errors.New("hosted agents: resolve request is required")
	}
	if body.Outcome == "" || body.Outcome == HostedAgentHITLOutcomeUnspecified {
		return nil, errors.New("hosted agents: outcome is required")
	}
	path := fmt.Sprintf(hostedAgentSessionHITLPath, sessionID, requestID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// StartProviderAuth begins (or resumes) the team-scoped connect flow for an
// external provider. The server derives the team from the authenticated
// principal, so the POST carries an empty JSON object body.
func (s *HostedAgentsServiceOp) StartProviderAuth(ctx context.Context, provider string) (*HostedAgentProviderAuthStart, *Response, error) {
	if provider == "" {
		return nil, nil, errors.New("hosted agents: provider is required")
	}
	path := fmt.Sprintf(hostedAgentProviderAuthPath, provider)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, struct{}{})
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentProviderAuthStart)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// PollProviderAuth checks whether a pending connect link has been authorized.
// pollURL is the PollURL returned by StartProviderAuth; it is forwarded to the
// server as the poll_url query parameter.
func (s *HostedAgentsServiceOp) PollProviderAuth(ctx context.Context, provider, pollURL string) (*HostedAgentProviderAuthPoll, *Response, error) {
	if provider == "" {
		return nil, nil, errors.New("hosted agents: provider is required")
	}
	if pollURL == "" {
		return nil, nil, errors.New("hosted agents: poll url is required")
	}
	path := fmt.Sprintf(hostedAgentProviderAuthPollPath, provider)
	q := url.Values{}
	q.Set("poll_url", pollURL)
	path += "?" + q.Encode()
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentProviderAuthPoll)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// ExecInSandbox runs a command inside the session sandbox.
func (s *HostedAgentsServiceOp) ExecInSandbox(ctx context.Context, sessionID string, body *HostedAgentSandboxExecRequest) (*HostedAgentSandboxExecResponse, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if body == nil {
		return nil, nil, errors.New("hosted agents: exec request is required")
	}
	if len(body.Argv) == 0 {
		return nil, nil, errors.New("hosted agents: argv is required")
	}
	path := fmt.Sprintf(hostedAgentSessionSandboxExecPath, sessionID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentSandboxExecResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// UploadWorkspace streams a file (or tar archive) into the session workspace.
func (s *HostedAgentsServiceOp) UploadWorkspace(ctx context.Context, sessionID string, input *HostedAgentWorkspaceUploadRequest) (*HostedAgentWorkspaceUploadResponse, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if input == nil {
		return nil, nil, errors.New("hosted agents: upload request is required")
	}
	if input.Path == "" {
		return nil, nil, errors.New("hosted agents: path is required")
	}
	if input.Body == nil {
		return nil, nil, errors.New("hosted agents: body is required")
	}

	path := fmt.Sprintf(hostedAgentSessionWorkspaceUploadPath, sessionID)
	u, err := s.client.BaseURL.Parse(path)
	if err != nil {
		return nil, nil, err
	}
	q := url.Values{}
	q.Set("path", input.Path)
	if input.IsArchive {
		q.Set("is_archive", "true")
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), input.Body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", hostedAgentWorkspaceMediaType)
	req.Header.Set("Accept", mediaType)
	req.Header.Set("User-Agent", s.client.UserAgent)
	if input.ContentSHA256 != "" {
		req.Header.Set(workspaceContentSHA256Header, input.ContentSHA256)
	}

	root := new(HostedAgentWorkspaceUploadResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// DownloadWorkspace streams a file (or tar archive) out of the session
// workspace. Callers MUST read the returned Body to EOF and then Close it;
// the body strips the trailing integrity footer and verifies SHA-256 of the
// payload.
func (s *HostedAgentsServiceOp) DownloadWorkspace(ctx context.Context, sessionID string, input *HostedAgentWorkspaceDownloadRequest) (*HostedAgentWorkspaceDownload, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if input == nil {
		return nil, nil, errors.New("hosted agents: download request is required")
	}
	if input.Path == "" {
		return nil, nil, errors.New("hosted agents: path is required")
	}

	path := fmt.Sprintf(hostedAgentSessionWorkspaceDownloadPath, sessionID)
	q := url.Values{}
	q.Set("path", input.Path)
	if input.AsArchive {
		q.Set("as_archive", "true")
	}
	path += "?" + q.Encode()

	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", hostedAgentWorkspaceMediaType)

	resp, err := s.client.DoStream(ctx, req)
	if err != nil {
		return nil, resp, err
	}

	out := &HostedAgentWorkspaceDownload{
		Body: &workspaceDownloadBody{
			body:   resp.Body,
			hasher: sha256.New(),
		},
	}
	if archive, perr := strconv.ParseBool(strings.TrimSpace(resp.Header.Get(workspaceIsArchiveHeader))); perr == nil {
		out.IsArchive = archive
	}
	if hint := strings.TrimSpace(resp.Header.Get(workspaceSizeBytesHeader)); hint != "" {
		if n, perr := strconv.ParseInt(hint, 10, 64); perr == nil {
			out.SizeBytes = n
		}
	}
	return out, resp, nil
}

// CreateWorkspaceTransfer starts a large-file staged upload or download.
// For uploads, response includes part_size for client-side chunking.
func (s *HostedAgentsServiceOp) CreateWorkspaceTransfer(ctx context.Context, sessionID string, input *HostedAgentWorkspaceTransferCreateRequest) (*HostedAgentWorkspaceTransfer, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if input == nil {
		return nil, nil, errors.New("hosted agents: transfer create request is required")
	}
	if input.Direction == "" {
		return nil, nil, errors.New("hosted agents: direction is required")
	}
	if input.Path == "" {
		return nil, nil, errors.New("hosted agents: path is required")
	}

	path := fmt.Sprintf(hostedAgentSessionWorkspaceTransfersPath, sessionID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, input)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentWorkspaceTransfer)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// CreateWorkspaceTransferPartUploadURLs returns presigned URLs to PUT one or more
// upload parts. Upload only. Part numbers start at 1. If a URL expires, request
// that part_number again.
func (s *HostedAgentsServiceOp) CreateWorkspaceTransferPartUploadURLs(ctx context.Context, sessionID, transferID string, input *HostedAgentWorkspaceTransferPartUploadURLsRequest) (*HostedAgentWorkspaceTransferPartUploadURLs, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if transferID == "" {
		return nil, nil, errors.New("hosted agents: transfer id is required")
	}
	if input == nil {
		return nil, nil, errors.New("hosted agents: part upload URLs request is required")
	}
	if len(input.PartNumbers) == 0 {
		return nil, nil, errors.New("hosted agents: part_numbers must not be empty")
	}
	for _, n := range input.PartNumbers {
		if n < 1 {
			return nil, nil, errors.New("hosted agents: part_numbers must all be >= 1")
		}
	}

	path := fmt.Sprintf(hostedAgentSessionWorkspaceTransferPartURLsPath, sessionID, transferID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, input)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentWorkspaceTransferPartUploadURLs)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// CommitWorkspaceTransfer finalizes an upload after all parts have been PUT and
// starts applying the file into the workspace. Poll GetWorkspaceTransfer afterward.
func (s *HostedAgentsServiceOp) CommitWorkspaceTransfer(ctx context.Context, sessionID, transferID string, input *HostedAgentWorkspaceTransferCommitRequest) (*HostedAgentWorkspaceTransfer, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if transferID == "" {
		return nil, nil, errors.New("hosted agents: transfer id is required")
	}
	if input == nil {
		input = &HostedAgentWorkspaceTransferCommitRequest{}
	}

	path := fmt.Sprintf(hostedAgentSessionWorkspaceTransferCommitPath, sessionID, transferID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, input)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentWorkspaceTransfer)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetWorkspaceTransfer polls transfer status. On a completed download, DownloadURL
// and SHA256 are set; verify SHA256(file) == SHA256 after fetching DownloadURL.
// This path has no DOWSSHA1 body footer.
func (s *HostedAgentsServiceOp) GetWorkspaceTransfer(ctx context.Context, sessionID, transferID string) (*HostedAgentWorkspaceTransfer, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if transferID == "" {
		return nil, nil, errors.New("hosted agents: transfer id is required")
	}

	path := fmt.Sprintf(hostedAgentSessionWorkspaceTransferByIDPath, sessionID, transferID)
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentWorkspaceTransfer)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// CancelWorkspaceTransfer aborts an in-flight transfer. Idempotent.
func (s *HostedAgentsServiceOp) CancelWorkspaceTransfer(ctx context.Context, sessionID, transferID string, input *HostedAgentWorkspaceTransferCancelRequest) (*HostedAgentWorkspaceTransferCancelResponse, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if transferID == "" {
		return nil, nil, errors.New("hosted agents: transfer id is required")
	}
	if input == nil {
		input = &HostedAgentWorkspaceTransferCancelRequest{}
	}

	path := fmt.Sprintf(hostedAgentSessionWorkspaceTransferCancelPath, sessionID, transferID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, input)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentWorkspaceTransferCancelResponse)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// workspaceDownloadBody holds back the trailing integrity footer while
// streaming, hashes the payload, and verifies the footer at EOF.
type workspaceDownloadBody struct {
	body     io.ReadCloser
	hasher   hash.Hash
	pending  []byte
	scratch  []byte
	sawEOF   bool
	verified bool
	verr     error
}

func (b *workspaceDownloadBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.verified {
		if b.verr != nil {
			return 0, b.verr
		}
		return 0, io.EOF
	}

	for {
		if overflow := len(b.pending) - workspaceDownloadFooterLen; overflow > 0 {
			n := overflow
			if n > len(p) {
				n = len(p)
			}
			copy(p, b.pending[:n])
			b.hasher.Write(b.pending[:n])
			b.pending = b.pending[n:]
			return n, nil
		}

		if b.sawEOF {
			if err := b.verifyFooter(); err != nil {
				return 0, err
			}
			return 0, io.EOF
		}

		if b.scratch == nil {
			b.scratch = make([]byte, 32*1024)
		}
		nr, err := b.body.Read(b.scratch)
		if nr > 0 {
			b.pending = append(b.pending, b.scratch[:nr]...)
		}
		if errors.Is(err, io.EOF) {
			b.sawEOF = true
			continue
		}
		if err != nil {
			return 0, err
		}
	}
}

func (b *workspaceDownloadBody) verifyFooter() error {
	if b.verified {
		return b.verr
	}
	b.verified = true

	if len(b.pending) != workspaceDownloadFooterLen {
		b.verr = errors.New("hosted agents: missing or truncated workspace download integrity footer")
		return b.verr
	}
	want, ok := parseWorkspaceDownloadFooter(b.pending)
	if !ok {
		b.verr = errors.New("hosted agents: invalid workspace download integrity footer")
		return b.verr
	}
	got := hex.EncodeToString(b.hasher.Sum(nil))
	if !strings.EqualFold(want, got) {
		b.verr = fmt.Errorf("hosted agents: workspace download checksum mismatch: want %q, got %q", want, got)
		return b.verr
	}
	b.pending = nil
	return nil
}

// parseWorkspaceDownloadFooter validates DOWSSHA1<64-hex>\n and returns the digest.
func parseWorkspaceDownloadFooter(footer []byte) (string, bool) {
	if len(footer) != workspaceDownloadFooterLen {
		return "", false
	}
	if string(footer[:len(workspaceDownloadFooterPrefix)]) != workspaceDownloadFooterPrefix {
		return "", false
	}
	if footer[workspaceDownloadFooterLen-1] != '\n' {
		return "", false
	}
	digest := string(footer[len(workspaceDownloadFooterPrefix) : workspaceDownloadFooterLen-1])
	for i := 0; i < len(digest); i++ {
		c := digest[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return "", false
		}
	}
	return digest, true
}

func (b *workspaceDownloadBody) Close() error {
	if cerr := b.body.Close(); cerr != nil {
		return cerr
	}
	if b.verified {
		return b.verr
	}
	return nil
}

// HostedAgentSessionStream is a typed iterator over a session SSE stream.
type HostedAgentSessionStream struct {
	raw     *SSEReader
	body    io.ReadCloser
	current HostedAgentEvent
	err     error
	done    bool
	hasMore bool
}

// hasMoreComment is the SSE comment a history page (see
// HostedAgentSessionStreamOptions.Before) ends with, reporting whether events
// older than the page remain.
const hasMoreComment = "has_more="

// observeComment records the has_more trailer of a history page. Wired as the
// reader's OnComment hook, so it runs while Next drains the stream.
func (s *HostedAgentSessionStream) observeComment(comment []byte) {
	value, ok := strings.CutPrefix(string(comment), hasMoreComment)
	if !ok {
		return
	}
	s.hasMore = value == "true"
}

// HasMore reports whether the history page just read has older events behind
// it. Only meaningful once a stream opened with
// HostedAgentSessionStreamOptions.Before has been drained to its end: the
// trailer arrives after the last event.
func (s *HostedAgentSessionStream) HasMore() bool { return s.hasMore }

// Next advances to the next event. Returns false on EOF or error.
func (s *HostedAgentSessionStream) Next() bool {
	for {
		if s.done || s.err != nil {
			return false
		}
		ev, err := s.raw.Next()
		if errors.Is(err, io.EOF) {
			s.done = true
			return false
		}
		if err != nil {
			s.err = err
			return false
		}
		if len(ev.Data) == 0 {
			continue
		}
		var event HostedAgentEvent
		if err := json.Unmarshal(ev.Data, &event); err != nil {
			s.err = err
			return false
		}
		if event.EventID == "" && ev.ID != "" {
			event.EventID = ev.ID
		}
		if event.Kind == "" && ev.Event != "" {
			event.Kind = HostedAgentEventKind(ev.Event)
		}
		s.current = event
		return true
	}
}

// Current returns the most recent event produced by Next.
func (s *HostedAgentSessionStream) Current() HostedAgentEvent { return s.current }

// Err returns any non-EOF error encountered during iteration.
func (s *HostedAgentSessionStream) Err() error { return s.err }

// Close releases the underlying HTTP response body. Always call Close.
func (s *HostedAgentSessionStream) Close() error {
	if s.body == nil {
		return nil
	}
	return s.body.Close()
}
