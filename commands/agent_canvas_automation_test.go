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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func canvasReq(t *testing.T, h http.Handler, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, "http://127.0.0.1:8000"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}
	return rec, out
}

func cronTrigger(id, session, expr string, status godo.HostedAgentTriggerStatus) do.HostedAgentTrigger {
	return do.HostedAgentTrigger{HostedAgentTrigger: &godo.HostedAgentTrigger{
		TriggerID:      id,
		Name:           "n-" + id,
		Kind:           godo.HostedAgentTriggerKindCron,
		Status:         status,
		SessionMode:    godo.HostedAgentTriggerSessionModeReuse,
		PromptTemplate: "say hi",
		BoundSessionID: session,
		Cron:           &godo.HostedAgentCronConfig{CronExpr: expr, Timezone: "UTC"},
		CreatedAt:      godo.Timestamp{Time: time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)},
		UpdatedAt:      godo.Timestamp{Time: time.Date(2026, 11, 10, 9, 5, 0, 0, time.UTC)},
	}}
}

// The tab's first two calls: health says the service is there, capabilities
// says scheduled prompts only, so Canvas hides events, plugins and bundles.
func TestCanvasAutomation_HealthAndCapabilities(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		rec, body := canvasReq(t, h, http.MethodGet, "/api/automation/health", "")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", body["status"])

		rec, body = canvasReq(t, h, http.MethodGet, "/api/automation/v1/capabilities", "")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, true, body["ready"])
		assert.Equal(t, []any{"cron"}, body["triggerKinds"])
		assert.Equal(t, []any{}, body["eventSources"])
		assert.Equal(t, []any{"presetPrompt"}, body["features"])
		cron := body["triggers"].(map[string]any)["cron"].(map[string]any)
		assert.EqualValues(t, canvasCronMinIntervalSeconds, cron["minIntervalSeconds"])
		assert.Contains(t, cron["timezones"], "UTC")
	})
}

// Listing shows only the triggers bound to this session, mapped to the
// automation the tab reads: enabled from the status, the schedule from cron.
func TestCanvasAutomation_ListIsScopedToTheSession(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().
			List(&godo.HostedAgentTriggerListOptions{PageSize: 100}).
			Return([]do.HostedAgentTrigger{
				cronTrigger("tr_mine", "sess_1", "0 9 * * 1-5", godo.HostedAgentTriggerStatusActive),
				cronTrigger("tr_other", "sess_2", "0 1 * * *", godo.HostedAgentTriggerStatusActive),
				cronTrigger("tr_paused", "sess_1", "30 1 * * *", godo.HostedAgentTriggerStatusPaused),
			}, "", nil)
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		rec, body := canvasReq(t, h, http.MethodGet, "/api/automation/v1?limit=50&offset=0", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.EqualValues(t, 2, body["total"])
		items := body["automations"].([]any)
		require.Len(t, items, 2)
		ids := []string{items[0].(map[string]any)["id"].(string), items[1].(map[string]any)["id"].(string)}
		assert.ElementsMatch(t, []string{"tr_mine", "tr_paused"}, ids)
		for _, it := range items {
			a := it.(map[string]any)
			tr := a["trigger"].(map[string]any)
			assert.Equal(t, "cron", tr["type"])
			assert.Equal(t, "UTC", tr["timezone"])
			assert.Equal(t, "say hi", a["prompt"])
			if a["id"] == "tr_paused" {
				assert.Equal(t, false, a["enabled"])
				assert.Equal(t, "30 1 * * *", tr["schedule"])
			} else {
				assert.Equal(t, true, a["enabled"])
			}
			assert.Equal(t, "2026-11-10T09:00:00Z", a["created_at"])
		}
	})
}

