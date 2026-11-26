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
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSandboxWorkspaceID = "11111111-2222-4333-8444-555555555555"

func testSandboxWorkspace(state godo.SandboxWorkspaceState) do.SandboxWorkspace {
	return do.SandboxWorkspace{SandboxWorkspace: &godo.SandboxWorkspace{
		ID:                testSandboxWorkspaceID,
		Name:              "my-workspace",
		SizeLimitGiB:      10,
		State:             state,
		AttachedSandboxID: testSandboxID,
		BytesUsed:         "4096",
		CreatedAt:         godo.Timestamp{Time: time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)},
	}}
}

func TestSandboxWorkspaceCommandNames(t *testing.T) {
	var ws *Command
	for _, sub := range Sandbox().Commands() {
		if sub.Name() == "workspace" {
			ws = &Command{Command: sub}
		}
	}
	require.NotNil(t, ws, "doctl sandbox workspace")
	assert.Equal(t, []string{"workspaces"}, ws.Aliases)
	assertCommandNames(t, ws, "create", "delete", "get", "list")
}

func TestSandboxWorkspaceCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSandboxWorkspaceName, " my-workspace ")
		config.Doit.Set(config.NS, doctl.ArgSandboxWorkspaceSizeLimitGiB, 10)

		creating := testSandboxWorkspace(godo.SandboxWorkspaceStateCreating)
		tm.sandboxes.EXPECT().CreateWorkspace(&godo.SandboxWorkspaceCreateRequest{
			Name:         "my-workspace",
			SizeLimitGiB: 10,
		}).Return(&creating, nil)

		require.NoError(t, RunSandboxWorkspaceCreate(config))
	})
}

func TestSandboxWorkspaceCreateValidatesBeforeCalling(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSandboxWorkspaceSizeLimitGiB, 0)
		assert.Error(t, RunSandboxWorkspaceCreate(config), "a size limit is required")

		config.Doit.Set(config.NS, doctl.ArgSandboxWorkspaceSizeLimitGiB, 10)
		config.Args = []string{"extra"}
		assert.Error(t, RunSandboxWorkspaceCreate(config))
	})
}

func TestSandboxWorkspaceCreateReturnsAPIErrors(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSandboxWorkspaceSizeLimitGiB, 1000)
		tm.sandboxes.EXPECT().CreateWorkspace(&godo.SandboxWorkspaceCreateRequest{SizeLimitGiB: 1000}).
			Return(nil, errors.New("size_limit_gib must be at most 100"))

		assert.ErrorContains(t, RunSandboxWorkspaceCreate(config), "at most 100")
	})
}

func TestSandboxWorkspaceListFiltersByState(t *testing.T) {
	cases := map[string]godo.SandboxWorkspaceState{
		"":                          "",
		"available":                 godo.SandboxWorkspaceStateAvailable,
		"ATTACHED":                  godo.SandboxWorkspaceStateAttached,
		"WORKSPACE_STATE_RELEASING": godo.SandboxWorkspaceStateReleasing,
		" failed ":                  godo.SandboxWorkspaceStateFailed,
		"workspace_state_creating":  godo.SandboxWorkspaceStateCreating,
		"deleting":                  godo.SandboxWorkspaceStateDeleting,
		"attaching":                 godo.SandboxWorkspaceStateAttaching,
	}
	for flag, want := range cases {
		flag, want := flag, want
		t.Run(flag, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSandboxWorkspaceState, flag)
				tm.sandboxes.EXPECT().ListWorkspaces(want).Return(do.SandboxWorkspaces{testSandboxWorkspace(godo.SandboxWorkspaceStateAvailable)}, nil)

				assert.NoError(t, RunSandboxWorkspaceList(config))
			})
		})
	}
}

func TestSandboxWorkspaceListRejectsAnUnknownState(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		for _, bad := range []string{"running", "WORKSPACE_STATE_UNSPECIFIED"} {
			config.Doit.Set(config.NS, doctl.ArgSandboxWorkspaceState, bad)
			assert.ErrorContains(t, RunSandboxWorkspaceList(config), "invalid --state")
		}

		config.Doit.Set(config.NS, doctl.ArgSandboxWorkspaceState, "")
		config.Args = []string{"extra"}
		assert.Error(t, RunSandboxWorkspaceList(config))
	})
}

func TestSandboxWorkspaceGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testSandboxWorkspaceID}
		w := testSandboxWorkspace(godo.SandboxWorkspaceStateAttached)
		tm.sandboxes.EXPECT().GetWorkspace(testSandboxWorkspaceID).Return(&w, nil)
		require.NoError(t, RunSandboxWorkspaceGet(config))

		config.Args = nil
		assert.Error(t, RunSandboxWorkspaceGet(config))
	})
}

func TestSandboxWorkspaceDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{"w-1", "w-2"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		tm.sandboxes.EXPECT().DeleteWorkspace("w-1").Return(nil)
		tm.sandboxes.EXPECT().DeleteWorkspace("w-2").Return(errors.New("boom"))

		err := RunSandboxWorkspaceDelete(config)
		assert.ErrorContains(t, err, "w-2")
		assert.ErrorContains(t, err, "boom")

		config.Args = nil
		assert.Error(t, RunSandboxWorkspaceDelete(config))
	})
}

func TestSandboxWorkspaceDisplayer(t *testing.T) {
	d := &displayers.SandboxWorkspace{Workspaces: do.SandboxWorkspaces{testSandboxWorkspace(godo.SandboxWorkspaceStateAttached)}}

	assert.Equal(t, []string{"ID", "Name", "State", "SizeLimitGiB", "AttachedSandbox", "BytesUsed", "Created"}, d.Cols())
	kv := d.KV()
	require.Len(t, kv, 1)
	assert.Equal(t, "attached", kv[0]["State"])
	assert.Equal(t, uint64(4096), kv[0]["BytesUsed"])
	assert.Equal(t, testSandboxID, kv[0]["AttachedSandbox"])
	assert.Equal(t, "2026-11-20T12:00:00Z", kv[0]["Created"])

	var buf bytes.Buffer
	require.NoError(t, d.JSON(&buf))
	assert.Contains(t, buf.String(), "WORKSPACE_STATE_ATTACHED", "JSON keeps the API value")
}
