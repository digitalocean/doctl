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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testADSRegion    = "nyc3"
	testADSWorkspace = "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"
)

var testADSWorkspaceItem = godo.ADSWorkspace{
	UUID:              testADSWorkspace,
	Name:              "analytics",
	Region:            testADSRegion,
	Status:            "active",
	QueryInstanceSize: "small",
	QueryEndpoint:     "query.example.com",
}

func TestADSCommand(t *testing.T) {
	cmd := ADS()
	assert.Equal(t, "ads", cmd.Name())
	assert.False(t, cmd.Hidden)
	assertCommandNames(t, cmd, "plan", "workspace", "token", "secret", "data-flow", "metrics")
}

func TestADSSubcommands(t *testing.T) {
	assertCommandNames(t, adsPlan(), "list")
	assertCommandNames(t, adsWorkspace(), "create", "list", "get", "delete", "query-credentials")
	assertCommandNames(t, adsToken(), "create", "list")
	assertCommandNames(t, adsSecret(), "put", "get", "list", "delete")
}

// TestADSHelpUsesPublicNames keeps the legacy product names out of every
// ads command's help text.
func TestADSHelpUsesPublicNames(t *testing.T) {
	var walk func(*Command)
	walk = func(cmd *Command) {
		text := strings.ToLower(cmd.Use + " " + cmd.Short + " " + cmd.Long + " " + cmd.Example + " " + cmd.Flags().FlagUsages())
		for _, legacy := range []string{"sandbox", "pipeline", "aidc", "aidp"} {
			assert.NotContains(t, text, legacy, "help for %q", cmd.CommandPath())
		}
		for _, child := range cmd.ChildCommands() {
			walk(child)
		}
	}
	walk(ADS())
}

func TestADSRootRegistered(t *testing.T) {
	found := false
	for _, cmd := range DoitCmd.Commands() {
		if cmd.Name() == "ads" {
			found = true
		}
	}
	assert.True(t, found, "ads is not registered on the root command")
}

func TestRunADSPlanList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().ListPlans().Return(&godo.ADSPlansResponse{Plans: []godo.ADSPlan{{
			RegionSlug:         testADSRegion,
			AvailableForCreate: true,
			QueryEngineSizes:   []godo.ADSQueryEngineSize{{Size: "small"}, {Size: "large"}},
		}}}, nil)
		buf := withInsightsOutput(t, config, "text")

		require.NoError(t, RunADSPlanList(config))
		assert.Contains(t, buf.String(), "small,large")
	})
}

func TestRunADSWorkspaceCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().CreateWorkspace(testADSRegion, &godo.ADSWorkspaceCreateRequest{
			Name:              "analytics",
			QueryInstanceSize: "medium",
		}).Return(&testADSWorkspaceItem, nil)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgADSName, "analytics")
		config.Doit.Set(config.NS, doctl.ArgADSQueryInstanceSize, "medium")

		require.NoError(t, RunADSWorkspaceCreate(config))
	})
}

func TestRunADSWorkspaceCreateRequiresRegion(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		err := RunADSWorkspaceCreate(config)
		var missing *doctl.MissingArgsErr
		require.ErrorAs(t, err, &missing)
		assert.Contains(t, err.Error(), doctl.ArgRegionSlug)
	})
}

func TestRunADSWorkspaceList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().ListWorkspaces(testADSRegion).Return(&godo.ADSWorkspacesResponse{
			Workspaces: []godo.ADSWorkspace{testADSWorkspaceItem},
		}, nil)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		buf := withInsightsOutput(t, config, "text")

		require.NoError(t, RunADSWorkspaceList(config))
		assert.Contains(t, buf.String(), "analytics")
	})
}

func TestRunADSWorkspaceGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().GetWorkspace(testADSRegion, testADSWorkspace).Return(&testADSWorkspaceItem, nil)
		config.Args = []string{testADSWorkspace}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)

		require.NoError(t, RunADSWorkspaceGet(config))
	})
}

func TestRunADSWorkspaceGetSurfacesForbidden(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		forbidden := errors.New("403 you do not have access to this workspace")
		tm.ads.EXPECT().GetWorkspace(testADSRegion, testADSWorkspace).Return(nil, forbidden)
		config.Args = []string{testADSWorkspace}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)

		require.ErrorIs(t, RunADSWorkspaceGet(config), forbidden)
	})
}

func TestRunADSWorkspaceDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().DeleteWorkspace(testADSRegion, testADSWorkspace).Return(nil)
		config.Args = []string{testADSWorkspace}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		require.NoError(t, RunADSWorkspaceDelete(config))
	})
}

func TestRunADSWorkspaceDeleteRequiresConfirmation(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Args = []string{testADSWorkspace}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)

		require.ErrorIs(t, RunADSWorkspaceDelete(config), errConfirmationRequired)
	})
}

func TestRunADSWorkspaceQueryCredentials(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().GetWorkspaceQueryCredentials(testADSRegion, testADSWorkspace).Return(&godo.ADSQueryCredentials{
			User:     "reader",
			Password: "secret_value",
		}, nil)
		config.Args = []string{testADSWorkspace}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		buf := withInsightsOutput(t, config, "text")

		require.NoError(t, RunADSWorkspaceQueryCredentials(config))
		assert.Contains(t, buf.String(), "reader")
		assert.Contains(t, buf.String(), "secret_value")
	})
}

func TestRunADSTokenCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().CreateToken(&godo.ADSTokenCreateRequest{
			Region:           testADSRegion,
			Purpose:          godo.ADSTokenPurposeQuery,
			WorkspaceUUIDs:   []string{testADSWorkspace},
			Name:             "reporting",
			ExpiresInSeconds: 3600,
			Permissions:      []string{},
		}).Return(&godo.ADSTokenCreateResponse{
			ID:          "token-id",
			Name:        "reporting",
			AccessToken: "secret_value",
		}, nil)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgADSName, "reporting")
		config.Doit.Set(config.NS, doctl.ArgADSTokenPurpose, "query")
		config.Doit.Set(config.NS, doctl.ArgADSTokenWorkspaceUUIDs, []string{testADSWorkspace})
		config.Doit.Set(config.NS, doctl.ArgADSTokenExpiresIn, 3600)
		config.Doit.Set(config.NS, doctl.ArgADSTokenPermissions, []string{})
		buf := withInsightsOutput(t, config, "text")

		require.NoError(t, RunADSTokenCreate(config))
		assert.Contains(t, buf.String(), "secret_value", "the one-time access token must be printed")
	})
}

func TestRunADSTokenCreateRejectsUnknownPurpose(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgADSTokenPurpose, "admin")

		assert.ErrorContains(t, RunADSTokenCreate(config), "the only supported purpose is query")
	})
}

func TestRunADSTokenList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().ListTokens(&godo.ADSTokenListRequest{
			Region:        testADSRegion,
			WorkspaceUUID: testADSWorkspace,
			Page:          2,
			PerPage:       10,
		}).Return(&godo.ADSListTokensResponse{Tokens: []godo.ADSTokenSummary{{
			ID:      "token-id",
			Name:    "reporting",
			Purpose: godo.ADSTokenPurposeQuery,
		}}}, nil)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgADSTokenWorkspaceUUID, testADSWorkspace)
		config.Doit.Set(config.NS, doctl.ArgADSPage, 2)
		config.Doit.Set(config.NS, doctl.ArgADSPerPage, 10)
		buf := withInsightsOutput(t, config, "text")

		require.NoError(t, RunADSTokenList(config))
		assert.Contains(t, buf.String(), "reporting")
		assert.Contains(t, buf.String(), "query")
	})
}

func TestRunADSSecretPutFromFlag(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().PutWorkspaceSecret(&godo.ADSPutWorkspaceSecretRequest{
			Region:    testADSRegion,
			UUID:      testADSWorkspace,
			SecretKey: "warehouse-db",
			Data:      map[string]string{"password": "secret_value"},
		}).Return(&godo.ADSPutWorkspaceSecretResponse{Data: &godo.ADSSecretMetadata{Version: 3}}, nil).Times(2)
		config.Args = []string{"warehouse-db"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgADSWorkspace, testADSWorkspace)
		config.Doit.Set(config.NS, doctl.ArgADSSecretData, map[string]string{"password": "secret_value"})

		for _, format := range []string{"text", "json"} {
			buf := withInsightsOutput(t, config, format)
			require.NoError(t, RunADSSecretPut(config))
			assert.NotContains(t, buf.String(), "secret_value", "%s output echoed a secret value", format)
			assert.Contains(t, buf.String(), "warehouse-db")
		}
	})
}

