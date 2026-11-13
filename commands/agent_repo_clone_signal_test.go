//go:build !windows

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
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/digitalocean/godo"
)

// These tests deliver a real SIGINT to the test process, which Windows cannot
// do, so they build everywhere else.

// interruptSelf delivers Ctrl-C to the process; the clone's signal context
// catches it, so the test binary survives.
func interruptSelf(t *testing.T) {
	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGINT))
	time.Sleep(100 * time.Millisecond)
}

func captureInterruptExit(t *testing.T) (*int, *bytes.Buffer) {
	code := new(int)
	prevExit := execExit
	execExit = func(c int) { *code = c }
	var stderr bytes.Buffer
	prevErr := execStderr
	execStderr = &stderr
	t.Cleanup(func() { execExit = prevExit; execStderr = prevErr })
	return code, &stderr
}

// Ctrl-C at any point means the prompt is not sent, including when the guest
// call itself reports success.
func TestFinishSessionStart_InterruptDuringCheckSkipsPrompt(t *testing.T) {
	repoTestSetup(t)
	code, stderr := captureInterruptExit(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ready := repoTestSession("sess_int")
		ready.Status = godo.HostedAgentSessionStatusReady
		tm.hostedAgents.EXPECT().GetSession("sess_int").Return(ready, nil)
		// The check answers "exists" at the very moment of the interrupt.
		tm.hostedAgents.EXPECT().ExecInSandbox(gomock.Any(), "sess_int", gomock.Any()).
			DoAndReturn(func(context.Context, string, *godo.HostedAgentSandboxExecRequest) (*godo.HostedAgentSandboxExecResponse, error) {
				interruptSelf(t)
				return &godo.HostedAgentSandboxExecResponse{ExitCode: 0}, nil
			})
		// No SendInput expectation: sending would fail the test.
		_, _, err := finishSessionStart(config, repoTestSession("sess_int"),
			&agentCreationSource{repo: "o/r", prompt: "go"}, nil)
		assert.ErrorIs(t, err, ErrExitSilently)
		assert.Equal(t, promptInterruptExit, *code)
		assert.Contains(t, stderr.String(), "still running and the prompt was not sent")
	})
}

func TestFinishSessionStart_InterruptDuringCloneSkipsPrompt(t *testing.T) {
	repoTestSetup(t)
	code, _ := captureInterruptExit(t)
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		ready := repoTestSession("sess_int2")
		ready.Status = godo.HostedAgentSessionStatusReady
		tm.hostedAgents.EXPECT().GetSession("sess_int2").Return(ready, nil)
		tm.hostedAgents.EXPECT().ExecInSandbox(gomock.Any(), "sess_int2", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, req *godo.HostedAgentSandboxExecRequest) (*godo.HostedAgentSandboxExecResponse, error) {
				if req.Argv[0] == "test" {
					return &godo.HostedAgentSandboxExecResponse{ExitCode: 1}, nil
				}
				interruptSelf(t)
				return &godo.HostedAgentSandboxExecResponse{ExitCode: 0}, nil
			}).Times(2)
		_, _, err := finishSessionStart(config, repoTestSession("sess_int2"),
			&agentCreationSource{repo: "o/r", prompt: "go"}, nil)
		assert.ErrorIs(t, err, ErrExitSilently)
		assert.Equal(t, promptInterruptExit, *code)
	})
}
