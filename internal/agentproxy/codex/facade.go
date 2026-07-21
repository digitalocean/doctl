// Package codex implements agentproxy.Facade for the codex CLI's app-server
// JSON-RPC protocol (github.com/openai/codex, codex-rs/app-server).
package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/internal/agentproxy"
	"github.com/digitalocean/godo"
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

	// Sessions is the harness bridge: turn/start calls SendInput on it, and
	// the background streaming goroutine reads StreamSession's SSE events to
	// translate into codex notifications. The bootstrap methods (initialize,
	// thread/start, etc.) don't touch it.
	Sessions do.HostedAgentsService

	// notifier is set once per connection by the bridge (see SetNotifier) and
	// used by the background turn-streaming goroutine to push notifications
	// (turn/started, item/agentMessage/delta, turn/completed, ...) on the
	// harness's timeline rather than in direct reply to a request.
	notifier agentproxy.Notifier

	// mu guards turns and streamStarted. The harness allows exactly one
	// connected SSE consumer per session (the same constraint that makes a
	// concurrent `doctl agents attach` conflict with this proxy's own
	// stream) — so this facade must never have more than one StreamSession
	// call open at a time, no matter how many turns run, one after another or
	// overlapping. Opening a second one mid-turn silently displaces the
	// first, cutting off that turn's remaining events: typing a second
	// message before the first turn finishes made only the second message
	// ever get a response, because the first turn's stream got evicted the
	// moment the second one opened its own. streamStarted tracks whether a
	// StreamSession call is currently outstanding, not merely whether one was
	// ever made: stopEventLoop clears it on exit so a later turn/start can
	// open a fresh one if the previous one died.
	mu            sync.Mutex
	turns         map[string]*turnState
	streamStarted bool
}

// turnState is the in-flight accumulation state for one turn, keyed by run
// id in Facade.turns. All turns share one event loop (runEventLoop), so this
// replaces what used to be local variables inside a per-turn goroutine.
type turnState struct {
	itemID      string
	text        strings.Builder
	startedAt   int64
	itemStarted bool
}

var _ agentproxy.Facade = (*Facade)(nil)
var _ agentproxy.NotifierAware = (*Facade)(nil)

// SetNotifier implements agentproxy.NotifierAware.
func (f *Facade) SetNotifier(n agentproxy.Notifier) {
	f.notifier = n
}

// initializeResult is the shape a real codex app-server returns for
// `initialize`, captured both over stdio (send-message-v2) and from the live
// TUI over WebSocket: {"result":{"userAgent":...,"codexHome":...,
// "platformFamily":...,"platformOs":...}}. codexHome/platformFamily/platformOs
// describe codex's own local runtime environment in a real app-server; since
// this facade proxies to a hosted sandbox rather than running codex locally,
// these are synthetic placeholders, not a reflection of anything the sandbox
// actually reports.
//
// TODO(M3-M5): replace with the sandbox's real values once the harness
// exposes them.
type initializeResult struct {
	UserAgent      string `json:"userAgent"`
	CodexHome      string `json:"codexHome"`
	PlatformFamily string `json:"platformFamily"`
	PlatformOs     string `json:"platformOs"`
}

// accountReadResult is the shape a real codex app-server returns for
// `account/read`. Discovered to be load-bearing by testing against a real
// `codex --remote`: the TUI treats a failed account/read as fatal during
// bootstrap ("Error: account/read failed during TUI bootstrap") rather than
// tolerating it like other unhandled methods, so it needs a real (if
// synthetic) success response for the TUI to render at all.
// requiresOpenaiAuth is false because auth to the model happens server-side
// in the hosted sandbox, not via any local OpenAI login flow the TUI would
// otherwise try to trigger.
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
// agent-spec.yaml's default model (gpt-5.5) is enough to clear bootstrap.
//
// TODO(M3-M5): a real per-session model catalog, once the harness exposes
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
// errors out if any fails. This is a synthesized stub — SessionID as the
// thread id, static policy/sandbox defaults matching what the real capture
// showed.
//
// TODO(M3-M5): Turns always reports empty even once real turns run through
// turn/start; feed real turn history back into this shape.
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

