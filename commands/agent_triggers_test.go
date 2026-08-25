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
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAgentTriggersCommand(t *testing.T) {
	cmd := AgentTriggers()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd,
		"list", "create", "get", "update", "delete", "pause", "resume",
		"rotate-secret", "list-executions", "get-execution", "get-by-session",
		"list-reusable-sessions", "list-providers",
	)
}

func TestAgentTriggersList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().
			List(&godo.HostedAgentTriggerListOptions{
				Kind:   godo.HostedAgentTriggerKindWebhook,
				Status: godo.HostedAgentTriggerStatusActive,
			}).
			Return([]do.HostedAgentTrigger{{
				HostedAgentTrigger: &godo.HostedAgentTrigger{
					TriggerID: "tr_1",
					Name:      "gh-ci",
					Kind:      godo.HostedAgentTriggerKindWebhook,
					Status:    godo.HostedAgentTriggerStatusActive,
				},
			}}, "", nil)

		config.Doit.Set(config.NS, doctl.ArgAgentTriggerKind, "webhook")
		config.Doit.Set(config.NS, doctl.ArgAgentStatus, "active")
		require.NoError(t, RunAgentTriggersList(config))
	})
}

func TestAgentTriggersCreateWebhook(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dir := t.TempDir()
		specPath := filepath.Join(dir, "agent.yaml")
		require.NoError(t, os.WriteFile(specPath, []byte("apiVersion: agents.digitalocean.com/v1alpha1\nkind: Agent\nspec:\n  runtime:\n    adapter: opencode\n"), 0o600))

		tm.hostedAgentTriggers.EXPECT().
			Create(gomock.Any()).
			DoAndReturn(func(req *godo.HostedAgentTriggerCreateRequest) (*do.HostedAgentTriggerCreateResult, error) {
				assert.Equal(t, godo.HostedAgentTriggerKindWebhook, req.Kind)
				assert.Equal(t, "gh-ci", req.Name)
				assert.Equal(t, godo.HostedAgentTriggerSessionModeFresh, req.SessionMode)
				assert.Equal(t, "run it", req.PromptTemplate)
				assert.Equal(t, godo.HostedAgentTriggerOutputModeNone, req.Output.Mode)
				assert.Contains(t, req.SessionTemplate, "kind: Agent")
				require.NotNil(t, req.Webhook)
				assert.Equal(t, godo.HostedAgentWebhookProviderGitHub, req.Webhook.Provider)
				return &do.HostedAgentTriggerCreateResult{
					WebhookSecret: "sec_once",
					Trigger: &do.HostedAgentTrigger{
						HostedAgentTrigger: &godo.HostedAgentTrigger{
							TriggerID: "tr_new",
							Name:      "gh-ci",
							Kind:      godo.HostedAgentTriggerKindWebhook,
							Webhook: &godo.HostedAgentWebhookConfig{
								Provider:   godo.HostedAgentWebhookProviderGitHub,
								WebhookURL: "https://api.digitalocean.com/v2/agents/triggers/tr_new/webhook",
							},
						},
					},
				}, nil
			})

		config.Doit.Set(config.NS, doctl.ArgAgentTriggerKind, "webhook")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "gh-ci")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerSessionMode, "fresh")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerPrompt, "run it")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerOutputMode, "none")
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerProvider, "github")

		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, RunAgentTriggersCreate(config))
		assert.Contains(t, buf.String(), "sec_once")
		assert.Contains(t, buf.String(), "https://api.digitalocean.com/v2/agents/triggers/tr_new/webhook")
		assert.Contains(t, buf.String(), "Trigger created")
	})
}

