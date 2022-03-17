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
	"sort"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunctionsCommand(t *testing.T) {
	cmd := Functions()
	assert.NotNil(t, cmd)
	expected := []string{"get", "invoke", "list"}

	names := []string{}
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}

	sort.Strings(expected)
	sort.Strings(names)
	assert.Equal(t, expected, names)
}

func TestFunctionsGet(t *testing.T) {
	tests := []struct {
		name            string
		doctlArgs       string
		doctlFlags      map[string]string
		expectedNimArgs []string
	}{
		{
			name:            "no flags",
			doctlArgs:       "hello",
			expectedNimArgs: []string{"hello"},
		},
		{
			name:            "code flag",
			doctlArgs:       "hello",
			doctlFlags:      map[string]string{"code": ""},
			expectedNimArgs: []string{"hello", "--code"},
		},
		{
			name:            "url flag",
			doctlArgs:       "hello",
			doctlFlags:      map[string]string{"url": ""},
			expectedNimArgs: []string{"hello", "--url"},
		},
		{
			name:            "save flag",
			doctlArgs:       "hello",
			doctlFlags:      map[string]string{"save": ""},
			expectedNimArgs: []string{"hello", "--save"},
		},
		{
			name:            "save-as flag",
			doctlArgs:       "hello",
			doctlFlags:      map[string]string{"save-as": "/path/to/code.py"},
			expectedNimArgs: []string{"hello", "--save-as", "/path/to/code.py"},
		},
		{
			name:            "save-env flag",
			doctlArgs:       "hello",
			doctlFlags:      map[string]string{"save-env": "/path/to/code.env"},
			expectedNimArgs: []string{"hello", "--save-env", "/path/to/code.env"},
		},
		{
			name:            "save-env-json flag",
			doctlArgs:       "hello",
			doctlFlags:      map[string]string{"save-env-json": "/path/to/code.json"},
			expectedNimArgs: []string{"hello", "--save-env-json", "/path/to/code.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				fakeCmd := &exec.Cmd{
					Stdout: config.Out,
				}

				config.Args = append(config.Args, tt.doctlArgs)
				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}

				tm.sandbox.EXPECT().Cmd("action/get", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.SandboxOutput{}, nil)

				err := RunFunctionsGet(config)
				require.NoError(t, err)
			})
		})
	}
}

func TestFunctionsInvoke(t *testing.T) {
	tests := []struct {
		name            string
		doctlArgs       string
		doctlFlags      map[string]interface{}
		expectedNimArgs []string
	}{
		{
			name:            "no flags",
			doctlArgs:       "hello",
			expectedNimArgs: []string{"hello"},
		},
		{
			name:            "full flag",
			doctlArgs:       "hello",
			doctlFlags:      map[string]interface{}{"full": ""},
			expectedNimArgs: []string{"hello", "--full"},
		},
		{
			name:            "param flag",
			doctlArgs:       "hello",
			doctlFlags:      map[string]interface{}{"param": "name:world"},
			expectedNimArgs: []string{"hello", "--param", "name", "world"},
		},
		{
			name:            "param flag list",
			doctlArgs:       "hello",
			doctlFlags:      map[string]interface{}{"param": []string{"name:world", "address:everywhere"}},
			expectedNimArgs: []string{"hello", "--param", "name", "world", "--param", "address", "everywhere"},
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

				config.Args = append(config.Args, tt.doctlArgs)
				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}

				tm.sandbox.EXPECT().Cmd("action/invoke", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.SandboxOutput{
					Entity: map[string]interface{}{"body": "Hello world!"},
				}, nil)
				expectedOut := `{
  "body": "Hello world!"
}
`
				err := RunFunctionsInvoke(config)
				require.NoError(t, err)
				assert.Equal(t, expectedOut, buf.String())
			})
		})
	}
}

func TestFunctionsList(t *testing.T) {
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
			name:            "count flag",
			doctlFlags:      map[string]string{"count": ""},
			expectedNimArgs: []string{"--count"},
		},
		{
			name:            "limit flag",
			doctlFlags:      map[string]string{"limit": "1"},
			expectedNimArgs: []string{"--limit", "1"},
		},
		{
			name:            "name flag",
			doctlFlags:      map[string]string{"name": ""},
			expectedNimArgs: []string{"--name"},
		},
		{
			name:            "name-sort flag",
			doctlFlags:      map[string]string{"name-sort": ""},
			expectedNimArgs: []string{"--name-sort"},
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

				tm.sandbox.EXPECT().Cmd("action/list", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.sandbox.EXPECT().Exec(fakeCmd).Return(do.SandboxOutput{}, nil)

				err := RunFunctionsList(config)
				require.NoError(t, err)
			})
		})
	}
}
