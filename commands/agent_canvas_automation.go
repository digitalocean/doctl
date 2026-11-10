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
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

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
// What the tab can do here follows the capabilities answer: scheduled (cron)
// automations with a prompt. Event triggers, plugin presets, script bundles,
// uploads and the KV store are not offered, and Canvas hides what the answer
// does not list. Run now (dispatch) waits for a run-now call on the triggers
// API and answers 501 until then.
const canvasAutomationPrefix = "/api/automation"

// canvasRunStatus* are the run states Canvas renders (its AutomationRunStatus
// enum); a trigger execution's status maps onto them in canvasRunStatus.
const (
	canvasRunStatusPending   = "PENDING"
	canvasRunStatusRunning   = "RUNNING"
	canvasRunStatusCompleted = "COMPLETED"
	canvasRunStatusFailed    = "FAILED"
)

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
}

func newCanvasAutomationServer(sessionID string, triggers do.HostedAgentTriggersService) *canvasAutomationServer {
	return &canvasAutomationServer{sessionID: sessionID, triggers: triggers, now: time.Now}
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

// canvasCreateBody is what the tab posts to /v1/preset/prompt.
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

var canvasAutomationID = regexp.MustCompile(`^/v1/([^/]+)(/runs|/dispatch|/tarball)?/?$`)

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
			if _, ok := s.owned(w, id); !ok {
				return
			}
			writeDetail(w, http.StatusNotImplemented, "Run now is not available yet for DigitalOcean triggers; the automation runs on its schedule")
		case sub == "/tarball":
			writeDetail(w, http.StatusNotImplemented, "a DigitalOcean trigger has no script bundle to download")
		default:
			writeDetail(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// capabilities is the "cron only" shape of the Canvas automation interface:
// scheduled prompts, no event sources, no bundles. Canvas hides what is not
// listed, so the tab offers exactly what the triggers API behind it does.
func (s *canvasAutomationServer) capabilities() map[string]any {
	return map[string]any{
		"ready":        true,
		"triggerKinds": []string{"cron"},
		"eventSources": []string{},
		"eventTypes":   []string{},
		"triggers": map[string]any{
			"cron": map[string]any{
				"minIntervalSeconds": canvasCronMinIntervalSeconds,
				"timezones":          canvasTimezones,
			},
		},
		"features": []string{"presetPrompt"},
	}
}

// owned fetches a trigger and checks it is bound to this session; anything
// else is "not found" to the tab, so one session's Canvas can neither read
// nor change another session's triggers.
func (s *canvasAutomationServer) owned(w http.ResponseWriter, id string) (*godo.HostedAgentTrigger, bool) {
	t, err := s.triggers.Get(id)
	if err != nil {
		writeGodoError(w, err)
		return nil, false
	}
	if t == nil || t.HostedAgentTrigger == nil || t.BoundSessionID != s.sessionID {
		writeDetail(w, http.StatusNotFound, "Automation not found")
		return nil, false
	}
	return t.HostedAgentTrigger, true
}

// sessionTriggers lists every trigger bound to this session, all pages.
func (s *canvasAutomationServer) sessionTriggers() ([]godo.HostedAgentTrigger, error) {
	var out []godo.HostedAgentTrigger
	token := ""
	for page := 0; page < 50; page++ {
		items, next, err := s.triggers.List(&godo.HostedAgentTriggerListOptions{PageSize: 100, PageToken: token})
		if err != nil {
			return nil, err
		}
		for _, t := range items {
			if t.HostedAgentTrigger != nil && t.BoundSessionID == s.sessionID {
				out = append(out, *t.HostedAgentTrigger)
			}
		}
		if next == "" {
			break
		}
		token = next
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Time.After(out[j].CreatedAt.Time) })
	return out, nil
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
		page = append(page, toCanvasAutomation(all[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"automations": page, "total": len(all)})
}

func (s *canvasAutomationServer) get(w http.ResponseWriter, id string) {
	t, ok := s.owned(w, id)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toCanvasAutomation(*t))
}

func (s *canvasAutomationServer) delete(w http.ResponseWriter, id string) {
	if _, ok := s.owned(w, id); !ok {
		return
	}
	if err := s.triggers.Delete(id); err != nil {
		writeGodoError(w, err)
		return
	}
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
	if body.Trigger.Type != "cron" {
		writeDetail(w, http.StatusBadRequest, "only scheduled (cron) automations are available for this session")
		return
	}
	if errs := validateCanvasTrigger(&body.Trigger); len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": errs[0].Message, "errors": errs})
		return
	}
	tz := body.Trigger.Timezone
	if tz == "" {
		tz = "UTC"
	}
	result, err := s.triggers.Create(&godo.HostedAgentTriggerCreateRequest{
		Kind:           godo.HostedAgentTriggerKindCron,
		Name:           body.Name,
		SessionMode:    godo.HostedAgentTriggerSessionModeReuse,
		PromptTemplate: body.Prompt,
		Output:         godo.HostedAgentTriggerOutputWrite{Mode: godo.HostedAgentTriggerOutputModeNone},
		BoundSessionID: s.sessionID,
		Cron:           &godo.HostedAgentCreateCronConfig{CronExpr: body.Trigger.Schedule, Timezone: tz},
	})
	if err != nil {
		writeGodoError(w, err)
		return
	}
	if result == nil || result.Trigger == nil || result.Trigger.HostedAgentTrigger == nil {
		writeDetail(w, http.StatusBadGateway, "the triggers API returned no trigger")
		return
	}
	writeJSON(w, http.StatusCreated, toCanvasAutomation(*result.Trigger.HostedAgentTrigger))
}

