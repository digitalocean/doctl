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
	"sort"
	"strings"
	"testing"
	"time"

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
	// The displayer for function list is time-zone sensitive so we need to pre-convert the timestamps using the local
	// time-zone to get exact matches.
	timestamps := []int64{1664538810000, 1664538820000, 1664538830000}
	symbols := []string{"%DATE1%", "%DATE2%", "%DATE3%"}
	dates := []string{
		time.UnixMilli(timestamps[0]).Format("01/02 03:04:05"),
		time.UnixMilli(timestamps[1]).Format("01/02 03:04:05"),
		time.UnixMilli(timestamps[2]).Format("01/02 03:04:05"),
	}

	tests := []struct {
		name           string
		doctlFlags     map[string]string
		doctlArg       string
		skip           int
		limit          int
		expectedOutput string
	}{
		{
			name:  "no flags or args",
			skip:  0,
			limit: 0,
			expectedOutput: `%DATE1%    0.0.1    nodejs:14    daily/hello
%DATE2%    0.0.2    nodejs:14    daily/goodbye
%DATE3%    0.0.3    nodejs:14    sometimes/meAgain
`,
		},
		{
			name:     "with package arg",
			doctlArg: "daily",
			skip:     0,
			limit:    0,
			expectedOutput: `%DATE1%    0.0.1    nodejs:14    daily/hello
%DATE2%    0.0.2    nodejs:14    daily/goodbye
`,
		},
		{
			name:           "count flag",
			doctlFlags:     map[string]string{"count": ""},
			skip:           0,
			limit:          0,
			expectedOutput: "There are 3 functions in this namespace.\n",
		},
		{
			name:           "limit flag",
			doctlFlags:     map[string]string{"limit": "1"},
			skip:           0,
			limit:          1,
			expectedOutput: "%DATE1%    0.0.1    nodejs:14    daily/hello\n",
		},
		{
			name:       "name flag",
			doctlFlags: map[string]string{"name": ""},
			skip:       0,
			limit:      0,
			expectedOutput: `%DATE2%    0.0.2    nodejs:14    daily/goodbye
%DATE1%    0.0.1    nodejs:14    daily/hello
%DATE3%    0.0.3    nodejs:14    sometimes/meAgain
`,
		},
		{
			name:       "name-sort flag",
			doctlFlags: map[string]string{"name-sort": ""},
			skip:       0,
			limit:      0,
			expectedOutput: `%DATE2%    0.0.2    nodejs:14    daily/goodbye
%DATE1%    0.0.1    nodejs:14    daily/hello
%DATE3%    0.0.3    nodejs:14    sometimes/meAgain
`,
		},
		{
			name:       "skip flag",
			doctlFlags: map[string]string{"skip": "1"},
			skip:       1,
			limit:      0,
			expectedOutput: `%DATE2%    0.0.2    nodejs:14    daily/goodbye
%DATE3%    0.0.3    nodejs:14    sometimes/meAgain
`,
		},
	}

	theList := []whisk.Action{
		{
			Name:      "hello",
			Namespace: "theNamespace/daily",
			Updated:   timestamps[0],
			Version:   "0.0.1",
			Annotations: whisk.KeyValueArr{
				whisk.KeyValue{
					Key:   "exec",
					Value: "nodejs:14",
				},
			},
		},
		{
			Name:      "goodbye",
			Namespace: "theNamespace/daily",
			Updated:   timestamps[1],
			Version:   "0.0.2",
			Annotations: whisk.KeyValueArr{
				whisk.KeyValue{
					Key:   "exec",
					Value: "nodejs:14",
				},
			},
		},
		{
			Name:      "meAgain",
			Namespace: "theNamespace/sometimes",
			Version:   "0.0.3",
			Updated:   timestamps[2],
			Annotations: whisk.KeyValueArr{
				whisk.KeyValue{
					Key:   "exec",
					Value: "nodejs:14",
				},
			},
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
				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}
				config.Doit.Set(config.NS, "no-header", true)

				answer := selectPackage(theList, tt.doctlArg)[tt.skip:]
				if tt.limit != 0 {
					answer = answer[0:tt.limit]
				}
				tm.serverless.EXPECT().ListFunctions(tt.doctlArg, tt.skip, tt.limit).Return(answer, nil)

				err := RunFunctionsList(config)
				require.NoError(t, err)
				expected := tt.expectedOutput
				for i := range symbols {
					expected = strings.Replace(expected, symbols[i], dates[i], 1)
				}
				assert.Equal(t, expected, buf.String())
			})
		})
	}
}

// selectPackage is a testing support utility to trim a master list of functions by package membership
// Also ensures the array is copied, because the logic being tested may sort it in place.
func selectPackage(masterList []whisk.Action, pkg string) []whisk.Action {
	if pkg == "" {
		copiedList := make([]whisk.Action, len(masterList))
		copy(copiedList, masterList)
		return copiedList
	}
	namespace := "theNamespace/" + pkg
	answer := []whisk.Action{}
	for _, action := range masterList {
		if action.Namespace == namespace {
			answer = append(answer, action)
		}
	}
	return answer
}