// Create maps the tab's prompt preset onto a reuse-mode cron trigger bound to
// the session, with no output delivery, and answers 201 with the automation.
func TestCanvasAutomation_CreateBindsACronTriggerToTheSession(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().
			Create(&godo.HostedAgentTriggerCreateRequest{
				Kind:           godo.HostedAgentTriggerKindCron,
				Name:           "Nightly digest",
				SessionMode:    godo.HostedAgentTriggerSessionModeReuse,
				PromptTemplate: "Summarise yesterday",
				Output:         godo.HostedAgentTriggerOutputWrite{Mode: godo.HostedAgentTriggerOutputModeNone},
				BoundSessionID: "sess_1",
				Cron:           &godo.HostedAgentCreateCronConfig{CronExpr: "30 1 * * *", Timezone: "Europe/London"},
			}).
			Return(&do.HostedAgentTriggerCreateResult{Trigger: func() *do.HostedAgentTrigger {
				tr := cronTrigger("tr_new", "sess_1", "30 1 * * *", godo.HostedAgentTriggerStatusActive)
				tr.Name = "Nightly digest"
				tr.Cron.Timezone = "Europe/London"
				return &tr
			}()}, nil)
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		rec, body := canvasReq(t, h, http.MethodPost, "/api/automation/v1/preset/prompt",
			`{"name":"Nightly digest","prompt":"Summarise yesterday","trigger":{"type":"cron","schedule":"30 1 * * *","timezone":"Europe/London"}}`)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		assert.Equal(t, "tr_new", body["id"])
		assert.Equal(t, "Europe/London", body["timezone"])
	})
}

// An event draft is refused before any API call: the capabilities answer
// offered cron only, and the refusal says so.
func TestCanvasAutomation_CreateRefusesEventAndBadSchedules(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		rec, body := canvasReq(t, h, http.MethodPost, "/api/automation/v1/preset/prompt",
			`{"name":"x","prompt":"y","trigger":{"type":"event","source":"github","on":"push"}}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, body["detail"], "scheduled")

		rec, body = canvasReq(t, h, http.MethodPost, "/api/automation/v1/preset/prompt",
			`{"name":"x","prompt":"y","trigger":{"type":"cron","schedule":"* * * * *"}}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, body["detail"], "Minimum interval")

		rec, body = canvasReq(t, h, http.MethodPost, "/api/automation/v1/preset/prompt",
			`{"name":"x","prompt":"y","trigger":{"type":"cron","schedule":"0 9 * *"}}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, body["detail"], "five fields")
	})
}

// Validate answers the way the tab's live check expects: valid, or a list
// with the field, a code and a message.
func TestCanvasAutomation_Validate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		rec, body := canvasReq(t, h, http.MethodPost, "/api/automation/v1/validate",
			`{"draft":{"trigger":{"type":"cron","schedule":"*/5 * * * *","timezone":"UTC"}}}`)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, true, body["valid"])

		rec, body = canvasReq(t, h, http.MethodPost, "/api/automation/v1/validate",
			`{"draft":{"trigger":{"type":"cron","schedule":"*/2 * * * *","timezone":"Mars/Olympus"}}}`)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, false, body["valid"])
		errs := body["errors"].([]any)
		require.Len(t, errs, 2)
		assert.Equal(t, "interval_too_short", errs[0].(map[string]any)["code"])
		assert.Equal(t, "invalid_timezone", errs[1].(map[string]any)["code"])
	})
}

// Enable/disable from the tab is a status update; a name, prompt or schedule
// change goes along only when it differs, and nothing is sent when nothing
// changed.
func TestCanvasAutomation_PatchTogglesAndEdits(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		current := cronTrigger("tr_1", "sess_1", "0 9 * * *", godo.HostedAgentTriggerStatusActive)
		paused := cronTrigger("tr_1", "sess_1", "0 9 * * *", godo.HostedAgentTriggerStatusPaused)
		tm.hostedAgentTriggers.EXPECT().Get("tr_1").Return(&current, nil).Times(3)
		tm.hostedAgentTriggers.EXPECT().
			Update("tr_1", &godo.HostedAgentTriggerUpdateRequest{Status: godo.HostedAgentTriggerStatusPaused}).
			Return(&paused, nil)
		tm.hostedAgentTriggers.EXPECT().
			Update("tr_1", &godo.HostedAgentTriggerUpdateRequest{
				Name:           "renamed",
				PromptTemplate: "new prompt",
				Cron:           &godo.HostedAgentCreateCronConfig{CronExpr: "0 10 * * *", Timezone: "UTC"},
			}).
			Return(&current, nil)
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)

		rec, body := canvasReq(t, h, http.MethodPatch, "/api/automation/v1/tr_1", `{"enabled":false}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, false, body["enabled"])

		rec, _ = canvasReq(t, h, http.MethodPatch, "/api/automation/v1/tr_1",
			`{"name":"renamed","prompt":"new prompt","trigger":{"type":"cron","schedule":"0 10 * * *"}}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		// The tab echoes the whole automation back with one toggle untouched:
		// same values, no Update call (gomock would fail on an unexpected one).
		rec, _ = canvasReq(t, h, http.MethodPatch, "/api/automation/v1/tr_1", `{"name":"n-tr_1","prompt":"say hi","enabled":true}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	})
}

