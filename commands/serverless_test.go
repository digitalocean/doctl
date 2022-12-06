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
	"bufio"
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerlessConnect(t *testing.T) {
	tests := []struct {
		name           string
		namespaceList  []do.OutputNamespace
		expectedOutput string
		expectedError  error
		doctlArg       string
	}{
		{
			name:          "no namespaces",
			namespaceList: []do.OutputNamespace{},
			expectedError: errors.New("you must create a namespace with `doctl namespace create`, specifying a region and label"),
		},
		{
			name: "one namespace",
			namespaceList: []do.OutputNamespace{
				{
					Namespace: "ns1",
					Region:    "nyc1",
					Label:     "something",
				},
			},
			expectedOutput: "Connected to functions namespace 'ns1' on API host 'https://api.example.com' (label=something)\n\n",
		},
		{
			name: "two namespaces",
			namespaceList: []do.OutputNamespace{
				{
					Namespace: "ns1",
					Region:    "nyc1",
					Label:     "something",
				},
				{
					Namespace: "ns2",
					Region:    "lon1",
					Label:     "another",
				},
			},
			expectedOutput: "0: ns1 in nyc1, label=something\n1: ns2 in lon1, label=another\nChoose a namespace by number or 'x' to exit\nConnected to functions namespace 'ns1' on API host 'https://api.example.com' (label=something)\n\n",
		},
		{
			name: "use argument",
			namespaceList: []do.OutputNamespace{
				{
					Namespace: "ns1",
					Region:    "nyc1",
					Label:     "something",
				},
				{
					Namespace: "ns2",
					Region:    "lon1",
					Label:     "another",
				},
			},
			doctlArg:       "thing",
			expectedOutput: "Connected to functions namespace 'ns1' on API host 'https://api.example.com' (label=something)\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				buf := &bytes.Buffer{}
				config.Out = buf
				if tt.doctlArg != "" {
					config.Args = append(config.Args, tt.doctlArg)
				}
				connectChoiceReader = bufio.NewReader(strings.NewReader("0\n"))
				nsResponse := do.NamespaceListResponse{Namespaces: tt.namespaceList}
				creds := do.ServerlessCredentials{Namespace: "ns1", APIHost: "https://api.example.com", Label: "something"}

				tm.serverless.EXPECT().CheckServerlessStatus().Return(do.ErrServerlessNotConnected)
				ctx := context.TODO()
				tm.serverless.EXPECT().ListNamespaces(ctx).Return(nsResponse, nil)
				if tt.expectedError == nil {
					tm.serverless.EXPECT().GetNamespace(ctx, "ns1").Return(creds, nil)
					tm.serverless.EXPECT().WriteCredentials(creds).Return(nil)
				}

				err := RunServerlessConnect(config)
				if tt.expectedError != nil {
					assert.Equal(t, tt.expectedError, err)
				} else {
					require.NoError(t, err)
				}
				if tt.expectedOutput != "" {
					assert.Equal(t, tt.expectedOutput, buf.String())
				}

			})
		})
	}
}

func TestServerlessStatusWhenConnected(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		tm.serverless.EXPECT().CheckServerlessStatus().MinTimes(1).Return(nil)
		tm.serverless.EXPECT().ReadCredentials().Return(do.ServerlessCredentials{
			APIHost:   "https://api.example.com",
			Namespace: "hello",
			Credentials: map[string]map[string]do.ServerlessCredential{
				"https://api.example.com": {
					"hello": do.ServerlessCredential{
						Auth: "here-are-some-credentials",
					},
				},
			},
		}, nil)
		tm.serverless.EXPECT().GetNamespaceFromCluster("https://api.example.com", "here-are-some-credentials").Return("hello", nil)

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

		tm.serverless.EXPECT().CheckServerlessStatus().MinTimes(1).Return(nil)
		tm.serverless.EXPECT().ReadCredentials().Return(do.ServerlessCredentials{
			APIHost:   "https://api.example.com",
			Namespace: "hello",
			Credentials: map[string]map[string]do.ServerlessCredential{
				"https://api.example.com": {
					"hello": do.ServerlessCredential{
						Auth: "here-are-some-credentials",
					},
				},
			},
		}, nil)
		tm.serverless.EXPECT().GetNamespaceFromCluster("https://api.example.com", "here-are-some-credentials").Return("hello", nil)
		tm.serverless.EXPECT().GetHostInfo("https://api.example.com").Return(fakeHostInfo, nil)

		err := RunServerlessStatus(config)
		require.NoError(t, err)
		assert.Contains(t, buf.String(), expectedDisplay)
	})
}

