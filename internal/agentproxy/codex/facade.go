// Package codex implements agentproxy.Facade for the codex CLI's app-server
// JSON-RPC protocol (github.com/openai/codex, codex-rs/app-server).
package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
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
	// stream) — so this facade must open at most one StreamSession call at a
	// time, no matter how many turns run, one after another or overlapping.
	// Opening a second one mid-turn silently displaces the first, cutting
	// off that turn's remaining events: typing a second message before the
	// first turn finishes made only the second message ever get a response,
	// because the first turn's stream got evicted the moment the second one
	// opened its own.
	//
	// streamStarted tracks "is a runEventLoop currently running," not "has
	// one ever run": runEventLoop resets it (and clears turns) on every exit
	// path so a later turn/start on this same connection can open a fresh
	// loop via ensureEventLoop after the previous one died (idle timeout with
	// no turns left, terminal stream error, etc.). Each WebSocket connection
	// gets its own Facade (see ServeListener's newFacade factory), so this
	// flag does not need to survive across disconnect/reconnect.
	mu            sync.Mutex
	turns         map[string]*turnState
	streamStarted bool
	// expectingTurn is set just before SendInput and cleared once trackTurn
	// registers the run (or SendInput fails). While true, lookupTurn retries
	// a miss so the shared loop can claim events that arrive before the map
	// write — without charging a wait on every permanently-untracked event.
	// expectTurnDeadline is only a safety cap for a hung SendInput; the
	// handoff itself is not bounded by a short wall-clock window that can
	// expire during a slow SendInput RPC.
	expectingTurn      bool
	expectTurnDeadline time.Time

	// streamCursor is this connection's Last-Event-ID resume point for the
	// live SSE stream. Held on the Facade (not a local in runEventLoop) so a
	// loop that exits via noTurnsLeft and is later restarted by
	// ensureEventLoop reattaches with ReplayFrom rather than a forward-only
	// empty cursor — otherwise every turn after an idle exit reproduces the
	// attach-before-SendInput race against a blank cursor.
	streamCursor string

	// totalUsage accumulates every live run.usage_recorded event seen across
	// this connection (not per-turn — see
	// threadTokenUsageUpdatedNotification's "total" field, which is
	// thread-scoped, unlike "last" which is one turn's own delta). Historical
	// usage from --replay is deliberately not folded in (see translateEvent's
	// replay path). Unlike turns/streamStarted, this is not cleared when
	// runEventLoop exits: an idle stream drop between messages is normal, and
	// resetting here would make the TUI's running total jump backwards.
	totalUsage tokenUsageBreakdown

	// Replay, when true, feeds this session's full durable event history
	// into the first thread this facade bootstraps before that thread would
	// otherwise appear to start with no prior conversation — see
	// replaySessionHistory. Set once at construction from the --replay CLI
	// flag; never toggled per-connection.
	Replay bool

	// replayMu guards replaying/replayDone. Not sync.Once: that can only
	// express "ran once, ever," with no way to distinguish a genuinely
	// completed replay from one that was aborted early (the initial
	// StreamSession call failing, or the client disconnecting mid-fetch) —
	// which matters here, since an aborted attempt must be retryable on a
	// later thread/start or thread/resume on this same connection, while a
	// completed one must not re-deliver history a second time to a client
	// that already saw it. Per-connection (this Facade), not process-wide:
	// a reconnecting client is a new TUI with empty scrollback and needs
	// --replay again.
	replayMu sync.Mutex
	// replaying is true while a replaySessionHistory attempt is in flight on
	// this Facade — guards thread/start and thread/resume from starting two
	// fetches concurrently.
	replaying bool
	// replayPending is set by maybeReplay when Dispatch wants history, and
	// cleared by AfterReply once the thread/start|resume reply is on the
	// wire — so replayed turn/started cannot race ahead of the thread it
	// refers to.
	replayPending bool
	// replayDone is set only once replaySessionHistory reaches the natural
	// end of the replay-only stream — never on an aborted attempt.
	replayDone bool
}

// turnState is the in-flight accumulation state for one turn, keyed by run
// id in Facade.turns. All turns share one event loop (runEventLoop), so this
// replaces what used to be local variables inside a per-turn goroutine.
type turnState struct {
	itemID      string
	text        strings.Builder
	startedAt   int64
	itemStarted bool

	// commands tracks in-flight tool calls within this turn, keyed by the
	// canonical event's own tool_call_id (confirmed real and stable via a
	// live capture — no id-minting needed, unlike the message item above).
	// Needed because run.tool_call_completed's payload does NOT repeat
	// command/cwd — only { tool_call_id, ok, duration_ms, summary } — so
	// those two fields must be remembered from the started event.
	commands map[string]*commandState

	// fileChanges tracks in-flight file_change tool calls, keyed the same
	// way as commands. See fileChangeState for why this needs to exist at
	// all despite having (almost) nothing to remember.
	fileChanges map[string]*fileChangeState

	// replayHITLs maps historical hitl_id → item id during --replay only,
	// so a following human_input_received can mark fileChangeState.declined
	// without re-POSTing ResolveHITL. Unused on the live path.
	replayHITLs map[string]string
}

// commandState is the remembered-from-started half of one tool call's
// CommandExecution item; see the turnState.commands doc comment above.
type commandState struct {
	command string
	cwd     string
}

