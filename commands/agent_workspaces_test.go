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
	"errors"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func testWorkspace() godo.HostedAgentWorkspace {
	return godo.HostedAgentWorkspace{
		WorkspaceID:       "ws_abc123",
		Name:              "notes",
		State:             godo.HostedAgentWorkspaceStateAttached,
		AttachedSessionID: "sess_abc123",
		SizeGibibytes:     10,
		BytesUsed:         3 * 1024 * 1024,
	}
}

func TestAgentWorkspacesCommand(t *testing.T) {
	cmd := AgentWorkspaces()
	require.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "get", "delete")
	assert.Contains(t, cmd.Aliases, "workspaces")

	root := Agents()
	found, _, err := root.Find([]string{"workspaces", "list"})
	require.NoError(t, err)
	assert.Equal(t, "list", found.Name())
}

// The server rejects a manifest that names a workspace, so the session-create
// wire fields have to stay where the server reads them: the body for a config,
// the query string for a manifest.
func TestWorkspaceWireNames(t *testing.T) {
	f, ok := reflect.TypeOf(godo.HostedAgentManifestCreateOptions{}).FieldByName("WorkspaceID")
	require.True(t, ok)
	assert.Equal(t, "workspace_id,omitempty", f.Tag.Get("url"))

	f, ok = reflect.TypeOf(godo.HostedAgentSessionFromConfigRequest{}).FieldByName("WorkspaceID")
	require.True(t, ok)
	assert.Equal(t, "workspace_id,omitempty", f.Tag.Get("json"))

	f, ok = reflect.TypeOf(godo.HostedAgentSession{}).FieldByName("WorkspaceID")
	require.True(t, ok)
	assert.Equal(t, "workspace_id,omitempty", f.Tag.Get("json"))
}

func TestAgentWorkspaceCreateFlags(t *testing.T) {
	var create *Command
	for _, c := range AgentWorkspaces().ChildCommands() {
		if c.Name() == "create" {
			create = c
		}
	}
	require.NotNil(t, create)
	for _, name := range []string{doctl.ArgAgentWorkspaceSizeGiB, doctl.ArgAgentName, doctl.ArgAgentIdempotencyKey} {
		assert.NotNil(t, create.Flags().Lookup(name), "workspace create needs --%s", name)
	}
	assert.Equal(t, "size-gib", doctl.ArgAgentWorkspaceSizeGiB)
	assert.Equal(t, "idempotency-key", doctl.ArgAgentIdempotencyKey)
}

// Words the product avoids for a workspace must not appear in its help.
func TestAgentWorkspaceHelpAvoidsStorageJargon(t *testing.T) {
	root := AgentWorkspaces()
	walkCommands(root.Command, func(cmd *cobra.Command) {
		text := strings.ToLower(cmd.Short + " " + cmd.Long + " " + cmd.Example)
		for _, banned := range []string{"disk", "volume", "device", "layer", "slot"} {
			assert.NotContains(t, text, banned, "%s help", cmd.CommandPath())
		}
	})
	text := strings.ToLower(agentWorkspaceFlagDesc)
	for _, banned := range []string{"disk", "volume", "device", "layer", "slot"} {
		assert.NotContains(t, text, banned)
	}
	assert.Contains(t, agentsWorkspaceDeleteHelpMD, "Permanently")
	assert.Contains(t, agentsWorkspaceDeleteHelpMD, "never deletes")
	assert.Contains(t, agentsRemoveHelpMD, "never deletes a workspace")
}

func TestRunAgentsWorkspaceCreate_SizeIsRequired(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// No CreateWorkspace expectation: gomock fails the test if it is called.
		err := RunAgentsWorkspaceCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--size-gib is required")
	})
}

func TestRunAgentsWorkspaceCreate_RefusesInvalidSize(t *testing.T) {
	for _, size := range []int{0, -1, -100, math.MaxInt32 + 1} {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			config.Doit.Set(config.NS, doctl.ArgAgentWorkspaceSizeGiB, size)
			err := RunAgentsWorkspaceCreate(config)
			require.Error(t, err, "size %d", size)
			assert.Contains(t, err.Error(), "invalid --size-gib")
			assert.Contains(t, err.Error(), "whole number from 1 to 100")
		})
	}
}

