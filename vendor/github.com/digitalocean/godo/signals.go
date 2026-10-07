package godo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	signalsBasePath = "v1/signals"
	consentBasePath = "v1/consent"
)

// SignalsService is the MCP v1 client: GETs on /v1/signals plus consent
// enable/disable on /v1/consent (consent-gateway).
type SignalsService interface {
	ListConsents(context.Context) (*SignalsListConsentsResponse, *Response, error)
	GetAgentConsent(context.Context, string) (*SignalsAgentConsent, *Response, error)
	SetAgentConsent(context.Context, string, bool) (*SignalsConsentRecord, *Response, error)
	ListAgentSessions(context.Context, string, *SignalsListAgentSessionsOptions) (*SignalsListAgentSessionsResponse, *Response, error)
	ListSessionSegments(context.Context, string, *SignalsListSegmentsOptions) (*SignalsListSegmentsResponse, *Response, error)
	ListSessionDialogues(context.Context, string, *SignalsListDialoguesOptions) (*SignalsSessionDialoguesResponse, *Response, error)
	GetSegment(context.Context, string, *SignalsCursorPageOptions) (*SignalsSegmentDetailResponse, *Response, error)
	GetSegmentSignalReport(context.Context, string) (*SignalsReport, *Response, error)
	CreateExport(context.Context, *SignalsCreateExportRequest) (*SignalsExportJob, *Response, error)
	ListExports(context.Context, *SignalsListExportsOptions) (*SignalsListExportsResponse, *Response, error)
	GetExport(context.Context, string) (*SignalsExportJob, *Response, error)
	GetExportDownload(context.Context, string) (*SignalsExportDownload, *Response, error)
	GetExportOptions(context.Context) (*SignalsExportOptions, *Response, error)
}

// SignalsServiceOp communicates with the Signals consumption API.
type SignalsServiceOp struct {
	client *Client
}

var _ SignalsService = &SignalsServiceOp{}

// SignalsCursorPageOptions is shared limit/after paging.
type SignalsCursorPageOptions struct {
	Limit int    `url:"limit,omitempty"`
	After string `url:"after,omitempty"`
}

// SignalsListAgentSessionsOptions are query params for GET /agents/{id}/sessions.
type SignalsListAgentSessionsOptions struct {
	SignalsCursorPageOptions
	StartTime  *int64   `url:"start_time,omitempty"`
	EndTime    *int64   `url:"end_time,omitempty"`
	SignalType []string `url:"signal_type,omitempty"`
}

// SignalsListSegmentsOptions are query params for GET /sessions/{id}/segments.
type SignalsListSegmentsOptions struct {
	SignalsCursorPageOptions
	Before         string   `url:"before,omitempty"`
	SignalType     []string `url:"signal_type,omitempty"`
	SignalCategory string   `url:"signal_category,omitempty"`
	SignalLayer    string   `url:"signal_layer,omitempty"`
	Concerning     *bool    `url:"concerning,omitempty"`
	StartedAfter   string   `url:"started_after,omitempty"`
	StartedBefore  string   `url:"started_before,omitempty"`
	StartTime      *int64   `url:"start_time,omitempty"`
	EndTime        *int64   `url:"end_time,omitempty"`
	Sort           string   `url:"sort,omitempty"`
	Order          string   `url:"order,omitempty"`
}

// SignalsListDialoguesOptions are query params for GET /sessions/{id}/dialogues.
type SignalsListDialoguesOptions struct {
	SignalsCursorPageOptions
	Before          string   `url:"before,omitempty"`
	ContinueSession bool     `url:"continue_session,omitempty"`
	SignalType      []string `url:"signal_type,omitempty"`
	StartTime       *int64   `url:"start_time,omitempty"`
	EndTime         *int64   `url:"end_time,omitempty"`
}

// SignalsListExportsOptions are query params for GET /exports.
type SignalsListExportsOptions struct {
	SignalsCursorPageOptions
	AgentID string `url:"agent_id,omitempty"`
}

