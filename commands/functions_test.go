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
	"os"
	"os/exec"
	"sort"
	"testing"

	"github.com/apache/openwhisk-client-go/whisk"
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
		name           string
		doctlArgs      string
		doctlFlags     map[string]string
		fetchCode      bool
		expectAPIHost  bool
		expectSaved    string
		expectOutput   string
		expectPlainEnv string
		expectJSONEnv  string
	}{
		{
			name:         "no flags",
			doctlArgs:    "hello",
			expectOutput: "{\n  \"namespace\": \"thenamespace\",\n  \"name\": \"hello\",\n  \"exec\": {\n    \"kind\": \"nodejs:14\",\n    \"code\": \"code of the function\",\n    \"binary\": false\n  },\n  \"annotations\": [\n    {\n      \"key\": \"web-export\",\n      \"value\": true\n    }\n  ]\n}\n",
		},
		{
			name:         "code flag",
			doctlArgs:    "hello",
			doctlFlags:   map[string]string{"code": ""},
			fetchCode:    true,
			expectOutput: "code of the function\n",
		},
		{
			name:          "url flag",
			doctlArgs:     "hello",
			doctlFlags:    map[string]string{"url": ""},
			expectAPIHost: true,
			expectOutput:  "https://example.com/api/v1/web/thenamespace/default/hello\n",
		},
		{
			name:          "save flag",
			doctlArgs:     "hello",
			doctlFlags:    map[string]string{"save": ""},
			fetchCode:     true,
			expectAPIHost: false,
			expectSaved:   "hello.js",
		},
		{
			name:        "save-as flag",
			doctlArgs:   "hello",
			doctlFlags:  map[string]string{"save-as": "savedcode"},
			fetchCode:   true,
			expectSaved: "savedcode",
		},
		{
			name:           "save-env flag",
			doctlArgs:      "hello",
			doctlFlags:     map[string]string{"save-env": "code.env"},
			expectPlainEnv: "code.env",
		},
		{
			name:          "save-env-json flag",
			doctlArgs:     "hello",
			doctlFlags:    map[string]string{"save-env-json": "code.json"},
			expectJSONEnv: "code.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				code := "code of the function"
				binaryFalse := false
				actionResponse := whisk.Action{
					Exec: &whisk.Exec{
						Code:   &code,
						Binary: &binaryFalse,
						Kind:   "nodejs:14",
					},
					Annotations: whisk.KeyValueArr{
						whisk.KeyValue{
							Key:   "web-export",
							Value: true,
						},
					},
					Name:      "hello",
					Namespace: "thenamespace",
				}
				param := do.FunctionParameter{Init: true, Key: "foo", Value: "bar"}
				paramResponse := []do.FunctionParameter{param}
				buf := &bytes.Buffer{}
				config.Out = buf
				plainEnv := "foo=bar"
				jsonEnv := `{
  "foo": "bar"
}`

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

				tm.serverless.EXPECT().GetFunction("hello", tt.fetchCode).Return(actionResponse, paramResponse, nil)
				if tt.expectAPIHost {
					tm.serverless.EXPECT().GetConnectedAPIHost().Return("https://example.com", nil)
				}

				savedFiles := []string{tt.expectSaved, tt.expectPlainEnv, tt.expectJSONEnv}
				defer func() {
					for _, file := range savedFiles {
						if file != "" {
							os.Remove(file)
						}
					}
				}()

				err := RunFunctionsGet(config)
				require.NoError(t, err)
				assert.Equal(t, tt.expectOutput, buf.String())
				expectedContents := []string{code, plainEnv, jsonEnv}
				for i, file := range savedFiles {
					if file != "" {
						contents, err := os.ReadFile(file)
						require.NoError(t, err)
						assert.Equal(t, string(contents), expectedContents[i])
					}
				}
			})
		})
	}
}

func TestFunctionsInvoke(t *testing.T) {
	tests := []struct {
		name          string
		doctlArgs     string
		doctlFlags    map[string]interface{}
		requestResult bool
		passedParams  interface{}
	}{
		{
			name:          "no flags",
			doctlArgs:     "hello",
			requestResult: true,
			passedParams:  nil,
		},
		{
			name:          "full flag",
			doctlArgs:     "hello",
			doctlFlags:    map[string]interface{}{"full": ""},
			requestResult: false,
			passedParams:  nil,
		},
		{
			name:          "param flag",
			doctlArgs:     "hello",
			doctlFlags:    map[string]interface{}{"param": "name:world"},
			requestResult: true,
			passedParams:  map[string]interface{}{"name": "world"},
		},
		{
			name:          "param flag list",
			doctlArgs:     "hello",
			doctlFlags:    map[string]interface{}{"param": []string{"name:world", "address:everywhere"}},
			requestResult: true,
			passedParams:  map[string]interface{}{"name": "world", "address": "everywhere"},
		},
		{
			name:          "param flag colon-value",
			doctlArgs:     "hello",
			doctlFlags:    map[string]interface{}{"param": []string{"url:https://example.com"}},
			requestResult: true,
			passedParams:  map[string]interface{}{"url": "https://example.com"},
		},
	}

	expectedRemoteResult := map[string]interface{}{
		"body": "Hello world!",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				buf := &bytes.Buffer{}
				config.Out = buf

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

				tm.serverless.EXPECT().InvokeFunction(tt.doctlArgs, tt.passedParams, true, tt.requestResult).Return(expectedRemoteResult, nil)
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
			expectedNimArgs: []string{"--json"},
		},
		{
			name:            "count flag",
			doctlFlags:      map[string]string{"count": ""},
			expectedNimArgs: []string{"--count"},
		},
		{
			name:            "limit flag",
			doctlFlags:      map[string]string{"limit": "1"},
			expectedNimArgs: []string{"--json", "--limit", "1"},
		},
		{
			name:            "name flag",
			doctlFlags:      map[string]string{"name": ""},
			expectedNimArgs: []string{"--name", "--json"},
		},
		{
			name:            "name-sort flag",
			doctlFlags:      map[string]string{"name-sort": ""},
			expectedNimArgs: []string{"--name-sort", "--json"},
		},
		{
			name:            "skip flag",
			doctlFlags:      map[string]string{"skip": "1"},
			expectedNimArgs: []string{"--json", "--skip", "1"},
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

				tm.serverless.EXPECT().CheckServerlessStatus().MinTimes(1).Return(nil)
				tm.serverless.EXPECT().Cmd("action/list", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.serverless.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{}, nil)

				err := RunFunctionsList(config)
				require.NoError(t, err)
			})
		})
	}
}
