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
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// textOutput puts the global output format back to what a `doctl` process
// actually runs with, since the flag's default never reaches these tests.
func textOutput(t *testing.T) {
	t.Helper()
	prev := Output
	Output = "text"
	t.Cleanup(func() { Output = prev })
}

func testCheckpoint() godo.HostedAgentCheckpoint {
	return godo.HostedAgentCheckpoint{
		CheckpointID: "cp_e9afcc6b10d3",
		SessionID:    "sess_abc123",
		Status:       godo.HostedAgentCheckpointStatusReady,
		Kind:         godo.HostedAgentCheckpointKindExplicit,
		Label:        "j5-ckpt",
		SizeBytes:    84000000000,
	}
}

func TestRunAgentsCheckpointList_FormatSelectsColumns(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListCheckpoints("sess_abc123", gomock.Any()).
			Return([]godo.HostedAgentCheckpoint{testCheckpoint()}, "", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_abc123"}
		config.Doit.Set(config.NS, doctl.ArgFormat, "CheckpointID,Label,Status,SizeBytes,Kind")

		require.NoError(t, RunAgentsCheckpointList(config))

		out := buf.String()
		assert.NotContains(t, out, "1 checkpoint", "--format asked for columns, not the card")

		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		require.Len(t, lines, 2, "one header row and one checkpoint")
		assert.Equal(t, []string{"Checkpoint", "Label", "Status", "SizeBytes", "Kind"}, strings.Fields(lines[0]))
		assert.Equal(t, []string{"cp_e9afcc6b10d3", "j5-ckpt", "READY", "84000000000", "explicit"}, strings.Fields(lines[1]))
	})
}

func TestRunAgentsCheckpointList_NoHeaderDropsTheHeaderRow(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListCheckpoints("sess_abc123", gomock.Any()).
			Return([]godo.HostedAgentCheckpoint{testCheckpoint()}, "", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_abc123"}
		config.Doit.Set(config.NS, doctl.ArgFormat, "CheckpointID")
		config.Doit.Set(config.NS, doctl.ArgNoHeader, true)

		require.NoError(t, RunAgentsCheckpointList(config))
		assert.Equal(t, "cp_e9afcc6b10d3\n", buf.String())
	})
}

// The page token is commentary on the listing rather than part of it, so it
// must not land in the column output a caller is about to parse.
func TestRunAgentsCheckpointList_ColumnsKeepPageTokenOffStdout(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListCheckpoints("sess_abc123", gomock.Any()).
			Return([]godo.HostedAgentCheckpoint{testCheckpoint()}, "next_page_cursor", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_abc123"}
		config.Doit.Set(config.NS, doctl.ArgFormat, "CheckpointID")
		config.Doit.Set(config.NS, doctl.ArgNoHeader, true)

		require.NoError(t, RunAgentsCheckpointList(config))
		assert.Equal(t, "cp_e9afcc6b10d3\n", buf.String())
	})
}

func TestRunAgentsCheckpointList_DefaultsToTheCard(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListCheckpoints("sess_abc123", gomock.Any()).
			Return([]godo.HostedAgentCheckpoint{testCheckpoint()}, "", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_abc123"}

		require.NoError(t, RunAgentsCheckpointList(config))

		out := buf.String()
		assert.Contains(t, out, "1 checkpoint")
		assert.Contains(t, out, "j5-ckpt")
	})
}

func TestRunAgentsCheckpointCreate_FormatSelectsColumns(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		cp := testCheckpoint()
		tm.hostedAgents.EXPECT().
			CreateCheckpoint("sess_abc123", &godo.HostedAgentCheckpointCreateRequest{Label: "j5-ckpt"}).
			Return(&cp, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_abc123"}
		config.Doit.Set(config.NS, doctl.ArgAgentCheckpointLabel, "j5-ckpt")
		config.Doit.Set(config.NS, doctl.ArgFormat, "CheckpointID")
		config.Doit.Set(config.NS, doctl.ArgNoHeader, true)

		require.NoError(t, RunAgentsCheckpointCreate(config))
		assert.Equal(t, "cp_e9afcc6b10d3\n", buf.String())
	})
}

// The bug this guards: `SID=$(doctl harness-runtime show <session> --format
// SessionID --no-header)` used to capture the whole rendered card, borders and
// all, and the command still exited 0 — so the first sign of trouble was a
// malformed URL somewhere downstream.
func TestRunAgentsShow_FormatSelectsColumns(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			GetSession("sess_abc123").
			Return(&do.HostedAgentSession{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_abc123",
					Name:      "t1",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"sess_abc123"}
		config.Doit.Set(config.NS, doctl.ArgFormat, "SessionID")
		config.Doit.Set(config.NS, doctl.ArgNoHeader, true)

		require.NoError(t, RunAgentsShow(config))
		assert.Equal(t, "sess_abc123\n", buf.String())
	})
}

func TestRunAgentsList_FormatSelectsColumns(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListSessions(gomock.Any()).
			Return([]do.HostedAgentSession{{
				HostedAgentSession: &godo.HostedAgentSession{
					SessionID: "sess_abc123",
					Name:      "t1",
					Status:    godo.HostedAgentSessionStatusReady,
				},
			}}, "", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgFormat, "Name")
		config.Doit.Set(config.NS, doctl.ArgNoHeader, true)

		require.NoError(t, RunAgentsList(config))
		assert.Equal(t, "t1\n", buf.String())
	})
}

func TestAgentStructuredOutput(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		assert.False(t, agentStructuredOutput(config), "plain text output is for a person")

		config.Doit.Set(config.NS, doctl.ArgFormat, "   ")
		assert.False(t, agentStructuredOutput(config), "an empty --format selects no columns")

		config.Doit.Set(config.NS, doctl.ArgFormat, "SessionID")
		assert.True(t, agentStructuredOutput(config))
		assert.True(t, agentColumnOutput(config))
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgNoHeader, true)
		assert.True(t, agentStructuredOutput(config), "--no-header only means anything to the table")
	})

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		prev := Output
		Output = "json"
		t.Cleanup(func() { Output = prev })
		assert.True(t, agentStructuredOutput(config))
		assert.False(t, agentColumnOutput(config), "-o json is a document, not columns")
	})
}
