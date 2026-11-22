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
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakeStream replays events, then io.EOF.
type fakeStream struct {
	events []*godo.SandboxCommandEvent
	closed bool
}

func (f *fakeStream) Next() (*godo.SandboxCommandEvent, error) {
	if len(f.events) == 0 {
		return nil, io.EOF
	}
	ev := f.events[0]
	f.events = f.events[1:]
	return ev, nil
}

func (f *fakeStream) Close() error { f.closed = true; return nil }

func stdoutEv(s string) *godo.SandboxCommandEvent {
	return &godo.SandboxCommandEvent{Stdout: []byte(s)}
}
func stderrEv(s string) *godo.SandboxCommandEvent {
	return &godo.SandboxCommandEvent{Stderr: []byte(s)}
}
func exitEv(code int32) *godo.SandboxCommandEvent {
	return &godo.SandboxCommandEvent{Exit: &godo.SandboxCommandExit{ExitCode: code, Status: "exited"}}
}

func TestSandboxCommandGroupNames(t *testing.T) {
	cmd := Sandbox()
	assertCommandNames(t, cmd, "command", "create", "delete", "exec", "get", "list", "pause", "resume", "template")
	for _, sub := range cmd.Commands() {
		if sub.Name() == "command" {
			assert.Equal(t, []string{"commands"}, sub.Aliases)
			names := []string{}
			for _, c := range sub.Commands() {
				names = append(names, c.Name())
			}
			assert.ElementsMatch(t, []string{"attach", "kill", "list"}, names)
		}
	}
}

func TestSandboxExecStreamsOutputAndReportsSuccess(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out
		config.Args = []string{testSandboxID, "bash", "-lc", "echo hi"}
		config.Doit.Set(config.NS, doctl.ArgSandboxCwd, "/workspace")
		config.Doit.Set(config.NS, doctl.ArgSandboxEnv, []string{"A=1", "B=x=y,z"})
		config.Doit.Set(config.NS, doctl.ArgSandboxTimeout, 30)
		config.Doit.Set(config.NS, doctl.ArgSandboxStdin, false)

		tm.sandboxes.EXPECT().StartCommand(testSandboxID, &godo.SandboxCommandStartRequest{
			Cmd:            "bash",
			Args:           []string{"-lc", "echo hi"},
			Cwd:            "/workspace",
			EnvVars:        map[string]string{"A": "1", "B": "x=y,z"},
			TimeoutSeconds: 30,
		}).Return(int64(42), nil)
		stream := &fakeStream{events: []*godo.SandboxCommandEvent{
			{Started: &godo.SandboxCommandStarted{PID: 42}},
			{Keepalive: &struct{}{}},
			stdoutEv("hi\n"),
			exitEv(0),
		}}
		tm.sandboxes.EXPECT().StreamCommand(gomock.Any(), testSandboxID, int64(42)).Return(stream, nil)

		require.NoError(t, RunSandboxExec(config))
		assert.Equal(t, "hi\n", out.String())
		assert.True(t, stream.closed)
	})
}

func TestSandboxExecReturnsAnErrorForANonZeroExitCode(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testSandboxID, "false"}
		tm.sandboxes.EXPECT().StartCommand(testSandboxID, &godo.SandboxCommandStartRequest{Cmd: "false"}).Return(int64(7), nil)
		tm.sandboxes.EXPECT().StreamCommand(gomock.Any(), testSandboxID, int64(7)).
			Return(&fakeStream{events: []*godo.SandboxCommandEvent{stderrEv("boom\n"), exitEv(3)}}, nil)

		assert.EqualError(t, RunSandboxExec(config), "command exited with code 3")
	})
}

func TestSandboxExecTellsWhenTheCommandExitedBeforeAttach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testSandboxID, "true"}
		tm.sandboxes.EXPECT().StartCommand(testSandboxID, &godo.SandboxCommandStartRequest{Cmd: "true"}).Return(int64(7), nil)
		notFound := &godo.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}, Message: "not found"}
		tm.sandboxes.EXPECT().StreamCommand(gomock.Any(), testSandboxID, int64(7)).Return(nil, notFound)

		err := RunSandboxExec(config)
		assert.ErrorContains(t, err, "already exited")
		assert.ErrorIs(t, err, notFound)
	})
}

