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
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// A UUID-shaped ref so resolveSessionRef short-circuits instead of listing.
const testExecSessionID = "01a01e58-6209-7c75-8b31-6cb80f7301ff"

// noExit is the sentinel a stubbed execExit records when the runner never asked
// to terminate.
const noExit = -1

// captureExec points stdout, stderr, and the exit call at the test instead of
// the process, and returns the recorded exit code by pointer.
func captureExec(t *testing.T, config *CmdConfig) (stdout, stderr *bytes.Buffer, exitCode *int) {
	t.Helper()
	stdout, stderr = &bytes.Buffer{}, &bytes.Buffer{}
	config.Out = stdout

	prevErr, prevExit := execStderr, execExit
	execStderr = stderr
	code := noExit
	execExit = func(c int) { code = c }
	t.Cleanup(func() { execStderr, execExit = prevErr, prevExit })

	// Registering the root flags sets RetryMax to its default as a side effect,
	// which would make the runner swap the mock for a live no-retry client. The
	// swap has its own test below; here the mock has to survive.
	prevRetry := RetryMax
	RetryMax = 0
	t.Cleanup(func() { RetryMax = prevRetry })

	return stdout, stderr, &code
}

// exec is a flat verb like upload/download, not a nested group: it is a
// session-scoped action, and the tree reserves subcommands for sub-resources
// with their own CRUD (checkpoint, triggers, config).
func TestAgentsExecIsAFlatVerb(t *testing.T) {
	cmd := Agents()
	require.NotNil(t, cmd)

	var exec *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "exec" {
			exec = c
		}
	}
	require.NotNil(t, exec, "agents exec must be registered on the root tree")
	assert.False(t, exec.HasSubCommands(), "exec takes a command to run, not subcommands")
}

func TestAgentsExec(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, stderr, exitCode := captureExec(t, config)

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, req *godo.HostedAgentSandboxExecRequest) (*godo.HostedAgentSandboxExecResponse, error) {
				assert.Equal(t, []string{"ls", "-la"}, req.Argv, "everything after the session is the guest command")
				assert.Equal(t, "/workspace/src", req.Workdir)
				assert.Equal(t, int64(30), req.TimeoutSeconds)
				return &godo.HostedAgentSandboxExecResponse{Stdout: "total 0\n"}, nil
			})

		config.Args = []string{testExecSessionID, "ls", "-la"}
		config.Doit.Set(config.NS, doctl.ArgAgentExecWorkdir, "/workspace/src")
		config.Doit.Set(config.NS, doctl.ArgAgentExecTimeout, 30)

		require.NoError(t, RunAgentsExec(config))
		assert.Equal(t, "total 0\n", stdout.String())
		assert.Empty(t, stderr.String())
		assert.Equal(t, noExit, *exitCode, "a successful command must not force an exit status")
	})
}

// Guest output is the contract: it must arrive byte-for-byte, with the streams
// kept apart and no newline added by doctl.
func TestAgentsExecPassesOutputThroughVerbatim(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, stderr, _ := captureExec(t, config)

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(&godo.HostedAgentSandboxExecResponse{
				Stdout: "no trailing newline",
				Stderr: "warning: careful\n",
			}, nil)

		config.Args = []string{testExecSessionID, "echo", "-n", "no trailing newline"}
		require.NoError(t, RunAgentsExec(config))

		assert.Equal(t, "no trailing newline", stdout.String())
		assert.Equal(t, "warning: careful\n", stderr.String())
	})
}

// A command that ran and failed is not a doctl error: the output still comes
// through, and the guest's code becomes doctl's exit status so `&&` chains and
// `if` tests behave.
func TestAgentsExecPropagatesGuestExitCode(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, stderr, exitCode := captureExec(t, config)

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(&godo.HostedAgentSandboxExecResponse{
				ExitCode: 2,
				Stdout:   "partial work\n",
				Stderr:   "make: *** Error 2\n",
			}, nil)

		config.Args = []string{testExecSessionID, "make"}
		err := RunAgentsExec(config)

		assert.Equal(t, 2, *exitCode)
		// Silent: the guest already explained itself on stderr, so doctl must
		// not print an error of its own on top.
		assert.ErrorIs(t, err, ErrExitSilently)
		assert.Equal(t, "partial work\n", stdout.String())
		assert.Equal(t, "make: *** Error 2\n", stderr.String())
	})
}

func TestAgentsExecJSONOutput(t *testing.T) {
	prev := Output
	Output = "json"
	t.Cleanup(func() { Output = prev })

	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		stdout, _, exitCode := captureExec(t, config)

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(&godo.HostedAgentSandboxExecResponse{
				ExitCode: 7,
				Stdout:   "out\n",
				Stderr:   "err\n",
			}, nil)

		config.Args = []string{testExecSessionID, "false"}
		_ = RunAgentsExec(config)

		// The exit code still propagates under -o json, so a script can branch
		// on status without parsing the body.
		assert.Equal(t, 7, *exitCode)

		var got godo.HostedAgentSandboxExecResponse
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &got), "body: %q", stdout.String())
		assert.Equal(t, 7, got.ExitCode)
		assert.Equal(t, "out\n", got.Stdout)
		assert.Equal(t, "err\n", got.Stderr)
	})
}

func TestAgentsExecRequiresACommand(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Out = io.Discard
		// A session with no argv cannot be run, so the API is never called.
		config.Args = []string{testExecSessionID}
		assert.Error(t, RunAgentsExec(config))

		config.Args = []string{}
		assert.Error(t, RunAgentsExec(config))
	})
}

// Retrying an exec would run the guest command again, so the call must go
// through a client with retries off whenever doctl's shared client has them on.
func TestAgentsExecDoesNotRetry(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		prev := RetryMax
		t.Cleanup(func() { RetryMax = prev })

		RetryMax = 0
		mocked := config.HostedAgents()
		require.NoError(t, disableRetriesForExec(config))
		assert.Same(t, mocked, config.HostedAgents(),
			"with retries already off there is nothing to swap")

		RetryMax = 5
		require.NoError(t, disableRetriesForExec(config))
		assert.NotSame(t, mocked, config.HostedAgents(),
			"exec must not run against the retrying client")
	})
}

func TestAgentsExecSurfacesTransportErrors(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		_, _, exitCode := captureExec(t, config)

		tm.hostedAgents.EXPECT().
			ExecInSandbox(gomock.Any(), testExecSessionID, gomock.Any()).
			Return(nil, errors.New("session is not operable"))

		config.Args = []string{testExecSessionID, "true"}
		err := RunAgentsExec(config)

		require.Error(t, err)
		// A real failure keeps its message and the default status, unlike a
		// non-zero guest exit.
		assert.Contains(t, err.Error(), "session is not operable")
		assert.NotErrorIs(t, err, ErrExitSilently)
		assert.Equal(t, noExit, *exitCode)
	})
}
