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
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxConnectNamespace(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf
		fakeCmd := &exec.Cmd{
			Stdout: config.Out,
		}

		config.Args = append(config.Args, "hello")
		tm.sandbox.EXPECT().ResolveNamespace(context.TODO(), "hello").Return(do.SandboxCredentials{Auth: "xyzzy", ApiHost: "https://api.example.com"}, nil)
		tm.sandbox.EXPECT().Cmd("auth/login", []string{"--auth", "xyzzy", "--apihost", "https://api.example.com"}).Return(fakeCmd, nil)
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

func TestSandboxConnectToken(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf
		fakeCmd := &exec.Cmd{
			Stdout: config.Out,
		}

		fakeJWT := "a-very-fake-JWT.a-very-fake-JWT.a-very-fake-JWT" // very unimaginative also, but this is enough to trigger JWT recognition
		config.Args = append(config.Args, fakeJWT)
		tm.sandbox.EXPECT().ResolveToken(context.TODO(), fakeJWT).Return(do.SandboxCredentials{Auth: "xyzzy", ApiHost: "https://api.example.com"}, nil)
		tm.sandbox.EXPECT().Cmd("auth/login", []string{"--auth", "xyzzy", "--apihost", "https://api.example.com"}).Return(fakeCmd, nil)
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
		assert.ErrorIs(t, err, SandboxNotConnectedErr)
	})
}

func TestSandboxStatusWhenNotInstalled(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.checkSandboxStatus = func() error {
			return SandboxNotInstalledErr
		}

		err := RunSandboxStatus(config)

		require.Error(t, err)
		assert.ErrorIs(t, err, SandboxNotInstalledErr)
	})
}

func TestSandboxStatusWhenNotUpToDate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.checkSandboxStatus = func() error {
			return SandboxNeedsUpgradeErr
		}

		err := RunSandboxStatus(config)

		require.Error(t, err)
		assert.ErrorIs(t, err, SandboxNeedsUpgradeErr)
	})
}

func TestSandboxInstallFromScratch(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		config.installSandbox = func(dir string, upgrade bool) error {
			fmt.Fprintf(config.Out, "Installed with upgrade %v\n", upgrade)
			return nil
		}
		config.checkSandboxStatus = func() error {
			return SandboxNotInstalledErr
		}

		err := RunSandboxInstall(config)
		require.NoError(t, err)
		assert.Equal(t, "Installed with upgrade false\n", buf.String())
	})
}

func TestSandboxInstallWhenInstalledNotCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		config.installSandbox = func(dir string, upgrade bool) error {
			fmt.Fprintf(config.Out, "Installed with upgrade %v\n", upgrade)
			return nil
		}
		config.checkSandboxStatus = func() error {
			return SandboxNeedsUpgradeErr
		}

		err := RunSandboxInstall(config)
		require.NoError(t, err)
		assert.Equal(t, "Sandbox support is already installed, but needs an upgrade for this version of `doctl`.\nUse `doctl sandbox upgrade` to upgrade the support.\n", buf.String())
	})
}

func TestSandboxInstallWhenInstalledAndCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		config.installSandbox = func(dir string, upgrade bool) error {
			fmt.Fprintf(config.Out, "Installed with upgrade %v\n", upgrade)
			return nil
		}
		config.checkSandboxStatus = func() error {
			return nil
		}

		err := RunSandboxInstall(config)
		require.NoError(t, err)
		assert.Equal(t, "Sandbox support is already installed at an appropriate version.  No action needed.\n", buf.String())
	})
}

func TestSandboxUpgradeWhenNotInstalled(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		config.installSandbox = func(dir string, upgrade bool) error {
			fmt.Fprintf(config.Out, "Installed with upgrade %v\n", upgrade)
			return nil
		}
		config.checkSandboxStatus = func() error {
			return SandboxNotInstalledErr
		}

		err := RunSandboxUpgrade(config)
		require.NoError(t, err)
		assert.Equal(t, "Sandbox support was never installed.  Use `doctl sandbox install`.\n", buf.String())
	})
}

func TestSandboxUpgradeWhenInstalledAndCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		config.installSandbox = func(dir string, upgrade bool) error {
			fmt.Fprintf(config.Out, "Installed with upgrade %v\n", upgrade)
			return nil
		}
		config.checkSandboxStatus = func() error {
			return nil
		}

		err := RunSandboxUpgrade(config)
		require.NoError(t, err)
		assert.Equal(t, "Sandbox support is already installed at an appropriate version.  No action needed.\n", buf.String())
	})
}

func TestSandboxUpgradeWhenInstalledAndNotCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		config.installSandbox = func(dir string, upgrade bool) error {
			fmt.Fprintf(config.Out, "Installed with upgrade %v\n", upgrade)
			return nil
		}
		config.checkSandboxStatus = func() error {
			return SandboxNeedsUpgradeErr
		}

		err := RunSandboxUpgrade(config)
		require.NoError(t, err)
		assert.Equal(t, "Installed with upgrade true\n", buf.String())
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