func TestAgentTriggersCreateCronSlack(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().
			Create(&godo.HostedAgentTriggerCreateRequest{
				Kind:           godo.HostedAgentTriggerKindCron,
				Name:           "daily",
				SessionMode:    godo.HostedAgentTriggerSessionModeReuse,
				PromptTemplate: "summarize",
				Output: godo.HostedAgentTriggerOutputWrite{
					Mode: godo.HostedAgentTriggerOutputModeSlack,
					Slack: &godo.HostedAgentTriggerSlackOutputWrite{
						WebhookURL: "https://hooks.slack.com/services/T/B/xxx",
					},
				},
				BoundSessionID: "sess_paused",
				Cron: &godo.HostedAgentCreateCronConfig{
					CronExpr: "0 9 * * *",
					Timezone: "UTC",
				},
			}).
			Return(&do.HostedAgentTriggerCreateResult{
				Trigger: &do.HostedAgentTrigger{
					HostedAgentTrigger: &godo.HostedAgentTrigger{
						TriggerID: "tr_cron",
						Name:      "daily",
						Kind:      godo.HostedAgentTriggerKindCron,
					},
				},
			}, nil)

		config.Doit.Set(config.NS, doctl.ArgAgentTriggerKind, "cron")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "daily")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerSessionMode, "reuse")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerPrompt, "summarize")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerOutputMode, "slack")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerOutputSlackWebhook, "https://hooks.slack.com/services/T/B/xxx")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerBoundSessionID, "sess_paused")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerCronExpr, "0 9 * * *")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerTimezone, "UTC")
		require.NoError(t, RunAgentTriggersCreate(config))
	})
}

func TestAgentTriggersGetUpdatePauseDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		trig := &do.HostedAgentTrigger{
			HostedAgentTrigger: &godo.HostedAgentTrigger{
				TriggerID: "tr_1",
				Name:      "gh-ci",
				Status:    godo.HostedAgentTriggerStatusActive,
			},
		}

		tm.hostedAgentTriggers.EXPECT().Get("tr_1").Return(trig, nil)
		config.Args = []string{"tr_1"}
		require.NoError(t, RunAgentTriggersGet(config))

		tm.hostedAgentTriggers.EXPECT().
			Update("tr_1", &godo.HostedAgentTriggerUpdateRequest{
				Status: godo.HostedAgentTriggerStatusPaused,
			}).
			Return(&do.HostedAgentTrigger{
				HostedAgentTrigger: &godo.HostedAgentTrigger{
					TriggerID: "tr_1",
					Status:    godo.HostedAgentTriggerStatusPaused,
				},
			}, nil)
		config.Doit.Set(config.NS, doctl.ArgAgentStatus, "paused")
		require.NoError(t, RunAgentTriggersUpdate(config))

		tm.hostedAgentTriggers.EXPECT().
			Update("tr_1", &godo.HostedAgentTriggerUpdateRequest{
				Status: godo.HostedAgentTriggerStatusActive,
			}).
			Return(trig, nil)
		require.NoError(t, RunAgentTriggersResume(config))

		tm.hostedAgentTriggers.EXPECT().Delete("tr_1").Return(nil)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		require.NoError(t, RunAgentTriggersDelete(config))
	})
}

// rotateExpiry is the previous-secret expiry the API returns on a default
// (grace-window) rotation.
const rotateExpiry = "2026-08-12T12:05:00Z"

func TestAgentTriggersRotateSecretAndExecutions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().RotateSecret("tr_1", false).Return("new_sec", rotateExpiry, nil)
		config.Args = []string{"tr_1"}
		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, RunAgentTriggersRotateSecret(config))
		assert.Contains(t, buf.String(), "new_sec")

		tm.hostedAgentTriggers.EXPECT().
			ListExecutions("tr_1", &godo.HostedAgentTriggerExecutionListOptions{
				Status: godo.HostedAgentTriggerExecutionStatusFailed,
			}).
			Return([]do.HostedAgentTriggerExecution{{
				HostedAgentTriggerExecution: &godo.HostedAgentTriggerExecution{
					ExecutionID: "ex_1",
					TriggerID:   "tr_1",
					Status:      godo.HostedAgentTriggerExecutionStatusFailed,
				},
			}}, "", nil)
		config.Doit.Set(config.NS, doctl.ArgAgentStatus, "failed")
		require.NoError(t, RunAgentTriggersListExecutions(config))

		tm.hostedAgentTriggers.EXPECT().
			GetExecution("tr_1", "ex_1").
			Return(&do.HostedAgentTriggerExecution{
				HostedAgentTriggerExecution: &godo.HostedAgentTriggerExecution{
					ExecutionID:     "ex_1",
					Payload:         `{"ok":true}`,
					OutputText:      "hello from run",
					OutputTruncated: true,
				},
			}, nil)
		config.Args = []string{"tr_1", "ex_1"}
		buf.Reset()
		config.Out = &buf
		require.NoError(t, RunAgentTriggersGetExecution(config))
		assert.Contains(t, buf.String(), "hello from run")
		assert.Contains(t, buf.String(), "output truncated")
	})
}

