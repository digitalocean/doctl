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
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const updateConfigSessionID = "11111111-1111-1111-1111-111111111111"

const updateConfigManifest = "tools:\n  - do.actions:\n      tools: [toolbelt:read-only@1]\n"

func writeUpdateSpec(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tools.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

func updateSession(status godo.HostedAgentSessionStatus, u *godo.HostedAgentSessionConfigUpdate) *godo.HostedAgentSession {
	return &godo.HostedAgentSession{
		SessionID: updateConfigSessionID,
		Name:      "demo",
		AgentKind: godo.HostedAgentKindClaudeCode,
		Status:    status,
		Update:    u,
	}
}

func TestRunAgentsUpdate_ConfigAcceptedThenApplied(t *testing.T) {
	old := sessionReadyPollInterval
	sessionReadyPollInterval = time.Millisecond
	defer func() { sessionReadyPollInterval = old }()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		changes := &godo.HostedAgentSessionConfigChanges{
			Tools: &godo.HostedAgentSessionConfigSectionChanges{Added: []string{"do.actions"}, Removed: []string{"jira"}},
		}
		applying := &godo.HostedAgentSessionConfigUpdate{UpdateID: "u1", State: godo.HostedAgentSessionConfigUpdateStateApplying, Changes: changes}
		applied := &godo.HostedAgentSessionConfigUpdate{UpdateID: "u1", State: godo.HostedAgentSessionConfigUpdateStateApplied, Changes: changes}

		tm.hostedAgents.EXPECT().
			UpdateSessionConfig(updateConfigSessionID, &godo.HostedAgentSessionConfigUpdateRequest{ManifestYAML: updateConfigManifest}).
			Return(&godo.HostedAgentSessionConfigUpdateResponse{
				Session:  updateSession(godo.HostedAgentSessionStatusUpdating, applying),
				Outcome:  godo.HostedAgentSessionConfigUpdateOutcomeAccepted,
				Warnings: []string{"heads up"},
			}, nil)
		gomockGet := tm.hostedAgents.EXPECT().GetSession(updateConfigSessionID)
		gomockGet.Return(&do.HostedAgentSession{HostedAgentSession: updateSession(godo.HostedAgentSessionStatusUpdating, applying)}, nil).Times(2)
		tm.hostedAgents.EXPECT().GetSession(updateConfigSessionID).
			Return(&do.HostedAgentSession{HostedAgentSession: updateSession(godo.HostedAgentSessionStatusReady, applied)}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{updateConfigSessionID}
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, writeUpdateSpec(t, updateConfigManifest))

		require.NoError(t, RunAgentsUpdate(config))
		out := buf.String()
		assert.Contains(t, out, "heads up")
		assert.Contains(t, out, "updated")
		assert.Contains(t, out, "tools +do.actions −jira")
	})
}

func TestRunAgentsUpdate_ConfigUnchanged(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			UpdateSessionConfig(updateConfigSessionID, &godo.HostedAgentSessionConfigUpdateRequest{ManifestYAML: updateConfigManifest}).
			Return(&godo.HostedAgentSessionConfigUpdateResponse{
				Session: updateSession(godo.HostedAgentSessionStatusReady, nil),
				Outcome: godo.HostedAgentSessionConfigUpdateOutcomeUnchanged,
			}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{updateConfigSessionID}
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, writeUpdateSpec(t, updateConfigManifest))

		// No GetSession expectation: an unchanged update must not poll.
		require.NoError(t, RunAgentsUpdate(config))
		assert.Contains(t, buf.String(), "nothing to change")
	})
}

func TestRunAgentsUpdate_ConfigNoWait(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			UpdateSessionConfig(updateConfigSessionID, &godo.HostedAgentSessionConfigUpdateRequest{ManifestYAML: updateConfigManifest}).
			Return(&godo.HostedAgentSessionConfigUpdateResponse{
				Session: updateSession(godo.HostedAgentSessionStatusUpdating, &godo.HostedAgentSessionConfigUpdate{State: godo.HostedAgentSessionConfigUpdateStateApplying}),
				Outcome: godo.HostedAgentSessionConfigUpdateOutcomeAccepted,
			}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{updateConfigSessionID}
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, writeUpdateSpec(t, updateConfigManifest))
		config.Doit.Set(config.NS, doctl.ArgAgentNoWait, true)

		require.NoError(t, RunAgentsUpdate(config))
		assert.Contains(t, buf.String(), "accepted")
	})
}