// threadResumeParams is ThreadResumeParams from the codex source
// (codex-rs/app-server-protocol/src/protocol/v2/thread.rs): only ThreadID is
// read here. history/path are experimental, Codex-Cloud-only fields that
// don't apply to a hosted-session proxy.
type threadResumeParams struct {
	ThreadID string `json:"threadId"`
}

// threadResumeResult is a distinct name for documentation purposes only —
// ThreadResumeResponse is byte-for-byte identical to ThreadStartResponse in
// the codex source, so the underlying fields (and JSON tags) are shared via
// threadStartResult.
type threadResumeResult threadStartResult

// hooksListResult is HooksListResponse (plugin.rs). data is required but an
// empty list is valid — no hosted-session hooks catalog exists yet.
type hooksListResult struct {
	Data []any `json:"data"`
}

// skillsListResult is SkillsListResponse (plugin.rs). Same story as hooks:
// empty is valid, and clears the "failed to load skills" toast the TUI
// otherwise shows when this method 404s.
type skillsListResult struct {
	Data []any `json:"data"`
}

// pluginListResult is PluginListResponse (plugin.rs). marketplaces is
// required; the other two fields have #[serde(default)] on the codex side
// but are included explicitly since Go's zero value for a nil slice
// marshals to `null`, not `[]`, and the real server always sends arrays.
type pluginListResult struct {
	Marketplaces          []any    `json:"marketplaces"`
	MarketplaceLoadErrors []any    `json:"marketplaceLoadErrors"`
	FeaturedPluginIDs     []string `json:"featuredPluginIds"`
}

// appListResult is AppsListResponse (apps.rs). Both fields are required keys
// on the wire (nextCursor is a required key with a nullable value, not an
// omittable one).
type appListResult struct {
	Data       []any   `json:"data"`
	NextCursor *string `json:"nextCursor"`
}

// threadUnsubscribeResult is ThreadUnsubscribeResponse (thread.rs). status is
// one of "notLoaded" | "notSubscribed" | "unsubscribed" — "unsubscribed" is
// always a truthful, safe answer here since this facade holds no real
// subscription state to be wrong about.
type threadUnsubscribeResult struct {
	Status string `json:"status"`
}

// --- turn/start, turn/interrupt, and the notifications a turn produces ---
//
// Shapes confirmed against the codex source
// (codex-rs/app-server-protocol/src/protocol/v2/turn.rs and thread_data.rs),
// not guessed. Worth calling out: there is no `turn/failed` method anywhere
// in the codex protocol. Failure is represented purely via `turn/completed`
// with `turn.status == "failed"` and `turn.error` populated — finishTurn
// below reflects that; there is no separate failed-turn notification to
// send.

// turnStartParams is TurnStartParams (v2/turn.rs) — only the two fields this
// v1 "one-way text" facade reads; the many optional per-turn overrides
// (model, sandboxPolicy, approvalPolicy, ...) aren't applied here.
type turnStartParams struct {
	ThreadID string          `json:"threadId"`
	Input    []userInputItem `json:"input"`
}

// userInputItem is one variant of the UserInput enum (v2/turn.rs), internally
// tagged by "type". Only the "text" variant is read; image/localImage/skill/
// mention inputs are silently ignored in v1.
type userInputItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// turnStartResult is TurnStartResponse (v2/turn.rs) — just the Turn itself.
type turnStartResult struct {
	Turn turnObj `json:"turn"`
}