// fileChangeState is the remembered-from-approval half of one file_change
// tool call's FileChange item. Unlike commandState, run.tool_call_started
// carries nothing worth remembering for file_change (its input is always
// `{}` — confirmed via a live capture, see fileChangeItem) — what this
// tracks instead is declined, set if this facade resolved the file's HITL
// as Reject. run.tool_call_completed only ever carries a success boolean,
// which is ambiguous between codex's PatchApplyStatus::Failed (the patch
// didn't apply) and ::Declined (the user said no) — declined disambiguates
// which one to report.
type fileChangeState struct {
	declined bool
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
// Item is `any`, not a concrete struct: the real protocol's `item` field is
// itself a tagged union (ThreadItem, item.rs) — agentMessageItem and
// commandExecutionItem are two of its variants — so this mirrors that shape
// rather than forcing every item kind through one Go type.
type itemStartedNotification struct {
	Item        any    `json:"item"`
	ThreadID    string `json:"threadId"`
	TurnID      string `json:"turnId"`
	StartedAtMs int64  `json:"startedAtMs"`
}

// itemCompletedNotification is ItemCompletedNotification, method
// "item/completed". Item is `any` for the same reason as above.
type itemCompletedNotification struct {
	Item          any    `json:"item"`
	ThreadID      string `json:"threadId"`
	TurnID        string `json:"turnId"`
	CompletedAtMs int64  `json:"completedAtMs"`
}

// agentMessageItem is the ThreadItem::AgentMessage variant, internally
// tagged "type": "agentMessage". phase/memoryCitation are omitted — both
// optional on the codex side.
type agentMessageItem struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Text string `json:"text"`
}

// commandExecutionItem is the ThreadItem::CommandExecution variant,
// internally tagged "type": "commandExecution" (item.rs:269), confirmed
// against a live capture of a real tool-using turn, not guessed:
//   - run.tool_call_started's payload is {tool_call_id, name, input:
//     {command, cwd}} — command/cwd come from here.
//   - run.tool_call_completed's payload is {tool_call_id, ok, duration_ms,
//     summary} — no command/cwd (see turnState.commands), no real exit code
//     (only a success boolean — exitCode below is synthesized: 0 if ok,
//     1 otherwise, not a genuine process exit code), and no raw output text
//     (aggregatedOutput is populated from `summary` when non-empty, which in
//     the one capture taken was always empty — a real, currently-unresolved
//     fidelity gap, not something this facade can improve on today; see the
//     RFC's passthrough evidence).
//
// processId is omitted (nullable, not available canonically). source is
// always "agent": every tool call this facade ever sees was agent-initiated
// (CommandExecutionSource's own #[default] variant). commandActions is sent
// as an empty array, not omitted — the field has no #[serde(default)] on the
// codex side, the same class of bug that broke thread/start's `source` field
// in M1 (missing required key -> decode error) if this were ever left nil.
type commandExecutionItem struct {
	Type             string  `json:"type"`
	ID               string  `json:"id"`
	Command          string  `json:"command"`
	Cwd              string  `json:"cwd"`
	ProcessID        *string `json:"processId"`
	Source           string  `json:"source"`
	Status           string  `json:"status"`
	CommandActions   []any   `json:"commandActions"`
	AggregatedOutput *string `json:"aggregatedOutput"`
	ExitCode         *int    `json:"exitCode"`
	DurationMs       *int64  `json:"durationMs"`
}

// fileChangeItem is the ThreadItem::FileChange variant, internally tagged
// "type": "fileChange" (item.rs:294). Much simpler than commandExecutionItem
// — id, changes, status, nothing else (no duration/exit-code equivalent).
//
// changes is always sent as an empty array, not populated: confirmed via a
// live capture that run.tool_call_started's payload for a file_change call
// is {tool_call_id, name, input: {}} — input carries no file path or diff
// content at all, unlike command_execution's {command, cwd}. Canonical has
// nothing to reconstruct FileUpdateChange entries from, so the codex TUI
// will show "a file changed" with no diff preview — a real fidelity gap,
// not a bug, same class as commandExecutionItem's empty commandActions.
//
// status is PatchApplyStatus (item.rs): "inProgress" | "completed" |
// "failed" | "declined" — one more value than CommandExecutionStatus, since
// codex distinguishes "the patch didn't apply" from "the user declined it"
// where CommandExecution doesn't. See fileChangeState for how this facade
// tells them apart despite canonical only carrying one success boolean.
type fileChangeItem struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Changes []any  `json:"changes"`
	Status  string `json:"status"`
}

// agentMessageDeltaNotification is AgentMessageDeltaNotification, method
// "item/agentMessage/delta" — maps from canonical run.token_delta.
type agentMessageDeltaNotification struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
	Delta    string `json:"delta"`
}

// threadTokenUsageUpdatedNotification is ThreadTokenUsageUpdatedNotification
// (v2/thread.rs), method "thread/tokenUsage/updated" — maps from canonical
// run.usage_recorded, confirmed against harness-api's own event schema
// (harness-api/public/events.proto: RunUsageRecorded{step_id, model_id,
// usage: {input_tokens, output_tokens, cached_input_tokens,
// reasoning_tokens, tool_seconds}}) — a per-turn delta, not a running total
// (the proto's own doc comment says so directly).
//
// run.cost_accrued (the plan's original table lumped this in with
// usage_recorded) does NOT map here, and is intentionally left unhandled:
// its payload is {running_total_micros, delta_micros} — a running dollar
// total from the billing layer, not token counts — and codex's
// ThreadTokenUsageUpdatedNotification has no cost/dollar field at all to
// put it in. There's nothing in the codex protocol to translate it to.
type threadTokenUsageUpdatedNotification struct {
	ThreadID   string           `json:"threadId"`
	TurnID     string           `json:"turnId"`
	TokenUsage threadTokenUsage `json:"tokenUsage"`
}

// threadTokenUsage is ThreadTokenUsage (v2/thread.rs). ModelContextWindow is
// always omitted (sent null): no source data for this exists anywhere in
// canonical (confirmed — no context-window field anywhere in
// harness-api's event schema), and codex's own struct already has a
// "TODO: make this not optional" on the same field, implying it's commonly
// absent in real usage too.
type threadTokenUsage struct {
	Total              tokenUsageBreakdown `json:"total"`
	Last               tokenUsageBreakdown `json:"last"`
	ModelContextWindow *int64              `json:"modelContextWindow"`
}

