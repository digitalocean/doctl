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
	"testing"

	"github.com/digitalocean/doctl/do"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testValidName       = "my-key"
	testValidBucketName = "my-bucket"
	testValidAccessKey  = "DOACCESSKEY"
	testSpacesKey       = do.SpacesKey{
		SpacesKey: &godo.SpacesKey{
			Name:   testValidName,
			Grants: []*godo.Grant{{Bucket: testValidBucketName, Permission: godo.SpacesKeyReadWrite}},
		},
	}
)

func TestSpacesKeysCommand(t *testing.T) {
	cmd := SpacesKeys()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "delete", "update")
}

func TestRunSpacesKeysCreate(t *testing.T) {
	testCases := []struct {
		name      string
		args      []string
		grants    []string
		expectErr bool
	}{
		{
			name:      "success",
			args:      []string{testValidName},
			grants:    []string{"bucket=my-bucket;permission=readwrite"},
			expectErr: false,
		},
		{
			name:      "missing key name",
			args:      []string{},
			grants:    []string{"bucket=my-bucket;permission=readwrite"},
			expectErr: true,
		},
		{
			name:      "invalid grant format",
			args:      []string{testValidName},
			grants:    []string{"bucket=my-bucket;permission"},
			expectErr: true,
		},
		{
			name:      "unsupported permission",
			args:      []string{testValidName},
			grants:    []string{"bucket=my-bucket;permission=invalid"},
			expectErr: true,
		},
		{
			name:      "too many arguments",
			args:      []string{testValidName, "extra-arg"},
			grants:    []string{"bucket=my-bucket;permission=readwrite"},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					req := godo.SpacesKeyCreateRequest{
						Name: testValidName,
						Grants: []*godo.Grant{
							{Bucket: testValidBucketName, Permission: godo.SpacesKeyReadWrite},
						},
					}
					tm.spacesKeys.EXPECT().Create(&req).Return(&testSpacesKey, nil)
				}

				config.Args = tc.args
				config.Doit.Set(config.NS, "grants", tc.grants)

				err := spacesKeysCreate(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestRunSpacesKeysList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.spacesKeys.EXPECT().List().Return([]do.SpacesKey{testSpacesKey}, nil)

		err := spacesKeysList(config)
		require.NoError(t, err)
	})
}

func TestRunSpacesKeysDelete(t *testing.T) {
	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name:      "success",
			args:      []string{testValidAccessKey},
			expectErr: false,
		},
		{
			name:      "missing key id",
			args:      []string{},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					tm.spacesKeys.EXPECT().Delete(testValidAccessKey).Return(nil)
				}

				config.Args = tc.args

				err := spacesKeysDelete(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestRunSpacesKeysUpdate(t *testing.T) {
	testCases := []struct {
		name      string
		args      []string
		grants    []string
		expectErr bool
	}{
		{
			name:      "success",
			args:      []string{testValidAccessKey},
			grants:    []string{"bucket=my-bucket;permission=readwrite"},
			expectErr: false,
		},
		{
			name:      "missing key id",
			args:      []string{},
			grants:    []string{"bucket=my-bucket;permission=readwrite"},
			expectErr: true,
		},
		{
			name:      "invalid grant format",
			args:      []string{testValidAccessKey},
			grants:    []string{"bucket=my-bucket;permission"},
			expectErr: true,
		},
		{
			name:      "unsupported permission",
			args:      []string{testValidAccessKey},
			grants:    []string{"bucket=my-bucket;permission=invalid"},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				if !tc.expectErr {
					req := godo.SpacesKeyUpdateRequest{
						Name: testValidName,
						Grants: []*godo.Grant{
							{Bucket: testValidBucketName, Permission: godo.SpacesKeyReadWrite},
						},
					}
					tm.spacesKeys.EXPECT().Update(testValidAccessKey, &req).Return(&testSpacesKey, nil)
				}

				config.Args = tc.args
				config.Doit.Set(config.NS, "grants", tc.grants)
				config.Doit.Set(config.NS, "name", testValidName)

				err := spacesKeysUpdate(config)
				if tc.expectErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}

func TestParseGrantsFromArg(t *testing.T) {
	grants := []string{"bucket=my-bucket;permission=readwrite"}
	parsedGrants, err := parseGrantsFromArg(grants)
	require.NoError(t, err)
	assert.Equal(t, "my-bucket", parsedGrants[0].Bucket)
	assert.Equal(t, godo.SpacesKeyReadWrite, parsedGrants[0].Permission)
}

func TestParseGrant(t *testing.T) {
	grant := "bucket=my-bucket;permission=readwrite"
	parsedGrant, err := parseGrant(grant)
	require.NoError(t, err)
	assert.Equal(t, "my-bucket", parsedGrant.Bucket)
	assert.Equal(t, godo.SpacesKeyReadWrite, parsedGrant.Permission)
}

func TestParseGrantInvalidFormat(t *testing.T) {
	grant := "bucket=my-bucket;permission"
	_, err := parseGrant(grant)
	assert.Error(t, err)
}

func TestParseGrantUnsupportedPermission(t *testing.T) {
	grant := "bucket=my-bucket;permission=invalid"
	_, err := parseGrant(grant)
	assert.Error(t, err)
}

func TestParseGrantUnsupportedArgument(t *testing.T) {
	grant := "unsupported=my-bucket;permission=readwrite"
	_, err := parseGrant(grant)
	assert.Error(t, err)
}