// turnObj is Turn (thread_data.rs). itemsView is omitted: it has
// #[serde(default)] on the codex side, so leaving it out entirely (rather
// than sending a zero value that might not match the default variant) is the
// safer decode-safe choice, same reasoning as the omitted optional fields on
// thread/start's Thread.
type turnObj struct {
	ID          string     `json:"id"`
	Items       []any      `json:"items"`
	Status      string     `json:"status"`
	Error       *turnError `json:"error"`
	StartedAt   *int64     `json:"startedAt"`
	CompletedAt *int64     `json:"completedAt"`
	DurationMs  *int64     `json:"durationMs"`
}

type turnError struct {
	Message string `json:"message"`
}

// turnStartedNotification is TurnStartedNotification (v2/turn.rs), method
// "turn/started" — maps from the canonical run.started event.
type turnStartedNotification struct {
	ThreadID string  `json:"threadId"`
	Turn     turnObj `json:"turn"`
}

// turnCompletedNotification is TurnCompletedNotification, method
// "turn/completed" — maps from both canonical run.completed AND run.failed
// (see the no-turn/failed note above).
type turnCompletedNotification struct {
	ThreadID string  `json:"threadId"`
	Turn     turnObj `json:"turn"`
}

// itemStartedNotification is ItemStartedNotification, method "item/started".
type itemStartedNotification struct {
	Item        agentMessageItem `json:"item"`
	ThreadID    string           `json:"threadId"`
	TurnID      string           `json:"turnId"`
	StartedAtMs int64            `json:"startedAtMs"`
}

// itemCompletedNotification is ItemCompletedNotification, method
// "item/completed".
type itemCompletedNotification struct {
	Item          agentMessageItem `json:"item"`
	ThreadID      string           `json:"threadId"`
	TurnID        string           `json:"turnId"`
	CompletedAtMs int64            `json:"completedAtMs"`
}

// agentMessageItem is the ThreadItem::AgentMessage variant, internally
// tagged "type": "agentMessage". phase/memoryCitation are omitted — both
// optional on the codex side.
type agentMessageItem struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Text string `json:"text"`
}

// agentMessageDeltaNotification is AgentMessageDeltaNotification, method
// "item/agentMessage/delta" — maps from canonical run.token_delta.
type agentMessageDeltaNotification struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
	Delta    string `json:"delta"`
}

// turnInterruptResult is TurnInterruptResponse (v2/turn.rs) — genuinely
// empty on the wire. Best-effort no-op for now: the harness has no
// cancel-input surface yet.
//
// TODO(M3-M5): wire this to a real cancel once the harness exposes one.
type turnInterruptResult struct{}