// tokenUsageBreakdown is TokenUsageBreakdown (v2/thread.rs). TotalTokens is
// synthesized as InputTokens+OutputTokens: canonical's run.usage_recorded
// payload has no total_tokens field of its own (confirmed against
// harness-api's events.proto), so this is a best-effort sum, not a value
// the harness actually reports — same "synthesized, not real" caveat as
// commandExecutionItem's ExitCode.
type tokenUsageBreakdown struct {
	TotalTokens           int64 `json:"totalTokens"`
	InputTokens           int64 `json:"inputTokens"`
	CachedInputTokens     int64 `json:"cachedInputTokens"`
	OutputTokens          int64 `json:"outputTokens"`
	ReasoningOutputTokens int64 `json:"reasoningOutputTokens"`
}

// --- M4: approval round-trip ---
//
// item/commandExecution/requestApproval is the first server->client REQUEST
// this facade sends (everything above only ever sends notifications, which
// expect no reply) — maps from canonical run.human_input_requested. Shapes
// confirmed against the codex source (codex-rs/app-server-protocol/src/
// protocol/v2/item.rs, permissions.rs) and a real captured
// run.human_input_requested payload (see the implementation plan's
// Corrections section), not guessed.

// hitlRequestedPayload is canonical run.human_input_requested's data shape.
// Two kinds confirmed so far, both against live captures:
//   - "command_execution": {"hitl_id":..., "payload":{"kind":
//     "command_execution", "itemId":..., "turnId":..., "startedAtMs":...,
//     "environmentId":..., "command":..., "cwd":..., "commandActions":[...],
//     "proposedExecpolicyAmendment":[...]}}
//   - "file_change": {"hitl_id":..., "payload":{"category":"permission",
//     "grantRoot":null, "itemId":..., "kind":"file_change", "reason":null,
//     "startedAtMs":..., "turnId":...}} — notably no environmentId/command/
//     cwd/commandActions at all; grantRoot is omitted here too (always null
//     in the one capture taken, and marked [UNSTABLE] on the codex side).
//
// kind is read to route to the right approval request (see
// HostedAgentEventKindHITLRequested in runEventLoop); any kind besides those
// two is logged and skipped rather than guessed at — item/permissions/
// requestApproval is a real third method (confirmed in the codex source) but
// its canonical trigger has never been observed.
type hitlRequestedPayload struct {
	HitlID  string `json:"hitl_id"`
	Payload struct {
		Kind           string `json:"kind"`
		ItemID         string `json:"itemId"`
		StartedAtMs    int64  `json:"startedAtMs"`
		Environment    string `json:"environmentId"`
		Command        string `json:"command"`
		Cwd            string `json:"cwd"`
		CommandActions []any  `json:"commandActions"`

		// ProposedExecpolicyAmendment is parsed but deliberately never
		// forwarded to the client — see commandExecutionRequestApprovalParams'
		// doc comment for why sending it is actively harmful, not just
		// unhelpful.
		ProposedExecpolicyAmendment []string `json:"proposedExecpolicyAmendment"`

		// Reason is file_change-specific (always null in the one capture
		// taken so far); command_execution has no equivalent field.
		Reason *string `json:"reason"`

		// TurnID is deliberately unused below: in the one live capture taken,
		// this canonical field's value differed from the event envelope's own
		// run_id (which the rest of this facade already uses as the codex
		// turnId for every other notification about the same turn — see
		// requestCommandExecutionApproval). Using this field instead would
		// send a turnId that doesn't match what item/started and
		// turn/started already told the client about this same turn.
		// Unresolved discrepancy, flagged for live verification rather than
		// guessed at.
		TurnID string `json:"turnId"`
	} `json:"payload"`
}

// commandExecutionRequestApprovalParams is CommandExecutionRequestApprovalParams
// (item.rs). approvalId is deliberately omitted: per the codex source's own
// doc comment, it's null for "regular shell/unified_exec approvals" (it only
// disambiguates multiple zsh-exec-bridge callbacks sharing one itemId, which
// doesn't apply here) — correlating the eventual reply to this specific
// approval is the JSON-RPC request id's job (agentproxy.Notifier.Request),
// not a field inside params.
//
// availableDecisions and proposedExecpolicyAmendment are both deliberately
// omitted — and omitting availableDecisions alone is NOT enough, a real bug
// found live-testing this exact facade: the codex TUI's fallback decision set
// (used whenever availableDecisions is absent — see
// default_exec_approval_decisions, codex-rs/tui/src/app.rs) is NOT just
// [Accept, Cancel]. It's [Accept, Cancel], PLUS AcceptWithExecpolicyAmendment
// whenever the request itself carries a non-nil proposedExecpolicyAmendment.
// This facade used to forward that field straight from canonical, which
// silently re-introduced the exact "remembered approval the harness can't
// honor" problem availableDecisions was omitted to avoid in the first place
// — confirmed live: a user saw "yes, allow this command and don't ask
// again," picked it, and the harness asked again next time anyway, because
// AcceptWithExecpolicyAmendment collapses to a one-shot Approve just like
// AcceptForSession does (see decodeCommandExecutionApprovalDecision). Leaving
// this field out entirely is what actually restricts the TUI to
// [Accept, Cancel] — both of which this facade can honor correctly.
type commandExecutionRequestApprovalParams struct {
	ThreadID       string `json:"threadId"`
	TurnID         string `json:"turnId"`
	ItemID         string `json:"itemId"`
	StartedAtMs    int64  `json:"startedAtMs"`
	EnvironmentID  string `json:"environmentId"`
	Command        string `json:"command"`
	Cwd            string `json:"cwd"`
	CommandActions []any  `json:"commandActions"`
}

