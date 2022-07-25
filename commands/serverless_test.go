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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerlessConnect(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).Return(do.ErrServerlessNotConnected)
		creds := do.ServerlessCredentials{Namespace: "hello", APIHost: "https://api.example.com"}
		tm.serverless.EXPECT().GetServerlessNamespace(context.TODO()).Return(creds, nil)
		tm.serverless.EXPECT().WriteCredentials(creds).Return(nil)

		err := RunServerlessConnect(config)
		require.NoError(t, err)
		assert.Equal(t, "Connected to functions namespace 'hello' on API host 'https://api.example.com'\n\n", buf.String())
	})
}

func TestServerlessStatusWhenConnected(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf
		fakeCmd := &exec.Cmd{
			Stdout: config.Out,
		}

		tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
		tm.serverless.EXPECT().Cmd("auth/current", []string{"--apihost", "--name"}).Return(fakeCmd, nil)
		tm.serverless.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{
			Entity: map[string]interface{}{
				"name":    "hello",
				"apihost": "https://api.example.com",
			},
		}, nil)

		err := RunServerlessStatus(config)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Connected to functions namespace 'hello' on API host 'https://api.example.com'\nServerless software version is")
	})
}

func TestServerlessStatusWithLanguages(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf
		config.Doit.Set(config.NS, "languages", true)
		fakeCmd := &exec.Cmd{
			Stdout: config.Out,
		}
		fakeHostInfo := do.ServerlessHostInfo{
			Runtimes: map[string][]do.ServerlessRuntime{
				"go": {
					{
						Kind:       "go:1.20",
						Deprecated: true,
						Default:    false,
					},
					{
						Kind:       "go:1.21",
						Deprecated: false,
						Default:    false,
					},
					{
						Kind:       "go:1.22",
						Deprecated: false,
						Default:    true,
					},
				},
			},
		}
		expectedDisplay := `go:
  Keywords: go, golang
  Runtime versions:
    go:1.20 (deprecated)
    go:1.21
    go:1.22 (go:default)
`

		tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
		tm.serverless.EXPECT().Cmd("auth/current", []string{"--apihost", "--name"}).Return(fakeCmd, nil)
		tm.serverless.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{
			Entity: map[string]interface{}{
				"name":    "hello",
				"apihost": "https://api.example.com",
			},
		}, nil)
		tm.serverless.EXPECT().GetHostInfo("https://api.example.com").Return(fakeHostInfo, nil)
		err := RunServerlessStatus(config)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), expectedDisplay)
	})
}

func TestServerlessStatusWhenNotConnected(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		fakeCmd := &exec.Cmd{
			Stdout: config.Out,
		}

		tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
		tm.serverless.EXPECT().Cmd("auth/current", []string{"--apihost", "--name"}).Return(fakeCmd, nil)
		tm.serverless.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{
			Error: "403",
		}, nil)

		err := RunServerlessStatus(config)
		require.Error(t, err)
		assert.ErrorIs(t, err, do.ErrServerlessNotConnected)
	})
}

func TestServerlessStatusWhenNotInstalled(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).Return(do.ErrServerlessNotInstalled)

		err := RunServerlessStatus(config)

		require.Error(t, err)
		assert.ErrorIs(t, err, do.ErrServerlessNotInstalled)
	})
}

func TestServerlessStatusWhenNotUpToDate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).Return(do.ErrServerlessNeedsUpgrade)

		err := RunServerlessStatus(config)

		require.Error(t, err)
		assert.ErrorIs(t, err, do.ErrServerlessNeedsUpgrade)
	})
}

func TestServerlessInstallFromScratch(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		credsToken := hashAccessToken(config)
		tm.serverless.EXPECT().CheckServerlessStatus(credsToken).Return(do.ErrServerlessNotInstalled)
		tm.serverless.EXPECT().InstallServerless(credsToken, false).Return(nil)

		err := RunServerlessInstall(config)
		require.NoError(t, err)
	})
}

func TestServerlessInstallWhenInstalledNotCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		credsToken := hashAccessToken(config)
		tm.serverless.EXPECT().CheckServerlessStatus(credsToken).Return(do.ErrServerlessNeedsUpgrade)

		err := RunServerlessInstall(config)
		require.NoError(t, err)
		assert.Equal(t, "Serverless support is already installed, but needs an upgrade for this version of `doctl`.\nUse `doctl serverless upgrade` to upgrade the support.\n", buf.String())
	})
}

func TestServerlessInstallWhenInstalledAndCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).Return(nil)

		err := RunServerlessInstall(config)
		require.NoError(t, err)
		assert.Equal(t, "Serverless support is already installed at an appropriate version.  No action needed.\n", buf.String())
	})
}