func TestAgentTriggersProvidersReusableBySession(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().
			ListWebhookProviders().
			Return([]do.HostedAgentWebhookProvider{{
				HostedAgentWebhookProvider: &godo.HostedAgentWebhookProvider{
					Key:         godo.HostedAgentWebhookProviderGitHub,
					DisplayName: "GitHub",
				},
			}}, nil)
		require.NoError(t, RunAgentTriggersListProviders(config))

		tm.hostedAgentTriggers.EXPECT().
			ListReusableSessions(nil).
			Return([]do.HostedAgentReusableSession{{
				HostedAgentReusableSession: &godo.HostedAgentReusableSession{
					SessionID: "sess_1",
					Name:      "paused-agent",
					Status:    godo.HostedAgentSessionStatusPaused,
				},
			}}, "", nil)
		require.NoError(t, RunAgentTriggersListReusableSessions(config))

		tm.hostedAgentTriggers.EXPECT().
			GetBySession("sess_1").
			Return(&do.HostedAgentTrigger{
				HostedAgentTrigger: &godo.HostedAgentTrigger{TriggerID: "tr_1", BoundSessionID: "sess_1"},
			}, nil)
		config.Args = []string{"sess_1"}
		require.NoError(t, RunAgentTriggersGetBySession(config))
	})
}

// --- JSON output contract tests (MARSOHS-867, MARSOHS-868) ------------------

// captureProcessStderr swaps os.Stderr for a pipe while fn runs and returns
// everything fn wrote to it. Needed because the JSON-output contract is about
// the real process streams, not the injectable CmdConfig.Out.
func captureProcessStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	require.NoError(t, w.Close())
	return <-done
}

// TestAgentTriggersCreateWebhook_JSONMode verifies that in -o json mode:
//   - stdout is a single valid JSON object
//   - webhook_secret IS in the JSON payload (it's a one-time value; consumers must be able to parse it)
//   - trigger fields (trigger_id, name) are also in the JSON payload
//   - the human-readable banner appears on neither stdout nor stderr
func TestAgentTriggersCreateWebhook_JSONMode(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dir := t.TempDir()
		specPath := filepath.Join(dir, "agent.yaml")
		require.NoError(t, os.WriteFile(specPath, []byte("apiVersion: agents.digitalocean.com/v1alpha1\nkind: Agent\nspec:\n  runtime:\n    adapter: opencode\n"), 0o600))

		tm.hostedAgentTriggers.EXPECT().
			Create(gomock.Any()).
			Return(&do.HostedAgentTriggerCreateResult{
				WebhookSecret: "sec_once",
				Trigger: &do.HostedAgentTrigger{
					HostedAgentTrigger: &godo.HostedAgentTrigger{
						TriggerID: "tr_new",
						Name:      "gh-ci",
						Kind:      godo.HostedAgentTriggerKindWebhook,
						Webhook: &godo.HostedAgentWebhookConfig{
							WebhookURL: "https://api.digitalocean.com/v2/agents/triggers/tr_new/webhook",
						},
					},
				},
			}, nil)

		config.Doit.Set(config.NS, doctl.ArgAgentTriggerKind, "webhook")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "gh-ci")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerSessionMode, "fresh")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerPrompt, "run it")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerOutputMode, "none")
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)

		var stdout bytes.Buffer
		config.Out = &stdout

		// Simulate -o json
		prev := Output
		Output = "json"
		defer func() { Output = prev }()

		stderr := captureProcessStderr(t, func() {
			require.NoError(t, RunAgentTriggersCreate(config))
		})

		raw := stdout.String()
		// stdout must be a single valid JSON object from the first byte.
		var parsed map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &parsed), "stdout must be valid JSON in -o json mode, got: %q", raw)

		// The one-time secret MUST be in the JSON payload — it is shown once and
		// is unrecoverable afterwards, so it cannot live only in a human banner.
		assert.Equal(t, "sec_once", parsed["webhook_secret"], "webhook_secret must be in the JSON payload")

		// Trigger fields must also be present (flat, not nested under "trigger").
		assert.Equal(t, "tr_new", parsed["trigger_id"], "trigger_id must be in the JSON payload")
		assert.Equal(t, "gh-ci", parsed["name"], "name must be in the JSON payload")

		// Human-readable banner text must NOT appear on stdout.
		assert.NotContains(t, raw, "store it now", "banner must not appear on stdout in JSON mode")
		assert.NotContains(t, raw, "Webhook URL:", "webhook URL banner must not appear on stdout in JSON mode")

		// Nor on stderr: the secret is already in the payload, so echoing it to a
		// second stream would just duplicate a secret into callers' logs.
		assert.Empty(t, stderr, "nothing may be written to stderr in -o json mode")
	})
}