// commandExecutionRequestApprovalResponse is
// CommandExecutionRequestApprovalResponse (item.rs). Decision is left as
// json.RawMessage rather than a concrete type: CommandExecutionApprovalDecision
// (item.rs) is externally tagged with #[serde(rename_all = "camelCase")] —
// unit variants ("accept", "acceptForSession", "decline", "cancel") serialize
// as bare strings, struct variants (AcceptWithExecpolicyAmendment,
// ApplyNetworkPolicyAmendment) as {"<camelCaseVariant>": {...}} — see
// decodeCommandExecutionApprovalDecision.
type commandExecutionRequestApprovalResponse struct {
	Decision json.RawMessage `json:"decision"`
}

// decodeCommandExecutionApprovalDecision maps a codex CommandExecutionApprovalDecision
// (item.rs) onto the harness's 3-value HostedAgentHITLOutcome.
//
// acceptForSession/acceptWithExecpolicyAmendment/applyNetworkPolicyAmendment
// all collapse to a one-shot Approve: the harness has no concept of a
// persisted approval, so the user's "always allow" choice won't suppress the
// next identical request (see the implementation plan's Corrections
// section) — a known, accepted limitation, not a bug to fix here.
//
// decline/cancel both collapse to Reject too: cancel's "also interrupt the
// turn immediately" semantic (vs. decline's "agent continues the turn") has
// no HostedAgentHITLOutcome equivalent either — a second, smaller known gap.
func decodeCommandExecutionApprovalDecision(raw json.RawMessage) (godo.HostedAgentHITLOutcome, error) {
	var kind string
	if err := json.Unmarshal(raw, &kind); err == nil {
		switch kind {
		case "accept", "acceptForSession":
			return godo.HostedAgentHITLOutcomeApprove, nil
		case "decline", "cancel":
			return godo.HostedAgentHITLOutcomeReject, nil
		default:
			return "", fmt.Errorf("unrecognized decision %q", kind)
		}
	}

	var variant map[string]json.RawMessage
	if err := json.Unmarshal(raw, &variant); err != nil {
		return "", fmt.Errorf("decision is neither a string nor an object: %s", raw)
	}
	for key := range variant {
		switch key {
		case "acceptWithExecpolicyAmendment", "applyNetworkPolicyAmendment":
			return godo.HostedAgentHITLOutcomeApprove, nil
		}
	}
	return "", fmt.Errorf("unrecognized decision object: %s", raw)
}