// The upper bound belongs to the server, since a team limit can lower it.
func TestRunAgentsWorkspaceCreate_LeavesUpperBoundToServer(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateWorkspace(gomock.Any()).
			Return(nil, godoStatusErr(http.StatusUnprocessableEntity, "size_gibibytes must be between 1 and 100"))

		config.Doit.Set(config.NS, doctl.ArgAgentWorkspaceSizeGiB, 500)
		err := RunAgentsWorkspaceCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "size_gibibytes must be between 1 and 100")
	})
}

func TestRunAgentsWorkspaceCreate_GeneratesOneKeyPerInvocation(t *testing.T) {
	textOutput(t)

	var keys []string
	for i := 0; i < 2; i++ {
		withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				CreateWorkspace(gomock.Any()).
				DoAndReturn(func(req *godo.HostedAgentWorkspaceCreateRequest) (*godo.HostedAgentWorkspace, error) {
					assert.Equal(t, int32(10), req.SizeGibibytes)
					assert.Equal(t, "notes", req.Name)
					_, err := uuid.Parse(req.IdempotencyKey)
					assert.NoError(t, err, "a missing --idempotency-key is replaced by a UUID")
					keys = append(keys, req.IdempotencyKey)
					ws := testWorkspace()
					return &ws, nil
				})

			var buf bytes.Buffer
			config.Out = &buf
			config.Doit.Set(config.NS, doctl.ArgAgentWorkspaceSizeGiB, 10)
			config.Doit.Set(config.NS, doctl.ArgAgentName, "notes")
			require.NoError(t, RunAgentsWorkspaceCreate(config))
			assert.Contains(t, buf.String(), "Workspace created")
			assert.Contains(t, buf.String(), "ws_abc123")
		})
	}
	require.Len(t, keys, 2)
	assert.NotEqual(t, keys[0], keys[1], "a re-run without --idempotency-key is a new workspace")
}

func TestRunAgentsWorkspaceCreate_HonoursGivenKey(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateWorkspace(&godo.HostedAgentWorkspaceCreateRequest{
				SizeGibibytes:  5,
				IdempotencyKey: "nightly-1",
			}).
			Return(&godo.HostedAgentWorkspace{WorkspaceID: "ws_1", SizeGibibytes: 5}, nil)

		config.Doit.Set(config.NS, doctl.ArgAgentWorkspaceSizeGiB, 5)
		config.Doit.Set(config.NS, doctl.ArgAgentIdempotencyKey, "nightly-1")
		require.NoError(t, RunAgentsWorkspaceCreate(config))
	})
}

func TestRunAgentsWorkspaceList(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListWorkspaces(&godo.HostedAgentWorkspaceListOptions{PageSize: 2, PageToken: "tok1"}).
			Return([]godo.HostedAgentWorkspace{testWorkspace()}, "tok2", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentPageSize, 2)
		config.Doit.Set(config.NS, doctl.ArgAgentPageToken, "tok1")
		require.NoError(t, RunAgentsWorkspaceList(config))

		assert.Contains(t, buf.String(), "1 workspace")
		assert.Contains(t, buf.String(), "notes")
		assert.Contains(t, buf.String(), "Next page token:")
		assert.Contains(t, buf.String(), "tok2")
	})
}

func TestRunAgentsWorkspaceList_Empty(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListWorkspaces(&godo.HostedAgentWorkspaceListOptions{}).
			Return(nil, "", nil)

		var buf bytes.Buffer
		config.Out = &buf
		require.NoError(t, RunAgentsWorkspaceList(config))
		assert.Contains(t, buf.String(), "No workspaces")
	})
}

func TestRunAgentsWorkspaceList_FormatSelectsColumns(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		saved := godo.Timestamp{Time: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)}
		ws := testWorkspace()
		ws.LastSavedAt = &saved
		tm.hostedAgents.EXPECT().
			ListWorkspaces(gomock.Any()).
			Return([]godo.HostedAgentWorkspace{ws}, "", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgFormat, "WorkspaceID,State,SizeGibibytes,BytesUsed,LastSavedAt")
		require.NoError(t, RunAgentsWorkspaceList(config))

		lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
		require.Len(t, lines, 2)
		assert.Equal(t, []string{"ID", "State", "Size", "(GiB)", "Used", "Last", "Saved"}, strings.Fields(lines[0]))
		assert.Equal(t, []string{"ws_abc123", "ATTACHED", "10", "3.00", "MiB", "2026-10-05T12:00:00Z"}, strings.Fields(lines[1]))
	})
}