// synthesizedThread builds the one thread this facade ever knows about: a
// stand-in for the hosted session, id equal to SessionID, with static
// policy/sandbox defaults matching what the protocol capture showed for a
// real thread/start. Shared by thread/start and thread/resume so the two
// can't drift apart. Turns is always an empty placeholder — see the
// threadStartResult TODO above.
func (f *Facade) synthesizedThread() threadStartResult {
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
	}
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
		// Client notification (no id, no reply expected); this facade has
		// nothing to do in response.
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
		return f.synthesizedThread(), nil

	case "thread/resume":
		// ThreadResumeParams.thread_id (camelCase threadId on the wire) is the
		// only field this stub reads — history/path are experimental,
		// codex-cloud-only fields per the codex source and don't apply here.
		var p threadResumeParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &agentproxy.RPCError{Code: -32602, Message: "invalid params: " + err.Error()}
		}
		// This proxy is bound 1:1 to a single hosted session for its whole
		// lifetime (see Facade.SessionID), so the only thread id it can ever
		// resume is its own — anything else is a real "not found," not a gap
		// to stub around.
		if p.ThreadID != f.SessionID {
			return nil, &agentproxy.RPCError{
				Code:    -32001,
				Message: fmt.Sprintf("thread %q not found; this proxy only serves thread %q", p.ThreadID, f.SessionID),
			}
		}
		// ThreadResumeResponse is identical in shape to ThreadStartResponse
		// (verified against the codex source, codex-rs/app-server-protocol/
		// src/protocol/v2/thread.rs) — same synthesized thread either way.
		return threadResumeResult(f.synthesizedThread()), nil

	// The remaining cases are real methods the TUI calls during/after
	// bootstrap that don't crash it (unlike account/read, model/list,
	// thread/start above) but, left unhandled, get logged and — for
	// skills/list — surface a visible (non-fatal) error toast. Shapes
	// confirmed against the codex source
	// (codex-rs/app-server-protocol/src/protocol/v2/{plugin,apps,thread}.rs),
	// not guessed — empty catalogs are valid, decode-safe responses.
	//
	// TODO(M3-M5): a real per-session hooks/skills/plugin/app catalog, once
	// the harness exposes one.
	case "hooks/list":
		return hooksListResult{Data: []any{}}, nil

	case "skills/list":
		return skillsListResult{Data: []any{}}, nil

	case "plugin/list":
		return pluginListResult{
			Marketplaces:          []any{},
			MarketplaceLoadErrors: []any{},
			FeaturedPluginIDs:     []string{},
		}, nil

	case "app/list":
		return appListResult{Data: []any{}}, nil

	case "thread/unsubscribe":
		// thread_id isn't checked: unsubscribing from the one thread this
		// facade ever serves (or from anything else) is always a safe no-op.
		return threadUnsubscribeResult{Status: "unsubscribed"}, nil

	case "turn/start":
		var p turnStartParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &agentproxy.RPCError{Code: -32602, Message: "invalid params: " + err.Error()}
		}
		if p.ThreadID != f.SessionID {
			return nil, &agentproxy.RPCError{
				Code:    -32001,
				Message: fmt.Sprintf("thread %q not found; this proxy only serves thread %q", p.ThreadID, f.SessionID),
			}
		}
		// Open (or confirm) the event-loop stream before SendInput, not after:
		// SendInput is what makes the harness start the run, so a reader
		// attached only afterward can race a fast run to completion and miss
		// every event it emits. See ensureEventLoop.
		if err := f.ensureEventLoop(ctx); err != nil {
			return nil, &agentproxy.RPCError{Code: -32000, Message: "opening event stream failed: " + err.Error()}
		}

		var text strings.Builder
		for _, item := range p.Input {
			if item.Type != "text" {
				continue // image/localImage/skill/mention: out of scope for v1 "one-way text"
			}
			if text.Len() > 0 {
				text.WriteString("\n")
			}
			text.WriteString(item.Text)
		}
		resp, err := f.Sessions.SendInput(f.SessionID, &godo.HostedAgentSendInputRequest{Text: text.String()})
		if err != nil {
			return nil, &agentproxy.RPCError{Code: -32000, Message: "SendInput failed: " + err.Error()}
		}
		// The harness's run id doubles as this facade's turn id — already a
		// unique per-turn identifier, no separate id scheme needed.
		f.trackTurn(resp.RunID)
		return turnStartResult{
			Turn: turnObj{ID: resp.RunID, Items: []any{}, Status: "inProgress"},
		}, nil

	case "turn/interrupt":
		return turnInterruptResult{}, nil

	default:
		// Every other method is logged by the bridge as "unhandled: <method>"
		// and, if it was a request, answered with a JSON-RPC error so codex
		// never hangs.
		return nil, agentproxy.ErrMethodNotFound
	}
}

// notify pushes one notification and logs (rather than propagating) a
// failure — from a background goroutine there's no request to answer with an
// error, and if the connection is dead every subsequent Notify in this turn
// will fail the same way, so the caller uses the bool to stop early instead
// of error-spamming a dead connection for the rest of the stream.
func (f *Facade) notify(method string, params any) bool {
	if err := f.notifier.Notify(method, params); err != nil {
		log.Printf("codex facade: notify %s failed: %v", method, err)
		return false
	}
	return true
}