// SignalsPageInfo is Relay-style cursor paging from signals-api.
type SignalsPageInfo struct {
	HasNextPage bool   `json:"has_next_page"`
	HasPrevPage bool   `json:"has_prev_page,omitempty"`
	StartCursor string `json:"start_cursor,omitempty"`
	EndCursor   string `json:"end_cursor,omitempty"`
}

// SignalsAgentConsent is GET /v1/consent/{agent_id} (consent-gateway).
type SignalsAgentConsent struct {
	TeamID    int64  `json:"team_id"`
	AgentID   string `json:"agent_id"`
	Enabled   bool   `json:"enabled"`
	Allowed   bool   `json:"allowed"`
	ID        int64  `json:"id,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// SignalsConsentRecord is the consents row returned by PUT /v1/consent/{agent_id}.
type SignalsConsentRecord struct {
	ID        int64  `json:"id"`
	TeamID    int64  `json:"team_id"`
	AgentID   string `json:"agent_id"`
	Enabled   bool   `json:"enabled"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type signalsSetConsentRequest struct {
	Enabled bool `json:"enabled"`
}

type signalsSetConsentRoot struct {
	Consent *SignalsConsentRecord `json:"consent"`
}

// SignalsListConsentsResponse is GET /v1/consent (consent-gateway).
type SignalsListConsentsResponse struct {
	TeamID   int64                  `json:"team_id"`
	Consents []SignalsConsentRecord `json:"consents"`
}

// SignalsCreateExportRequest is the body for POST /v1/signals/exports.
type SignalsCreateExportRequest struct {
	AgentID    string   `json:"agent_id"`
	SignalType []string `json:"signal_type,omitempty"`
	StartTime  *int64   `json:"start_time,omitempty"`
	EndTime    *int64   `json:"end_time,omitempty"`
}

// SignalsSession is a session list node.
type SignalsSession struct {
	SessionID       string `json:"session_id"`
	TotalTurns      int    `json:"total_turns"`
	StartedAt       string `json:"started_at"`
	DurationSeconds int    `json:"duration_seconds"`
	SignalCount     int    `json:"signal_count"`
}

// SignalsSessionEdge is one edge in a session list.
type SignalsSessionEdge struct {
	Cursor string         `json:"cursor"`
	Node   SignalsSession `json:"node"`
}

// SignalsListAgentSessionsResponse is GET /agents/{agent_id}/sessions.
type SignalsListAgentSessionsResponse struct {
	Edges    []SignalsSessionEdge `json:"edges"`
	PageInfo SignalsPageInfo      `json:"page_info"`
}

// SignalsSummary is a compact signal chip on a segment index row.
type SignalsSummary struct {
	SignalType string `json:"signal_type"`
	Category   string `json:"category,omitempty"`
	Layer      string `json:"layer,omitempty"`
	Concerning bool   `json:"concerning,omitempty"`
}

// SignalsSegment is a segment list node.
type SignalsSegment struct {
	SegmentID        string           `json:"segment_id"`
	SessionID        string           `json:"session_id"`
	SegmentSeq       int              `json:"segment_seq"`
	StartedAt        string           `json:"started_at"`
	EndedAt          *string          `json:"ended_at,omitempty"`
	Status           string           `json:"status"`
	AnnotationStatus string           `json:"annotation_status"`
	CloseReason      *string          `json:"close_reason,omitempty"`
	TotalTurns       int              `json:"total_turns"`
	UserTurns        *int             `json:"user_turns,omitempty"`
	AssistantTurns   *int             `json:"assistant_turns,omitempty"`
	DurationSeconds  int              `json:"duration_seconds"`
	OverallQuality   *string          `json:"overall_quality,omitempty"`
	Concerning       *bool            `json:"concerning,omitempty"`
	Signals          []SignalsSummary `json:"signals"`
	InTimeRange      bool             `json:"in_time_range,omitempty"`
}

// SignalsSegmentEdge is one edge in a segment list.
type SignalsSegmentEdge struct {
	Cursor string         `json:"cursor"`
	Node   SignalsSegment `json:"node"`
}

// SignalsListSegmentsResponse is GET /sessions/{session_id}/segments.
type SignalsListSegmentsResponse struct {
	Edges    []SignalsSegmentEdge `json:"edges"`
	PageInfo SignalsPageInfo      `json:"page_info"`
}

// SignalsDialogue is one user/assistant turn.
type SignalsDialogue struct {
	ID          int64             `json:"id"`
	RunID       string            `json:"run_id"`
	CreatedAt   string            `json:"created_at"`
	Sequence    int               `json:"sequence"`
	UserMessage string            `json:"user_message"`
	Steps       []json.RawMessage `json:"steps"`
	RunStatus   string            `json:"run_status"`
	FailureCode *string           `json:"failure_code,omitempty"`
}

// SignalsInstance is one detector hit on a dialogue.
type SignalsInstance struct {
	SignalType string  `json:"signal_type"`
	Label      string  `json:"label"`
	RunUUID    string  `json:"run_uuid"`
	StepIndex  int     `json:"step_index"`
	Confidence float64 `json:"confidence"`
	Snippet    *string `json:"snippet,omitempty"`
	Layer      string  `json:"layer"`
	Category   string  `json:"category"`
}

// SignalsSessionDialogue is a dialogue plus session-scoped chips.
type SignalsSessionDialogue struct {
	SignalsDialogue
	SegmentID  string            `json:"segment_id"`
	SegmentSeq int               `json:"segment_seq"`
	Signals    []SignalsInstance `json:"signals,omitempty"`
}

// SignalsSessionDialogueEdge is one edge in a session dialogue list.
type SignalsSessionDialogueEdge struct {
	Cursor string                 `json:"cursor"`
	Node   SignalsSessionDialogue `json:"node"`
}

// SignalsSessionDialoguesResponse is GET /sessions/{session_id}/dialogues.
type SignalsSessionDialoguesResponse struct {
	SessionID string                       `json:"session_id"`
	Edges     []SignalsSessionDialogueEdge `json:"edges"`
	PageInfo  SignalsPageInfo              `json:"page_info"`
}

// SignalsDialogueEdge is one edge in a segment detail list.
type SignalsDialogueEdge struct {
	Cursor string          `json:"cursor"`
	Node   SignalsDialogue `json:"node"`
}

// SignalsSegmentDetailResponse is GET /segments/{segment_id}.
type SignalsSegmentDetailResponse struct {
	SegmentID string                `json:"segment_id"`
	SessionID string                `json:"session_id"`
	Edges     []SignalsDialogueEdge `json:"edges"`
	PageInfo  SignalsPageInfo       `json:"page_info"`
}

// SignalsGroup is a grouped set of instances on a report.
type SignalsGroup struct {
	ID        int64             `json:"id"`
	Layer     string            `json:"layer"`
	Category  string            `json:"category"`
	HitCount  int               `json:"hit_count"`
	Severity  int               `json:"severity"`
	Instances []SignalsInstance `json:"instances"`
}

// SignalsReport is GET /segments/{segment_id}/signal-report.
type SignalsReport struct {
	SegmentID       string         `json:"segment_id"`
	SessionID       string         `json:"session_id"`
	OverallQuality  *string        `json:"overall_quality,omitempty"`
	QualityScore    float64        `json:"quality_score"`
	Concerning      bool           `json:"concerning"`
	Summary         *string        `json:"summary,omitempty"`
	TotalTurns      int            `json:"total_turns"`
	UserTurns       int            `json:"user_turns"`
	AssistantTurns  int            `json:"assistant_turns"`
	IsDragging      bool           `json:"is_dragging"`
	EfficiencyScore *float64       `json:"efficiency_score,omitempty"`
	Groups          []SignalsGroup `json:"groups"`
}

// SignalsExportFilters is the snapshot stored on an export job.
type SignalsExportFilters struct {
	SessionIDs     []string `json:"session_ids,omitempty"`
	SignalType     []string `json:"signal_type,omitempty"`
	SignalCategory string   `json:"signal_category,omitempty"`
	SignalLayer    string   `json:"signal_layer,omitempty"`
	Concerning     *bool    `json:"concerning,omitempty"`
	StartTime      *int64   `json:"start_time,omitempty"`
	EndTime        *int64   `json:"end_time,omitempty"`
}

// SignalsExportJob is GET /exports/{export_id} (and list nodes).
type SignalsExportJob struct {
	ExportID     string               `json:"export_id"`
	AgentID      string               `json:"agent_id,omitempty"`
	Status       string               `json:"status"`
	Filters      SignalsExportFilters `json:"filters"`
	CreatedAt    int64                `json:"created_at"`
	CompletedAt  *int64               `json:"completed_at,omitempty"`
	ErrorMessage *string              `json:"error_message,omitempty"`
	ExpiresAt    *int64               `json:"expires_at,omitempty"`
}

// SignalsExportEdge is one edge in an export list.
type SignalsExportEdge struct {
	Cursor string           `json:"cursor"`
	Node   SignalsExportJob `json:"node"`
}

// SignalsListExportsResponse is GET /exports.
type SignalsListExportsResponse struct {
	Edges    []SignalsExportEdge `json:"edges"`
	PageInfo SignalsPageInfo     `json:"page_info"`
}

// SignalsExportDownload is GET /exports/{export_id}/download.
type SignalsExportDownload struct {
	DownloadURL string `json:"download_url"`
	ExpiresAt   int64  `json:"expires_at"`
}

// SignalsExportOptions is GET /exports/options.
type SignalsExportOptions struct {
	Filters SignalsExportFilterOptions `json:"filters"`
}

// SignalsExportFilterOptions is the static detector catalog.
type SignalsExportFilterOptions struct {
	SignalType []string `json:"signal_type"`
}

func (s *SignalsServiceOp) get(ctx context.Context, path string, out interface{}) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, out)
}