// A trigger bound to another session is "not found" to this tab, for get,
// patch and delete alike, and nothing is changed or deleted.
func TestCanvasAutomation_OtherSessionsTriggersAreInvisible(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		other := cronTrigger("tr_x", "sess_other", "0 9 * * *", godo.HostedAgentTriggerStatusActive)
		tm.hostedAgentTriggers.EXPECT().Get("tr_x").Return(&other, nil).Times(3)
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		for _, m := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
			rec, _ := canvasReq(t, h, m, "/api/automation/v1/tr_x", `{"enabled":false}`)
			assert.Equal(t, http.StatusNotFound, rec.Code, m)
		}
	})
}

// Delete answers 204 once the API has removed the session's own trigger.
func TestCanvasAutomation_Delete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		mine := cronTrigger("tr_1", "sess_1", "0 9 * * *", godo.HostedAgentTriggerStatusActive)
		tm.hostedAgentTriggers.EXPECT().Get("tr_1").Return(&mine, nil)
		tm.hostedAgentTriggers.EXPECT().Delete("tr_1").Return(nil)
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		rec, _ := canvasReq(t, h, http.MethodDelete, "/api/automation/v1/tr_1", "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

// Run history is the trigger's executions in the tab's shape: its status
// words, the failure reason as error detail, counts per status.
func TestCanvasAutomation_RunsMapExecutions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		mine := cronTrigger("tr_1", "sess_1", "0 9 * * *", godo.HostedAgentTriggerStatusActive)
		tm.hostedAgentTriggers.EXPECT().Get("tr_1").Return(&mine, nil)
		at := godo.Timestamp{Time: time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)}
		done := godo.Timestamp{Time: time.Date(2026, 11, 10, 9, 2, 0, 0, time.UTC)}
		tm.hostedAgentTriggers.EXPECT().
			ListExecutions("tr_1", &godo.HostedAgentTriggerExecutionListOptions{PageSize: 100}).
			Return([]do.HostedAgentTriggerExecution{
				{HostedAgentTriggerExecution: &godo.HostedAgentTriggerExecution{ExecutionID: "ex_3", Status: godo.HostedAgentTriggerExecutionStatusRunning, CreatedAt: at}},
				{HostedAgentTriggerExecution: &godo.HostedAgentTriggerExecution{ExecutionID: "ex_2", Status: godo.HostedAgentTriggerExecutionStatusFailed, FailureReason: "session gone", CreatedAt: at, UpdatedAt: done}},
				{HostedAgentTriggerExecution: &godo.HostedAgentTriggerExecution{ExecutionID: "ex_1", Status: godo.HostedAgentTriggerExecutionStatusSucceeded, CreatedAt: at, UpdatedAt: done}},
			}, "page2", nil)
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		rec, body := canvasReq(t, h, http.MethodGet, "/api/automation/v1/tr_1/runs?limit=2&offset=0", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.EqualValues(t, 4, body["total"], "one page filled the window; a page remained, so total says there is more")
		runs := body["runs"].([]any)
		require.Len(t, runs, 2)
		assert.Equal(t, "RUNNING", runs[0].(map[string]any)["status"])
		assert.Nil(t, runs[0].(map[string]any)["completed_at"])
		assert.Equal(t, "FAILED", runs[1].(map[string]any)["status"])
		assert.Equal(t, "session gone", runs[1].(map[string]any)["error_detail"])
		assert.Equal(t, "2026-11-10T09:02:00Z", runs[1].(map[string]any)["completed_at"])
		counts := body["status_counts"].(map[string]any)
		assert.EqualValues(t, 1, counts["COMPLETED"])
		assert.EqualValues(t, 1, counts["FAILED"])
		assert.EqualValues(t, 1, counts["RUNNING"])
	})
}

