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

package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // the timezone pick list must load on a Windows build too

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// Agent Canvas's Automate tab talks to an "automation service" on the
// agent-server's origin under /api/automation (health, capabilities, list,
// create, update, delete, runs, dispatch). The openhands template ships no
// such service: on DigitalOcean the same job is done by the Hosted Agents
// triggers API. So when `port-forward` serves an openhands session's Canvas
// port, doctl answers /api/automation itself, on the local listener, with the
// user's token, and maps every call onto triggers. A trigger made from the
// tab is bound to the session (session_mode reuse), so it is the one trigger
// the console's Triggers page and `doctl harness-runtime triggers` show too.
//
// The page that drives this API is served from the sandbox, which the agent
// itself can change, so a write that costs or acts (create, enable, run
// now, an edit of what runs or when) is confirmed out of band, with the
// prompt shown, in the port-forward terminal unless the user
// started with --automate=allow (canvasConfirmFunc). Reads are free.
//
// What the tab can do here follows the capabilities answer: scheduled (cron)
// automations with a prompt, and event automations on GitHub or GitLab, which
// are webhook triggers with an event filter (webhook.events); the webhook URL
// and the one-time secret the user must paste into the provider are printed
// in the terminal where port-forward runs, since the tab has no place for
// them, and the URL rides along in the automation's summary line. Each is created with keep_session_awake, since
// the tab's session is open in a browser and never paused: the trigger runs
// while the session is awake and leaves it awake after the run. Event triggers, plugin presets, script bundles,
// uploads and the KV store are not offered, and Canvas hides what the answer
// does not list. Run now (dispatch) fires the trigger through the triggers
// API's run-now route, and answers 501 in a region that does not have that
// route yet.
const canvasAutomationPrefix = "/api/automation"

// canvasEventSources are the webhook providers the tab may pick as an event
// source: the ones that name their events in a delivery header, so the
// tab's "on" field can filter on them. custom has no event names.
var canvasEventSources = []string{"github", "gitlab"}

// canvasEventTypes is the tab's "on" pick list: GitHub event[.action] names
// an automation commonly runs on. GitLab deliveries are named the same way
// by the triggers API (merge_request.open, push).
var canvasEventTypes = []string{
	"push",
	"pull_request.opened", "pull_request.synchronize", "pull_request.reopened", "pull_request.closed", "pull_request.labeled",
	"pull_request_review.submitted", "pull_request_review_comment.created",
	"issues.opened", "issues.labeled", "issues.closed",
	"issue_comment.created",
	"release.published",
	"workflow_run.completed",
	"merge_request.open", "merge_request.update", "merge_request.merge",
}

