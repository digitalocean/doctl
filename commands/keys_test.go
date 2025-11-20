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
	"testing"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testAccessKey = do.AccessKey{
		ID:        "dof_v1_abc123def456",
		Name:      "test-key",
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: nil,
		Secret:    "secret123", // Only present during creation
	}

	testAccessKeyWithoutSecret = do.AccessKey{
		ID:        "dof_v1_abc123def456",
		Name:      "test-key",
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: nil,
		Secret:    "", // Empty for list operations
	}

	testAccessKeyList = []do.AccessKey{testAccessKeyWithoutSecret}

	testServerlessCredentials = do.ServerlessCredentials{
		Namespace: "fn-test-namespace",
		APIHost:   "https://test-api.co",
	}
)

func TestKeysCommand(t *testing.T) {
	cmd := Keys()
	assert.NotNil(t, cmd)
	expected := []string{"create", "list", "revoke"}

	names := []string{}
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}

	assert.ElementsMatch(t, expected, names)

	// Test command properties
	assert.Equal(t, "key", cmd.Use)
	assert.Equal(t, "Manage access keys for functions namespaces", cmd.Short)
	assert.Contains(t, cmd.Long, "Access keys provide secure authentication")
	assert.Contains(t, cmd.Aliases, "keys")
}

func TestAccessKeyCreate(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         map[string]interface{}
		expectedCalls func(*tcMocks)
		expectedError string
	}{
		{
			name: "create with connected namespace",
			flags: map[string]interface{}{
				"name": "my-key",
			},
			expectedCalls: func(tm *tcMocks) {
				tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)
				tm.serverless.EXPECT().ReadCredentials().Return(testServerlessCredentials, nil)
				tm.serverless.EXPECT().CreateNamespaceAccessKey(context.TODO(), "fn-test-namespace", "my-key").Return(testAccessKey, nil)
			},
		},
		{
			name: "create with explicit namespace",
			flags: map[string]interface{}{
				"name":      "my-key",
				"namespace": "fn-explicit-namespace",
			},
			expectedCalls: func(tm *tcMocks) {
				tm.serverless.EXPECT().GetNamespace(context.TODO(), "fn-explicit-namespace").Return(do.ServerlessCredentials{Namespace: "fn-explicit-namespace", APIHost: "https://test.api.host"}, nil)
				tm.serverless.EXPECT().CreateNamespaceAccessKey(context.TODO(), "fn-explicit-namespace", "my-key").Return(testAccessKey, nil)
			},
		},
		{
			name: "create without name flag",
			flags: map[string]interface{}{
				// name is required, but we'll pass empty string
				"name": "",
			},
			expectedCalls: func(tm *tcMocks) {
				// It will still try to resolve namespace and then call create with empty name
				tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)
				tm.serverless.EXPECT().ReadCredentials().Return(testServerlessCredentials, nil)
				tm.serverless.EXPECT().CreateNamespaceAccessKey(context.TODO(), "fn-test-namespace", "").Return(do.AccessKey{}, assert.AnError)
			},
			expectedError: "assert.AnError", // API will reject empty name
		},
		{
			name: "create with disconnected namespace",
			flags: map[string]interface{}{
				"name": "my-key",
			},
			expectedCalls: func(tm *tcMocks) {
				tm.serverless.EXPECT().CheckServerlessStatus().Return(do.ErrServerlessNotConnected)
			},
			expectedError: "serverless support is installed but not connected to a functions namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if tt.expectedCalls != nil {
					tt.expectedCalls(tm)
				}

				// Set flags
				for key, value := range tt.flags {
					config.Doit.Set(config.NS, key, value)
				}

				// Set args
				config.Args = tt.args

				err := RunAccessKeyCreate(config)

				if tt.expectedError != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestAccessKeyList(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         map[string]interface{}
		expectedCalls func(*tcMocks)
		expectedError string
	}{
		{
			name: "list with connected namespace",
			expectedCalls: func(tm *tcMocks) {
				tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)
				tm.serverless.EXPECT().ReadCredentials().Return(testServerlessCredentials, nil)
				tm.serverless.EXPECT().ListNamespaceAccessKeys(context.TODO(), "fn-test-namespace").Return(testAccessKeyList, nil)
			},
		},
		{
			name: "list with explicit namespace",
			flags: map[string]interface{}{
				"namespace": "fn-explicit-namespace",
			},
			expectedCalls: func(tm *tcMocks) {
				tm.serverless.EXPECT().GetNamespace(context.TODO(), "fn-explicit-namespace").Return(do.ServerlessCredentials{Namespace: "fn-explicit-namespace", APIHost: "https://test.api.host"}, nil)
				tm.serverless.EXPECT().ListNamespaceAccessKeys(context.TODO(), "fn-explicit-namespace").Return(testAccessKeyList, nil)
			},
		},
		{
			name:          "list with too many args",
			args:          []string{"extra-arg"},
			expectedError: "command contains unsupported arguments",
		},
		{
			name: "list with disconnected namespace",
			expectedCalls: func(tm *tcMocks) {
				tm.serverless.EXPECT().CheckServerlessStatus().Return(do.ErrServerlessNotConnected)
			},
			expectedError: "serverless support is installed but not connected to a functions namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if tt.expectedCalls != nil {
					tt.expectedCalls(tm)
				}

				// Set flags
				for key, value := range tt.flags {
					config.Doit.Set(config.NS, key, value)
				}

				// Set args
				config.Args = tt.args

				err := RunAccessKeyList(config)

				if tt.expectedError != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestAccessKeyRevoke(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         map[string]interface{}
		expectedCalls func(*tcMocks)
		expectedError string
	}{
		{
			name: "revoke with connected namespace and force",
			args: []string{"dof_v1_abc123def456"},
			flags: map[string]interface{}{
				"force": true,
			},
			expectedCalls: func(tm *tcMocks) {
				tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)
				tm.serverless.EXPECT().ReadCredentials().Return(testServerlessCredentials, nil)
				tm.serverless.EXPECT().DeleteNamespaceAccessKey(context.TODO(), "fn-test-namespace", "dof_v1_abc123def456").Return(nil)
			},
		},
		{
			name: "revoke with explicit namespace",
			args: []string{"dof_v1_abc123def456"},
			flags: map[string]interface{}{
				"namespace": "fn-explicit-namespace",
				"force":     true,
			},
			expectedCalls: func(tm *tcMocks) {
				tm.serverless.EXPECT().GetNamespace(context.TODO(), "fn-explicit-namespace").Return(do.ServerlessCredentials{Namespace: "fn-explicit-namespace", APIHost: "https://test.api.host"}, nil)
				tm.serverless.EXPECT().DeleteNamespaceAccessKey(context.TODO(), "fn-explicit-namespace", "dof_v1_abc123def456").Return(nil)
			},
		},
		{
			name:          "revoke without key ID",
			args:          []string{},
			expectedError: "command is missing required arguments",
		},
		{
			name:          "revoke with too many args",
			args:          []string{"key1", "key2"},
			expectedError: "command contains unsupported arguments",
		},
		{
			name: "revoke with disconnected namespace",
			args: []string{"dof_v1_abc123def456"},
			flags: map[string]interface{}{
				"force": true,
			},
			expectedCalls: func(tm *tcMocks) {
				tm.serverless.EXPECT().CheckServerlessStatus().Return(do.ErrServerlessNotConnected)
			},
			expectedError: "serverless support is installed but not connected to a functions namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if tt.expectedCalls != nil {
					tt.expectedCalls(tm)
				}

				// Set flags
				for key, value := range tt.flags {
					config.Doit.Set(config.NS, key, value)
				}

				// Set args
				config.Args = tt.args

				err := RunAccessKeyRevoke(config)

				if tt.expectedError != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestResolveTargetNamespace(t *testing.T) {
	tests := []struct {
		name              string
		explicitNamespace string
		credentialsReturn do.ServerlessCredentials
		credentialsError  error
		statusError       error
		expectedNamespace string
		expectedError     string
	}{
		{
			name:              "explicit namespace provided",
			explicitNamespace: "fn-explicit",
			expectedNamespace: "fn-explicit",
		},
		{
			name:              "use connected namespace",
			explicitNamespace: "",
			credentialsReturn: do.ServerlessCredentials{Namespace: "fn-connected"},
			expectedNamespace: "fn-connected",
		},
		{
			name:              "not connected to serverless",
			explicitNamespace: "",
			statusError:       do.ErrServerlessNotConnected,
			expectedError:     "serverless support is installed but not connected to a functions namespace",
		},
		{
			name:              "credentials read error",
			explicitNamespace: "",
			credentialsError:  assert.AnError,
			expectedError:     "not connected to any namespace",
		},
		{
			name:              "empty namespace in credentials",
			explicitNamespace: "",
			credentialsReturn: do.ServerlessCredentials{Namespace: ""},
			expectedError:     "not connected to any namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if tt.explicitNamespace == "" {
					if tt.statusError != nil {
						tm.serverless.EXPECT().CheckServerlessStatus().Return(tt.statusError)
					} else {
						tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)
						if tt.credentialsError != nil {
							tm.serverless.EXPECT().ReadCredentials().Return(do.ServerlessCredentials{}, tt.credentialsError)
						} else {
							tm.serverless.EXPECT().ReadCredentials().Return(tt.credentialsReturn, nil)
						}
					}
				} else {
					// For explicit namespace, we now need to mock GetNamespace validation call
					mockCredentials := do.ServerlessCredentials{
						Namespace: tt.explicitNamespace,
						APIHost:   "https://test.api.host",
					}
					tm.serverless.EXPECT().GetNamespace(context.TODO(), tt.explicitNamespace).Return(mockCredentials, nil)
				}

				namespace, err := resolveTargetNamespace(config, tt.explicitNamespace)

				if tt.expectedError != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedNamespace, namespace)
				}
			})
		})
	}
}