func TestRunAgentsUpdate_ConfigFailedReverted(t *testing.T) {
	old := sessionReadyPollInterval
	sessionReadyPollInterval = time.Millisecond
	defer func() { sessionReadyPollInterval = old }()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			UpdateSessionConfig(updateConfigSessionID, &godo.HostedAgentSessionConfigUpdateRequest{ManifestYAML: updateConfigManifest}).
			Return(&godo.HostedAgentSessionConfigUpdateResponse{
				Session: updateSession(godo.HostedAgentSessionStatusUpdating, &godo.HostedAgentSessionConfigUpdate{State: godo.HostedAgentSessionConfigUpdateStateApplying}),
				Outcome: godo.HostedAgentSessionConfigUpdateOutcomeAccepted,
			}, nil)
		tm.hostedAgents.EXPECT().GetSession(updateConfigSessionID).
			Return(&do.HostedAgentSession{HostedAgentSession: updateSession(godo.HostedAgentSessionStatusReady, &godo.HostedAgentSessionConfigUpdate{
				State:    godo.HostedAgentSessionConfigUpdateStateFailed,
				Error:    godo.HostedAgentSessionConfigUpdateErrorUpdateFailed,
				Detail:   "mcp server did not start",
				Reverted: true,
			})}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{updateConfigSessionID}
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, writeUpdateSpec(t, updateConfigManifest))

		err := RunAgentsUpdate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mcp server did not start")
		assert.Contains(t, err.Error(), "reverted")
	})
}

func TestRunAgentsUpdate_ConfigRejectsResumeOnTopoffAlongside(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{updateConfigSessionID}
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, writeUpdateSpec(t, updateConfigManifest))
		config.Doit.Set(config.NS, doctl.ArgAgentResumeOnTopoff, true)

		err := RunAgentsUpdate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be combined")
	})
}

func TestRunAgentsUpdate_ConfigEmptyManifest(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{updateConfigSessionID}
		config.Doit.Set(config.NS, doctl.ArgAgentSpec, writeUpdateSpec(t, "  \n"))

		err := RunAgentsUpdate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "manifest is empty")
	})
}

func TestWaitForSessionConfigUpdate_Timeout(t *testing.T) {
	old := sessionReadyPollInterval
	sessionReadyPollInterval = time.Millisecond
	defer func() { sessionReadyPollInterval = old }()

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().GetSession(updateConfigSessionID).
			Return(&do.HostedAgentSession{HostedAgentSession: updateSession(godo.HostedAgentSessionStatusUpdating, nil)}, nil).
			AnyTimes()

		ctx, cancel := contextWithTimeoutForTest(20 * time.Millisecond)
		defer cancel()
		_, err := waitForSessionConfigUpdate(ctx, tm.hostedAgents, updateConfigSessionID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})
}

func TestConfigDiffSummary(t *testing.T) {
	assert.Equal(t, "", configDiffSummary(nil))
	assert.Equal(t, "tools +a −b; env ~FOO; secrets +K", configDiffSummary(&godo.HostedAgentSessionConfigChanges{
		Tools:   &godo.HostedAgentSessionConfigSectionChanges{Added: []string{"a"}, Removed: []string{"b"}},
		Env:     &godo.HostedAgentSessionConfigSectionChanges{Changed: []string{"FOO"}},
		Secrets: &godo.HostedAgentSessionConfigSectionChanges{Added: []string{"K"}},
	}))
}

func TestPrintSessionShowCard_ConfigDriftAndUpdateRows(t *testing.T) {
	sess := &do.HostedAgentSession{HostedAgentSession: updateSession(godo.HostedAgentSessionStatusReady, &godo.HostedAgentSessionConfigUpdate{
		State:     godo.HostedAgentSessionConfigUpdateStateAppliedRestarted,
		StartedAt: godo.Timestamp{Time: time.Now().Add(-time.Minute)},
	})}
	sess.ConfigDrifted = true
	sess.ConfigDiff = &godo.HostedAgentSessionConfigChanges{Env: &godo.HostedAgentSessionConfigSectionChanges{Changed: []string{"MODE"}}}

	var buf bytes.Buffer
	printSessionShowCard(&buf, sess)
	assert.Contains(t, buf.String(), "Config drifted")
	assert.Contains(t, buf.String(), "env ~MODE")
	assert.Contains(t, buf.String(), "Last update")
	assert.Contains(t, buf.String(), "applied restarted")

	plain := &do.HostedAgentSession{HostedAgentSession: updateSession(godo.HostedAgentSessionStatusReady, nil)}
	buf.Reset()
	printSessionShowCard(&buf, plain)
	assert.NotContains(t, buf.String(), "Config drifted")
	assert.NotContains(t, buf.String(), "Last update")
}

func TestUpdatingStatusDisplay(t *testing.T) {
	assert.Equal(t, "Applying config update…", runStatusLabel(godo.HostedAgentSessionStatusUpdating))
	assert.Equal(t, "updating", humanSessionStatus(godo.HostedAgentSessionStatusUpdating))
}

func contextWithTimeoutForTest(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