// ListConsents returns all consent records for the authenticated team.
// Uses GET /v1/consent on consent-gateway.
func (s *SignalsServiceOp) ListConsents(ctx context.Context) (*SignalsListConsentsResponse, *Response, error) {
	root := new(SignalsListConsentsResponse)
	resp, err := s.get(ctx, consentBasePath, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetAgentConsent returns collection consent for one agent (default deny if no row).
// Uses GET /v1/consent/{agent_id} on consent-gateway (same public path as Cloud UI / SetAgentConsent).
func (s *SignalsServiceOp) GetAgentConsent(ctx context.Context, agentID string) (*SignalsAgentConsent, *Response, error) {
	path := fmt.Sprintf("%s/%s", consentBasePath, agentID)
	root := new(SignalsAgentConsent)
	resp, err := s.get(ctx, path, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// SetAgentConsent enables or disables collection for one agent (PUT /v1/consent/{id}).
// Requires IAM signals:update. Oceanus routes this to consent-gateway, not signals-api.
func (s *SignalsServiceOp) SetAgentConsent(ctx context.Context, agentID string, enabled bool) (*SignalsConsentRecord, *Response, error) {
	path := fmt.Sprintf("%s/%s", consentBasePath, agentID)
	req, err := s.client.NewRequest(ctx, http.MethodPut, path, &signalsSetConsentRequest{Enabled: enabled})
	if err != nil {
		return nil, nil, err
	}
	root := new(signalsSetConsentRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root.Consent, resp, nil
}

// ListAgentSessions lists sessions for an agent.
func (s *SignalsServiceOp) ListAgentSessions(ctx context.Context, agentID string, opts *SignalsListAgentSessionsOptions) (*SignalsListAgentSessionsResponse, *Response, error) {
	path := fmt.Sprintf("%s/agents/%s/sessions", signalsBasePath, agentID)
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}
	root := new(SignalsListAgentSessionsResponse)
	resp, err := s.get(ctx, path, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// ListSessionSegments lists annotated segments for a session.
func (s *SignalsServiceOp) ListSessionSegments(ctx context.Context, sessionID string, opts *SignalsListSegmentsOptions) (*SignalsListSegmentsResponse, *Response, error) {
	path := fmt.Sprintf("%s/sessions/%s/segments", signalsBasePath, sessionID)
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}
	root := new(SignalsListSegmentsResponse)
	resp, err := s.get(ctx, path, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// ListSessionDialogues lists replay dialogues for a session.
func (s *SignalsServiceOp) ListSessionDialogues(ctx context.Context, sessionID string, opts *SignalsListDialoguesOptions) (*SignalsSessionDialoguesResponse, *Response, error) {
	path := fmt.Sprintf("%s/sessions/%s/dialogues", signalsBasePath, sessionID)
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}
	root := new(SignalsSessionDialoguesResponse)
	resp, err := s.get(ctx, path, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetSegment lists dialogues for one segment.
func (s *SignalsServiceOp) GetSegment(ctx context.Context, segmentID string, opts *SignalsCursorPageOptions) (*SignalsSegmentDetailResponse, *Response, error) {
	path := fmt.Sprintf("%s/segments/%s", signalsBasePath, segmentID)
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}
	root := new(SignalsSegmentDetailResponse)
	resp, err := s.get(ctx, path, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetSegmentSignalReport returns the stored report (404 if none).
func (s *SignalsServiceOp) GetSegmentSignalReport(ctx context.Context, segmentID string) (*SignalsReport, *Response, error) {
	path := fmt.Sprintf("%s/segments/%s/signal-report", signalsBasePath, segmentID)
	root := new(SignalsReport)
	resp, err := s.get(ctx, path, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// CreateExport starts a bulk signal export job (POST /v1/signals/exports).
// The create is idempotent: if an active export already exists for the same
// parameters the server returns it instead of creating a duplicate.
func (s *SignalsServiceOp) CreateExport(ctx context.Context, body *SignalsCreateExportRequest) (*SignalsExportJob, *Response, error) {
	if body == nil {
		return nil, nil, fmt.Errorf("signals: create export request is required")
	}
	if body.AgentID == "" {
		return nil, nil, fmt.Errorf("signals: agent_id is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, signalsBasePath+"/exports", body)
	if err != nil {
		return nil, nil, err
	}
	root := new(SignalsExportJob)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// ListExports lists export jobs (no download URLs).
func (s *SignalsServiceOp) ListExports(ctx context.Context, opts *SignalsListExportsOptions) (*SignalsListExportsResponse, *Response, error) {
	path, err := addOptions(signalsBasePath+"/exports", opts)
	if err != nil {
		return nil, nil, err
	}
	root := new(SignalsListExportsResponse)
	resp, err := s.get(ctx, path, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetExport polls one export job.
func (s *SignalsServiceOp) GetExport(ctx context.Context, exportID string) (*SignalsExportJob, *Response, error) {
	path := fmt.Sprintf("%s/exports/%s", signalsBasePath, exportID)
	root := new(SignalsExportJob)
	resp, err := s.get(ctx, path, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetExportDownload returns a 15-minute Spaces presign for a completed export.
func (s *SignalsServiceOp) GetExportDownload(ctx context.Context, exportID string) (*SignalsExportDownload, *Response, error) {
	path := fmt.Sprintf("%s/exports/%s/download", signalsBasePath, exportID)
	root := new(SignalsExportDownload)
	resp, err := s.get(ctx, path, root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}

// GetExportOptions returns the static signal_type catalog.
func (s *SignalsServiceOp) GetExportOptions(ctx context.Context) (*SignalsExportOptions, *Response, error) {
	root := new(SignalsExportOptions)
	resp, err := s.get(ctx, signalsBasePath+"/exports/options", root)
	if err != nil {
		return nil, resp, err
	}
	return root, resp, nil
}