func TestAccessKeyListOutput(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		// Test data for output formatting
		keys := []do.AccessKey{
			{
				ID:        "dof_v1_abc123def456ghi789",
				Name:      "laptop-key",
				CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				ExpiresAt: nil,
				Secret:    "", // Empty for list operations
			},
			{
				ID:        "dof_v1_xyz789abc123def456",
				Name:      "ci-cd-key",
				CreatedAt: time.Date(2023, 2, 15, 9, 30, 0, 0, time.UTC),
				ExpiresAt: func() *time.Time { t := time.Date(2024, 2, 15, 9, 30, 0, 0, time.UTC); return &t }(),
				Secret:    "", // Empty for list operations
			},
		}

		tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)
		tm.serverless.EXPECT().ReadCredentials().Return(testServerlessCredentials, nil)
		tm.serverless.EXPECT().ListNamespaceAccessKeys(context.TODO(), "fn-test-namespace").Return(keys, nil)

		err := RunAccessKeyList(config)

		require.NoError(t, err)

		// Test output contains expected elements
		output := buf.String()
		assert.Contains(t, output, "dof_v1_abc12...") // ID truncated to 12 chars + ...
		assert.Contains(t, output, "laptop-key")
		assert.Contains(t, output, "dof_v1_xyz78...") // ID truncated to 12 chars + ...
		assert.Contains(t, output, "ci-cd-key")
		assert.Contains(t, output, "<hidden>")
		assert.Contains(t, output, "2023-01-01 12:00:00 UTC")
		assert.Contains(t, output, "2023-02-15 09:30:00 UTC")
		assert.Contains(t, output, "2024-02-15 09:30:00 UTC")
	})
}

func TestAccessKeyRevokeOutput(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := &bytes.Buffer{}
		config.Out = buf

		config.Args = []string{"dof_v1_abc123def456"}
		config.Doit.Set(config.NS, "force", true)

		expectedOutput := "Key dof_v1_abc123def456 has been revoked.\n"

		tm.serverless.EXPECT().CheckServerlessStatus().Return(nil)
		tm.serverless.EXPECT().ReadCredentials().Return(testServerlessCredentials, nil)
		tm.serverless.EXPECT().DeleteNamespaceAccessKey(context.TODO(), "fn-test-namespace", "dof_v1_abc123def456").Return(nil)

		err := RunAccessKeyRevoke(config)

		require.NoError(t, err)
		assert.Equal(t, expectedOutput, buf.String())
	})
}