func TestSandboxExecFailsWhenTheStreamEndsWithoutAnExit(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testSandboxID, "sleep", "9"}
		tm.sandboxes.EXPECT().StartCommand(testSandboxID, &godo.SandboxCommandStartRequest{Cmd: "sleep", Args: []string{"9"}}).Return(int64(7), nil)
		tm.sandboxes.EXPECT().StreamCommand(gomock.Any(), testSandboxID, int64(7)).
			Return(&fakeStream{events: []*godo.SandboxCommandEvent{stdoutEv("partial")}}, nil)

		assert.ErrorContains(t, RunSandboxExec(config), "outcome is unknown")
	})
}

func TestSandboxExecValidatesBeforeStarting(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testSandboxID}
		assert.Error(t, RunSandboxExec(config), "a command is required")

		config.Args = []string{testSandboxID, "ls"}
		config.Doit.Set(config.NS, doctl.ArgSandboxEnv, []string{"NOEQUALS"})
		assert.ErrorContains(t, RunSandboxExec(config), "KEY=VALUE")

		config.Doit.Set(config.NS, doctl.ArgSandboxEnv, []string{})
		config.Doit.Set(config.NS, doctl.ArgSandboxTimeout, -1)
		assert.ErrorContains(t, RunSandboxExec(config), "negative")
	})
}

func TestForwardStdinSendsChunksThenEOF(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.sandboxes.EXPECT().WriteCommandStdin(testSandboxID, int64(7), []byte("hello\n"), false).Return(nil)
		tm.sandboxes.EXPECT().WriteCommandStdin(testSandboxID, int64(7), []byte(nil), true).Return(nil)

		forwardStdin(config.Sandboxes(), testSandboxID, 7, strings.NewReader("hello\n"))
	})
}

func TestSandboxCommandList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out
		config.Args = []string{testSandboxID}
		tm.sandboxes.EXPECT().ListCommands(testSandboxID).Return(do.SandboxCommands{
			{SandboxCommand: &godo.SandboxCommand{PID: 42, Cmd: "bash", Cwd: "/workspace"}},
		}, nil)

		require.NoError(t, RunSandboxCommandList(config))
	})
}

func TestSandboxCommandAttach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var out bytes.Buffer
		config.Out = &out
		config.Args = []string{testSandboxID, "42"}
		tm.sandboxes.EXPECT().StreamCommand(gomock.Any(), testSandboxID, int64(42)).
			Return(&fakeStream{events: []*godo.SandboxCommandEvent{stdoutEv("a"), stdoutEv("b"), exitEv(0)}}, nil)

		require.NoError(t, RunSandboxCommandAttach(config))
		assert.Equal(t, "ab", out.String())
	})
}

func TestSandboxCommandKill(t *testing.T) {
	cases := map[string]godo.SandboxCommandSignal{
		"":        godo.SandboxCommandSignalTerm,
		"term":    godo.SandboxCommandSignalTerm,
		"SIGTERM": godo.SandboxCommandSignalTerm,
		"kill":    godo.SandboxCommandSignalKill,
		"SIGKILL": godo.SandboxCommandSignalKill,
	}
	for flag, want := range cases {
		flag, want := flag, want
		t.Run(flag, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Args = []string{testSandboxID, "42"}
				config.Doit.Set(config.NS, doctl.ArgSandboxSignal, flag)
				tm.sandboxes.EXPECT().SignalCommand(testSandboxID, int64(42), want).Return(nil)

				assert.NoError(t, RunSandboxCommandKill(config))
			})
		})
	}
}

func TestSandboxCommandArgumentValidation(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgSandboxSignal, "hup")
		config.Args = []string{testSandboxID, "42"}
		assert.ErrorContains(t, RunSandboxCommandKill(config), "term or kill")

		for _, args := range [][]string{{}, {testSandboxID}, {testSandboxID, "abc"}, {testSandboxID, "-1"}, {testSandboxID, "1", "2"}} {
			config.Args = args
			assert.Error(t, RunSandboxCommandKill(config), "args %v", args)
			assert.Error(t, RunSandboxCommandAttach(config), "args %v", args)
		}
		config.Args = nil
		assert.Error(t, RunSandboxCommandList(config))
	})
}

func TestStreamSandboxCommandPropagatesStreamErrors(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.sandboxes.EXPECT().StreamCommand(gomock.Any(), testSandboxID, int64(1)).Return(nil, errors.New("denied"))
		_, err := streamSandboxCommand(context.Background(), config.Sandboxes(), testSandboxID, 1, io.Discard, io.Discard)
		assert.EqualError(t, err, "denied")
	})
}
