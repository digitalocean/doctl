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
	"strings"
	"testing"
	"time"

	"github.com/apache/openwhisk-client-go/whisk"
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

// theActivations is the set of activation assumed to be present, used to mock whisk API behavior
var theActivations = []whisk.Activation{
	{
		Namespace:    "my-namespace",
		Name:         "hello1",
		Version:      "0.0.1",
		ActivationID: "activation-1",
		Start:        1664538810000,
		End:          1664538820000,
		Response: whisk.Response{
			Status:     "success",
			StatusCode: 0,
			Success:    true,
			Result: &whisk.Result{
				"body": "Hello stranger!",
			},
		},
		Logs: []string{
			"2022-09-30T11:53:50.567914279Z stdout: Hello stranger!",
		},
	},
	{
		Namespace:    "my-namespace",
		Name:         "hello2",
		Version:      "0.0.2",
		ActivationID: "activation-2",
		Start:        1664538830000,
		End:          1664538840000,
		Response: whisk.Response{
			Status:     "success",
			StatusCode: 0,
			Success:    true,
			Result: &whisk.Result{
				"body": "Hello Archie!",
			},
		},
		Logs: []string{
			"2022-09-30T11:53:50.567914279Z stdout: Hello Archie!",
		},
	},
	{
		Namespace:    "my-namespace",
		Name:         "hello3",
		Version:      "0.0.3",
		ActivationID: "activation-3",
		Start:        1664538850000,
		End:          1664538860000,
		Response: whisk.Response{
			Result: &whisk.Result{
				"error": "Missing main/no code to execute.",
			},
			Status:  "developer error",
			Success: false,
		},
	},
}

// Timestamps in the activations are converted to dates using local time so, to make this test capable of running
// in any timezone, we need to abstract things a bit.  Following the conventions in aio, the banner dates are computed
// from End and the activation record dates from Start.
var (
	timestamps  = []int64{1664538810000, 1664538820000, 1664538830000, 1664538840000, 1664538850000, 1664538860000}
	actvSymbols = []string{"%START1%", "%START2%", "%START3%"}
	actvDates   = []string{
		time.UnixMilli(timestamps[0]).Format("2006-01-02 03:04:05"),
		time.UnixMilli(timestamps[2]).Format("2006-01-02 03:04:05"),
		time.UnixMilli(timestamps[4]).Format("2006-01-02 03:04:05"),
	}
	bannerSymbols = []string{"%END1%", "%END2%", "%END3%"}
	bannerDates   = []string{
		time.UnixMilli(timestamps[1]).Format("01/02 03:04:05"),
		time.UnixMilli(timestamps[3]).Format("01/02 03:04:05"),
		time.UnixMilli(timestamps[5]).Format("01/02 03:04:05"),
	}
)

// convertDates operates on the expected output (containing symbols) to substitute actual dates
func convertDates(expected string) string {
	for i, symbol := range actvSymbols {
		expected = strings.Replace(expected, symbol, actvDates[i], 1)
	}
	for i, symbol := range bannerSymbols {
		expected = strings.Replace(expected, symbol, bannerDates[i], 1)
	}
	return expected
}

// findActivation finds the activation with a given id (in these tests, assumed to be present)
func findActivation(id string) whisk.Activation {
	for _, activation := range theActivations {
		if activation.ActivationID == id {
			return activation
		}
	}
	// Should not happen
	panic("could not find " + id)
}

