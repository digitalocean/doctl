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
		WorkspaceID:       "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f",
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

// The session-create wire fields have to stay where the server reads them: the
// request body when creating from a config, and the session's own field when
// reading one back.
func TestWorkspaceWireNames(t *testing.T) {
	f, ok := reflect.TypeOf(godo.HostedAgentSessionFromConfigRequest{}).FieldByName("WorkspaceID")
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
			assert.Contains(t, buf.String(), "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f")
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

func TestRunAgentsWorkspaceList_SendsState(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListWorkspaces(&godo.HostedAgentWorkspaceListOptions{State: godo.HostedAgentWorkspaceStateAvailable, PageSize: 2, PageToken: "tok1"}).
			Return([]godo.HostedAgentWorkspace{testWorkspace()}, "", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspaceState, "AVAILABLE")
		config.Doit.Set(config.NS, doctl.ArgAgentPageSize, 2)
		config.Doit.Set(config.NS, doctl.ArgAgentPageToken, "tok1")
		require.NoError(t, RunAgentsWorkspaceList(config))
		assert.Contains(t, buf.String(), "notes")
		assert.NotContains(t, buf.String(), "Next page token:")
	})
}

func TestRunAgentsWorkspaceList_StateIsLeftToTheServer(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListWorkspaces(&godo.HostedAgentWorkspaceListOptions{State: "busy"}).
			Return(nil, "", godoStatusErr(http.StatusBadRequest, "state must be one of AVAILABLE, ATTACHING, ATTACHED, RELEASING, FAILED"))

		config.Out = &bytes.Buffer{}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspaceState, "busy")
		err := RunAgentsWorkspaceList(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "state must be one of")
	})
}

func TestRunAgentsWorkspaceList_EmptyPageWithMoreToCome(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			ListWorkspaces(&godo.HostedAgentWorkspaceListOptions{State: godo.HostedAgentWorkspaceStateFailed}).
			Return(nil, "tok9", nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspaceState, "FAILED")
		require.NoError(t, RunAgentsWorkspaceList(config))
		assert.Contains(t, buf.String(), "No workspaces on this page")
		assert.Contains(t, buf.String(), "tok9")
	})
}

func TestAgentWorkspaceListFlags(t *testing.T) {
	var list *Command
	for _, c := range AgentWorkspaces().ChildCommands() {
		if c.Name() == "list" {
			list = c
		}
	}
	require.NotNil(t, list)
	for _, name := range []string{doctl.ArgAgentWorkspaceState, doctl.ArgAgentPageSize, doctl.ArgAgentPageToken} {
		assert.NotNil(t, list.Flags().Lookup(name), "workspace list needs --%s", name)
	}
	assert.Equal(t, "state", doctl.ArgAgentWorkspaceState)
	for _, state := range []string{"AVAILABLE", "ATTACHING", "ATTACHED", "RELEASING", "FAILED"} {
		assert.Contains(t, agentsWorkspaceListHelpMD, state)
	}
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
		assert.Equal(t, []string{"018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f", "ATTACHED", "10", "3.00", "MiB", "2026-10-05T12:00:00Z"}, strings.Fields(lines[1]))
	})
}

func TestRunAgentsWorkspaceGet(t *testing.T) {
	textOutput(t)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ws := testWorkspace()
		tm.hostedAgents.EXPECT().GetWorkspace("018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f").Return(&ws, nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f"}
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
		tm.hostedAgents.EXPECT().DeleteWorkspace("018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f").Return(nil)

		var buf bytes.Buffer
		config.Out = &buf
		config.Args = []string{"018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		require.NoError(t, RunAgentsWorkspaceDelete(config))
		assert.Contains(t, buf.String(), "Deleted workspace 018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f")
	})
}

// Without --force a delete must ask first; with nobody to ask it must not go
// ahead. No DeleteWorkspace expectation, so a call fails the test.
func TestRunAgentsWorkspaceDelete_RequiresConfirmation(t *testing.T) {
	prev := Interactive
	Interactive = false
	t.Cleanup(func() { Interactive = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f"}
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
			DeleteWorkspace("018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f").
			Return(godoStatusErr(http.StatusConflict, serverMsg))

		config.Args = []string{"018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		err := RunAgentsWorkspaceDelete(config)
		require.Error(t, err)

		var pretty *agentPrettyError
		require.True(t, errors.As(beautifyAgentError(err), &pretty))
		assert.Equal(t, serverMsg, pretty.reason)
		assert.Equal(t, http.StatusConflict, pretty.status)
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
				WorkspaceID: "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f",
			}).
			Return(nil, assertCalledErr)

		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f")
		require.ErrorIs(t, RunAgentsCreate(config), assertCalledErr)
	})
}

// A workspace attaches only to a session made from a saved config. Every other
// source is refused before any request, and the message names the config
// command that saves a manifest.
func TestRunAgentsCreate_WorkspaceNeedsConfig(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	cases := map[string]func(config *CmdConfig){
		"spec": func(config *CmdConfig) {
			config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		},
		"positional manifest": func(config *CmdConfig) {
			config.Args = []string{specPath}
		},
		"harness": func(config *CmdConfig) {
			config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		},
		"template": func(config *CmdConfig) {
			config.Doit.Set(config.NS, doctl.ArgAgentTemplate, "sandbox")
		},
		"discovered agents.yaml": func(config *CmdConfig) {
			require.NoError(t, os.WriteFile("agents.yaml", []byte(sampleManifest), 0o644))
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			// An empty working directory, so nothing is discovered by accident.
			t.Chdir(t.TempDir())
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				// No expectations: gomock fails the test on any API call.
				setup(config)
				config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f")

				err := RunAgentsCreate(config)
				require.Error(t, err)
				assert.Equal(t, workspaceNeedsConfigMessage, err.Error())
			})
		})
	}
}