func (s *canvasAutomationServer) patch(w http.ResponseWriter, r *http.Request, id string) {
	current, ok := s.owned(w, id)
	if !ok {
		return
	}
	var body canvasPatchBody
	if err := readJSON(r, &body); err != nil {
		writeDetail(w, http.StatusBadRequest, err.Error())
		return
	}
	update := &godo.HostedAgentTriggerUpdateRequest{}
	changed := false
	if body.Name != nil && strings.TrimSpace(*body.Name) != "" && *body.Name != current.Name {
		update.Name = *body.Name
		changed = true
	}
	if body.Prompt != nil && *body.Prompt != current.PromptTemplate {
		update.PromptTemplate = *body.Prompt
		changed = true
	}
	if body.Enabled != nil {
		want := godo.HostedAgentTriggerStatusPaused
		if *body.Enabled {
			want = godo.HostedAgentTriggerStatusActive
		}
		if want != current.Status {
			update.Status = want
			changed = true
		}
	}
	if body.Trigger != nil && body.Trigger.Schedule != "" {
		if current.Kind != godo.HostedAgentTriggerKindCron {
			writeDetail(w, http.StatusBadRequest, "this automation is a webhook trigger; its schedule cannot be set")
			return
		}
		tz := body.Trigger.Timezone
		if tz == "" && body.Timezone != nil {
			tz = *body.Timezone
		}
		if tz == "" && current.Cron != nil {
			tz = current.Cron.Timezone
		}
		if tz == "" {
			tz = "UTC"
		}
		probe := canvasTrigger{Type: "cron", Schedule: body.Trigger.Schedule, Timezone: tz}
		if errs := validateCanvasTrigger(&probe); len(errs) > 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": errs[0].Message, "errors": errs})
			return
		}
		if current.Cron == nil || current.Cron.CronExpr != body.Trigger.Schedule || current.Cron.Timezone != tz {
			update.Cron = &godo.HostedAgentCreateCronConfig{CronExpr: body.Trigger.Schedule, Timezone: tz}
			changed = true
		}
	}
	if !changed {
		writeJSON(w, http.StatusOK, toCanvasAutomation(*current))
		return
	}
	t, err := s.triggers.Update(id, update)
	if err != nil {
		writeGodoError(w, err)
		return
	}
	if t == nil || t.HostedAgentTrigger == nil {
		writeDetail(w, http.StatusBadGateway, "the triggers API returned no trigger")
		return
	}
	writeJSON(w, http.StatusOK, toCanvasAutomation(*t.HostedAgentTrigger))
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
	for page := 0; page < 20 && len(all) < offset+limit; page++ {
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
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs, "total": len(all), "status_counts": counts})
}

// toCanvasAutomation maps a trigger onto the automation the tab reads. A
// webhook trigger bound to the session (made elsewhere) shows as an event
// automation on its provider, firing on every delivery.
func toCanvasAutomation(t godo.HostedAgentTrigger) canvasAutomation {
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
		source := "custom"
		if t.Webhook != nil && t.Webhook.Provider != "" {
			source = string(t.Webhook.Provider)
		}
		a.Trigger = canvasTrigger{Type: "event", Source: source, On: "*", ScheduleHuman: "On every " + source + " webhook delivery"}
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
	}
	if run.Status == canvasRunStatusCompleted || run.Status == canvasRunStatusFailed {
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
	return strings.ToUpper(string(s))
}

func canvasTime(t godo.Timestamp) string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// canvasEveryNMinutes matches the schedules Canvas itself reads as "every N
// minutes" when it checks the minimum interval.
var canvasEveryNMinutes = regexp.MustCompile(`^(?:\*|\*/(\d+)) \* \* \* \*$`)

// validateCanvasTrigger checks a cron draft the way the tab expects: five
// fields, an interval not under the minimum, a timezone Go knows.
func validateCanvasTrigger(t *canvasTrigger) []canvasValidationError {
	if t == nil {
		return nil
	}
	var errs []canvasValidationError
	if t.Type != "cron" {
		return append(errs, canvasValidationError{Field: "trigger.type", Code: "unsupported_trigger", Message: "only scheduled (cron) automations are available for this session"})
	}
	expr := strings.TrimSpace(t.Schedule)
	if len(strings.Fields(expr)) != 5 {
		errs = append(errs, canvasValidationError{Field: "trigger.schedule", Code: "invalid_cron", Message: "the schedule must have five fields: minute hour day-of-month month day-of-week"})
	} else if m := canvasEveryNMinutes.FindStringSubmatch(expr); m != nil {
		every := 1
		if m[1] != "" {
			every, _ = strconv.Atoi(m[1])
		}
		if every*60 < canvasCronMinIntervalSeconds {
			errs = append(errs, canvasValidationError{Field: "trigger.schedule", Code: "interval_too_short", Message: fmt.Sprintf("Minimum interval for this session is %d minutes.", canvasCronMinIntervalSeconds/60)})
		}
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