// TestAgentTriggersRotateSecret_JSONMode verifies that in -o json mode:
//   - stdout is a valid JSON object containing the secret key
//   - the banner is on neither stdout nor stderr
func TestAgentTriggersRotateSecret_JSONMode(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().RotateSecret("tr_1", false).Return("new_sec", rotateExpiry, nil)
		config.Args = []string{"tr_1"}

		var stdout bytes.Buffer
		config.Out = &stdout

		prev := Output
		Output = "json"
		defer func() { Output = prev }()

		stderr := captureProcessStderr(t, func() {
			require.NoError(t, RunAgentTriggersRotateSecret(config))
		})

		raw := stdout.String()
		var parsed map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &parsed), "stdout must be valid JSON in -o json mode, got: %q", raw)
		assert.Equal(t, "new_sec", parsed["webhook_secret"], "JSON must contain webhook_secret field")
		assert.Equal(t, rotateExpiry, parsed["previous_secret_expires_at"], "scripts need the expiry to schedule the provider-side update")
		assert.NotContains(t, parsed, "previous_secret_revoked", "the old secret is still live during the grace window")
		assert.NotContains(t, raw, "store it now", "banner must not appear on stdout in JSON mode")
		assert.Empty(t, stderr, "nothing may be written to stderr in -o json mode")
	})
}

// --revoke-previous is the breach path: the response reports the old secret as
// already dead rather than giving an expiry to wait out.
func TestAgentTriggersRotateSecret_RevokePrevious(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().RotateSecret("tr_1", true).Return("new_sec", "", nil)
		config.Args = []string{"tr_1"}
		config.Doit.Set(config.NS, doctl.ArgAgentRevokePrevious, true)

		var stdout bytes.Buffer
		config.Out = &stdout

		prev := Output
		Output = "json"
		defer func() { Output = prev }()

		require.NoError(t, RunAgentTriggersRotateSecret(config))

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &parsed))
		assert.Equal(t, "new_sec", parsed["webhook_secret"])
		assert.Equal(t, true, parsed["previous_secret_revoked"])
		assert.NotContains(t, parsed, "previous_secret_expires_at", "there is no window left to report")
	})
}

// TestAgentTriggersRotateSecret_TextMode verifies that in text mode the
// secret banner still appears on stdout (existing behaviour preserved).
func TestAgentTriggersRotateSecret_TextMode(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().RotateSecret("tr_1", false).Return("new_sec", rotateExpiry, nil)
		config.Args = []string{"tr_1"}

		var stdout bytes.Buffer
		config.Out = &stdout

		prev := Output
		Output = "text"
		defer func() { Output = prev }()

		require.NoError(t, RunAgentTriggersRotateSecret(config))
		assert.Contains(t, stdout.String(), "new_sec", "secret must appear on stdout in text mode")
		assert.Contains(t, stdout.String(), rotateExpiry, "an operator needs the exact instant the old secret dies")
	})
}

