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
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const testSandboxID = "01a0ef1a-d873-793d-ae6a-5dd6239be62a"

func testSandbox(state godo.SandboxState) do.Sandbox {
	return do.Sandbox{Sandbox: &godo.Sandbox{
		ID:        testSandboxID,
		Template:  "coding-base",
		State:     state,
		SizeSlug:  "mars-2vcpu-4gb",
		CreatedAt: godo.Timestamp{Time: time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)},
	}}
}

func TestSandboxCommands(t *testing.T) {
	cmd := Sandbox()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "command", "create", "delete", "exec", "get", "list", "pause", "resume", "snapshot", "template")
	assert.Equal(t, "sandbox", cmd.Use)
	assert.Equal(t, []string{"sandboxes", "sb"}, cmd.Aliases)
}

func TestSandboxCreateWithDefaults(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSandboxTemplate, "coding-base")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "mars-2vcpu-4gb")
		config.Doit.Set(config.NS, doctl.ArgSandboxAllowInternetAccess, true)

		running := testSandbox(godo.SandboxStateRunning)
		tm.sandboxes.EXPECT().Create(&godo.SandboxCreateRequest{
			Template: "coding-base",
			SizeSlug: "mars-2vcpu-4gb",
		}).Return(&running, nil)

		assert.NoError(t, RunSandboxCreate(config))
	})
}

func TestSandboxCreateWithOptions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		off := false
		config.Doit.Set(config.NS, doctl.ArgSandboxTemplate, "other")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "s-1vcpu-2gb")
		config.Doit.Set(config.NS, doctl.ArgSandboxAllowInternetAccess, false)
		config.Doit.Set(config.NS, doctl.ArgSandboxEnv, []string{"A=1", "B=x=y,z"})

		running := testSandbox(godo.SandboxStateRunning)
		tm.sandboxes.EXPECT().Create(&godo.SandboxCreateRequest{
			Template:            "other",
			SizeSlug:            "s-1vcpu-2gb",
			AllowInternetAccess: &off,
			EnvVars:             map[string]string{"A": "1", "B": "x=y,z"},
		}).Return(&running, nil)

		assert.NoError(t, RunSandboxCreate(config))
	})
}

func TestSandboxCreateValidatesBeforeCalling(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSandboxTemplate, " ")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "mars-2vcpu-4gb")
		assert.Error(t, RunSandboxCreate(config))

		config.Doit.Set(config.NS, doctl.ArgSandboxTemplate, "coding-base")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "")
		assert.Error(t, RunSandboxCreate(config))

		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "mars-2vcpu-4gb")
		config.Doit.Set(config.NS, doctl.ArgSandboxEnv, []string{"NOEQUALS"})
		assert.ErrorContains(t, RunSandboxCreate(config), "KEY=VALUE")

		config.Args = []string{"extra"}
		assert.Error(t, RunSandboxCreate(config))
	})
}

func TestSandboxCreateReturnsAPIErrors(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSandboxTemplate, "nope")
		config.Doit.Set(config.NS, doctl.ArgSizeSlug, "mars-2vcpu-4gb")
		config.Doit.Set(config.NS, doctl.ArgSandboxAllowInternetAccess, true)
		tm.sandboxes.EXPECT().Create(&godo.SandboxCreateRequest{Template: "nope", SizeSlug: "mars-2vcpu-4gb"}).
			Return(nil, errors.New("Template not found."))

		assert.ErrorContains(t, RunSandboxCreate(config), "Template not found.")
	})
}

func TestSandboxList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		list := do.Sandboxes{testSandbox(godo.SandboxStateRunning), testSandbox(godo.SandboxStatePaused)}
		tm.sandboxes.EXPECT().List(godo.SandboxState("")).Return(list, nil)

		assert.NoError(t, RunSandboxList(config))
	})
}

func TestSandboxListFiltersByState(t *testing.T) {
	cases := map[string]godo.SandboxState{
		"running":               godo.SandboxStateRunning,
		"PAUSED":                godo.SandboxStatePaused,
		"creating":              godo.SandboxStateCreating,
		"SANDBOX_STATE_RUNNING": godo.SandboxStateRunning,
	}
	for flag, want := range cases {
		flag, want := flag, want
		t.Run(flag, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgSandboxState, flag)
				tm.sandboxes.EXPECT().List(want).Return(do.Sandboxes{}, nil)

				assert.NoError(t, RunSandboxList(config))
			})
		})
	}
}

func TestSandboxListRejectsAnUnknownState(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSandboxState, "deleted")

		assert.ErrorContains(t, RunSandboxList(config), "running, paused or creating")
	})
}

func TestSandboxGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		running := testSandbox(godo.SandboxStateRunning)
		tm.sandboxes.EXPECT().Get(testSandboxID).Return(&running, nil)
		config.Args = []string{testSandboxID}

		assert.NoError(t, RunSandboxGet(config))
	})
}

func TestSandboxPauseAndResume(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		paused := testSandbox(godo.SandboxStatePaused)
		running := testSandbox(godo.SandboxStateRunning)
		tm.sandboxes.EXPECT().Pause(testSandboxID).Return(&paused, nil)
		tm.sandboxes.EXPECT().Resume(testSandboxID).Return(&running, nil)
		config.Args = []string{testSandboxID}

		assert.NoError(t, RunSandboxPause(config))
		assert.NoError(t, RunSandboxResume(config))
	})
}

func TestSandboxCommandsNeedExactlyOneID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		for _, run := range []CmdRunner{RunSandboxGet, RunSandboxPause, RunSandboxResume} {
			config.Args = nil
			assert.Error(t, run(config))
			config.Args = []string{"a", "b"}
			assert.Error(t, run(config))
		}
	})
}

func TestSandboxDeleteWithForce(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.sandboxes.EXPECT().Delete("sb-1").Return(nil)
		tm.sandboxes.EXPECT().Delete("sb-2").Return(nil)
		config.Args = []string{"sb-1", "sb-2"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		assert.NoError(t, RunSandboxDelete(config))
	})
}

func TestSandboxDeleteStopsAtTheFirstFailureAndNamesIt(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.sandboxes.EXPECT().Delete("sb-1").Return(errors.New("boom"))
		config.Args = []string{"sb-1", "sb-2"}
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		err := RunSandboxDelete(config)
		assert.ErrorContains(t, err, "sb-1")
		assert.ErrorContains(t, err, "boom")
	})
}

func TestSandboxDeleteNeedsAnID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		assert.Error(t, RunSandboxDelete(config))
	})
}

func TestSandboxDisplay(t *testing.T) {
	sb := displayers.Sandbox{Sandboxes: do.Sandboxes{testSandbox(godo.SandboxStatePaused)}}

	kv := sb.KV()
	assert.Equal(t, []map[string]any{{
		"ID":       testSandboxID,
		"Template": "coding-base",
		"State":    "paused",
		"Size":     "mars-2vcpu-4gb",
		"Created":  "2026-11-20T12:00:00Z",
	}}, kv)
	assert.Equal(t, []string{"ID", "Template", "State", "Size", "Created"}, sb.Cols())
	assert.Equal(t, "Created At", sb.ColMap()["Created"])

	// JSON keeps the API values, so scripts do not depend on the table's wording.
	var buf bytes.Buffer
	assert.NoError(t, sb.JSON(&buf))
	assert.Contains(t, buf.String(), `"state": "SANDBOX_STATE_PAUSED"`)
	assert.Contains(t, buf.String(), `"sandbox_id": "`+testSandboxID+`"`)
}

func TestSandboxStateFormatting(t *testing.T) {
	assert.Equal(t, "running", displayers.FormatSandboxState(godo.SandboxStateRunning))
	assert.Equal(t, "creating", displayers.FormatSandboxState(godo.SandboxStateCreating))
	assert.Equal(t, "unspecified", displayers.FormatSandboxState(godo.SandboxStateUnspecified))
	assert.Equal(t, "", displayers.FormatSandboxState(""))
}

// `doctl sandbox` used to be an alias of `doctl serverless`. The Sandboxes
// command owns that name now, and serverless must not shadow it.
func TestSandboxNameIsOnlyOwnedBySandboxes(t *testing.T) {
	assert.NotContains(t, Serverless().Aliases, "sandbox")
	assert.NotContains(t, Serverless().Aliases, "sandboxes")
	assert.Equal(t, []string{"sbx", "sls"}, Serverless().Aliases)

	root := &cobra.Command{Use: "doctl"}
	root.AddCommand(Serverless().Command, Sandbox().Command)
	for _, name := range []string{"sandbox", "sandboxes"} {
		found, _, err := root.Find([]string{name})
		assert.NoError(t, err)
		assert.Equal(t, "sandbox", found.Name(), "doctl %s", name)
	}
	for _, name := range []string{"serverless", "sls", "sbx"} {
		found, _, err := root.Find([]string{name})
		assert.NoError(t, err)
		assert.Equal(t, "serverless", found.Name(), "doctl %s", name)
	}
}
