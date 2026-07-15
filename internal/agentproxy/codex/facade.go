// Package codex implements agentproxy.Facade for the codex CLI's app-server
// JSON-RPC protocol (github.com/openai/codex, codex-rs/app-server).
package codex

import (
	"context"
	"encoding/json"
	"time"

	"github.com/digitalocean/doctl/internal/agentproxy"
)

// TestedVersion is the codex-cli version this facade's protocol handling was
// captured against (hosted-agents' docs/design/codex-app-server-protocol-capture.md:
// stdio send-message-v2 capture plus a live codex --remote / app-server pairing
// over WebSocket). Re-run that capture and update this pin on every codex
// upgrade — the WS/app-server transport is officially experimental and can
// change without notice.
const TestedVersion = "0.142.5"

// defaultModel is the single model this facade advertises via model/list and
// echoes back from thread/start, kept as one constant so the two stubbed
// responses can't drift apart. Matches agent-spec.yaml's MODEL today; a real
// per-session model comes from the harness, not this facade, once it's wired.
const defaultModel = "gpt-5.5"

// workspaceRoot is the fixed sandbox workspace root every hosted session
// uses (see doctl agents upload/download's --workspace-path help).
const workspaceRoot = "/workspace"

// Facade implements agentproxy.Facade for codex --remote.
type Facade struct {
	// SessionID is the hosted session this facade bridges to. thread/start's
	// synthesized thread id is exactly this value — simplest possible way to
	// satisfy "a synthesized thread whose id embeds the session id" while
	// matching what the real capture showed too (thread.id == thread.sessionId
	// there as well).
	SessionID string
}

var _ agentproxy.Facade = (*Facade)(nil)

// initializeResult is the shape a real codex app-server returns for
// `initialize`, captured both over stdio (send-message-v2) and from the live
// TUI over WebSocket: {"result":{"userAgent":...,"codexHome":...,
// "platformFamily":...,"platformOs":...}}. codexHome/platformFamily/platformOs
// describe codex's own local runtime environment in a real app-server; since
// v1 proxies to a hosted sandbox rather than running codex locally, these are
// synthetic placeholders until the harness exposes the sandbox's real values
// (M1+), not a reflection of anything the sandbox actually reports today.
type initializeResult struct {
	UserAgent      string `json:"userAgent"`
	CodexHome      string `json:"codexHome"`
	PlatformFamily string `json:"platformFamily"`
	PlatformOs     string `json:"platformOs"`
}

// accountReadResult is the shape a real codex app-server returns for
// `account/read`. Discovered to be load-bearing by testing against a real
// `codex --remote`, not scoped in the original M0 plan: the TUI treats a
// failed account/read as fatal during bootstrap ("Error: account/read failed
// during TUI bootstrap") rather than tolerating it like other unhandled
// methods, so it needs a real (if synthetic) success response to clear M0's
// "renders UI without crashing" bar. requiresOpenaiAuth is false because auth
// to the model happens server-side in the hosted sandbox, not via any local
// OpenAI login flow the TUI would otherwise try to trigger.
type accountReadResult struct {
	Account struct {
		Type     string `json:"type"`
		Email    string `json:"email"`
		PlanType string `json:"planType"`
	} `json:"account"`
	RequiresOpenaiAuth bool `json:"requiresOpenaiAuth"`
}

// modelListResult is the shape a real codex app-server returns for
// `model/list`. Discovered fatal-if-missing the same way as account/read: the
// TUI bootstrap errors out on a failed model/list too. One entry matching
// agent-spec.yaml's default model (gpt-5.5) is enough to clear bootstrap; a
// real per-session model catalog is out of scope until the harness exposes
// one.
type modelListResult struct {
	Data       []modelInfo `json:"data"`
	NextCursor *string     `json:"nextCursor"`
}

type modelInfo struct {
	ID                        string                `json:"id"`
	Model                     string                `json:"model"`
	Upgrade                   *string               `json:"upgrade"`
	UpgradeInfo               *string               `json:"upgradeInfo"`
	AvailabilityNux           *string               `json:"availabilityNux"`
	DisplayName               string                `json:"displayName"`
	Description               string                `json:"description"`
	Hidden                    bool                  `json:"hidden"`
	SupportedReasoningEfforts []reasoningEffortInfo `json:"supportedReasoningEfforts"`
	DefaultReasoningEffort    string                `json:"defaultReasoningEffort"`
	InputModalities           []string              `json:"inputModalities"`
	SupportsPersonality       bool                  `json:"supportsPersonality"`
	AdditionalSpeedTiers      []string              `json:"additionalSpeedTiers"`
	ServiceTiers              []string              `json:"serviceTiers"`
	DefaultServiceTier        *string               `json:"defaultServiceTier"`
	IsDefault                 bool                  `json:"isDefault"`
}

type reasoningEffortInfo struct {
	ReasoningEffort string `json:"reasoningEffort"`
	Description     string `json:"description"`
}