// ensureEventLoop makes sure exactly one StreamSession read loop is running
// for this facade's connection before returning, opening one synchronously
// if none is active yet — either because this is the first turn for this
// facade's connection, or because the previous loop died and stopEventLoop
// cleared streamStarted. Callers (turn/start) must call this, and see it
// succeed, BEFORE calling SendInput: SendInput is what makes the harness
// start actually running and emitting events, so opening the stream only
// afterward (e.g. lazily from inside a newly spawned goroutine) leaves a
// window where a fast run can start, emit every event, and complete before
// any reader is attached to see them — codex would then hang on a turn
// that silently never got its notifications. See the Facade.mu doc comment
// for why there must never be more than one StreamSession call open at a
// time.
func (f *Facade) ensureEventLoop(ctx context.Context) error {
	f.mu.Lock()
	if f.streamStarted {
		f.mu.Unlock()
		return nil
	}
	f.mu.Unlock()

	stream, err := f.Sessions.StreamSession(ctx, f.SessionID, nil)
	if err != nil {
		return err
	}

	f.mu.Lock()
	f.streamStarted = true
	f.mu.Unlock()

	go f.runEventLoop(stream)
	return nil
}

// trackTurn registers a new in-flight turn, keyed by run id, once
// ensureEventLoop has confirmed the read loop is running to see its events.
func (f *Facade) trackTurn(runID string) {
	f.mu.Lock()
	if f.turns == nil {
		f.turns = make(map[string]*turnState)
	}
	f.turns[runID] = &turnState{itemID: runID + "-msg"}
	f.mu.Unlock()
}

// eventClaimPoll/eventClaimWait bound how long lookupTurn retries an
// unrecognized run id before giving up on it.
const (
	eventClaimPoll = 2 * time.Millisecond
	eventClaimWait = 200 * time.Millisecond
)

// lookupTurn returns the tracked turnState for runID, retrying briefly if
// it's not there yet. turn/start calls SendInput, which is what makes the
// harness create the run and start emitting its events, and only registers
// runID in f.turns once SendInput returns — so this goroutine can observe
// that run's first event fractionally before trackTurn's map write becomes
// visible to it. A caller that still gets ok == false after this returns
// should treat the event as genuinely untracked (e.g. one that predates this
// connection), not as this race.
func (f *Facade) lookupTurn(runID string) (*turnState, bool) {
	deadline := time.Now().Add(eventClaimWait)
	for {
		f.mu.Lock()
		ts, ok := f.turns[runID]
		f.mu.Unlock()
		if ok || time.Now().After(deadline) {
			return ts, ok
		}
		time.Sleep(eventClaimPoll)
	}
}