func TestServerlessStatusWhenNotConnected(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {

		tm.serverless.EXPECT().CheckServerlessStatus().MinTimes(1).Return(nil)
		tm.serverless.EXPECT().ReadCredentials().Return(do.ServerlessCredentials{
			APIHost:   "https://api.example.com",
			Namespace: "hello",
			Credentials: map[string]map[string]do.ServerlessCredential{
				"https://api.example.com": {
					"hello": do.ServerlessCredential{
						Auth: "here-are-some-credentials",
					},
				},
			},
		}, nil)
		tm.serverless.EXPECT().GetNamespaceFromCluster("https://api.example.com", "here-are-some-credentials").Return("not-hello", errors.New("an error"))

		err := RunServerlessStatus(config)
		require.Error(t, err)
		assert.ErrorIs(t, err, do.ErrServerlessNotConnected)
	})
}

func TestServerlessStatusWhenNotInstalled(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.serverless.EXPECT().CheckServerlessStatus().Return(do.ErrServerlessNotInstalled)

		err := RunServerlessStatus(config)

		require.Error(t, err)
		assert.ErrorIs(t, err, do.ErrServerlessNotInstalled)
	})
}

func TestServerlessStatusWhenNotUpToDate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.serverless.EXPECT().CheckServerlessStatus().Return(do.ErrServerlessNeedsUpgrade)

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
		tm.serverless.EXPECT().CheckServerlessStatus().Return(do.ErrServerlessNotInstalled)
		tm.serverless.EXPECT().InstallServerless(credsToken, false).Return(nil)

		err := RunServerlessInstall(config)
		require.NoError(t, err)
	})
}

func TestServerlessInstallWhenInstalledNotCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		tm.serverless.EXPECT().CheckServerlessStatus().Return(do.ErrServerlessNeedsUpgrade)

		err := RunServerlessInstall(config)
		require.NoError(t, err)
		assert.Equal(t, "Serverless support is already installed, but needs an upgrade for this version of `doctl`.\nUse `doctl serverless upgrade` to upgrade the support.\n", buf.String())
	})
}

func TestServerlessInstallWhenInstalledAndCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)

		err := RunServerlessInstall(config)
		require.NoError(t, err)
		assert.Equal(t, "Serverless support is already installed at an appropriate version.  No action needed.\n", buf.String())
	})
}

func TestServerlessUpgradeWhenNotInstalled(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		tm.serverless.EXPECT().CheckServerlessStatus().Return(do.ErrServerlessNotInstalled)

		err := RunServerlessUpgrade(config)
		require.NoError(t, err)
		assert.Equal(t, "Serverless support was never installed.  Use `doctl serverless install`.\n", buf.String())
	})
}

func TestServerlessUpgradeWhenInstalledAndCurrent(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)

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
		tm.serverless.EXPECT().CheckServerlessStatus().Return(do.ErrServerlessNeedsUpgrade)
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
		out             map[string]interface{}
		expectCheck     bool
		expectOverwrite bool
	}{
		{
			name:      "no flags",
			doctlArgs: "path/to/foo",
			// The language flag has a default normally applied by cobra
			doctlFlags: map[string]string{"language": "javascript"},
			out:        map[string]interface{}{"project": "foo"},
		},
		{
			name:      "overwrite",
			doctlArgs: "path/to/foo",
			// The language flag has a default normally applied by cobra
			doctlFlags:      map[string]string{"overwrite": "", "language": "javascript"},
			out:             map[string]interface{}{"project": "foo"},
			expectOverwrite: true,
		},
		{
			name:        "language flag",
			doctlArgs:   "path/to/foo",
			doctlFlags:  map[string]string{"language": "go"},
			out:         map[string]interface{}{"project": "foo"},
			expectCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				buf := &bytes.Buffer{}
				config.Out = buf

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

				sawOverwrite := false
				// Grab the overrideable commands so they can be mocked
				writeAFile = func(path string, contents []byte) error {
					return nil
				}
				doMkdir = func(path string, parents bool) error {
					return nil
				}
				prepareProjectArea = func(project string, overwrite bool) error {
					sawOverwrite = overwrite
					return nil
				}

				if tt.expectCheck {
					tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)
					creds := do.ServerlessCredentials{APIHost: "https://example.com"}
					tm.serverless.EXPECT().ReadCredentials().Return(creds, nil)
					hostInfo := do.ServerlessHostInfo{
						Runtimes: map[string][]do.ServerlessRuntime{
							"go": []do.ServerlessRuntime{},
						},
					}
					tm.serverless.EXPECT().GetHostInfo("https://example.com").Return(hostInfo, nil)
				}
				err := RunServerlessExtraCreate(config)
				require.NoError(t, err)
				assert.Equal(t, tt.expectOverwrite, sawOverwrite)
				assert.Equal(t, `A local functions project directory 'path/to/foo' was created for you.
You may deploy it by running the command shown on the next line:
  doctl serverless deploy path/to/foo`+"\n", buf.String())
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

				tm.serverless.EXPECT().CheckServerlessStatus().MinTimes(1).Return(nil)
				tm.serverless.EXPECT().Cmd("deploy", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.serverless.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{}, nil)

				err := RunServerlessExtraDeploy(config)
				require.NoError(t, err)
			})
		})
	}
}