// What the triggers API cannot do yet answers 501 with a plain reason, and a
// webhook trigger bound to the session shows as an event automation.
func TestCanvasAutomation_NotImplementedAndWebhookShape(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		hook := do.HostedAgentTrigger{HostedAgentTrigger: &godo.HostedAgentTrigger{
			TriggerID: "tr_h", Name: "gh", Kind: godo.HostedAgentTriggerKindWebhook, Status: godo.HostedAgentTriggerStatusActive,
			BoundSessionID: "sess_1", Webhook: &godo.HostedAgentWebhookConfig{Provider: godo.HostedAgentWebhookProviderGitHub},
		}}
		tm.hostedAgentTriggers.EXPECT().Get("tr_h").Return(&hook, nil).Times(2)
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)

		rec, body := canvasReq(t, h, http.MethodGet, "/api/automation/v1/tr_h", "")
		require.Equal(t, http.StatusOK, rec.Code)
		tr := body["trigger"].(map[string]any)
		assert.Equal(t, "event", tr["type"])
		assert.Equal(t, "github", tr["source"])
		assert.Equal(t, "*", tr["on"])

		rec, _ = canvasReq(t, h, http.MethodPost, "/api/automation/v1/tr_h/dispatch", "")
		assert.Equal(t, http.StatusNotImplemented, rec.Code)
		rec, _ = canvasReq(t, h, http.MethodPost, "/api/automation/v1/preset/plugin", `{}`)
		assert.Equal(t, http.StatusNotImplemented, rec.Code)
		rec, _ = canvasReq(t, h, http.MethodPost, "/api/automation/v1/uploads?name=x", `{}`)
		assert.Equal(t, http.StatusNotImplemented, rec.Code)
	})
}

// The API's own status and message reach the tab (a 404 for a gone trigger,
// a 409 for a limit); a transport failure is a 502.
func TestCanvasAutomation_APIErrorsPassThrough(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().Get("tr_gone").Return(nil, &godo.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound}, Message: "trigger not found",
		})
		tm.hostedAgentTriggers.EXPECT().List(&godo.HostedAgentTriggerListOptions{PageSize: 100}).Return(nil, "", assert.AnError)
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		rec, body := canvasReq(t, h, http.MethodGet, "/api/automation/v1/tr_gone", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "trigger not found", body["detail"])
		rec, _ = canvasReq(t, h, http.MethodGet, "/api/automation/v1", "")
		assert.Equal(t, http.StatusBadGateway, rec.Code)
	})
}

// The minimum interval holds whatever shape the minute field takes: every
// minute written as a range or a list, a step that wraps the hour short,
// while a wide gap passes and a single-hour schedule is not read as wrapping.
func TestCanvasAutomation_MinimumIntervalCoversEveryMinuteShape(t *testing.T) {
	tooShort := []string{"* * * * *", "* * * * 1-5", "0-59 * * * *", "*/58 * * * *", "0,2 * * * *", "0,58 * * * *", "5/3 * * * *"}
	for _, expr := range tooShort {
		errs := validateCanvasTrigger(&canvasTrigger{Type: "cron", Schedule: expr})
		require.Len(t, errs, 1, expr)
		assert.Equal(t, "interval_too_short", errs[0].Code, expr)
	}
	fine := []string{"*/5 * * * *", "0 * * * *", "0,30 * * * *", "0,58 9 * * *", "0-59/10 * * * *", "15 9 * * 1-5", "0 */2 * * *", "0,58 */2 * * *", "0,58 1,3,5 * * *"}
	for _, expr := range []string{"0,58 9,10 * * *", "0,58 23,0 * * *", "0,58 8-10 * * *"} {
		errs := validateCanvasTrigger(&canvasTrigger{Type: "cron", Schedule: expr})
		require.Len(t, errs, 1, expr)
		assert.Equal(t, "interval_too_short", errs[0].Code, expr)
	}
	errs := validateCanvasTrigger(&canvasTrigger{Type: "cron", Schedule: "0 24 * * *"})
	require.Len(t, errs, 1)
	assert.Equal(t, "invalid_cron", errs[0].Code)
	for _, expr := range fine {
		assert.Empty(t, validateCanvasTrigger(&canvasTrigger{Type: "cron", Schedule: expr}), expr)
	}
	for _, expr := range []string{"61 * * * *", "a * * * *", "10-5 * * * *", "*/0 * * * *", "1,,2 * * * *"} {
		errs := validateCanvasTrigger(&canvasTrigger{Type: "cron", Schedule: expr})
		require.Len(t, errs, 1, expr)
		assert.Equal(t, "invalid_cron", errs[0].Code, expr)
	}
}

