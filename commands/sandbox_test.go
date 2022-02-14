/*
Copyright 2018 The Doctl Authors All rights reserved.
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
	"os/exec"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxConnect(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf
		fakeCmd := &exec.Cmd{
			Stdout: config.Out,
		}

		config.Args = append(config.Args, "token")
		tm.sandbox.EXPECT().Cmd("auth/login", []string{"token"}).Return(fakeCmd, nil)
		tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.SandboxOutput{
			Entity: map[string]interface{}{
				"namespace": "hello",
				"apihost":   "https://api.example.com",
			},
		}, nil)

		err := RunSandboxConnect(config)
		require.NoError(t, err)
		assert.Equal(t, "Connected to function namespace 'hello' on API host 'https://api.example.com'\n\n", buf.String())
	})
}

func TestSandboxStatusWhenConnected(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf
		fakeCmd := &exec.Cmd{
			Stdout: config.Out,
		}

		tm.sandbox.EXPECT().Cmd("auth/current", []string{"--apihost", "--name"}).Return(fakeCmd, nil)
		tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.SandboxOutput{
			Entity: map[string]interface{}{
				"name":    "hello",
				"apihost": "https://api.example.com",
			},
		}, nil)

		err := RunSandboxStatus(config)
		require.NoError(t, err)
		assert.Equal(t, "Connected to function namespace 'hello' on API host 'https://api.example.com'\n\n", buf.String())
	})
}

func TestSandboxStatusWhenNotConnected(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fakeCmd := &exec.Cmd{
			Stdout: config.Out,
		}

		tm.sandbox.EXPECT().Cmd("auth/current", []string{"--apihost", "--name"}).Return(fakeCmd, nil)
		tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.SandboxOutput{
			Error: "403",
		}, nil)

		err := RunSandboxStatus(config)
		require.Error(t, err)
		assert.EqualError(t, err, "A sandbox is installed but not connected to a function namespace (see 'doctl sandbox connect')")
	})
}

func TestSandboxStatusWhenNotInstalled(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.sandboxInstalled = func() bool {
			return false
		}

		err := RunSandboxStatus(config)

		require.Error(t, err)
		assert.EqualError(t, err, SandboxNotInstalledErr.Error())
	})
}

func TestSandboxInit(t *testing.T) {
	tests := []struct {
		name            string
		doctlArgs       string
		doctlFlags      map[string]string
		expectedNimArgs []string
		out             map[string]interface{}
	}{
		{
			name:            "no flags",
			doctlArgs:       "path/to/foo",
			expectedNimArgs: []string{"path/to/foo"},
			out:             map[string]interface{}{"project": "foo"},
		},
		{
			name:            "overwrite",
			doctlArgs:       "path/to/project",
			doctlFlags:      map[string]string{"overwrite": ""},
			expectedNimArgs: []string{"path/to/project", "--overwrite"},
			out:             map[string]interface{}{"project": "foo"},
		},
		{
			name:            "language flag",
			doctlArgs:       "path/to/project",
			doctlFlags:      map[string]string{"language": "go"},
			expectedNimArgs: []string{"path/to/project", "--language", "go"},
			out:             map[string]interface{}{"project": "foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				buf := &bytes.Buffer{}
				config.Out = buf
				fakeCmd := &exec.Cmd{
					Stdout: config.Out,
				}

				if tt.doctlArgs != "" {
					config.Args = append(config.Args, tt.doctlArgs)
				}

				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}

				tm.sandbox.EXPECT().Cmd("project/create", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.SandboxOutput{
					Entity: tt.out,
				}, nil)

				err := RunSandboxExtraCreate(config)
				require.NoError(t, err)
				assert.Equal(t, `A local sandbox area 'foo' was created for you.
You may deploy it by running the command shown on the next line:
  doctl sandbox deploy foo`+"\n\n", buf.String())
			})
		})
	}
}

func TestSandboxDeploy(t *testing.T) {
	tests := []struct {
		name            string
		doctlArgs       string
		doctlFlags      map[string]string
		expectedNimArgs []string
	}{
		{
			name:            "no flags with path",
			doctlArgs:       "path/to/project",
			expectedNimArgs: []string{"path/to/project", "--exclude", "web"},
		},
		// TODO: Add additional scenarios for other flags, etc.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				fakeCmd := &exec.Cmd{
					Stdout: config.Out,
				}

				if tt.doctlArgs != "" {
					config.Args = append(config.Args, tt.doctlArgs)
				}

				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}

				tm.sandbox.EXPECT().Cmd("project/deploy", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.SandboxOutput{}, nil)

				err := RunSandboxExtraDeploy(config)
				require.NoError(t, err)
			})
		})
	}
}

func TestSandboxWatch(t *testing.T) {
	tests := []struct {
		name            string
		doctlArgs       string
		doctlFlags      map[string]string
		expectedNimArgs []string
	}{
		{
			name:            "no flags with path",
			doctlArgs:       "path/to/project",
			expectedNimArgs: []string{"path/to/project", "--exclude", "web"},
		},
		// TODO: Add additional scenarios for other flags, etc.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				fakeCmd := &exec.Cmd{
					Stdout: config.Out,
				}

				if tt.doctlArgs != "" {
					config.Args = append(config.Args, tt.doctlArgs)
				}

				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}

				tm.sandbox.EXPECT().Cmd("project/watch", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.sandbox.EXPECT().Stream(fakeCmd).Return(nil)

				err := RunSandboxExtraWatch(config)
				require.NoError(t, err)
			})
		})
	}
}