func TestServerlessUndeploy(t *testing.T) {
	tests := []struct {
		name          string
		doctlArgs     []string
		doctlFlags    map[string]string
		expectedError error
	}{
		{
			name:          "no arguments or flags",
			doctlArgs:     nil,
			doctlFlags:    nil,
			expectedError: errUndeployTooFewArgs,
		},
		{
			name:          "with --all flag only",
			doctlArgs:     nil,
			doctlFlags:    map[string]string{"all": ""},
			expectedError: nil,
		},
		{
			name:          "mixed args, no flags",
			doctlArgs:     []string{"foo/bar", "baz"},
			doctlFlags:    nil,
			expectedError: nil,
		},
		{
			name:          "mixed args, --packages flag",
			doctlArgs:     []string{"foo/bar", "baz"},
			doctlFlags:    map[string]string{"packages": ""},
			expectedError: nil,
		},
		{
			name:          "mixed args, --all flag",
			doctlArgs:     []string{"foo/bar", "baz"},
			doctlFlags:    map[string]string{"all": ""},
			expectedError: errUndeployAllAndArgs,
		},
		{
			name:          "--triggers and --packages",
			doctlArgs:     []string{"foo/bar", "baz"},
			doctlFlags:    map[string]string{"triggers": "", "packages": ""},
			expectedError: errUndeployTrigPkg,
		},
		{
			name:          "--triggers and args",
			doctlArgs:     []string{"fire1", "fire2"},
			doctlFlags:    map[string]string{"triggers": ""},
			expectedError: nil,
		},
		{
			name:          "--triggers and --all",
			doctlArgs:     nil,
			doctlFlags:    map[string]string{"triggers": "", "all": ""},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				cannedTriggerList := []do.ServerlessTrigger{
					{Name: "fireA"},
					{Name: "fireB"},
				}

				if len(tt.doctlArgs) > 0 {
					config.Args = append(config.Args, tt.doctlArgs...)
				}

				var pkg bool = false
				var trig bool = false
				var all bool = false

				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if k == "all" {
							all = true
						}

						if k == "packages" {
							pkg = true
						}

						if k == "triggers" {
							trig = true
						}

						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}

				if all && !trig && !pkg && len(config.Args) == 0 {
					tm.serverless.EXPECT().CleanNamespace().Return(nil)
				} else if all && trig && len(config.Args) == 0 {
					tm.serverless.EXPECT().ListTriggers(context.TODO(), "").Return(cannedTriggerList, nil)
					for _, st := range cannedTriggerList {
						tm.serverless.EXPECT().DeleteTrigger(context.TODO(), st.Name)
					}

				} else if !all && !trig && len(config.Args) > 0 {
					for _, arg := range config.Args {
						if !pkg || strings.Contains(arg, "/") {
							tm.serverless.EXPECT().DeleteFunction(arg, true).Return(nil)
						} else {
							tm.serverless.EXPECT().DeletePackage(arg, true).Return(nil)
						}
					}
				} else if !all && trig && !pkg && len(config.Args) > 0 {
					for _, t := range config.Args {
						tm.serverless.EXPECT().DeleteTrigger(context.TODO(), t)
					}
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
			expectedNimArgs: []string{"watch", "path/to/project", "--exclude", "web"},
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

				tm.serverless.EXPECT().CheckServerlessStatus().MinTimes(1).Return(nil)
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
		serverlessCredsDir := filepath.Join(serverlessDir, "creds", "d5b388f2")
		err = os.MkdirAll(serverlessCredsDir, os.FileMode(0755))
		require.NoError(t, err)
		serverlessCreds := filepath.Join(serverlessCredsDir, "credentials.json")
		creds, err := os.Create(serverlessCreds)
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