// TestAgentTriggersGetExecution_JSONMode verifies that in -o json mode stdout
// is a single valid JSON document from the very first byte. The human-readable
// text preamble (bare OutputText + truncation notice) that used to be printed
// before the object would break any JSON parser at character 0.
// Note: output_text IS present as a field inside the JSON object — that is
// correct and expected. What must be absent is a bare text block *before* the
// opening '{'.
func TestAgentTriggersCreateRejectsInvalidName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		dir := t.TempDir()
		specPath := filepath.Join(dir, "agent.yaml")
		require.NoError(t, os.WriteFile(specPath, []byte("apiVersion: agents.digitalocean.com/v1alpha1\nkind: Agent\nspec:\n  runtime:\n    adapter: opencode\n"), 0o600))

		config.Doit.Set(config.NS, doctl.ArgAgentTriggerKind, "webhook")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "<script>alert(1)</script>")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerSessionMode, "fresh")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerPrompt, "run it")
		config.Doit.Set(config.NS, doctl.ArgAgentTriggerOutputMode, "none")
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)

		err := RunAgentTriggersCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name must be 1–64 characters")
	})
}

func TestAgentTriggersUpdateRejectsInvalidName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"tr_1"}
		config.Doit.Set(config.NS, doctl.ArgAgentName, "../../etc/passwd")

		err := RunAgentTriggersUpdate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name must be 1–64 characters")
	})
}

func TestAgentTriggersGetExecution_JSONMode(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().
			GetExecution("tr_1", "ex_1").
			Return(&do.HostedAgentTriggerExecution{
				HostedAgentTriggerExecution: &godo.HostedAgentTriggerExecution{
					ExecutionID:     "ex_1",
					OutputText:      "hello from run",
					OutputTruncated: true,
				},
			}, nil)
		config.Args = []string{"tr_1", "ex_1"}

		var stdout bytes.Buffer
		config.Out = &stdout

		prev := Output
		Output = "json"
		defer func() { Output = prev }()

		require.NoError(t, RunAgentTriggersGetExecution(config))

		raw := stdout.String()
		// stdout must be a valid JSON document — if bare text preceded the object
		// the Unmarshal would fail at character 0.
		var parsed map[string]any
		require.NoError(t, json.Unmarshal([]byte(raw), &parsed), "stdout must be valid JSON in -o json mode, got: %q", raw)
		// The structured field is still there inside the JSON (correct).
		assert.Equal(t, "ex_1", parsed["execution_id"])
		// The bare "(output truncated)" notice must NOT appear; it has no JSON equivalent.
		assert.NotContains(t, raw, "(output truncated)", "bare truncation notice must not appear in JSON mode")
	})
}

// TestAgentTriggersGetExecution_TextMode verifies the run output text block
// is still present on stdout in text mode (existing behaviour preserved).
func TestAgentTriggersGetExecution_TextMode(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgentTriggers.EXPECT().
			GetExecution("tr_1", "ex_1").
			Return(&do.HostedAgentTriggerExecution{
				HostedAgentTriggerExecution: &godo.HostedAgentTriggerExecution{
					ExecutionID:     "ex_1",
					OutputText:      "hello from run",
					OutputTruncated: true,
				},
			}, nil)
		config.Args = []string{"tr_1", "ex_1"}

		var stdout bytes.Buffer
		config.Out = &stdout

		prev := Output
		Output = "text"
		defer func() { Output = prev }()

		require.NoError(t, RunAgentTriggersGetExecution(config))
		assert.Contains(t, stdout.String(), "hello from run", "run output text must appear on stdout in text mode")
		assert.Contains(t, stdout.String(), "output truncated", "truncation notice must appear on stdout in text mode")
	})
}