// Note: this facade never sends a real item/fileChange/requestApproval to
// the client — see autoRejectFileChangeApproval for why. There is
// deliberately no FileChangeRequestApprovalParams/Response or
// FileChangeApprovalDecision decoder here, unlike the command_execution
// equivalents just above: nothing ever constructs or parses that shape.

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
		f.maybeReplay(ctx)
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
		f.maybeReplay(ctx)
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
		// every event it emits. Live delivery is forward-only from the moment
		// of attach (no Last-Event-ID recovery on a fresh cursor). See
		// ensureEventLoop.
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
		// Arm the claim window before SendInput: the shared loop can see
		// this run's first event as soon as the harness accepts it, which
		// is before trackTurn writes the map entry below. Cleared by
		// trackTurn (or on SendInput error) — not by a short deadline that
		// could expire while SendInput itself is still in flight.
		f.mu.Lock()
		f.expectingTurn = true
		f.expectTurnDeadline = time.Now().Add(eventClaimSafety)
		f.mu.Unlock()
		resp, err := f.Sessions.SendInput(f.SessionID, &godo.HostedAgentSendInputRequest{Text: text.String()})
		if err != nil {
			f.mu.Lock()
			f.expectingTurn = false
			f.expectTurnDeadline = time.Time{}
			f.mu.Unlock()
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
// facade's connection, or because the previous loop died and cleared
// streamStarted. Callers (turn/start) must call this, and see it succeed,
// BEFORE calling SendInput: SendInput is what makes the harness start
// actually running and emitting events, so opening the stream only afterward
// (e.g. lazily from inside a newly spawned goroutine) leaves a window where
// a fast run can start, emit every event, and complete before any reader is
// attached to see them — live delivery is forward-only from the moment of
// attach, and a fresh cursor has no Last-Event-ID to recover what was missed.
// See the Facade.mu doc comment for why there must never be more than one
// StreamSession call open at a time.
func (f *Facade) ensureEventLoop(ctx context.Context) error {
	f.mu.Lock()
	if f.streamStarted {
		f.mu.Unlock()
		return nil
	}
	cursor := f.streamCursor
	f.mu.Unlock()

	var opt *godo.HostedAgentSessionStreamOptions
	if cursor != "" {
		opt = &godo.HostedAgentSessionStreamOptions{ReplayFrom: cursor}
	}
	stream, err := f.Sessions.StreamSession(ctx, f.SessionID, opt)
	if err != nil {
		return err
	}

	f.mu.Lock()
	// A concurrent ensureEventLoop may have won the race and already started
	// a loop while we were opening — keep that one (the harness allows only
	// a single SSE consumer) and discard ours.
	if f.streamStarted {
		f.mu.Unlock()
		stream.Close()
		return nil
	}
	f.streamStarted = true
	f.mu.Unlock()

	go f.runEventLoop(ctx, stream)
	return nil
}

// trackTurn registers a new in-flight turn, keyed by run id, once
// ensureEventLoop has confirmed the read loop is running to see its events.
// Clears the expectingTurn claim window opened before SendInput.
func (f *Facade) trackTurn(runID string) {
	f.mu.Lock()
	if f.turns == nil {
		f.turns = make(map[string]*turnState)
	}
	f.turns[runID] = &turnState{itemID: runID + "-msg"}
	f.expectingTurn = false
	f.expectTurnDeadline = time.Time{}
	f.mu.Unlock()
}

// eventClaimPoll is how often lookupTurn rechecks the map while a
// SendInput→trackTurn handoff is in flight. eventClaimSafety caps how long
// that retry may last if SendInput hangs — the handoff itself is gated by
// expectingTurn, not by this duration.
const (
	eventClaimPoll   = 2 * time.Millisecond
	eventClaimSafety = 30 * time.Second
)

// lookupTurn returns the tracked turnState for runID. While turn/start has
// expectingTurn armed (see turn/start), it retries so the shared loop can
// claim a run's first event that arrives before trackTurn's map write —
// including events that land during a slow SendInput. Outside that handoff
// a miss is returned immediately so permanently-untracked events
// (already-finished runs, history that predates this connection) do not
// stall the drain loop.
func (f *Facade) lookupTurn(runID string) (*turnState, bool) {
	for {
		f.mu.Lock()
		ts, ok := f.turns[runID]
		expecting := f.expectingTurn && (f.expectTurnDeadline.IsZero() || time.Now().Before(f.expectTurnDeadline))
		f.mu.Unlock()
		if ok || !expecting {
			return ts, ok
		}
		time.Sleep(eventClaimPoll)
	}
}

// Reconnect schedule for runEventLoop's StreamSession loop. Mirrors
// commands/agents.go's doctl agents attach reconnect constants
// (maxAutoReconnectAttempts, initialReconnectBackoff, maxReconnectBackoff,
// healthyStreamDuration) — duplicated rather than shared, since extracting a
// common package would mean refactoring attach's already-working, tested
// code for this proxy's benefit alone. Worth revisiting if the two drift or
// a third caller appears.
const (
	maxAutoReconnectAttempts = 5
	initialReconnectBackoff  = 1 * time.Second
	maxReconnectBackoff      = 30 * time.Second
)

// healthyStreamDuration is how long a stream must stay connected before a
// drop is treated as a normal idle timeout (which resets the reconnect
// budget) rather than a failing connection — see runEventLoop. Var, not
// const, so tests can shrink it.
var healthyStreamDuration = 30 * time.Second

// nextReconnectBackoff doubles cur, capped at maxReconnectBackoff.
func nextReconnectBackoff(cur time.Duration) time.Duration {
	next := cur * 2
	if next > maxReconnectBackoff {
		return maxReconnectBackoff
	}
	return next
}

// sleepCtx waits for d or returns false immediately if ctx is done.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// reconnectSleep is the backoff wait runEventLoop uses between reconnect
// attempts, stored behind an atomic.Pointer rather than a plain var (unlike
// commands/agents.go's identical hook for doctl agents attach, which has no
// equivalent concern): a facade's runEventLoop goroutine can legitimately
// outlive the test that started it — some tests intentionally never finish
// their turn (e.g. ones only testing an approval hand-off) — and keep
// calling this hook concurrently with a *different*, later test reassigning
// it via setReconnectSleepForTest. A plain var raced under -race in
// practice, confirmed live across the test suite, not hypothetical.
var reconnectSleep atomic.Pointer[func(context.Context, time.Duration) bool]

func init() {
	fn := sleepCtx
	reconnectSleep.Store(&fn)
}

// sleepBeforeReconnect calls whatever reconnectSleep currently holds.
func sleepBeforeReconnect(ctx context.Context, d time.Duration) bool {
	fn := reconnectSleep.Load()
	return (*fn)(ctx, d)
}

// setReconnectSleepForTest replaces the reconnect backoff wait for the
// duration of a test, returning a restore func for t.Cleanup — see
// reconnectSleep's doc comment for why tests must go through this instead of
// assigning to a plain package var.
func setReconnectSleepForTest(fn func(context.Context, time.Duration) bool) (restore func()) {
	old := reconnectSleep.Load()
	reconnectSleep.Store(&fn)
	return func() { reconnectSleep.Store(old) }
}

// isTerminalStreamError reports whether err means reconnecting is pointless
// — auth failure, the session/run is gone, or a conflicting single-connection
// consumer (the same harness constraint that makes a concurrent `doctl
// agents attach` conflict with this proxy's own stream). Mirrors
// commands/agents.go's classifyStreamError — duplicated for the same reason
// as the reconnect constants above.
func isTerminalStreamError(err error) bool {
	var er *godo.ErrorResponse
	if !errors.As(err, &er) || er.Response == nil {
		return false
	}
	switch er.Response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict:
		return true
	}
	return false
}