func TestServerlessUpgradeWhenNotInstalled(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		credsToken := hashAccessToken(config)
		tm.serverless.EXPECT().CheckServerlessStatus(credsToken).Return(do.ErrServerlessNotInstalled)

		err := RunServerlessUpgrade(config)
		require.NoError(t, err)
		assert.Equal(t, "Serverless support was never installed.  Use `doctl serverless install`.\n", buf.String())
	})
}

func TestServerlessUpgradeWhenInstalledAndCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).Return(nil)

		err := RunServerlessUpgrade(config)
		require.NoError(t, err)
		assert.Equal(t, "Serverless support is already installed at an appropriate version.  No action needed.\n", buf.String())
	})
}

func TestServerlessUpgradeWhenInstalledAndNotCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		credsToken := hashAccessToken(config)
		tm.serverless.EXPECT().CheckServerlessStatus(credsToken).Return(do.ErrServerlessNeedsUpgrade)
		tm.serverless.EXPECT().InstallServerless(credsToken, true).Return(nil)

		err := RunServerlessUpgrade(config)
		require.NoError(t, err)
	})
}

func TestServerlessInit(t *testing.T) {
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

				tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
				tm.serverless.EXPECT().Cmd("project/create", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.serverless.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{
					Entity: tt.out,
				}, nil)

				err := RunServerlessExtraCreate(config)
				require.NoError(t, err)
				assert.Equal(t, `A local functions project directory 'foo' was created for you.
You may deploy it by running the command shown on the next line:
  doctl serverless deploy foo`+"\n\n", buf.String())
			})
		})
	}
}

func TestServerlessDeploy(t *testing.T) {
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
		{
			name:            "include flag for package 'web'",
			doctlArgs:       "path/to/project",
			doctlFlags:      map[string]string{"include": "web"},
			expectedNimArgs: []string{"path/to/project", "--include", "web/", "--exclude", "web"},
		},
		{
			name:            "exclude flag for package 'web'",
			doctlArgs:       "path/to/project",
			doctlFlags:      map[string]string{"exclude": "web"},
			expectedNimArgs: []string{"path/to/project", "--exclude", "web/,web"},
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

				tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
				tm.serverless.EXPECT().Cmd("project/deploy", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.serverless.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{}, nil)

				err := RunServerlessExtraDeploy(config)
				require.NoError(t, err)
			})
		})
	}
}

func TestServerlessUndeploy(t *testing.T) {
	type testNimCmd struct {
		cmd  string
		args []string
	}

	tests := []struct {
		name            string
		doctlArgs       []string
		doctlFlags      map[string]string
		expectedNimCmds []testNimCmd
		expectedError   error
	}{
		{
			name:            "no arguments or flags",
			doctlArgs:       nil,
			doctlFlags:      nil,
			expectedNimCmds: nil,
			expectedError:   errUndeployTooFewArgs,
		},
		{
			name:       "with --all flag only",
			doctlArgs:  nil,
			doctlFlags: map[string]string{"all": ""},
			expectedNimCmds: []testNimCmd{
				{
					cmd:  "namespace/clean",
					args: []string{"--force"},
				},
			},
			expectedError: nil,
		},
		{
			name:       "mixed args, no flags",
			doctlArgs:  []string{"foo/bar", "baz"},
			doctlFlags: nil,
			expectedNimCmds: []testNimCmd{
				{
					cmd:  "action/delete",
					args: []string{"foo/bar"},
				},
				{
					cmd:  "action/delete",
					args: []string{"baz"},
				},
			},
			expectedError: nil,
		},
		{
			name:       "mixed args, --packages flag",
			doctlArgs:  []string{"foo/bar", "baz"},
			doctlFlags: map[string]string{"packages": ""},
			expectedNimCmds: []testNimCmd{
				{
					cmd:  "action/delete",
					args: []string{"foo/bar"},
				},
				{
					cmd:  "package/delete",
					args: []string{"baz", "--recursive"},
				},
			},
			expectedError: nil,
		},
		{
			name:            "mixed args, --all flag",
			doctlArgs:       []string{"foo/bar", "baz"},
			doctlFlags:      map[string]string{"all": ""},
			expectedNimCmds: nil,
			expectedError:   errUndeployAllAndArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				fakeCmd := &exec.Cmd{
					Stdout: config.Out,
				}

				if len(tt.doctlArgs) > 0 {
					config.Args = append(config.Args, tt.doctlArgs...)
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

				if tt.expectedError == nil {
					tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
				}
				for i := range tt.expectedNimCmds {
					tm.serverless.EXPECT().Cmd(tt.expectedNimCmds[i].cmd, tt.expectedNimCmds[i].args).Return(fakeCmd, nil)
					tm.serverless.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{}, nil)
				}
				err := RunServerlessUndeploy(config)
				if tt.expectedError != nil {
					require.Error(t, err)
					assert.ErrorIs(t, err, tt.expectedError)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestServerlessWatch(t *testing.T) {
	tests := []struct {
		name            string
		doctlArgs       string
		doctlFlags      map[string]string
		expectedNimArgs []string
	}{
		{
			name:            "no flags with path",
			doctlArgs:       "path/to/project",
			expectedNimArgs: []string{"project/watch", "path/to/project", "--exclude", "web"},
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

				tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
				tm.serverless.EXPECT().Cmd("nocapture", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.serverless.EXPECT().Stream(fakeCmd).Return(nil)

				err := RunServerlessExtraWatch(config)
				require.NoError(t, err)
			})
		})
	}
}

func TestGetCredentialDirectory(t *testing.T) {
	testDir := "/home/foo/.config/doctl/sandbox/"
	tests := []struct {
		name      string
		tokenFunc func() string
		expected  string
	}{
		{
			name: "legacy token",
			tokenFunc: func() string {
				return "a7bbe7e8af7411ec912e47a270a2ee78a7bbe7e8af7411ec912e47a270a2ee78"
			},
			expected: filepath.Join(testDir, "creds", "3785870f"),
		},
		{
			name: "v1 token",
			tokenFunc: func() string {
				return "dop_v1_a7bbe7e8af7411ec912e47a270a2ee78a7bbe7e8af7411ec912e47a270a2ee78"
			},
			expected: filepath.Join(testDir, "creds", "7a1ae925"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.getContextAccessToken = tt.tokenFunc

				out := do.GetCredentialDirectory(hashAccessToken(config), testDir)
				require.Equal(t, tt.expected, out)
			})
		})
	}
}