// A timezone-only edit resends the schedule with the new zone; an empty
// prompt is refused as it is on create.
func TestCanvasAutomation_PatchTimezoneOnlyAndEmptyPrompt(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		current := cronTrigger("tr_1", "sess_1", "0 9 * * *", godo.HostedAgentTriggerStatusActive)
		tm.hostedAgentTriggers.EXPECT().Get("tr_1").Return(&current, nil).Times(2)
		tm.hostedAgentTriggers.EXPECT().
			Update("tr_1", &godo.HostedAgentTriggerUpdateRequest{Cron: &godo.HostedAgentCreateCronConfig{CronExpr: "0 9 * * *", Timezone: "Asia/Tokyo"}}).
			Return(&current, nil)
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		rec, _ := canvasReq(t, h, http.MethodPatch, "/api/automation/v1/tr_1", `{"timezone":"Asia/Tokyo"}`)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec, body := canvasReq(t, h, http.MethodPatch, "/api/automation/v1/tr_1", `{"prompt":"   "}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, body["detail"], "prompt")
	})
}

// A wrong method on a fixed path is 405, not a trigger lookup; statuses the
// mapping does not know read as the two Canvas has or as failed.
func TestCanvasAutomation_MethodNotAllowedAndStatusMapping(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		for _, p := range []string{"/api/automation/v1/validate", "/api/automation/v1/capabilities", "/api/automation/health"} {
			rec, _ := canvasReq(t, h, http.MethodDelete, p, "")
			assert.Equal(t, http.StatusMethodNotAllowed, rec.Code, p)
		}
	})
	assert.Equal(t, "CANCELLED", canvasRunStatus("cancelled"))
	assert.Equal(t, "SKIPPED", canvasRunStatus("skipped"))
	assert.Equal(t, "FAILED", canvasRunStatus("timed_out"))
	assert.Equal(t, "RUNNING", canvasRunStatus("queued"), "a listed in-flight status keeps the tab polling")
	assert.Equal(t, "RUNNING", canvasRunStatus("cancelling"))
	assert.Equal(t, "FAILED", canvasRunStatus("suspended"), "a parked state is not read as in flight")
	run := toCanvasRun(godo.HostedAgentTriggerExecution{ExecutionID: "e", Status: "timed_out"})
	require.NotNil(t, run.ErrorDetail)
	assert.Equal(t, "execution status timed_out", *run.ErrorDetail)
	assert.Nil(t, toCanvasRun(godo.HostedAgentTriggerExecution{ExecutionID: "q", Status: "queued"}).CompletedAt)
}

// The tab is on for an openhands session on a loopback bind only; any other
// bind, or another agent kind, gets the raw tunnel.
func TestCanvasAutomationOn(t *testing.T) {
	oh := &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{AgentKind: hostedAgentKindOpenHands}}
	cc := &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{AgentKind: godo.HostedAgentKindClaudeCode}}
	assert.True(t, canvasAutomationOn(oh, "127.0.0.1"))
	assert.True(t, canvasAutomationOn(oh, "::1"))
	assert.False(t, canvasAutomationOn(oh, "0.0.0.0"))
	assert.False(t, canvasAutomationOn(oh, "192.168.1.5"))
	assert.False(t, canvasAutomationOn(cc, "127.0.0.1"))
	assert.False(t, canvasAutomationOn(nil, "127.0.0.1"))
}

// The gates port-forward mounts the API behind, through the wrapped handler:
// a page on another local port cannot drive it (its Origin is refused and a
// text/plain write is refused), a non-loopback Host is refused, and the
// Canvas page itself (same origin, JSON) is served.
func TestCanvasAutomationHandler_GatesAgainstOtherLocalPages(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		h := canvasAutomationHandler("sess_1", tm.hostedAgentTriggers)
		send := func(method, origin, contentType, host string) *httptest.ResponseRecorder {
			req := httptest.NewRequest(method, "http://"+host+"/api/automation/v1/preset/prompt", strings.NewReader(`{"name":"x","prompt":"y","trigger":{"type":"cron","schedule":"*/5 * * * *"}}`))
			req.Host = host
			if origin != "" {
				req.Header.Set("Origin", origin)
			}
			if contentType != "" {
				req.Header.Set("Content-Type", contentType)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			return rec
		}
		// Another forwarded port's page: refused on Origin, and on body type.
		assert.Equal(t, http.StatusForbidden, send(http.MethodPost, "http://localhost:3000", "text/plain", "127.0.0.1:8000").Code)
		assert.Equal(t, http.StatusForbidden, send(http.MethodPost, "http://127.0.0.1:3000", "application/json", "127.0.0.1:8000").Code)
		assert.Equal(t, http.StatusForbidden, send(http.MethodPost, "null", "application/json", "127.0.0.1:8000").Code)
		assert.Equal(t, http.StatusUnsupportedMediaType, send(http.MethodPost, "", "text/plain", "127.0.0.1:8000").Code, "a write without JSON is refused even with no Origin")
		// DNS rebinding: a loopback Origin with a foreign Host.
		assert.Equal(t, http.StatusForbidden, send(http.MethodPost, "http://evil.example:8000", "application/json", "evil.example:8000").Code)
		// A foreign Origin that happens to equal a foreign Host is still refused by the Host check.
		assert.Equal(t, http.StatusForbidden, send(http.MethodGet, "http://evil.example:8000", "", "evil.example:8000").Code)

		// The Canvas page: same origin as the listener, JSON, served (and
		// the API call behind it is made).
		tm.hostedAgentTriggers.EXPECT().Create(gomock.Any()).Return(&do.HostedAgentTriggerCreateResult{Trigger: func() *do.HostedAgentTrigger {
			tr := cronTrigger("tr_ok", "sess_1", "*/5 * * * *", godo.HostedAgentTriggerStatusActive)
			return &tr
		}()}, nil)
		assert.Equal(t, http.StatusCreated, send(http.MethodPost, "http://127.0.0.1:8000", "application/json; charset=utf-8", "127.0.0.1:8000").Code)
		// A read without an Origin (curl) is served too.
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8000/api/automation/health", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// A model in the create body is dropped, not refused: a trigger runs on the
// session's active profile, and a form that pre-fills the model must still
// create. The automation comes back with model null.
func TestCanvasAutomation_CreateDropsTheModel(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().Create(gomock.Any()).Return(&do.HostedAgentTriggerCreateResult{Trigger: func() *do.HostedAgentTrigger {
			tr := cronTrigger("tr_m", "sess_1", "*/5 * * * *", godo.HostedAgentTriggerStatusActive)
			return &tr
		}()}, nil)
		h := newCanvasAutomationServer("sess_1", tm.hostedAgentTriggers)
		rec, body := canvasReq(t, h, http.MethodPost, "/api/automation/v1/preset/prompt",
			`{"name":"x","prompt":"y","model":"gpt-5","trigger":{"type":"cron","schedule":"*/5 * * * *"}}`)
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Nil(t, body["model"])
	})
}

// Run now as Canvas sends it, a POST with no body and no Content-Type,
// reaches its own answer (501 for now) instead of the JSON gate.
func TestCanvasAutomationHandler_BodilessPostPassesTheJSONGate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		mine := cronTrigger("tr_1", "sess_1", "0 9 * * *", godo.HostedAgentTriggerStatusActive)
		tm.hostedAgentTriggers.EXPECT().Get("tr_1").Return(&mine, nil)
		h := canvasAutomationHandler("sess_1", tm.hostedAgentTriggers)
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8000/api/automation/v1/tr_1/dispatch", nil)
		req.Host = "127.0.0.1:8000"
		req.Header.Set("Origin", "http://127.0.0.1:8000")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotImplemented, rec.Code, rec.Body.String())
	})
}

// The server stores the bound session id in lower case; a ref typed in upper
// case still owns its triggers, and an id that is not an id is 404 before
// any API call.
func TestCanvasAutomation_SessionIDCaseAndTriggerIDShape(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		mine := cronTrigger("tr_1", "01a1c5e5-5f96-71da-bca5-f5921275d3b0", "0 9 * * *", godo.HostedAgentTriggerStatusActive)
		tm.hostedAgentTriggers.EXPECT().Get("tr_1").Return(&mine, nil)
		tm.hostedAgentTriggers.EXPECT().List(&godo.HostedAgentTriggerListOptions{PageSize: 100}).Return([]do.HostedAgentTrigger{mine}, "", nil)
		h := newCanvasAutomationServer("01A1C5E5-5F96-71DA-BCA5-F5921275D3B0", tm.hostedAgentTriggers)
		rec, _ := canvasReq(t, h, http.MethodGet, "/api/automation/v1/tr_1", "")
		assert.Equal(t, http.StatusOK, rec.Code)
		rec, body := canvasReq(t, h, http.MethodGet, "/api/automation/v1", "")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.EqualValues(t, 1, body["total"])
		for _, p := range []string{"/api/automation/v1/..", "/api/automation/v1/a%3Fb=1", "/api/automation/v1/x%2Fy"} {
			rec, _ = canvasReq(t, h, http.MethodGet, p, "")
			assert.Equal(t, http.StatusNotFound, rec.Code, p)
		}
	})
}