// runEventLoop is the single, long-lived StreamSession reader for this
// facade's connection lifetime — started by ensureEventLoop with an already-
// open stream (so turn/start never races a fast run against attach), and
// shared by every turn thereafter. It dispatches each event to whichever
// tracked turn it belongs to (by run id) and translates into notifications:
// turn/started -> item/started -> item/agentMessage/delta* -> item/completed
// -> turn/completed for text, and item/started -> item/completed
// (commandExecution) for tool calls, interleaved as they occur. Everything
// else falls through untouched. Runs until ctx is canceled (the connection
// closes), the client connection dies, or reconnecting gives up for good —
// NOT until one turn finishes, since later turns depend on this same loop
// still running.
//
// Reconnects on a transient StreamSession failure or an unexpected stream
// drop (including a clean EOF, which is how a server idle-timeout close
// looks), with backoff — mirroring doctl agents attach's streamWithReconnect
// pattern exactly: maxAutoReconnectAttempts consecutive failures before
// giving up, reset by any connection that stayed healthy for
// healthyStreamDuration, so a long-lived proxy connection survives an
// unbounded number of idle drops while still giving up on a session that
// keeps failing immediately. A terminal error (see isTerminalStreamError)
// gives up immediately, no retries. Giving up for any reason fails every
// still-tracked turn (see failAllTrackedTurns) rather than leaving them
// silently hanging — codex has no "lost connection to backend" notification
// of its own to translate.
//
// TODO(M3): item/commandExecution/outputDelta (live streaming command
// output) is deliberately not sent — canonical run.tool_call_completed
// carries no output text at all today (see commandExecutionItem), so there's
// nothing to stream even if this were wired up.
func (f *Facade) runEventLoop(ctx context.Context, initial *godo.HostedAgentSessionStream) {
	// Reset streamStarted (and drop any turns this loop was still tracking)
	// on every exit path — ensureEventLoop only starts a loop when
	// streamStarted is false, so a later turn/start on this same connection
	// can recover after the previous loop died (idle timeout with no turns
	// left, give-up after reconnect exhaustion, etc.).
	defer func() {
		f.mu.Lock()
		f.streamStarted = false
		f.turns = nil
		// totalUsage is deliberately not reset: it is connection-scoped
		// (thread-scoped on the wire), and a normal idle noTurnsLeft exit
		// between user messages must not make the TUI's running total jump
		// backwards.
		f.mu.Unlock()
	}()

	backoff := initialReconnectBackoff
	failures := 0
	stream := initial

	for {
		if stream == nil {
			if ctx.Err() != nil {
				return
			}

			var opt *godo.HostedAgentSessionStreamOptions
			if f.streamCursor != "" {
				opt = &godo.HostedAgentSessionStreamOptions{ReplayFrom: f.streamCursor}
			}
			var err error
			stream, err = f.Sessions.StreamSession(ctx, f.SessionID, opt)
			if err != nil {
				if isTerminalStreamError(err) {
					log.Printf("codex facade: StreamSession failed (terminal), giving up: %v", err)
					f.failAllTrackedTurns("hosted session stream unavailable: " + err.Error())
					return
				}
				failures++
				if failures >= maxAutoReconnectAttempts {
					log.Printf("codex facade: StreamSession failed %d times in a row, giving up: %v", failures, err)
					f.failAllTrackedTurns("lost connection to hosted session")
					return
				}
				log.Printf("codex facade: StreamSession failed, reconnecting: %v", err)
				if !sleepBeforeReconnect(ctx, backoff) {
					return
				}
				backoff = nextReconnectBackoff(backoff)
				continue
			}
		}

		connectedAt := time.Now()
		clientDead := f.drainStream(ctx, stream, &f.streamCursor)
		streamErr := stream.Err()
		stream.Close()
		stream = nil

		if clientDead || ctx.Err() != nil {
			return
		}

		if streamErr != nil && isTerminalStreamError(streamErr) {
			log.Printf("codex facade: stream error (terminal), giving up: %v", streamErr)
			f.failAllTrackedTurns("hosted session stream unavailable: " + streamErr.Error())
			return
		}

		// Nothing left to reconnect for: every tracked turn already finished
		// (finishTurn deletes each one as it completes), so there's no
		// in-flight turn waiting on more events. Unlike doctl agents attach
		// (a long-lived interactive session that always expects more input),
		// this loop doesn't need to stay connected on the chance a new turn
		// starts later — ensureEventLoop already opens a fresh stream
		// synchronously before the next SendInput (see its own doc comment),
		// so exiting here is free, not a missed opportunity. Also avoids an
		// unbounded background retry loop outliving every turn it was ever
		// started for.
		f.mu.Lock()
		noTurnsLeft := len(f.turns) == 0
		f.mu.Unlock()
		if noTurnsLeft {
			return
		}

		// A drop after a healthy, long-lived connection is a normal idle
		// timeout, not a failing session: reset the budget and backoff so
		// this loop keeps recovering. Only rapid, back-to-back drops
		// accumulate toward the give-up limit.
		if time.Since(connectedAt) >= healthyStreamDuration {
			failures = 0
			backoff = initialReconnectBackoff
		} else {
			failures++
		}
		if failures >= maxAutoReconnectAttempts {
			log.Printf("codex facade: reconnected %d times without staying healthy, giving up", failures)
			f.failAllTrackedTurns("lost connection to hosted session")
			return
		}
		log.Printf("codex facade: stream ended (err=%v), reconnecting", streamErr)
		if !sleepBeforeReconnect(ctx, backoff) {
			return
		}
		backoff = nextReconnectBackoff(backoff)
	}
}

// failAllTrackedTurns synthesizes a failed turn/completed for every turn
// this loop was still tracking when runEventLoop is about to give up
// reconnecting for good. codex has no "lost connection to backend"
// notification of its own — leaving these turns silently unanswered would
// just recreate the exact "TUI hangs forever" failure mode this facade
// exists to avoid, just triggered by a dead stream instead of an unhandled
// HITL kind.
// The map is copied under f.mu rather than aliased: finishTurn below deletes
// from f.turns (and a concurrent turn/start inserts into it, and a replay
// goroutine's own finishTurn deletes from it) while this loop is ranging, so
// ranging the live map is a "concurrent map iteration and map write" crash,
// not merely a benign stale read. Copying (rather than detaching via
// f.turns = nil) keeps a racing turn/start's registration visible to this
// loop's deferred cleanup instead of widening that window.
func (f *Facade) failAllTrackedTurns(message string) {
	f.mu.Lock()
	turns := make(map[string]*turnState, len(f.turns))
	for runID, ts := range f.turns {
		turns[runID] = ts
	}
	f.mu.Unlock()

	for runID, ts := range turns {
		// Still delete every tracked turn even after the first notify
		// failure — the client is gone, but leaving entries in f.turns
		// would only confuse a later ensureEventLoop on this Facade.
		_ = f.finishTurn(runID, ts, "failed", &turnError{Message: message}, time.Now())
	}
}

