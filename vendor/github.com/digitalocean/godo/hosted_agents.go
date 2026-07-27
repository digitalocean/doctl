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
	// repo_hint, idle_timeout_seconds). Prefer CreateSessionFromManifest for the
	// agents.digitalocean.com/v1alpha1 Agent manifest.
	CreateSession(context.Context, *HostedAgentSessionCreateRequest) (*HostedAgentSession, *Response, error)
	// CreateSessionFromManifest uploads a customer Agent manifest YAML document.
	// The request uses Content-Type: application/x-yaml.
	CreateSessionFromManifest(context.Context, []byte) (*HostedAgentSession, *Response, error)
	ListSessions(context.Context, *HostedAgentSessionListOptions) (*HostedAgentSessionsListResponse, *Response, error)
	GetSession(context.Context, string) (*HostedAgentSession, *Response, error)
	DestroySession(context.Context, string) (*Response, error)
	PauseSession(context.Context, string) (*Response, error)
	ResumeSession(context.Context, string) (*Response, error)
	StreamSession(context.Context, string, *HostedAgentSessionStreamOptions) (*HostedAgentSessionStream, *Response, error)
	SendInput(context.Context, string, *HostedAgentSendInputRequest) (*HostedAgentSendInputResponse, *Response, error)
	ResolveHITL(context.Context, string, string, *HostedAgentResolveHITLRequest) (*Response, error)
	ExecInSandbox(context.Context, string, *HostedAgentSandboxExecRequest) (*HostedAgentSandboxExecResponse, *Response, error)
	UploadWorkspace(context.Context, string, *HostedAgentWorkspaceUploadRequest) (*HostedAgentWorkspaceUploadResponse, *Response, error)
	DownloadWorkspace(context.Context, string, *HostedAgentWorkspaceDownloadRequest) (*HostedAgentWorkspaceDownload, *Response, error)

	// Large-file (>~50 MiB) staged workspace transfer APIs. Streaming
	// UploadWorkspace / DownloadWorkspace remain for smaller payloads.
	CreateWorkspaceTransfer(context.Context, string, *HostedAgentWorkspaceTransferCreateRequest) (*HostedAgentWorkspaceTransfer, *Response, error)
	CreateWorkspaceTransferPartUploadURL(context.Context, string, string, *HostedAgentWorkspaceTransferPartUploadURLRequest) (*HostedAgentWorkspaceTransferPartUploadURL, *Response, error)
	CommitWorkspaceTransfer(context.Context, string, string, *HostedAgentWorkspaceTransferCommitRequest) (*HostedAgentWorkspaceTransfer, *Response, error)
	GetWorkspaceTransfer(context.Context, string, string) (*HostedAgentWorkspaceTransfer, *Response, error)
	CancelWorkspaceTransfer(context.Context, string, string, *HostedAgentWorkspaceTransferCancelRequest) (*HostedAgentWorkspaceTransferCancelResponse, *Response, error)
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
	HostedAgentKindNone        HostedAgentKind = "AGENT_KIND_NONE"
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
}

// hostedAgentEventWire is the on-the-wire SPI canonical event envelope.
type hostedAgentEventWire struct {
	EventID   string               `json:"event_id"`
	RunID     string               `json:"run_id"`
	TenantID  string               `json:"tenant_id"`
	SessionID string               `json:"session_id"`
	Timestamp Timestamp            `json:"timestamp"`
	Seq       uint64               `json:"seq"`
	Type      HostedAgentEventKind `json:"type"`
	Data      json.RawMessage      `json:"data"`
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
}

// HostedAgentSessionListOptions specifies optional list filters.
type HostedAgentSessionListOptions struct {
	PageToken string                   `url:"page_token,omitempty"`
	PageSize  int                      `url:"page_size,omitempty"`
	Status    HostedAgentSessionStatus `url:"status,omitempty"`
	Name      string                   `url:"name,omitempty"`
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
}

// HostedAgentSendInputRequest is the body for POST .../input.
type HostedAgentSendInputRequest struct {
	Text string `json:"text"`
}

// HostedAgentSendInputResponse is returned by POST .../input.
type HostedAgentSendInputResponse struct {
	RunID string `json:"run_id"`
}

// HostedAgentResolveHITLRequest is the body for POST .../hitl/{requestID}.
type HostedAgentResolveHITLRequest struct {
	Outcome HostedAgentHITLOutcome      `json:"outcome"`
	Reason  string                      `json:"reason,omitempty"`
	Source  HostedAgentResolutionSource `json:"source,omitempty"`
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

// HostedAgentWorkspaceTransferPartUploadURLRequest requests a presigned URL for one part.
type HostedAgentWorkspaceTransferPartUploadURLRequest struct {
	// PartNumber is required and starts at 1.
	PartNumber int `json:"part_number"`
}

// HostedAgentWorkspaceTransferPartUploadURL is a presigned URL for one upload part.
// PUT the part bytes directly to UploadURL (not through OHS).
type HostedAgentWorkspaceTransferPartUploadURL struct {
	TransferID string     `json:"transfer_id"`
	PartNumber int        `json:"part_number"`
	UploadURL  string     `json:"upload_url"`
	ExpiresAt  *Timestamp `json:"expires_at,omitempty"`
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

func (s *HostedAgentsServiceOp) CreateSessionFromManifest(ctx context.Context, manifest []byte) (*HostedAgentSession, *Response, error) {
	if len(bytes.TrimSpace(manifest)) == 0 {
		return nil, nil, errors.New("hosted agents: manifest is required")
	}
	req, err := s.newCreateSessionPostRequest(ctx, bytes.NewReader(manifest), hostedAgentManifestMediaType)
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

func (s *HostedAgentsServiceOp) newCreateSessionPostRequest(ctx context.Context, body io.Reader, contentType string) (*http.Request, error) {
	u, err := s.client.BaseURL.Parse(hostedAgentsSessionsBasePath)
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
		q := url.Values{}
		if opt.ReplayFrom != "" {
			q.Set("replay_from", opt.ReplayFrom)
		}
		if opt.ReplayOnly {
			q.Set("replay_only", "true")
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
	return &HostedAgentSessionStream{
		raw:  NewSSEReader(resp.Body),
		body: resp.Body,
	}, resp, nil
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

// CreateWorkspaceTransferPartUploadURL returns a presigned URL to PUT one upload part.
// Upload only. Call once per part_number (starts at 1). If the URL expires, call again
// for the same part_number.
func (s *HostedAgentsServiceOp) CreateWorkspaceTransferPartUploadURL(ctx context.Context, sessionID, transferID string, input *HostedAgentWorkspaceTransferPartUploadURLRequest) (*HostedAgentWorkspaceTransferPartUploadURL, *Response, error) {
	if sessionID == "" {
		return nil, nil, errors.New("hosted agents: session id is required")
	}
	if transferID == "" {
		return nil, nil, errors.New("hosted agents: transfer id is required")
	}
	if input == nil {
		return nil, nil, errors.New("hosted agents: part upload URL request is required")
	}
	if input.PartNumber < 1 {
		return nil, nil, errors.New("hosted agents: part_number must be >= 1")
	}

	path := fmt.Sprintf(hostedAgentSessionWorkspaceTransferPartURLsPath, sessionID, transferID)
	req, err := s.client.NewRequest(ctx, http.MethodPost, path, input)
	if err != nil {
		return nil, nil, err
	}
	root := new(HostedAgentWorkspaceTransferPartUploadURL)
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
}

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