// threadStartResult is the shape a real codex app-server returns for
// thread/start. Discovered fatal-if-missing the same way as account/read and
// model/list: the TUI's bootstrap sequence calls all three synchronously and
// errors out if any fails, so a real M1 can't be deferred past M0 the way the
// milestone ladder originally assumed. This is a synthesized stub only —
// SessionID as the thread id, static policy/sandbox defaults matching what
// the real capture showed — not yet backed by the harness bridge: no real
// turns, no SendInput/StreamSession wiring. That's M2.
type threadStartResult struct {
	Thread                thread      `json:"thread"`
	Model                 string      `json:"model"`
	ModelProvider         string      `json:"modelProvider"`
	Cwd                   string      `json:"cwd"`
	RuntimeWorkspaceRoots []string    `json:"runtimeWorkspaceRoots"`
	ApprovalPolicy        string      `json:"approvalPolicy"`
	ApprovalsReviewer     string      `json:"approvalsReviewer"`
	Sandbox               sandboxInfo `json:"sandbox"`
	MultiAgentMode        string      `json:"multiAgentMode"`
}

type sandboxInfo struct {
	Type          string `json:"type"`
	NetworkAccess bool   `json:"networkAccess"`
}

type thread struct {
	ID             string       `json:"id"`
	SessionID      string       `json:"sessionId"`
	ForkedFromID   *string      `json:"forkedFromId"`
	ParentThreadID *string      `json:"parentThreadId"`
	Preview        string       `json:"preview"`
	Ephemeral      bool         `json:"ephemeral"`
	ModelProvider  string       `json:"modelProvider"`
	CreatedAt      int64        `json:"createdAt"`
	UpdatedAt      int64        `json:"updatedAt"`
	RecencyAt      int64        `json:"recencyAt"`
	Status         threadStatus `json:"status"`
	Path           string       `json:"path"`
	Cwd            string       `json:"cwd"`
	CliVersion     string       `json:"cliVersion"`
	// Source is a fixed placeholder, not a reflection of the real client: the
	// live-TUI protocol capture showed a real app-server return "vscode" here
	// even when the connecting client was the terminal TUI, not VS Code. Kept
	// identical since that's the one value already proven to decode cleanly.
	Source        string  `json:"source"`
	ThreadSource  string  `json:"threadSource"`
	AgentNickname *string `json:"agentNickname"`
	AgentRole     *string `json:"agentRole"`
	GitInfo       *string `json:"gitInfo"`
	Name          *string `json:"name"`
	Turns         []any   `json:"turns"`
}

type threadStatus struct {
	Type string `json:"type"`
}

// Dispatch implements agentproxy.Facade.
func (f *Facade) Dispatch(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		return initializeResult{
			UserAgent:      "doctl-agents-proxy/" + TestedVersion,
			CodexHome:      "/hosted",
			PlatformFamily: "unix",
			PlatformOs:     "linux",
		}, nil

	case "initialized":
		// Client notification (no id, no reply expected). Nothing to do until
		// M1 wires thread/start against a real session.
		return nil, nil

	case "account/read":
		result := accountReadResult{RequiresOpenaiAuth: false}
		result.Account.Type = "chatgpt"
		result.Account.Email = "hosted-session@digitalocean.com"
		result.Account.PlanType = "free"
		return result, nil

	case "model/list":
		return modelListResult{
			Data: []modelInfo{{
				ID:          defaultModel,
				Model:       defaultModel,
				DisplayName: defaultModel,
				Description: "Model configured for this hosted session.",
				SupportedReasoningEfforts: []reasoningEffortInfo{
					{ReasoningEffort: "medium", Description: "Balances speed and reasoning depth for everyday tasks"},
				},
				DefaultReasoningEffort: "medium",
				InputModalities:        []string{"text"},
				AdditionalSpeedTiers:   []string{},
				ServiceTiers:           []string{},
				IsDefault:              true,
			}},
		}, nil

	case "thread/start":
		now := time.Now().Unix()
		return threadStartResult{
			Thread: thread{
				ID:            f.SessionID,
				SessionID:     f.SessionID,
				ModelProvider: "openai",
				CreatedAt:     now,
				UpdatedAt:     now,
				RecencyAt:     now,
				Status:        threadStatus{Type: "idle"},
				Path:          workspaceRoot + "/.codex-session-" + f.SessionID + ".jsonl",
				Cwd:           workspaceRoot,
				CliVersion:    TestedVersion,
				Source:        "vscode",
				ThreadSource:  "user",
				Turns:         []any{},
			},
			Model:                 defaultModel,
			ModelProvider:         "openai",
			Cwd:                   workspaceRoot,
			RuntimeWorkspaceRoots: []string{workspaceRoot},
			ApprovalPolicy:        "on-request",
			ApprovalsReviewer:     "user",
			Sandbox:               sandboxInfo{Type: "readOnly", NetworkAccess: false},
			MultiAgentMode:        "explicitRequestOnly",
		}, nil

	default:
		// M0: every other method is logged by the bridge as "unhandled: <method>"
		// and, if it was a request, answered with a JSON-RPC error so codex
		// never hangs. This log is the M1-M3 backlog.
		return nil, agentproxy.ErrMethodNotFound
	}
}
