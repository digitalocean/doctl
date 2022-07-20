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
	"os/exec"
	"sort"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivationsCommand(t *testing.T) {
	cmd := Activations()
	assert.NotNil(t, cmd)
	expected := []string{"get", "list", "logs", "result"}

	names := []string{}
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}

	sort.Strings(expected)
	sort.Strings(names)
	assert.Equal(t, expected, names)
}

func TestActivationsGet(t *testing.T) {
	tests := []struct {
		name            string
		doctlArgs       string
		doctlFlags      map[string]string
		expectedNimArgs []string
	}{
		{
			name:            "no flags with ID",
			doctlArgs:       "activationid",
			expectedNimArgs: []string{"activationid"},
		},
		{
			name:            "no flags or args",
			expectedNimArgs: []string{},
		},
		{
			name:            "last flag",
			doctlArgs:       "activationid",
			doctlFlags:      map[string]string{"last": ""},
			expectedNimArgs: []string{"activationid", "--last"},
		},
		{
			name:            "logs flag",
			doctlArgs:       "activationid",
			doctlFlags:      map[string]string{"logs": ""},
			expectedNimArgs: []string{"activationid", "--logs"},
		},
		{
			name:            "skip flag",
			doctlArgs:       "activationid",
			doctlFlags:      map[string]string{"skip": "10"},
			expectedNimArgs: []string{"activationid", "--skip", "10"},
		},
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

				tm.sandbox.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
				tm.sandbox.EXPECT().Cmd("activation/get", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{}, nil)

				err := RunActivationsGet(config)
				require.NoError(t, err)
			})
		})
	}
}

func TestActivationsList(t *testing.T) {
	tests := []struct {
		name            string
		doctlArgs       string
		doctlFlags      map[string]string
		expectedNimArgs []string
	}{
		{
			name:            "no flags or args",
			expectedNimArgs: []string{},
		},
		{
			name:            "full flag",
			doctlFlags:      map[string]string{"full": ""},
			expectedNimArgs: []string{"--full"},
		},
		{
			name:            "count flag",
			doctlFlags:      map[string]string{"count": ""},
			expectedNimArgs: []string{"--count"},
		},
		{
			name:            "limit flag",
			doctlFlags:      map[string]string{"limit": "10"},
			expectedNimArgs: []string{"--limit", "10"},
		},
		{
			name:            "since flag",
			doctlFlags:      map[string]string{"since": "1644866670085"},
			expectedNimArgs: []string{"--since", "1644866670085"},
		},
		{
			name:            "skip flag",
			doctlFlags:      map[string]string{"skip": "1"},
			expectedNimArgs: []string{"--skip", "1"},
		},
		{
			name:            "upto flag",
			doctlFlags:      map[string]string{"upto": "1644866670085"},
			expectedNimArgs: []string{"--upto", "1644866670085"},
		},
		{
			name:            "multiple flags",
			doctlFlags:      map[string]string{"limit": "10", "count": ""},
			expectedNimArgs: []string{"--count", "--limit", "10"},
		},
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

				tm.sandbox.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
				tm.sandbox.EXPECT().Cmd("activation/list", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{}, nil)

				err := RunActivationsList(config)
				require.NoError(t, err)
			})
		})
	}
}

func TestActivationsLogs(t *testing.T) {
	tests := []struct {
		name            string
		doctlArgs       string
		doctlFlags      map[string]string
		expectedNimArgs []string
		expectStream    bool
	}{
		{
			name:            "no flags or args",
			expectedNimArgs: []string{},
		},
		{
			name:            "no flags with ID",
			doctlArgs:       "activationid",
			expectedNimArgs: []string{"activationid"},
		},
		{
			name:            "last flag",
			doctlFlags:      map[string]string{"last": ""},
			expectedNimArgs: []string{"--last"},
		},
		{
			name:            "limit flag",
			doctlFlags:      map[string]string{"limit": "10"},
			expectedNimArgs: []string{"--limit", "10"},
		},
		{
			name:            "function flag",
			doctlFlags:      map[string]string{"function": "sample"},
			expectedNimArgs: []string{"--action", "sample"},
		},
		{
			name:            "package flag",
			doctlFlags:      map[string]string{"package": "sample"},
			expectedNimArgs: []string{"--deployed", "--package", "sample"},
		},
		{
			name:            "follow flag",
			doctlFlags:      map[string]string{"follow": ""},
			expectedNimArgs: []string{"--watch"},
			expectStream:    true,
		},
		{
			name:            "strip flag",
			doctlFlags:      map[string]string{"strip": ""},
			expectedNimArgs: []string{"--strip"},
		},
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

				tm.sandbox.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
				if tt.expectStream {
					expectedArgs := append([]string{"activation/logs"}, tt.expectedNimArgs...)
					tm.sandbox.EXPECT().Cmd("nocapture", expectedArgs).Return(fakeCmd, nil)
					tm.sandbox.EXPECT().Stream(fakeCmd).Return(nil)
				} else {
					tm.sandbox.EXPECT().Cmd("activation/logs", tt.expectedNimArgs).Return(fakeCmd, nil)
					tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{}, nil)
				}

				err := RunActivationsLogs(config)
				require.NoError(t, err)
			})
		})
	}
}

func TestActivationsResult(t *testing.T) {
	tests := []struct {
		name            string
		doctlArgs       string
		doctlFlags      map[string]string
		expectedNimArgs []string
	}{
		{
			name:            "no flags or args",
			expectedNimArgs: []string{},
		},
		{
			name:            "no flags with ID",
			doctlArgs:       "activationid",
			expectedNimArgs: []string{"activationid"},
		},
		{
			name:            "last flag",
			doctlFlags:      map[string]string{"last": ""},
			expectedNimArgs: []string{"--last"},
		},
		{
			name:            "limit flag",
			doctlFlags:      map[string]string{"limit": "10"},
			expectedNimArgs: []string{"--limit", "10"},
		},
		{
			name:            "quiet flag",
			doctlFlags:      map[string]string{"quiet": ""},
			expectedNimArgs: []string{"--quiet"},
		},
		{
			name:            "skip flag",
			doctlFlags:      map[string]string{"skip": "1"},
			expectedNimArgs: []string{"--skip", "1"},
		},
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

				tm.sandbox.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
				tm.sandbox.EXPECT().Cmd("activation/result", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{}, nil)

				err := RunActivationsResult(config)
				require.NoError(t, err)
			})
		})
	}
}
