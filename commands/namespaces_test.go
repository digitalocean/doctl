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
	"errors"
	"sort"
	"testing"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespacesCommand(t *testing.T) {
	cmd := Namespaces()
	assert.NotNil(t, cmd)
	expected := []string{"create", "delete", "list", "list-regions"}

	names := []string{}
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}

	sort.Strings(expected)
	sort.Strings(names)
	assert.Equal(t, expected, names)
}

func TestNamespacesCreate(t *testing.T) {
	tests := []struct {
		name           string
		doctlFlags     map[string]interface{}
		expectedOutput string
		expectedError  error
		expectList     bool
		willConnect    bool
	}{
		{
			name:          "no flags",
			expectedError: errors.New("the '--label' and '--region' flags are both required"),
		},
		{
			name: "invalid region",
			doctlFlags: map[string]interface{}{
				"label":  "my_dog",
				"region": "dog",
			},
			expectedError: errors.New("'dog' is not a valid region value"),
		},
		{
			name: "legal flags, with no-connect",
			doctlFlags: map[string]interface{}{
				"label":      "something",
				"region":     "lon",
				"no-connect": true,
			},
			expectedOutput: "New namespace hello created, but not connected.\n",
			expectList:     true,
		},
		{
			name: "legal flags, with label conflict",
			doctlFlags: map[string]interface{}{
				"label":  "my_dog",
				"region": "lon",
			},
			expectList:    true,
			expectedError: errors.New("you are using  label 'my_dog' for another namespace; labels should be unique"),
		},
		{
			name: "legal flags, should connect",
			doctlFlags: map[string]interface{}{
				"label":  "something",
				"region": "lon",
			},
			expectList:     true,
			willConnect:    true,
			expectedOutput: "Connected to functions namespace 'hello' on API host 'https://api.example.com'\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				buf := &bytes.Buffer{}
				config.Out = buf
				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}

				ctx := context.TODO()
				if tt.expectList {
					initialList := do.NamespaceListResponse{Namespaces: []do.OutputNamespace{
						{Label: "my_dog"},
					}}
					tm.serverless.EXPECT().ListNamespaces(ctx).Return(initialList, nil)
				}
				if tt.willConnect {
					tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)
					creds := do.ServerlessCredentials{Namespace: "hello", APIHost: "https://api.example.com"}
					tm.serverless.EXPECT().WriteCredentials(creds).Return(nil)
				}
				if tt.expectedError == nil {
					label := tt.doctlFlags["label"]
					tm.serverless.EXPECT().CreateNamespace(ctx, label, "lon1").Return(do.ServerlessCredentials{
						Namespace: "hello",
						APIHost:   "https://api.example.com",
					}, nil)
				}

				err := RunNamespacesCreate(config)
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

func TestNamespacesListRegions(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		expectedOutput := "[ams ams3 blr blr1 fra fra1 lon lon1 nyc nyc1 sfo sfo3 sgp sgp1 syd syd1 tor tor1]\n"

		err := RunNamespacesListRegions(config)

		require.NoError(t, err)
		assert.Equal(t, expectedOutput, buf.String())
	})
}

func TestNamespacesList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		returnedList := do.NamespaceListResponse{Namespaces: []do.OutputNamespace{
			{
				Label:     "my_dog",
				Namespace: "ns1",
				Region:    "lon1",
				APIHost:   "https://lon1.example.com",
			},
			{
				Label:     "something",
				Namespace: "ns2",
				Region:    "sgp1",
				APIHost:   "https://sgp1.example.com",
			},
		}}
		expectedOutput := "Label        Region    Namespace ID    API Host\nmy_dog       lon1      ns1             https://lon1.example.com\nsomething    sgp1      ns2             https://sgp1.example.com\n"

		tm.serverless.EXPECT().ListNamespaces(context.TODO()).Return(returnedList, nil)

		err := RunNamespacesList(config)

		require.NoError(t, err)
		assert.Equal(t, expectedOutput, buf.String())
	})
}

func TestNamespacesDelete(t *testing.T) {
	tests := []struct {
		name               string
		doctlFlags         map[string]interface{}
		doctlArg           string
		expectedOutput     string
		expectedError      error
		expectConfirmation bool
	}{
		{
			name:          "nonmatching argument",
			doctlArg:      "xyzzy",
			expectedError: errors.New("'xyzzy' does not exactly match the label or id of any of your namespaces"),
		},
		{
			name:          "partial match argument",
			doctlArg:      "some",
			expectedError: errors.New("'some' does not exactly match the label or id of any of your namespaces"),
		},
		{
			name:       "valid argument with force",
			doctlArg:   "my_dog",
			doctlFlags: map[string]interface{}{"force": true},
		},
		{
			name:               "valid argument with prompt",
			doctlArg:           "my_dog",
			expectConfirmation: true,
			expectedError:      errors.New("deletion of 'ns1' not confirmed, doing nothing"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				buf := &bytes.Buffer{}
				config.Out = buf
				if tt.doctlFlags != nil {
					for k, v := range tt.doctlFlags {
						if v == "" {
							config.Doit.Set(config.NS, k, true)
						} else {
							config.Doit.Set(config.NS, k, v)
						}
					}
				}
				config.Args = append(config.Args, tt.doctlArg)
				listForMatching := do.NamespaceListResponse{Namespaces: []do.OutputNamespace{
					{
						Label:     "my_dog",
						Namespace: "ns1",
						Region:    "lon1",
						APIHost:   "https://lon1.example.com",
					},
					{
						Label:     "something",
						Namespace: "ns2",
						Region:    "sgp1",
						APIHost:   "https://sgp1.example.com",
					},
				}}

				ctx := context.TODO()
				tm.serverless.EXPECT().ListNamespaces(ctx).Return(listForMatching, nil)
				if tt.expectedError == nil {
					tm.serverless.EXPECT().DeleteNamespace(ctx, "ns1").Return(nil)
				}

				err := RunNamespacesDelete(config)
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