func TestRunAgentsWorkspaceGet(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ws := testWorkspace()
		tm.hostedAgents.EXPECT().GetWorkspace("ws_abc123").Return(&ws, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"ws_abc123"}
		require.NoError(t, RunAgentsWorkspaceGet(config))
		assert.Contains(t, buf.String(), "notes")
		assert.Contains(t, buf.String(), "10 GiB")
		assert.Contains(t, buf.String(), "sess_abc123")
		assert.NotContains(t, buf.String(), "Workspace created")
	})
}

func TestRunAgentsWorkspaceGetAndDelete_ArgValidation(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		assert.Error(t, RunAgentsWorkspaceGet(config))
		assert.Error(t, RunAgentsWorkspaceDelete(config))

		config.Args = []string{"ws_1", "ws_2"}
		assert.Error(t, RunAgentsWorkspaceGet(config))
		assert.Error(t, RunAgentsWorkspaceDelete(config))
	})
}

func TestRunAgentsWorkspaceDelete_ForceSkipsConfirmation(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().DeleteWorkspace("ws_abc123").Return(nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"ws_abc123"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		require.NoError(t, RunAgentsWorkspaceDelete(config))
		assert.Contains(t, buf.String(), "Deleted workspace ws_abc123")
	})
}

// Without --force a delete must ask first; with nobody to ask it must not go
// ahead. No DeleteWorkspace expectation, so a call fails the test.
func TestRunAgentsWorkspaceDelete_RequiresConfirmation(t *testing.T) {
	prev := Interactive
	Interactive = false
	t.Cleanup(func() { Interactive = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"ws_abc123"}
		err := RunAgentsWorkspaceDelete(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "operation aborted")
	})
}

// A 409 on delete shows the server's message as sent, under the generic
// conflict card.
func TestRunAgentsWorkspaceDelete_ConflictShowsServerMessage(t *testing.T) {
	const serverMsg = "workspace is attached to a session"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			DeleteWorkspace("ws_abc123").
			Return(godoStatusErr(http.StatusConflict, serverMsg))

		config.Args = []string{"ws_abc123"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		err := RunAgentsWorkspaceDelete(config)
		require.Error(t, err)

		var pretty *agentPrettyError
		require.True(t, errors.As(beautifyAgentError(err), &pretty))
		assert.Equal(t, serverMsg, pretty.reason)
		assert.Equal(t, http.StatusConflict, pretty.status)
		assert.NotContains(t, pretty.tips, workspaceSavingHint, "the saving hint is for session create")
	})
}

func TestBeautifyAgentError_NotImplementedReadsAsNotAvailableYet(t *testing.T) {
	out := beautifyAgentError(godoStatusErr(http.StatusNotImplemented, "workspaces are not enabled"))
	var pretty *agentPrettyError
	require.True(t, errors.As(out, &pretty))
	assert.Equal(t, "Not available yet", pretty.title)
	assert.Equal(t, "workspaces are not enabled", pretty.reason)
	assert.Empty(t, pretty.tips, "a 501 does not get better with a retry")
}

// --- --workspace on create / launch ------------------------------------------

func TestRunAgentsCreate_FromConfig_SendsWorkspaceInBody(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
				Name:        "demo",
				ConfigID:    "cfg_abc123",
				WorkspaceID: "ws_abc123",
			}).
			Return(nil, assertCalledErr)

		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "ws_abc123")
		require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
	})
}

// The workspace is per session, so it rides the query string and never the
// manifest the server would reject.
func TestRunAgentsCreate_Spec_SendsWorkspaceAsQueryOption(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			DoAndReturn(func(manifest []byte, opt *godo.HostedAgentManifestCreateOptions) (*do.HostedAgentSession, error) {
				assert.NotContains(t, string(manifest), "workspace")
				require.NotNil(t, opt)
				assert.Equal(t, "ws_abc123", opt.WorkspaceID)
				return nil, assertCalledErr
			})

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "ws_abc123")
		require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
	})
}

// Without the flag nothing about the create changes: no options object at all.
func TestRunAgentsCreate_Spec_NoWorkspaceNoOptions(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), nil).
			Return(nil, assertCalledErr)

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
	})
}

func TestRunAgentsCreate_HarnessRefusesWorkspace(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// No create expectations: nothing may reach the API.
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "ws_abc123")
		err := RunAgentsCreate(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--workspace needs --spec or --from-config")
	})
}