func TestRunADSSecretPutFromStdin(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		prev := adsStdin
		adsStdin = strings.NewReader("user: loader\npassword: secret_value\n")
		t.Cleanup(func() { adsStdin = prev })

		tm.ads.EXPECT().PutWorkspaceSecret(&godo.ADSPutWorkspaceSecretRequest{
			Region:    testADSRegion,
			UUID:      testADSWorkspace,
			SecretKey: "warehouse-db",
			Data:      map[string]string{"user": "loader", "password": "secret_value"},
		}).Return(&godo.ADSPutWorkspaceSecretResponse{}, nil)
		config.Args = []string{"warehouse-db"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgADSWorkspace, testADSWorkspace)
		config.Doit.Set(config.NS, doctl.ArgADSSecretDataFile, "-")

		require.NoError(t, RunADSSecretPut(config))
	})
}

func TestRunADSSecretPutRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested.json")
	require.NoError(t, os.WriteFile(nested, []byte(`{"password":{"inner":"secret_value"}}`), 0o600))

	tests := []struct {
		name    string
		data    map[string]string
		file    string
		wantErr string
	}{
		{name: "no properties", wantErr: "at least one property is required"},
		{name: "flag and file", data: map[string]string{"a": "b"}, file: nested, wantErr: "cannot be combined"},
		{name: "non-string value", file: nested, wantErr: "values are all strings"},
		{name: "missing file", file: filepath.Join(dir, "missing.json"), wantErr: "missing.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				config.Args = []string{"warehouse-db"}
				config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
				config.Doit.Set(config.NS, doctl.ArgADSWorkspace, testADSWorkspace)
				if tt.data != nil {
					config.Doit.Set(config.NS, doctl.ArgADSSecretData, tt.data)
				}
				if tt.file != "" {
					config.Doit.Set(config.NS, doctl.ArgADSSecretDataFile, tt.file)
				}

				err := RunADSSecretPut(config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.NotContains(t, err.Error(), "secret_value")
			})
		})
	}
}

func TestRunADSSecretGetShowsNamesNotValues(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().GetWorkspaceSecret(testADSRegion, testADSWorkspace, "warehouse-db").Return(&godo.ADSGetWorkspaceSecretResponse{
			Data: &godo.ADSGetSecretData{
				Keys:     []string{"password", "user"},
				Data:     map[string]any{"password": "secret_value"},
				Metadata: &godo.ADSSecretMetadata{Version: 2},
			},
		}, nil).Times(2)
		config.Args = []string{"warehouse-db"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgADSWorkspace, testADSWorkspace)

		for _, format := range []string{"text", "json"} {
			buf := withInsightsOutput(t, config, format)
			require.NoError(t, RunADSSecretGet(config))
			assert.Contains(t, buf.String(), "password")
			assert.Contains(t, buf.String(), "user")
			assert.NotContains(t, buf.String(), "secret_value", "%s output printed a secret value", format)
		}
	})
}

func TestRunADSSecretList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().ListWorkspaceSecrets(testADSRegion, testADSWorkspace).Return(&godo.ADSListWorkspaceSecretsResponse{
			SecretKeys: []string{"warehouse-db"},
		}, nil)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgADSWorkspace, testADSWorkspace)
		buf := withInsightsOutput(t, config, "text")

		require.NoError(t, RunADSSecretList(config))
		assert.Contains(t, buf.String(), "warehouse-db")
	})
}

func TestRunADSSecretListRequiresWorkspace(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)

		err := RunADSSecretList(config)
		var missing *doctl.MissingArgsErr
		require.ErrorAs(t, err, &missing)
		assert.Contains(t, err.Error(), doctl.ArgADSWorkspace)
	})
}

func TestRunADSSecretDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.ads.EXPECT().DeleteWorkspaceSecret(testADSRegion, testADSWorkspace, "warehouse-db").Return(nil)
		config.Args = []string{"warehouse-db"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgADSWorkspace, testADSWorkspace)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		require.NoError(t, RunADSSecretDelete(config))
	})
}

func TestRunADSSecretDeleteRequiresConfirmation(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Args = []string{"warehouse-db"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
		config.Doit.Set(config.NS, doctl.ArgADSWorkspace, testADSWorkspace)

		require.ErrorIs(t, RunADSSecretDelete(config), errConfirmationRequired)
	})
}