func TestPreserveCredsMovesExistingToStaging(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tmp, err := ioutil.TempDir("", "test-dir")
		require.NoError(t, err)
		defer func() {
			err := os.RemoveAll(tmp)
			require.NoError(t, err, "error cleaning tmp dir")
		}()

		// Set up "existing" creds in the "sandbox" dir
		serverlessDir := filepath.Join(tmp, "sandbox")
		sandboxCredsDir := filepath.Join(serverlessDir, "creds", "d5b388f2")
		err = os.MkdirAll(sandboxCredsDir, os.FileMode(0755))
		require.NoError(t, err)
		sandboxCreds := filepath.Join(sandboxCredsDir, "credentials.json")
		creds, err := os.Create(sandboxCreds)
		require.NoError(t, err)
		creds.Close()

		// Create staging dir
		stagingDir := filepath.Join(tmp, "staging")
		err = os.MkdirAll(stagingDir, os.FileMode(0755))
		require.NoError(t, err)

		err = do.PreserveCreds(hashAccessToken(config), stagingDir, serverlessDir)
		require.NoError(t, err)

		stagingCreds := filepath.Join(stagingDir, "creds", "d5b388f2", "credentials.json")
		_, err = os.Stat(stagingCreds)
		require.NoError(t, err, "expected creds to exist in staging dir")
	})
}

func TestPreserveCredsMovesLegacyCreds(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		// Mock token to get a stable hash (3785870f)
		config.getContextAccessToken = func() string {
			return "a7bbe7e8af7411ec912e47a270a2ee78a7bbe7e8af7411ec912e47a270a2ee78"
		}

		tmp, err := ioutil.TempDir("", "test-dir")
		require.NoError(t, err)
		defer func() {
			err := os.RemoveAll(tmp)
			require.NoError(t, err, "error cleaning tmp dir")
		}()

		// Set up "existing" legacy creds in the "sandbox" dir
		serverlessDir := filepath.Join(tmp, "sandbox")
		legacyCredsDir := filepath.Join(serverlessDir, ".nimbella")
		err = os.MkdirAll(legacyCredsDir, os.FileMode(0755))
		require.NoError(t, err)
		legacyCreds := filepath.Join(legacyCredsDir, "credentials.json")
		creds, err := os.Create(legacyCreds)
		require.NoError(t, err)
		creds.Close()

		stagingDir := filepath.Join(tmp, "staging")
		err = os.MkdirAll(stagingDir, os.FileMode(0755))
		require.NoError(t, err)

		err = do.PreserveCreds(hashAccessToken(config), stagingDir, serverlessDir)
		require.NoError(t, err)

		stagingCreds := filepath.Join(stagingDir, "creds", "3785870f", "credentials.json")
		_, err = os.Stat(stagingCreds)
		require.NoError(t, err, "expected new creds to exist in staging dir")
	})
}