// drainStream reads events from stream until it ends or errors, dispatching
// each to whichever tracked turn it belongs to and translating into
// notifications — runEventLoop's outer reconnect loop calls this once per
// successful StreamSession connection. Returns true if a notify failure
// indicates the client connection itself is dead, in which case the caller
// must stop entirely rather than try to reconnect to the harness (the
// problem isn't the harness stream).
//
// cursor is updated with every event's EventID as it's seen, before
// processing — even an event this facade fails to fully handle (bad JSON,
// e.g.) must never be re-requested via ReplayFrom on the next reconnect,
// since re-receiving it wouldn't fix the bad JSON.
//
// Also guards against a real, confirmed reconnect-boundary behavior: replay
// can re-deliver the last event(s) already processed before a drop (see
// doctl agents attach's tokenDeduper, added for the identical problem on
// run.token_delta specifically) — skipping any event whose id is <= cursor
// covers every event kind, not just token deltas, relying on canonical event
// ids being ULIDs (lexicographically ordered by time).
func (f *Facade) drainStream(ctx context.Context, stream *godo.HostedAgentSessionStream, cursor *string) (clientDead bool) {
	for stream.Next() {
		ev := stream.Current()

		if ev.EventID != "" {
			if *cursor != "" && ev.EventID <= *cursor {
				continue
			}
			*cursor = ev.EventID
		}

		// Control frames (stream.state) and anything else without a run id
		// belong to no turn — skip before lookupTurn rather than synthesizing
		// a phantom miss.
		if ev.RunID == "" {
			continue
		}

		ts, ok := f.lookupTurn(ev.RunID)
		if !ok {
			continue // event for a run this facade isn't tracking (e.g. predates this connection)
		}

		if f.translateEvent(ctx, ev, ts, false) {
			return true
		}
	}
	return false
}

// finishTurn sends the closing item/completed (if item/started ever landed)
// and turn/completed pair, then stops tracking runID. status is "completed"
// or "failed" — there is no turn/failed method in the codex protocol
// (confirmed against the source); failure is turn/completed with status
// "failed" and turnErr populated. at is the completion timestamp (event.At
// on the live/replay path, time.Now for synthesized failures). Returns true
// when a notify fails (client connection dead) so callers can stop draining.
// On the first notify failure the remaining notify is skipped (dead socket)
// but runID is still deleted so f.turns does not retain a zombie entry.
func (f *Facade) finishTurn(runID string, ts *turnState, status string, turnErr *turnError, at time.Time) (clientDead bool) {
	if at.IsZero() {
		at = time.Now()
	}
	completedAt := at.Unix()
	if ts.itemStarted {
		if !f.notify("item/completed", itemCompletedNotification{
			Item:          agentMessageItem{Type: "agentMessage", ID: ts.itemID, Text: ts.text.String()},
			ThreadID:      f.SessionID,
			TurnID:        runID,
			CompletedAtMs: at.UnixMilli(),
		}) {
			f.mu.Lock()
			delete(f.turns, runID)
			f.mu.Unlock()
			return true
		}
	}
	var durationMs *int64
	if ts.startedAt > 0 {
		d := (completedAt - ts.startedAt) * 1000
		durationMs = &d
	}
	if !f.notify("turn/completed", turnCompletedNotification{
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
	}) {
		clientDead = true
	}

	f.mu.Lock()
	delete(f.turns, runID)
	f.mu.Unlock()
	return clientDead
}

// requestCommandExecutionApproval sends the server->client
// item/commandExecution/requestApproval request for one HITL and, once the
// client replies, resolves the harness's HITL request accordingly. See the
// HostedAgentEventKindHITLRequested case in runEventLoop for why this runs
// in its own goroutine rather than inline.
func (f *Facade) requestCommandExecutionApproval(ctx context.Context, turnID string, hitl hitlRequestedPayload) {
	raw, err := f.notifier.Request(ctx, "item/commandExecution/requestApproval", commandExecutionRequestApprovalParams{
		ThreadID:       f.SessionID,
		TurnID:         turnID,
		ItemID:         hitl.Payload.ItemID,
		StartedAtMs:    hitl.Payload.StartedAtMs,
		EnvironmentID:  hitl.Payload.Environment,
		Command:        hitl.Payload.Command,
		Cwd:            hitl.Payload.Cwd,
		CommandActions: hitl.Payload.CommandActions,
	})
	if err != nil {
		f.rejectHITLAfterRequestFailure(hitl.HitlID, "command_execution", err)
		return
	}

	var resp commandExecutionRequestApprovalResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		log.Printf("codex facade: approval response %s: invalid shape: %v", hitl.HitlID, err)
		return
	}
	outcome, err := decodeCommandExecutionApprovalDecision(resp.Decision)
	if err != nil {
		log.Printf("codex facade: approval response %s: %v", hitl.HitlID, err)
		return
	}
	if err := f.Sessions.ResolveHITL(f.SessionID, hitl.HitlID, &godo.HostedAgentResolveHITLRequest{
		Outcome: outcome,
		Source:  godo.HostedAgentResolutionSourceInlineKeystroke,
	}); err != nil {
		log.Printf("codex facade: ResolveHITL %s failed: %v", hitl.HitlID, err)
	}
}