// canvasEventName is the shape the triggers API accepts for an event.
var canvasEventName = regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_]+)?$`)

// canvasRunStatus* are the run states Canvas renders (its AutomationRunStatus
// enum); a trigger execution's status maps onto them in canvasRunStatus.
const (
	canvasRunStatusPending   = "PENDING"
	canvasRunStatusRunning   = "RUNNING"
	canvasRunStatusCompleted = "COMPLETED"
	canvasRunStatusFailed    = "FAILED"
	canvasRunStatusCancelled = "CANCELLED"
	canvasRunStatusSkipped   = "SKIPPED"
)

// canvasRunPages bounds the executions read for one run-history answer.
// Pages are read until the tab's window (offset+limit) is filled, so the
// usual call is one page; when more pages remain, total is reported as one
// more than what was read, which keeps the tab's "more" alive without
// walking the whole history on every poll. The per-status counts cover
// what was read.
const canvasRunPages = 10

// canvasCronMinIntervalSeconds is the shortest schedule the tab offers; it is
// the figure the Canvas automation interface documents for its own service
// and keeps a mistyped "* * * * *" from firing a session every minute.
const canvasCronMinIntervalSeconds = 300

// canvasTimezones is the pick list the tab's timezone field shows. UTC first,
// then the zones a DigitalOcean team is most likely to schedule in.
var canvasTimezones = []string{
	"UTC",
	"America/New_York", "America/Chicago", "America/Denver", "America/Los_Angeles", "America/Sao_Paulo", "America/Toronto",
	"Europe/London", "Europe/Amsterdam", "Europe/Berlin", "Europe/Paris", "Europe/Madrid",
	"Asia/Kolkata", "Asia/Singapore", "Asia/Tokyo", "Asia/Shanghai", "Asia/Dubai",
	"Australia/Sydney",
}

// canvasAutomationServer answers Canvas's automation API for one session.
type canvasAutomationServer struct {
	sessionID string
	triggers  do.HostedAgentTriggersService
	now       func() time.Time
	// out is the port-forward terminal, where a new event automation's
	// webhook URL and one-time secret are printed; nil prints nothing.
	out io.Writer
	// confirm, when set, is asked before a write that costs or acts (create,
	// enable, run now, an edit of the prompt, schedule or events); it
	// answers no with a reason the tab shows. Reads, renames, disables and
	// deletes never ask: they cost nothing.
	confirm canvasConfirmFunc
	// listCache keeps the last team scan for canvasListCacheTTL, so a tab
	// refresh is not fifty API calls each time; invalidated by every write.
	listMu     sync.Mutex
	listCached []canvasTriggerRow
	listAt     time.Time
}

// canvasConfirmFunc asks the user to allow a write the tab requested; it
// returns the answer and, on no, the reason to show.
type canvasConfirmFunc func(what string) (bool, string)

// canvasListCacheTTL is how long a team scan is reused by the tab's list.
const canvasListCacheTTL = 10 * time.Second

func newCanvasAutomationServer(sessionID string, triggers do.HostedAgentTriggersService) *canvasAutomationServer {
	return &canvasAutomationServer{sessionID: sessionID, triggers: triggers, now: time.Now}
}

// canvasAutomationHandler is the automation API as port-forward mounts it:
// behind the loopback Host check every token-adding proxy here has
// (loopbackOnly), and behind canvasSameOrigin, which is what makes it safe
// next to other forwarded ports.
func canvasAutomationHandler(sessionID string, triggers do.HostedAgentTriggersService, out io.Writer, confirm canvasConfirmFunc) http.Handler {
	srv := newCanvasAutomationServer(sessionID, triggers)
	srv.out = out
	srv.confirm = confirm
	return loopbackOnly(canvasSameOrigin(srv))
}

// allowed asks the confirmation hook, when there is one, and answers the
// tab with 403 when the user says no (or nobody can be asked).
func (s *canvasAutomationServer) allowed(w http.ResponseWriter, what string) bool {
	if s.confirm == nil {
		return true
	}
	ok, why := s.confirm(what)
	if ok {
		return true
	}
	writeDetail(w, http.StatusForbidden, "not allowed: "+why)
	return false
}

// invalidateList drops the cached team scan after a write.
func (s *canvasAutomationServer) invalidateList() {
	s.listMu.Lock()
	s.listCached, s.listAt = nil, time.Time{}
	s.listMu.Unlock()
}

// canvasSameOrigin refuses what a loopback Origin check alone lets through:
// a page on another local port. port-forward often carries one next to
// Canvas (an app the agent built on :3000, say), and a page there could
// otherwise POST to this API with the user's token, as a text/plain body
// needs no CORS preflight. So a request that names an Origin must name this
// listener's own origin, and a write must be JSON, which a page from another
// origin can only send after a preflight that is never answered. Canvas is
// same-origin and sends JSON, so it is untouched; plain tools send no Origin.
func canvasSameOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// An opaque "null" Origin never gets here: loopbackOnly refuses it.
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Scheme != "http" || !strings.EqualFold(u.Host, r.Host) {
				writeDetail(w, http.StatusForbidden, "the automation API answers the Canvas page on this port only")
				return
			}
		}
		// A write with a body must be JSON; a bodiless POST (Canvas's Run now
		// sends none) carries nothing a page could smuggle and passes. A
		// chunked body reads as -1, so anything but 0 is a body.
		hasBody := r.ContentLength != 0
		if hasBody && (r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodPut) {
			ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if ct != "application/json" {
				writeDetail(w, http.StatusUnsupportedMediaType, "the automation API takes application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// canvasAutomation is an automation as the tab reads it.
type canvasAutomation struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Prompt          *string       `json:"prompt"`
	Model           *string       `json:"model"`
	Trigger         canvasTrigger `json:"trigger"`
	Timezone        string        `json:"timezone,omitempty"`
	Enabled         bool          `json:"enabled"`
	CreatedAt       string        `json:"created_at"`
	UpdatedAt       string        `json:"updated_at"`
	LastTriggeredAt *string       `json:"last_triggered_at"`
}

// canvasTrigger is the automation's trigger as the tab reads and writes it.
type canvasTrigger struct {
	Type          string `json:"type"`
	Schedule      string `json:"schedule,omitempty"`
	ScheduleHuman string `json:"schedule_human,omitempty"`
	Timezone      string `json:"timezone,omitempty"`
	Source        string `json:"source,omitempty"`
	On            any    `json:"on,omitempty"`
	Filter        string `json:"filter,omitempty"`
}

// canvasRun is one run as the tab reads it (the run history list).
type canvasRun struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	ConversationID *string `json:"conversation_id"`
	BashCommandID  *string `json:"bash_command_id"`
	ErrorDetail    *string `json:"error_detail"`
	PhaseCode      *string `json:"phase_code"`
	PhaseLabel     *string `json:"phase_label"`
	PhaseUpdatedAt *string `json:"phase_updated_at"`
	StartedAt      string  `json:"started_at"`
	CompletedAt    *string `json:"completed_at"`
}

// canvasValidationError is one entry of the validate answer's errors.
type canvasValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// canvasCreateBody is what the tab posts to /v1/preset/prompt. A model, when
// the form sends one, is read and dropped (see create). An event trigger's
// "on" is a string or a list of strings.
type canvasCreateBody struct {
	Name    string        `json:"name"`
	Prompt  string        `json:"prompt"`
	Trigger canvasTrigger `json:"trigger"`
	Model   *string       `json:"model"`
}

// canvasPatchBody is what the tab sends to PATCH /v1/{id}: any subset of the
// automation; a field left out is left alone.
type canvasPatchBody struct {
	Name     *string        `json:"name"`
	Prompt   *string        `json:"prompt"`
	Enabled  *bool          `json:"enabled"`
	Trigger  *canvasTrigger `json:"trigger"`
	Timezone *string        `json:"timezone"`
}

// canvasAutomationID picks the trigger id out of the path. Only the id
// characters the API issues are accepted, so what reaches the godo client's
// path is an id and never a path segment or a query.
var canvasAutomationID = regexp.MustCompile(`^/v1/([A-Za-z0-9_-]+)(/runs|/dispatch|/tarball)?/?$`)

func (s *canvasAutomationServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, canvasAutomationPrefix)
	if path == "" {
		path = "/"
	}
	path = strings.TrimSuffix(path, "/")
	switch {
	case path == "/health" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "doctl", "version": doctl.DoitVersion.String()})
	case path == "/v1/capabilities" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, s.capabilities())
	case path == "/v1" && r.Method == http.MethodGet:
		s.list(w, r)
	case path == "/v1/validate" && r.Method == http.MethodPost:
		s.validate(w, r)
	case path == "/v1/preset/prompt" && r.Method == http.MethodPost:
		s.create(w, r)
	case path == "/v1/preset/plugin" || path == "/v1/uploads" || (path == "/v1" && r.Method == http.MethodPost):
		writeDetail(w, http.StatusNotImplemented, "this session's automations are DigitalOcean triggers, which run a prompt on a schedule; plugin presets and script bundles are not available here")
	case path == "/health" || path == "/v1/capabilities" || path == "/v1/validate" || path == "/v1/preset/prompt" || path == "/v1" || path == "/v1/preset/plugin" || path == "/v1/uploads":
		writeDetail(w, http.StatusMethodNotAllowed, "method not allowed")
	default:
		m := canvasAutomationID.FindStringSubmatch(path)
		if m == nil {
			writeDetail(w, http.StatusNotFound, "not found")
			return
		}
		id, sub := m[1], m[2]
		switch {
		case sub == "" && r.Method == http.MethodGet:
			s.get(w, id)
		case sub == "" && r.Method == http.MethodPatch:
			s.patch(w, r, id)
		case sub == "" && r.Method == http.MethodDelete:
			s.delete(w, id)
		case sub == "/runs" && r.Method == http.MethodGet:
			s.runs(w, r, id)
		case sub == "/dispatch" && r.Method == http.MethodPost:
			s.dispatch(w, id)
		case sub == "/tarball":
			writeDetail(w, http.StatusNotImplemented, "a DigitalOcean trigger has no script bundle to download")
		default:
			writeDetail(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// capabilities is the shape of the Canvas automation interface this backend
// serves: scheduled prompts, and event automations on the providers that
// name their events; no plugins, bundles or filter expressions. Canvas hides
// what is not listed, so the tab offers exactly what the triggers API does.
func (s *canvasAutomationServer) capabilities() map[string]any {
	return map[string]any{
		"ready":        true,
		"triggerKinds": []string{"cron", "event"},
		"eventSources": canvasEventSources,
		"eventTypes":   canvasEventTypes,
		"triggers": map[string]any{
			"cron": map[string]any{
				"minIntervalSeconds": canvasCronMinIntervalSeconds,
				"timezones":          canvasTimezones,
			},
			// No filterLanguage: the triggers API filters on event names only,
			// and a filter expression is refused (validateCanvasTrigger).
			"event": map[string]any{},
		},
		"features": []string{"presetPrompt", "webhookDelivery"},
	}
}

// owned fetches a trigger and checks it is bound to this session; anything
// else is "not found" to the tab, so one session's Canvas can neither read
// nor change another session's triggers.
func (s *canvasAutomationServer) owned(w http.ResponseWriter, id string) (*godo.HostedAgentTrigger, bool) {
	t, _, ok := s.ownedWithDetails(w, id)
	return t, ok
}

// ownedWithDetails is owned plus the Canvas fields (the webhook's events).
func (s *canvasAutomationServer) ownedWithDetails(w http.ResponseWriter, id string) (*godo.HostedAgentTrigger, do.HostedAgentTriggerDetails, bool) {
	t, d, err := s.triggers.GetWithDetails(id)
	if err != nil {
		writeGodoError(w, err)
		return nil, do.HostedAgentTriggerDetails{}, false
	}
	if t == nil || t.HostedAgentTrigger == nil || !strings.EqualFold(t.BoundSessionID, s.sessionID) {
		writeDetail(w, http.StatusNotFound, "Automation not found")
		return nil, do.HostedAgentTriggerDetails{}, false
	}
	if d == nil {
		d = &do.HostedAgentTriggerDetails{}
	}
	return t.HostedAgentTrigger, *d, true
}

// sessionTriggers lists every trigger bound to this session: one call with
// the API's bound_session_id filter. A region whose triggers API predates
// the filter ignores it and returns the team's triggers, so the rows are
// still checked here and paged (100 a page, up to 50 pages). The session
// id is compared case-insensitively: the server stores it in lower case
// and a ref typed in upper case must still find its triggers.
func (s *canvasAutomationServer) sessionTriggers() ([]canvasTriggerRow, error) {
	s.listMu.Lock()
	if s.listCached != nil && s.now().Sub(s.listAt) < canvasListCacheTTL {
		cached := s.listCached
		s.listMu.Unlock()
		return cached, nil
	}
	s.listMu.Unlock()
	var out []canvasTriggerRow
	token := ""
	for page := 0; page < 50; page++ {
		items, details, next, err := s.triggers.ListWithDetails(strings.ToLower(s.sessionID), 100, token)
		if err != nil {
			return nil, err
		}
		for i, t := range items {
			if t.HostedAgentTrigger != nil && strings.EqualFold(t.BoundSessionID, s.sessionID) {
				row := canvasTriggerRow{trigger: *t.HostedAgentTrigger}
				if i < len(details) {
					row.details = details[i]
				}
				out = append(out, row)
			}
		}
		if next == "" {
			break
		}
		token = next
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].trigger.CreatedAt.Time.After(out[j].trigger.CreatedAt.Time) })
	if out == nil {
		out = []canvasTriggerRow{}
	}
	s.listMu.Lock()
	s.listCached, s.listAt = out, s.now()
	s.listMu.Unlock()
	return out, nil
}

// canvasTriggerRow is a trigger with its Canvas fields.
type canvasTriggerRow struct {
	trigger godo.HostedAgentTrigger
	details do.HostedAgentTriggerDetails
}

func (s *canvasAutomationServer) list(w http.ResponseWriter, r *http.Request) {
	limit, offset := pageParams(r, 50)
	all, err := s.sessionTriggers()
	if err != nil {
		writeGodoError(w, err)
		return
	}
	page := []canvasAutomation{}
	for i := offset; i < len(all) && i < offset+limit; i++ {
		page = append(page, toCanvasAutomation(all[i].trigger, all[i].details))
	}
	writeJSON(w, http.StatusOK, map[string]any{"automations": page, "total": len(all)})
}

func (s *canvasAutomationServer) get(w http.ResponseWriter, id string) {
	t, d, ok := s.ownedWithDetails(w, id)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toCanvasAutomation(*t, d))
}

func (s *canvasAutomationServer) delete(w http.ResponseWriter, id string) {
	if _, ok := s.owned(w, id); !ok {
		return
	}
	if err := s.triggers.Delete(id); err != nil {
		writeGodoError(w, err)
		return
	}
	s.invalidateList()
	w.WriteHeader(http.StatusNoContent)
}

func (s *canvasAutomationServer) validate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Draft struct {
			Trigger *canvasTrigger `json:"trigger"`
		} `json:"draft"`
	}
	if err := readJSON(r, &body); err != nil {
		writeDetail(w, http.StatusBadRequest, err.Error())
		return
	}
	errs := validateCanvasTrigger(body.Draft.Trigger)
	if errs == nil {
		errs = []canvasValidationError{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"valid": len(errs) == 0, "errors": errs})
}

func (s *canvasAutomationServer) create(w http.ResponseWriter, r *http.Request) {
	var body canvasCreateBody
	if err := readJSON(r, &body); err != nil {
		writeDetail(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Prompt) == "" {
		writeDetail(w, http.StatusBadRequest, "name and prompt are required")
		return
	}
	// A model the form may carry is not kept: a trigger runs on the
	// session's active LLM profile, so the automation comes back with model
	// null and the tab shows none, rather than a model that is not used.
	if errs := validateCanvasTrigger(&body.Trigger); len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": errs[0].Message, "errors": errs})
		return
	}
	// keep_session_awake on every trigger made here: the tab's session is
	// open in a browser and never paused, so the trigger must run while it
	// is awake and leave it awake.
	req := &godo.HostedAgentTriggerCreateRequest{
		Name:           body.Name,
		SessionMode:    godo.HostedAgentTriggerSessionModeReuse,
		PromptTemplate: body.Prompt,
		Output:         godo.HostedAgentTriggerOutputWrite{Mode: godo.HostedAgentTriggerOutputModeNone},
		BoundSessionID: s.sessionID,
	}
	var events []string
	switch body.Trigger.Type {
	case "cron":
		tz := body.Trigger.Timezone
		if tz == "" {
			tz = "UTC"
		}
		req.Kind = godo.HostedAgentTriggerKindCron
		req.Cron = &godo.HostedAgentCreateCronConfig{CronExpr: body.Trigger.Schedule, Timezone: tz}
	case "event":
		req.Kind = godo.HostedAgentTriggerKindWebhook
		req.Webhook = &godo.HostedAgentCreateWebhookConfig{Provider: godo.HostedAgentWebhookProviderKey(body.Trigger.Source)}
		events = canvasEventList(body.Trigger.On)
	}
	what := fmt.Sprintf("create automation %q", body.Name)
	switch body.Trigger.Type {
	case "cron":
		what += fmt.Sprintf(" on schedule %q (%s)", req.Cron.CronExpr, req.Cron.Timezone)
	case "event":
		what += fmt.Sprintf(" on %s events %v", body.Trigger.Source, events)
	}
	what += fmt.Sprintf("\n  prompt: %s\n  it runs in this session and keeps the session awake (billed) while it does", canvasPromptPreview(body.Prompt))
	if !s.allowed(w, what) {
		return
	}
	result, details, err := s.triggers.CreateForCanvas(req, events)
	if err != nil {
		writeGodoError(w, err)
		return
	}
	s.invalidateList()
	if result == nil || result.Trigger == nil || result.Trigger.HostedAgentTrigger == nil {
		writeDetail(w, http.StatusBadGateway, "the triggers API returned no trigger")
		return
	}
	if details == nil {
		details = &do.HostedAgentTriggerDetails{}
	}
	t := *result.Trigger.HostedAgentTrigger
	// A region whose triggers API predates keep_session_awake drops it
	// silently, and a trigger there waits for the session to pause, which
	// an open Canvas never does: say so in the terminal, since the tab has
	// nowhere to.
	if !details.KeepSessionAwake && s.out != nil {
		fmt.Fprintf(s.out, "\n  Note: this region's triggers API cannot keep the session awake yet, so %q runs only while this Canvas tab is closed and the session is paused; while the tab is open its runs fail.\n\n", t.Name)
	}
	if t.Kind == godo.HostedAgentTriggerKindWebhook {
		s.printWebhookWiring(t, result.WebhookSecret)
		// A region whose triggers API does not filter events yet drops the
		// filter silently; the user must know the trigger fires on every
		// delivery there, and the tab shows "*" for it.
		if len(events) > 0 && len(details.WebhookEvents) == 0 && s.out != nil {
			fmt.Fprintf(s.out, "  Note: this region's triggers API does not filter events yet, so this automation runs on every %s webhook delivery until it does.\n\n", string(t.Webhook.Provider))
		}
	}
	writeJSON(w, http.StatusCreated, toCanvasAutomation(t, *details))
}

// printWebhookWiring tells the user, in the terminal where port-forward
// runs, where to point the provider's webhook: the tab has no place for the
// URL or the secret, and the secret is shown by the API exactly once.
func (s *canvasAutomationServer) printWebhookWiring(t godo.HostedAgentTrigger, secret string) {
	if s.out == nil {
		return
	}
	webhookURL := ""
	provider := ""
	if t.Webhook != nil {
		webhookURL, provider = t.Webhook.WebhookURL, string(t.Webhook.Provider)
	}
	fmt.Fprintf(s.out, "\nAutomation %q (%s) listens for %s webhooks. Add a webhook in %s with:\n  Payload URL:  %s\n  Content type: application/json\n",
		t.Name, t.TriggerID, provider, provider, webhookURL)
	if strings.TrimSpace(secret) != "" {
		fmt.Fprintf(s.out, "  Secret (shown once, store it now): %s\n", secret)
	}
	fmt.Fprintln(s.out)
}

// canvasEventList is the tab's "on" (a string or a list) as the API's
// event filter; "*" or nothing is no filter.
func canvasEventList(on any) []string {
	var raw []string
	switch v := on.(type) {
	case string:
		raw = []string{v}
	case []any:
		for _, x := range v {
			if str, ok := x.(string); ok {
				raw = append(raw, str)
			}
		}
	case []string:
		raw = v
	}
	var out []string
	for _, e := range raw {
		e = strings.ToLower(strings.TrimSpace(e))
		if e != "" && e != "*" {
			out = append(out, e)
		}
	}
	return out
}

func (s *canvasAutomationServer) patch(w http.ResponseWriter, r *http.Request, id string) {
	current, currentDetails, ok := s.ownedWithDetails(w, id)
	if !ok {
		return
	}
	var body canvasPatchBody
	if err := readJSON(r, &body); err != nil {
		writeDetail(w, http.StatusBadRequest, err.Error())
		return
	}
	// Every change is worked out first and confirmed as one question, then
	// written: an edit that changes what runs, when, or on what (prompt,
	// schedule, event filter, enabling) is asked in the terminal like a
	// create is, since the page can rewrite an enabled automation. A
	// rename or a pause is not asked.
	update := &godo.HostedAgentTriggerUpdateRequest{}
	changed := false
	var asks []string
	if body.Name != nil && strings.TrimSpace(*body.Name) != "" && *body.Name != current.Name {
		update.Name = *body.Name
		changed = true
	}
	if body.Prompt != nil && *body.Prompt != current.PromptTemplate {
		if strings.TrimSpace(*body.Prompt) == "" {
			writeDetail(w, http.StatusBadRequest, "the prompt cannot be empty")
			return
		}
		update.PromptTemplate = *body.Prompt
		changed = true
		asks = append(asks, "prompt: "+canvasPromptPreview(*body.Prompt))
	}
	if body.Enabled != nil {
		want := godo.HostedAgentTriggerStatusPaused
		if *body.Enabled {
			want = godo.HostedAgentTriggerStatusActive
		}
		if want != current.Status {
			update.Status = want
			changed = true
			if want == godo.HostedAgentTriggerStatusActive {
				asks = append(asks, "enable it")
			}
		}
	}
	// An event automation's "on" is the webhook's event filter, its own
	// call on the API; the trigger row's own fields go below.
	var events []string
	setEvents := false
	if body.Trigger != nil && current.Kind == godo.HostedAgentTriggerKindWebhook && body.Trigger.On != nil {
		if strings.TrimSpace(body.Trigger.Filter) != "" {
			writeDetail(w, http.StatusBadRequest, "filter expressions are not supported; pick the events to run on")
			return
		}
		events = canvasEventList(body.Trigger.On)
		for _, e := range events {
			if !canvasEventName.MatchString(e) {
				writeDetail(w, http.StatusBadRequest, fmt.Sprintf("%q is not an event name", e))
				return
			}
		}
		setEvents = true
		asks = append(asks, fmt.Sprintf("events: %v", events))
	}
	// The schedule and the timezone travel together on the API, so an edit
	// to either resends both: a new schedule with the timezone kept, or the
	// current schedule with a new timezone.
	schedule, tz := "", ""
	if body.Trigger != nil {
		schedule, tz = body.Trigger.Schedule, body.Trigger.Timezone
	}
	if tz == "" && body.Timezone != nil {
		tz = *body.Timezone
	}
	if schedule != "" || tz != "" {
		if current.Kind != godo.HostedAgentTriggerKindCron || current.Cron == nil {
			writeDetail(w, http.StatusBadRequest, "this automation is a webhook trigger; it has no schedule or timezone to set")
			return
		}
		if schedule == "" {
			schedule = current.Cron.CronExpr
		}
		if tz == "" {
			tz = current.Cron.Timezone
		}
		if tz == "" {
			tz = "UTC"
		}
		probe := canvasTrigger{Type: "cron", Schedule: schedule, Timezone: tz}
		if errs := validateCanvasTrigger(&probe); len(errs) > 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": errs[0].Message, "errors": errs})
			return
		}
		if current.Cron.CronExpr != schedule || current.Cron.Timezone != tz {
			update.Cron = &godo.HostedAgentCreateCronConfig{CronExpr: schedule, Timezone: tz}
			changed = true
			asks = append(asks, fmt.Sprintf("schedule: %q (%s)", schedule, tz))
		}
	}
	if len(asks) > 0 && !s.allowed(w, fmt.Sprintf("change automation %q\n  %s", current.Name, strings.Join(asks, "\n  "))) {
		return
	}
	if setEvents {
		t, d, err := s.triggers.UpdateWebhookEvents(id, events)
		if err != nil {
			writeGodoError(w, err)
			return
		}
		s.invalidateList()
		if t != nil && t.HostedAgentTrigger != nil {
			current = t.HostedAgentTrigger
		}
		if d != nil {
			currentDetails = *d
		}
	}
	if !changed {
		writeJSON(w, http.StatusOK, toCanvasAutomation(*current, currentDetails))
		return
	}
	t, err := s.triggers.Update(id, update)
	if err != nil {
		writeGodoError(w, err)
		return
	}
	s.invalidateList()
	if t == nil || t.HostedAgentTrigger == nil {
		writeDetail(w, http.StatusBadGateway, "the triggers API returned no trigger")
		return
	}
	writeJSON(w, http.StatusOK, toCanvasAutomation(*t.HostedAgentTrigger, currentDetails))
}

// canvasPromptPreview is the prompt as shown in a terminal question: one
// line, cut at 160 characters, so the user sees what will run.
func canvasPromptPreview(prompt string) string {
	p := strings.Join(strings.Fields(prompt), " ")
	if len(p) > 160 {
		p = p[:157] + "..."
	}
	return fmt.Sprintf("%q", p)
}

// dispatch is the tab's Run now: the trigger is fired through the run-now
// route and the pending execution comes back as a run (201), which the tab
// then polls in the run history. The route is newer than some regions'
// harness-trigger: a 404 or 405 from it after the trigger itself was found
// means the region does not have it yet, which the tab is told as 501.
func (s *canvasAutomationServer) dispatch(w http.ResponseWriter, id string) {
	t, ok := s.owned(w, id)
	if !ok {
		return
	}
	if !s.allowed(w, fmt.Sprintf("run automation %q now", t.Name)) {
		return
	}
	e, err := s.triggers.Run(id)
	if err != nil {
		var er *godo.ErrorResponse
		if errors.As(err, &er) && er.Response != nil && (er.Response.StatusCode == http.StatusNotFound || er.Response.StatusCode == http.StatusMethodNotAllowed) {
			writeDetail(w, http.StatusNotImplemented, "Run now is not available in this region yet; the automation runs on its schedule")
			return
		}
		writeGodoError(w, err)
		return
	}
	if e == nil || e.HostedAgentTriggerExecution == nil {
		writeDetail(w, http.StatusBadGateway, "the triggers API returned no execution")
		return
	}
	writeJSON(w, http.StatusCreated, toCanvasRun(*e.HostedAgentTriggerExecution))
}

// runs is the automation's run history: the trigger's executions, newest
// first as the API lists them, cut to the tab's limit/offset window.
func (s *canvasAutomationServer) runs(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.owned(w, id); !ok {
		return
	}
	limit, offset := pageParams(r, 50)
	var all []godo.HostedAgentTriggerExecution
	token := ""
	more := false
	for page := 0; page < canvasRunPages; page++ {
		items, next, err := s.triggers.ListExecutions(id, &godo.HostedAgentTriggerExecutionListOptions{PageSize: 100, PageToken: token})
		if err != nil {
			writeGodoError(w, err)
			return
		}
		for _, e := range items {
			if e.HostedAgentTriggerExecution != nil {
				all = append(all, *e.HostedAgentTriggerExecution)
			}
		}
		if next == "" {
			break
		}
		token = next
		if len(all) >= offset+limit {
			more = true
			break
		}
	}
	counts := map[string]int{}
	runs := []canvasRun{}
	for i, e := range all {
		run := toCanvasRun(e)
		counts[run.Status]++
		if i >= offset && i < offset+limit {
			runs = append(runs, run)
		}
	}
	total := len(all)
	if more {
		total++
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs, "total": total, "status_counts": counts})
}

// toCanvasAutomation maps a trigger onto the automation the tab reads. A
// webhook trigger bound to the session (made elsewhere) shows as an event
// automation on its provider, firing on every delivery.
func toCanvasAutomation(t godo.HostedAgentTrigger, d do.HostedAgentTriggerDetails) canvasAutomation {
	prompt := t.PromptTemplate
	a := canvasAutomation{
		ID:        t.TriggerID,
		Name:      t.Name,
		Prompt:    &prompt,
		Enabled:   t.Status == godo.HostedAgentTriggerStatusActive,
		CreatedAt: canvasTime(t.CreatedAt),
		UpdatedAt: canvasTime(t.UpdatedAt),
	}
	switch {
	case t.Kind == godo.HostedAgentTriggerKindCron && t.Cron != nil:
		a.Trigger = canvasTrigger{Type: "cron", Schedule: t.Cron.CronExpr, Timezone: t.Cron.Timezone}
		a.Timezone = t.Cron.Timezone
	case t.Kind == godo.HostedAgentTriggerKindWebhook:
		source, webhookURL := "custom", ""
		if t.Webhook != nil {
			if t.Webhook.Provider != "" {
				source = string(t.Webhook.Provider)
			}
			webhookURL = t.Webhook.WebhookURL
		}
		var on any = "*"
		onText := "every " + source + " event"
		switch len(d.WebhookEvents) {
		case 0:
		case 1:
			on, onText = d.WebhookEvents[0], d.WebhookEvents[0]
		default:
			on, onText = d.WebhookEvents, strings.Join(d.WebhookEvents, ", ")
		}
		human := "On " + onText
		if webhookURL != "" {
			human += " · payload URL " + webhookURL
		}
		a.Trigger = canvasTrigger{Type: "event", Source: source, On: on, ScheduleHuman: human}
	default:
		a.Trigger = canvasTrigger{Type: string(t.Kind)}
	}
	return a
}

func toCanvasRun(e godo.HostedAgentTriggerExecution) canvasRun {
	run := canvasRun{ID: e.ExecutionID, Status: canvasRunStatus(e.Status), StartedAt: canvasTime(e.CreatedAt)}
	if e.FailureReason != "" {
		reason := e.FailureReason
		run.ErrorDetail = &reason
	} else if run.Status == canvasRunStatusFailed && e.Status != godo.HostedAgentTriggerExecutionStatusFailed {
		reason := "execution status " + string(e.Status)
		run.ErrorDetail = &reason
	}
	if run.Status != canvasRunStatusPending && run.Status != canvasRunStatusRunning {
		if done := canvasTime(e.UpdatedAt); done != "" {
			run.CompletedAt = &done
		}
	}
	return run
}

func canvasRunStatus(s godo.HostedAgentTriggerExecutionStatus) string {
	switch s {
	case godo.HostedAgentTriggerExecutionStatusPending:
		return canvasRunStatusPending
	case godo.HostedAgentTriggerExecutionStatusRunning:
		return canvasRunStatusRunning
	case godo.HostedAgentTriggerExecutionStatusSucceeded:
		return canvasRunStatusCompleted
	case godo.HostedAgentTriggerExecutionStatusFailed:
		return canvasRunStatusFailed
	}
	// Statuses the API may grow that this build does not know: the two
	// Canvas renders are passed as such; the in-flight names listed in
	// canvasInFlightStatuses show as running, so the tab keeps polling
	// them; anything else reads as failed (the run did not complete), with
	// the raw status kept in error_detail by toCanvasRun. A list, not a
	// pattern: a parked state such as "suspended" must not poll forever.
	raw := strings.ToLower(string(s))
	switch raw {
	case "cancelled", "canceled":
		return canvasRunStatusCancelled
	case "skipped":
		return canvasRunStatusSkipped
	}
	if canvasInFlightStatuses[raw] {
		return canvasRunStatusRunning
	}
	return canvasRunStatusFailed
}

// canvasInFlightStatuses are execution states, beyond pending and running,
// that mean a run is still going.
var canvasInFlightStatuses = map[string]bool{
	"queued": true, "scheduled": true, "starting": true, "dispatching": true,
	"resuming": true, "cancelling": true, "canceling": true,
}

func canvasTime(t godo.Timestamp) string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// validateCanvasTrigger checks a cron draft the way the tab expects: five
// fields, a minute field that parses, an interval not under the minimum, a
// timezone Go knows.
func validateCanvasTrigger(t *canvasTrigger) []canvasValidationError {
	if t == nil {
		return nil
	}
	var errs []canvasValidationError
	if t.Type == "event" {
		source := strings.ToLower(strings.TrimSpace(t.Source))
		if !slices.Contains(canvasEventSources, source) {
			errs = append(errs, canvasValidationError{Field: "trigger.source", Code: "unsupported_source", Message: "event automations run on " + strings.Join(canvasEventSources, " or ") + " webhooks"})
		}
		if strings.TrimSpace(t.Filter) != "" {
			errs = append(errs, canvasValidationError{Field: "trigger.filter", Code: "unsupported_filter", Message: "filter expressions are not supported; pick the events to run on"})
		}
		for _, e := range canvasEventList(t.On) {
			if !canvasEventName.MatchString(e) {
				errs = append(errs, canvasValidationError{Field: "trigger.on", Code: "invalid_event", Message: fmt.Sprintf("%q is not an event name (event or event.action, lower case)", e)})
				break
			}
		}
		return errs
	}
	if t.Type != "cron" {
		return append(errs, canvasValidationError{Field: "trigger.type", Code: "unsupported_trigger", Message: "only scheduled and event automations are available for this session"})
	}
	expr := strings.TrimSpace(t.Schedule)
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		errs = append(errs, canvasValidationError{Field: "trigger.schedule", Code: "invalid_cron", Message: "the schedule must have five fields: minute hour day-of-month month day-of-week"})
	} else if minutes, ok := cronField(fields[0], 59); !ok {
		errs = append(errs, canvasValidationError{Field: "trigger.schedule", Code: "invalid_cron", Message: fmt.Sprintf("the minute field %q is not a cron minute expression", fields[0])})
	} else if hours, ok := cronField(fields[1], 23); !ok {
		errs = append(errs, canvasValidationError{Field: "trigger.schedule", Code: "invalid_cron", Message: fmt.Sprintf("the hour field %q is not a cron hour expression", fields[1])})
	} else if gap := cronMinGapMinutes(minutes, cronHasConsecutive(hours, 23)); gap*60 < canvasCronMinIntervalSeconds {
		errs = append(errs, canvasValidationError{Field: "trigger.schedule", Code: "interval_too_short", Message: fmt.Sprintf("Minimum interval for this session is %d minutes.", canvasCronMinIntervalSeconds/60)})
	}
	if t.Timezone != "" {
		if _, err := time.LoadLocation(t.Timezone); err != nil {
			errs = append(errs, canvasValidationError{Field: "trigger.timezone", Code: "invalid_timezone", Message: fmt.Sprintf("%q is not a known timezone", t.Timezone)})
		}
	}
	return errs
}

func pageParams(r *http.Request, defaultLimit int) (limit, offset int) {
	limit, offset = defaultLimit, 0
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 500 {
		limit = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v >= 0 {
		offset = v
	}
	return limit, offset
}

func readJSON(r *http.Request, into any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("reading the request: %w", err)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return errors.New("a JSON body is required")
	}
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("the body is not the JSON expected: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeDetail is the error shape Canvas reads ({"detail": "..."}).
func writeDetail(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

// writeGodoError passes the API's own status and message to the tab (a 404,
// a 409 for a limit, a 403), and makes anything else a 502 with the text.
func writeGodoError(w http.ResponseWriter, err error) {
	var er *godo.ErrorResponse
	if errors.As(err, &er) && er.Response != nil && er.Response.StatusCode >= 400 {
		msg := er.Message
		if msg == "" {
			msg = http.StatusText(er.Response.StatusCode)
		}
		writeDetail(w, er.Response.StatusCode, msg)
		return
	}
	writeDetail(w, http.StatusBadGateway, "the DigitalOcean triggers API did not answer: "+err.Error())
}

// cronField expands a cron minute or hour field ("*", "*/15", "0,30",
// "0-59/5", "5-10,45") into the sorted set of values it names, 0..max; ok is
// false when the field is not such an expression.
func cronField(field string, max int) ([]int, bool) {
	set := map[int]bool{}
	for _, part := range strings.Split(field, ",") {
		step := 1
		if i := strings.Index(part, "/"); i >= 0 {
			n, err := strconv.Atoi(part[i+1:])
			if err != nil || n < 1 || n > max {
				return nil, false
			}
			step, part = n, part[:i]
		}
		lo, hi := 0, max
		switch {
		case part == "*":
		case strings.Contains(part, "-"):
			bounds := strings.SplitN(part, "-", 2)
			a, err1 := strconv.Atoi(bounds[0])
			b, err2 := strconv.Atoi(bounds[1])
			if err1 != nil || err2 != nil || a < 0 || b > max || a > b {
				return nil, false
			}
			lo, hi = a, b
		default:
			a, err := strconv.Atoi(part)
			if err != nil || a < 0 || a > max {
				return nil, false
			}
			lo, hi = a, a
			if step != 1 {
				hi = max // "5/10" means every 10 from 5, as cron reads it
			}
		}
		for m := lo; m <= hi; m += step {
			set[m] = true
		}
	}
	if len(set) == 0 {
		return nil, false
	}
	out := make([]int, 0, len(set))
	for m := range set {
		out = append(out, m)
	}
	sort.Ints(out)
	return out, true
}

// cronHasConsecutive reports whether a sorted value set names two values in
// a row, the last and the first wrapping (23 then 0 for hours).
func cronHasConsecutive(values []int, max int) bool {
	for i := 1; i < len(values); i++ {
		if values[i]-values[i-1] == 1 {
			return true
		}
	}
	return len(values) > 1 && values[0] == 0 && values[len(values)-1] == max
}

// cronMinGapMinutes is the shortest time between two firings a minute set
// allows: the smallest gap between its minutes within an hour, and, when the
// hour field names two hours in a row (cronHasConsecutive), the gap across
// the top of the hour as well. One minute a day is 60 or more.
func cronMinGapMinutes(minutes []int, hoursMayBeConsecutive bool) int {
	if len(minutes) == 0 {
		return 60
	}
	gap := 60
	for i := 1; i < len(minutes); i++ {
		if d := minutes[i] - minutes[i-1]; d < gap {
			gap = d
		}
	}
	if hoursMayBeConsecutive {
		if d := 60 - minutes[len(minutes)-1] + minutes[0]; d < gap {
			gap = d
		}
	}
	return gap
}