// runEventLoop is the single, long-lived reader of stream — opened by
// ensureEventLoop before this goroutine was even spawned — for this facade's
// connection lifetime. It dispatches each event to whichever tracked turn it
// belongs to (by run id) and translates into the "one-way text" notification
// sequence: turn/started -> item/started -> item/agentMessage/delta* ->
// item/completed -> turn/completed. Everything else falls through untouched.
// Runs until the ctx passed to ensureEventLoop's StreamSession call is
// canceled (the connection closes) or the stream ends/errors — NOT until one
// turn finishes, since later turns depend on this same loop still running.
// The deferred stopEventLoop is what lets a later turn/start recover:
// without it, streamStarted would stay true forever after any exit here and
// every subsequent turn/start would return "inProgress" with no loop left
// running to ever notify it.
//
// TODO(M3): translate tool-call events instead of dropping them.
// TODO(M4): translate HITL events instead of dropping them.
func (f *Facade) runEventLoop(stream *godo.HostedAgentSessionStream) {
	defer f.stopEventLoop()
	defer stream.Close()

	for stream.Next() {
		ev := stream.Current()

		ts, ok := f.lookupTurn(ev.RunID)
		if !ok {
			continue // event for a run this facade isn't tracking (e.g. predates this connection)
		}

		switch ev.Kind {
		case godo.HostedAgentEventKindRunStarted:
			ts.startedAt = time.Now().Unix()
			if !f.notify("turn/started", turnStartedNotification{
				ThreadID: f.SessionID,
				Turn:     turnObj{ID: ev.RunID, Items: []any{}, Status: "inProgress", StartedAt: &ts.startedAt},
			}) {
				return
			}
			ts.itemStarted = f.notify("item/started", itemStartedNotification{
				Item:        agentMessageItem{Type: "agentMessage", ID: ts.itemID, Text: ""},
				ThreadID:    f.SessionID,
				TurnID:      ev.RunID,
				StartedAtMs: ts.startedAt * 1000,
			})

		case godo.HostedAgentEventKindTokenChunk:
			var payload struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(ev.Payload, &payload); err != nil {
				continue
			}
			ts.text.WriteString(payload.Text)
			if !f.notify("item/agentMessage/delta", agentMessageDeltaNotification{
				ThreadID: f.SessionID,
				TurnID:   ev.RunID,
				ItemID:   ts.itemID,
				Delta:    payload.Text,
			}) {
				return
			}

		case godo.HostedAgentEventKindRunCompleted:
			f.finishTurn(ev.RunID, ts, "completed", nil)

		case godo.HostedAgentEventKindRunFailed:
			var payload struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(ev.Payload, &payload)
			f.finishTurn(ev.RunID, ts, "failed", &turnError{Message: payload.Message})
		}
	}
	if err := stream.Err(); err != nil {
		log.Printf("codex facade: stream error: %v", err)
	}
}

// stopEventLoop runs when runEventLoop returns for any reason — the stream
// ended, errored, or a notify failed (ensureEventLoop only spawns
// runEventLoop after StreamSession has already succeeded, so a failure to
// connect never reaches here; it fails turn/start directly instead). It
// clears streamStarted so the next turn/start reopens the stream instead of
// silently returning "inProgress" for a turn nothing will ever finish, and
// fails out any turns still tracked at that point (mid-flight when the loop
// died) so codex isn't left waiting forever on those either.
func (f *Facade) stopEventLoop() {
	f.mu.Lock()
	f.streamStarted = false
	stranded := f.turns
	f.turns = nil
	f.mu.Unlock()

	for runID, ts := range stranded {
		f.finishTurn(runID, ts, "failed", &turnError{Message: "lost connection to hosted session"})
	}
}

// finishTurn sends the closing item/completed (if item/started ever landed)
// and turn/completed pair, then stops tracking runID. status is "completed"
// or "failed" — there is no turn/failed method in the codex protocol
// (confirmed against the source); failure is turn/completed with status
// "failed" and turnErr populated.
func (f *Facade) finishTurn(runID string, ts *turnState, status string, turnErr *turnError) {
	completedAt := time.Now().Unix()
	if ts.itemStarted {
		f.notify("item/completed", itemCompletedNotification{
			Item:          agentMessageItem{Type: "agentMessage", ID: ts.itemID, Text: ts.text.String()},
			ThreadID:      f.SessionID,
			TurnID:        runID,
			CompletedAtMs: completedAt * 1000,
		})
	}
	var durationMs *int64
	if ts.startedAt > 0 {
		d := (completedAt - ts.startedAt) * 1000
		durationMs = &d
	}
	f.notify("turn/completed", turnCompletedNotification{
		ThreadID: f.SessionID,
		Turn: turnObj{
			ID:          runID,
			Items:       []any{},
			Status:      status,
			Error:       turnErr,
			StartedAt:   &ts.startedAt,
			CompletedAt: &completedAt,
			DurationMs:  durationMs,
		},
	})

	f.mu.Lock()
	delete(f.turns, runID)
	f.mu.Unlock()
}