// autoRejectFileChangeApproval immediately rejects every file_change HITL
// server-side — no item/fileChange/requestApproval is ever sent to the
// client. This is a deliberate default, not a stub for a gap: apply_patch's
// registration as a model-facing tool is a per-model, server-side catalog
// decision inside codex itself (ModelInfo.apply_patch_tool_type, gated in
// codex-rs/core/src/tools/spec_plan.rs), with no config lever this proxy —
// or even hosted-agents' own sandbox config — can reach to disable it
// outright (confirmed via source; hosted-agents' codex sidecar config is a
// small curated allowlist, not a passthrough, and the actual per-run
// config.toml is rendered in a separate "plano" repo this proxy has no
// access to). Instantly rejecting every attempt is the only lever available
// here to steer codex away from apply_patch and toward shell commands for
// file edits.
//
// This also sidesteps a real interactive-approval race that was found and
// reverted: a fixed timeout on the client-facing request was tried, but the
// codex protocol has no way to withdraw a request already sent, so timing
// out server-side left the client's own prompt dangling, and a human's real
// (late) answer got silently dropped by wsNotifier.Request/deliverReply
// (proxy.go). Never sending the request at all has no such race.
//
// Marks fc.declined so the later run.tool_call_completed reports
// PatchApplyStatus::Declined, not ::Failed — same disambiguation
// fileChangeState exists for (see its doc comment).
//
// Takes ts directly — the caller's own turnState — rather than a turn id to
// re-look-up via f.turns: this runs in its own goroutine (see
// translateEvent's HITLRequested case) concurrently with the goroutine that
// owns ts (the live event loop, or replaySessionHistory), so f.mu still
// guards the actual ts.fileChanges map access, but no longer depends on ts
// being registered in the shared f.turns map at all — which
// replaySessionHistory's synthesized turnStates never are (see replay.go).
func (f *Facade) autoRejectFileChangeApproval(ts *turnState, hitl hitlRequestedPayload) {
	log.Printf("codex facade: auto-rejecting file_change HITL %s (pushing codex toward shell commands for file edits)", hitl.HitlID)

	f.mu.Lock()
	if fc, ok := ts.fileChanges[hitl.Payload.ItemID]; ok {
		fc.declined = true
	}
	f.mu.Unlock()

	if err := f.Sessions.ResolveHITL(f.SessionID, hitl.HitlID, &godo.HostedAgentResolveHITLRequest{
		Outcome: godo.HostedAgentHITLOutcomeReject,
		Source:  godo.HostedAgentResolutionSourceOutOfBand,
	}); err != nil {
		log.Printf("codex facade: auto-reject ResolveHITL %s failed: %v", hitl.HitlID, err)
	}
}

// autoRejectUnknownHITL resolves a HITL request of a kind this facade
// doesn't implement as Reject, rather than leaving it unanswered forever.
// Left unhandled, the harness would wait on a decision that never comes
// until its own idle timeout kills the whole session — the exact failure
// mode the approval mechanism exists to prevent, just for a kind this facade
// hasn't been taught yet instead of one it never learns about.
//
// This happens entirely server-side: unlike
// requestCommandExecutionApproval, there's no client-facing request here at
// all, since an unrecognized kind's item/<kind>/requestApproval shape isn't
// known well enough to construct — the codex TUI never sees that anything
// was asked. Source is OutOfBand (not InlineKeystroke): no human made this
// decision — same as autoRejectFileChangeApproval, for the same reason.
//
// Whether the agent gracefully continues after a reject for a kind this
// facade has never implemented, the way it's confirmed to for
// command_execution/file_change's Decline, is inferred from that pattern,
// not verified — flagged here rather than assumed silently.
func (f *Facade) autoRejectUnknownHITL(hitlID, kind string) {
	log.Printf("codex facade: auto-rejecting unknown HITL kind %q (hitl_id=%s) to avoid stalling the run", kind, hitlID)
	if err := f.Sessions.ResolveHITL(f.SessionID, hitlID, &godo.HostedAgentResolveHITLRequest{
		Outcome: godo.HostedAgentHITLOutcomeReject,
		Source:  godo.HostedAgentResolutionSourceOutOfBand,
	}); err != nil {
		log.Printf("codex facade: auto-reject ResolveHITL %s failed: %v", hitlID, err)
	}
}

// rejectHITLAfterRequestFailure resolves hitlID as Reject when this facade's
// server->client approval Request itself failed — this only happens when the
// client disconnects (or this connection's own ctx is otherwise canceled)
// while the approval was still outstanding, unblocking Request with
// ctx.Err() before any reply ever arrived. Only requestCommandExecutionApproval
// calls this now — file_change never sends a client-facing request at all
// (see autoRejectFileChangeApproval), so it can't hit this path.
//
// Deliberately NOT reached on a fixed timeout: that was tried for file_change
// and reverted (see autoRejectFileChangeApproval's doc comment) because the
// codex protocol has no way to tell the client to withdraw a request it
// already sent to the human, so timing out server-side just abandons the
// request while the client's own prompt keeps dangling — and if the human
// then does answer it, that reply arrives with nothing left in
// wsNotifier.pending to receive it (proxy.go's deliverReply logs it as an
// unrecognized id and drops it). Without this function, a genuinely dead
// connection would leave the harness waiting on a decision nothing will
// ever send, until its own idle timeout kills the run — same failure class
// autoRejectUnknownHITL exists to prevent, just triggered by a dead
// connection instead of an unimplemented kind.
//
// Safe to call after the connection is already gone: f.Sessions.ResolveHITL
// makes its own outbound call with context.TODO() (see do.agents.go), not
// this connection's (already-canceled) ctx, so it still reaches the harness.
func (f *Facade) rejectHITLAfterRequestFailure(hitlID, kind string, cause error) {
	log.Printf("codex facade: %s approval request %s never got a client reply (%v); auto-rejecting so the run doesn't stall", kind, hitlID, cause)
	if err := f.Sessions.ResolveHITL(f.SessionID, hitlID, &godo.HostedAgentResolveHITLRequest{
		Outcome: godo.HostedAgentHITLOutcomeReject,
		Source:  godo.HostedAgentResolutionSourceOutOfBand,
	}); err != nil {
		log.Printf("codex facade: auto-reject ResolveHITL %s failed: %v", hitlID, err)
	}
}
