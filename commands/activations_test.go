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
			"2022-09-30T11:53:50.567914279Z stdout: Hello stranger!",
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
		name        string
		doctlArgs   string
		doctlFlags  map[string]string
		listOptions whisk.ActivationListOptions
	}{
		{
			name:      "no flags with ID",
			doctlArgs: "activation-2",
		},
		{
			name:        "no flags or args",
			listOptions: whisk.ActivationListOptions{Limit: 1},
		},
		{
			name:        "logs flag",
			doctlFlags:  map[string]string{"logs": ""},
			listOptions: whisk.ActivationListOptions{Limit: 1},
		},
		{
			name:        "result flag",
			doctlFlags:  map[string]string{"result": ""},
			listOptions: whisk.ActivationListOptions{Limit: 1},
		},
		{
			name:        "skip flag",
			doctlFlags:  map[string]string{"skip": "2"},
			listOptions: whisk.ActivationListOptions{Limit: 1, Skip: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
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
		name        string
		doctlArgs   string
		doctlFlags  map[string]string
		listOptions whisk.ActivationListOptions
	}{
		{
			name:        "no flags or args",
			listOptions: whisk.ActivationListOptions{Limit: 1},
		},
		{
			name:      "no flags with ID",
			doctlArgs: "activation-1",
		},
		{
			name:        "limit flag",
			doctlFlags:  map[string]string{"limit": "10"},
			listOptions: whisk.ActivationListOptions{Limit: 10},
		},
		{
			name:        "quiet flag",
			doctlFlags:  map[string]string{"quiet": ""},
			listOptions: whisk.ActivationListOptions{Limit: 1},
		},
		{
			name:        "skip flag",
			doctlFlags:  map[string]string{"skip": "1"},
			listOptions: whisk.ActivationListOptions{Limit: 1, Skip: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {

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
			})
		})
	}
}