const workspaceNeedsConfigMessage = "--workspace needs --from-config: save the manifest as an Agent Config first " +
	"(`doctl harness-runtime config create --spec <file> --name <name>`), then create the session from it"

func TestRunAgentsLaunch_WorkspaceNeedsConfig(t *testing.T) {
	stubInteractiveTerminal(t, true)
	dir := t.TempDir()
	specPath := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(sampleManifest), 0o644))

	cases := map[string]func(config *CmdConfig){
		"spec": func(config *CmdConfig) {
			config.Doit.Set(config.NS, doctl.ArgAgentSpec, specPath)
		},
		"positional manifest": func(config *CmdConfig) {
			config.Args = []string{specPath}
		},
		"harness": func(config *CmdConfig) {
			config.Doit.Set(config.NS, doctl.ArgAgentHarness, "opencode")
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				setup(config)
				config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f")

				err := RunAgentsLaunch(config)
				require.Error(t, err)
				assert.Equal(t, workspaceNeedsConfigMessage, err.Error())
			})
		})
	}
}

// Attaching to an existing session cannot take a workspace; the refusal comes
// before the session is even looked up.
func TestRunAgentsLaunch_ExistingSessionRefusesWorkspace(t *testing.T) {
	stubInteractiveTerminal(t, true)
	t.Chdir(t.TempDir())

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"my-session"}
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f")

		err := RunAgentsLaunch(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--workspace only applies when creating a new session from a saved config")
		assert.Contains(t, err.Error(), "create --from-config <config> --workspace <workspace-id>")
	})
}

// Without the flag nothing about a manifest create changes: no options object.
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

func TestRunAgentsLaunch_FromConfig_SendsWorkspace(t *testing.T) {
	stubInteractiveTerminal(t, true)

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
				Name:        "demo",
				ConfigID:    "cfg_abc123",
				WorkspaceID: "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f",
			}).
			Return(nil, assertCalledErr)

		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f")
		require.ErrorIs(t, RunAgentsLaunch(config), assertCalledErr)
	})
}

// A 409 on create with --workspace shows the server's message as sent and adds
// no advice of its own: the server words each conflict for what the caller can
// do, and a retry is wrong for some of them.
func TestRunAgentsCreate_WorkspaceConflictShowsServerMessage(t *testing.T) {
	const serverMsg = "the workspace is attached to another session; remove that session first"
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.hostedAgents.EXPECT().
			CreateSessionFromConfig(gomock.Any()).
			Return(nil, godoStatusErr(http.StatusConflict, serverMsg))
		config.Doit.Set(config.NS, doctl.ArgAgentFromConfig, "cfg_abc123")
		config.Doit.Set(config.NS, doctl.ArgAgentName, "demo")
		config.Doit.Set(config.NS, doctl.ArgAgentWorkspace, "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f")

		err := RunAgentsCreate(config)
		require.Error(t, err)

		var pretty *agentPrettyError
		require.True(t, errors.As(beautifyAgentError(err), &pretty))
		assert.Equal(t, serverMsg, pretty.reason)
		assert.Equal(t, http.StatusConflict, pretty.status)
		assert.Empty(t, pretty.tips)
	})
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
		WorkspaceID: "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f",
	}})
	printSessionShowCard(&without, &do.HostedAgentSession{HostedAgentSession: &godo.HostedAgentSession{
		SessionID: "sess_1",
		Status:    godo.HostedAgentSessionStatusReady,
	}})
	assert.Contains(t, with.String(), "Workspace")
	assert.Contains(t, with.String(), "018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f")
	assert.NotContains(t, without.String(), "Workspace", "the row only appears when a workspace is attached")
}

// Help must say --workspace works with --from-config only, and must not show
// it beside a manifest.
func TestWorkspaceHelpSaysFromConfigOnly(t *testing.T) {
	_, afterHeading, found := strings.Cut(agentsCreateHelpMD, "**Keeping files between sessions.**")
	require.True(t, found)
	createParagraph, _, _ := strings.Cut(afterHeading, "\n\nCreating from a manifest")

	for name, text := range map[string]string{
		"create help paragraph": createParagraph,
		"workspace help":        agentsWorkspaceRootHelpMD,
		"flag":                  agentWorkspaceFlagDesc,
	} {
		assert.Contains(t, text, "--from-config", name)
		assert.NotContains(t, text, "--spec agents.yaml --workspace", name)
		for _, banned := range []string{"disk", "volume", "device", "layer", "slot"} {
			assert.NotContains(t, strings.ToLower(text), banned, name)
		}
	}
	assert.Contains(t, agentsWorkspaceRootHelpMD, "create --from-config reviewer --name review-2 --workspace <workspace-id>")
	assert.NotContains(t, agentsCreateHelpMD, "--spec agents.yaml --workspace")
}