func TestRunAgentsLaunch_Spec_SendsWorkspace(t *testing.T) {
	stubInteractiveTerminal(t, true)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), &godo.HostedAgentManifestCreateOptions{WorkspaceID: "ws_abc123"}).
			Return(nil, assertCalledErr)

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "ws_abc123")
		require.ErrorIs(t, RunAgentsLaunch(config), assertCalledErr)
	})
}

func TestRunAgentsLaunch_FromConfig_SendsWorkspace(t *testing.T) {
	stubInteractiveTerminal(t, true)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
				Name:        "demo",
				ConfigID:    "cfg_abc123",
				WorkspaceID: "ws_abc123",
			}).
			Return(nil, assertCalledErr)

		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "ws_abc123")
		require.ErrorIs(t, RunAgentsLaunch(config), assertCalledErr)
	})
}

func TestRunAgentsLaunch_HarnessRefusesWorkspace(t *testing.T) {
	stubInteractiveTerminal(t, true)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "ws_abc123")
		err := RunAgentsLaunch(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--workspace needs --spec or --from-config")
	})
}

// A 409 on create with --workspace carries the server's message and the
// retry hint. It is decided by status code: the message here says nothing about
// workspaces.
func TestRunAgentsCreate_WorkspaceConflictAddsHint(t *testing.T) {
	const serverMsg = "that is not available right now"
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	cases := map[string]func(config *CmdConfig, tm *tcMocks){
		"manifest": func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				CreateSessionFromManifest(gomock.Any(), gomock.Any()).
				Return(nil, godoStatusErr(http.StatusConflict, serverMsg))
			config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		},
		"config": func(config *CmdConfig, tm *tcMocks) {
			tm.hostedAgents.EXPECT().
				CreateSessionFromConfig(gomock.Any()).
				Return(nil, godoStatusErr(http.StatusConflict, serverMsg))
			config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
			config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				setup(config, tm)
				config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "ws_abc123")

				err := RunAgentsCreate(config)
				require.Error(t, err)

				var pretty *agentPrettyError
				require.True(t, errors.As(beautifyAgentError(err), &pretty))
				assert.Equal(t, serverMsg, pretty.reason)
				assert.Equal(t, http.StatusConflict, pretty.status)
				assert.Contains(t, pretty.tips, workspaceSavingHint)
				assert.Contains(t, pretty.DisplayError(), "still being saved")
			})
		})
	}
}

func TestRunAgentsCreate_ConflictWithoutWorkspaceHasNoHint(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromManifest(gomock.Any(), gomock.Any()).
			Return(nil, godoStatusErr(http.StatusConflict, "name already in use"))

		config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		err := RunAgentsCreate(config)
		require.Error(t, err)

		var pretty *agentPrettyError
		require.True(t, errors.As(beautifyAgentError(err), &pretty))
		assert.NotContains(t, pretty.tips, workspaceSavingHint)
	})
}

// Only a 409 gets the hint; any other failure keeps its own message.
func TestWithWorkspaceConflictHint_OnlyForConflict(t *testing.T) {
	other := godoStatusErr(http.StatusNotFound, "workspace not found")
	assert.Same(t, other, withWorkspaceConflictHint(other))

	plain := errors.New("connection reset")
	assert.Same(t, plain, withWorkspaceConflictHint(plain))
}

func TestAgentCreationFlagsIncludeWorkspace(t *testing.T) {
	for _, name := range []string{"create", "launch"} {
		found, _, err := Agents().Find([]string{name})
		require.NoError(t, err)
		f := found.Flags().Lookup(doctl.ArgAgentWorkspace)
		require.NotNil(t, f, "%s needs --workspace", name)
		assert.Equal(t, "string", f.Value.Type())
	}
}

func TestPrintSessionShowCard_Workspace(t *testing.T) {
	var with, without bytes.Buffer
	printSessionShowCard(&with, &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
		SessionID:   "sess_1",
		Status:      godo.HostedAgentSessionStatusReady,
		WorkspaceID: "ws_abc123",
	}})
	printSessionShowCard(&without, &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
		SessionID: "sess_1",
		Status:    godo.HostedAgentSessionStatusReady,
	}})
	assert.Contains(t, with.String(), "Workspace")
	assert.Contains(t, with.String(), "ws_abc123")
	assert.NotContains(t, without.String(), "Workspace", "the row only appears when a workspace is attached")
}