func TestActivationsGet(t *testing.T) {
	tests := []struct {
		name           string
		doctlArgs      string
		doctlFlags     map[string]string
		listOptions    whisk.ActivationListOptions
		expectedOutput string
	}{
		{
			name:      "no flags with ID",
			doctlArgs: "activation-2",
			expectedOutput: `{
  "namespace": "my-namespace",
  "name": "hello2",
  "version": "0.0.2",
  "subject": "",
  "activationId": "activation-2",
  "start": 1664538830000,
  "end": 1664538840000,
  "duration": 0,
  "statusCode": 0,
  "response": {
    "status": "success",
    "statusCode": 0,
    "success": true,
    "result": {
      "body": "Hello Archie!"
    }
  },
  "logs": [
    "2022-09-30T11:53:50.567914279Z stdout: Hello Archie!"
  ],
  "annotations": null,
  "date": "%START2%"
}
`,
		},
		{
			name:        "no flags or args",
			listOptions: whisk.ActivationListOptions{Limit: 1},
			expectedOutput: `{
  "namespace": "my-namespace",
  "name": "hello1",
  "version": "0.0.1",
  "subject": "",
  "activationId": "activation-1",
  "start": 1664538810000,
  "end": 1664538820000,
  "duration": 0,
  "statusCode": 0,
  "response": {
    "status": "success",
    "statusCode": 0,
    "success": true,
    "result": {
      "body": "Hello stranger!"
    }
  },
  "logs": [
    "2022-09-30T11:53:50.567914279Z stdout: Hello stranger!"
  ],
  "annotations": null,
  "date": "%START1%"
}
`,
		},
		{
			name:        "logs flag",
			doctlFlags:  map[string]string{"logs": ""},
			listOptions: whisk.ActivationListOptions{Limit: 1},
			expectedOutput: `=== activation-1 success %END1% hello1:0.0.1
Hello stranger!
`,
		},
		{
			name:        "result flag",
			doctlFlags:  map[string]string{"result": ""},
			listOptions: whisk.ActivationListOptions{Limit: 1},
			expectedOutput: `=== activation-1 success %END1% hello1:0.0.1
{
  "body": "Hello stranger!"
}
`,
		},
		{
			name:        "skip flag",
			doctlFlags:  map[string]string{"skip": "2"},
			listOptions: whisk.ActivationListOptions{Limit: 1, Skip: 2},
			expectedOutput: `{
  "namespace": "my-namespace",
  "name": "hello3",
  "version": "0.0.3",
  "subject": "",
  "activationId": "activation-3",
  "start": 1664538850000,
  "end": 1664538860000,
  "duration": 0,
  "statusCode": 0,
  "response": {
    "status": "developer error",
    "statusCode": 0,
    "success": false,
    "result": {
      "error": "Missing main/no code to execute."
    }
  },
  "logs": null,
  "annotations": null,
  "date": "%START3%"
}
`,
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

				logs := false
				result := false
				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if k == "logs" {
							logs = true
						}
						if k == "result" {
							result = true
						}
						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}

				id := tt.doctlArgs
				var activation whisk.Activation
				if id != "" {
					activation = findActivation(id)
				}
				if tt.listOptions.Limit > 0 {
					fst := tt.listOptions.Skip
					lnth := tt.listOptions.Limit + fst
					tm.serverless.EXPECT().ListActivations(tt.listOptions).Return(theActivations[fst:lnth], nil)
					activation = theActivations[fst]
					id = activation.ActivationID
				}
				if logs {
					tm.serverless.EXPECT().GetActivationLogs(id).Return(activation, nil)
				} else if result {
					tm.serverless.EXPECT().GetActivationResult(id).Return(activation.Response, nil)
				} else {
					tm.serverless.EXPECT().GetActivation(id).Return(activation, nil)
				}

				err := RunActivationsGet(config)
				require.NoError(t, err)
				assert.Equal(t, convertDates(tt.expectedOutput), buf.String())
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

				tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
				tm.serverless.EXPECT().Cmd("activation/list", tt.expectedNimArgs).Return(fakeCmd, nil)
				tm.serverless.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{}, nil)

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

				tm.serverless.EXPECT().CheckServerlessStatus(hashAccessToken(config)).MinTimes(1).Return(nil)
				if tt.expectStream {
					expectedArgs := append([]string{"activation/logs"}, tt.expectedNimArgs...)
					tm.serverless.EXPECT().Cmd("nocapture", expectedArgs).Return(fakeCmd, nil)
					tm.serverless.EXPECT().Stream(fakeCmd).Return(nil)
				} else {
					tm.serverless.EXPECT().Cmd("activation/logs", tt.expectedNimArgs).Return(fakeCmd, nil)
					tm.serverless.EXPECT().Exec(fakeCmd).Return(do.ServerlessOutput{}, nil)
				}

				err := RunActivationsLogs(config)
				require.NoError(t, err)
			})
		})
	}
}

func TestActivationsResult(t *testing.T) {
	tests := []struct {
		name           string
		doctlArgs      string
		doctlFlags     map[string]string
		listOptions    whisk.ActivationListOptions
		expectedOutput string
	}{
		{
			name:        "no flags or args",
			listOptions: whisk.ActivationListOptions{Limit: 1},
			expectedOutput: `=== activation-1 success %END1% hello1:0.0.1
{
  "body": "Hello stranger!"
}
`,
		},
		{
			name:      "no flags with ID",
			doctlArgs: "activation-2",
			expectedOutput: `{
  "body": "Hello Archie!"
}
`,
		},
		{
			name:        "limit flag",
			doctlFlags:  map[string]string{"limit": "10"},
			listOptions: whisk.ActivationListOptions{Limit: 10},
			expectedOutput: `=== activation-3 success %END3% hello3:0.0.3
{
  "error": "Missing main/no code to execute."
}
=== activation-2 success %END2% hello2:0.0.2
{
  "body": "Hello Archie!"
}
=== activation-1 success %END1% hello1:0.0.1
{
  "body": "Hello stranger!"
}
`,
		},
		{
			name:        "quiet flag",
			doctlFlags:  map[string]string{"quiet": ""},
			listOptions: whisk.ActivationListOptions{Limit: 1},
			expectedOutput: `{
  "body": "Hello stranger!"
}
`,
		},
		{
			name:        "skip flag",
			doctlFlags:  map[string]string{"skip": "1"},
			listOptions: whisk.ActivationListOptions{Limit: 1, Skip: 1},
			expectedOutput: `=== activation-2 success %END2% hello2:0.0.2
{
  "body": "Hello Archie!"
}
`,
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

				var ids []string
				var activations []whisk.Activation
				if tt.doctlArgs != "" {
					ids = []string{tt.doctlArgs}
					activations = []whisk.Activation{findActivation(ids[0])}
				}
				limit := tt.listOptions.Limit
				if limit > 0 {
					if limit > len(theActivations) {
						limit = len(theActivations)
					}
					fst := tt.listOptions.Skip
					lnth := limit + fst
					// The command reverses the returned list in asking for the responses
					chosen := theActivations[fst:lnth]
					ids = make([]string, len(chosen))
					activations = make([]whisk.Activation, len(chosen))
					for i, activation := range chosen {
						activations[len(chosen)-i-1] = activation
						ids[len(chosen)-i-1] = activation.ActivationID
					}
					tm.serverless.EXPECT().ListActivations(tt.listOptions).Return(chosen, nil)
				}
				for i, id := range ids {
					tm.serverless.EXPECT().GetActivationResult(id).Return(activations[i].Response, nil)
				}
				err := RunActivationsResult(config)
				require.NoError(t, err)
				assert.Equal(t, convertDates(tt.expectedOutput), buf.String())
			})
		})
	}
}
