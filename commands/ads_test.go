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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/internal/ui"
	"github.com/digitalocean/godo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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

// TestADSFlagsReadableAtRuntime checks that every string flag on every ads
// command is bound to the config key its handler reads. A command attached
// to its parent after its flags were added binds them under the wrong key,
// and the value set on the command line is then never seen.
func TestADSFlagsReadableAtRuntime(t *testing.T) {
	// A fresh tree, because building one rebinds the config keys to its
	// flags; other tests in this package build ads trees too.
	ads := ADS()

	checked := 0
	var walk func(*Command)
	walk = func(cmd *Command) {
		for _, child := range cmd.ChildCommands() {
			walk(child)
		}
		if cmd.Run == nil {
			return
		}
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f.Value.Type() != "string" {
				return
			}
			prev, prevChanged := f.Value.String(), f.Changed
			require.NoError(t, cmd.Flags().Set(f.Name, "probe"))
			t.Cleanup(func() { _ = f.Value.Set(prev); f.Changed = prevChanged })

			key := cmdNS(cmd) + "." + f.Name
			assert.Equal(t, "probe", viper.GetString(key), "%s --%s is not readable at %s", cmd.CommandPath(), f.Name, key)
			checked++
		})
	}
	walk(ads)
	assert.Greater(t, checked, 50, "expected to check every string flag")
}

// TestADSNamespacesUnique checks that no other doctl command shares a config
// namespace with an ads command. Commands that share one bind the same flag
// keys, so one of them reads the other's flag values.
func TestADSNamespacesUnique(t *testing.T) {
	owners := map[string][]string{}
	var walk func(*Command)
	walk = func(cmd *Command) {
		for _, child := range cmd.ChildCommands() {
			walk(child)
		}
		if cmd.Run != nil {
			owners[cmdNS(cmd)] = append(owners[cmdNS(cmd)], cmd.CommandPath())
		}
	}
	walk(DoitCmd)

	for ns, paths := range owners {
		for _, path := range paths {
			if strings.HasPrefix(path, "doctl ads ") {
				assert.Len(t, paths, 1, "namespace %q is shared by %v", ns, paths)
				break
			}
		}
	}
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
	tests := []struct {
		name         string
		mask         bool
		format       string
		wantPassword bool
	}{
		{name: "masked text", mask: true, format: "text"},
		{name: "masked json", mask: true, format: "json"},
		{name: "show text", mask: false, format: "text", wantPassword: true},
		{name: "show json", mask: false, format: "json", wantPassword: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.ads.EXPECT().GetWorkspaceQueryCredentials(testADSRegion, testADSWorkspace).Return(&godo.ADSQueryCredentials{
					User:     "reader",
					Password: "secret_value",
				}, nil)
				config.Args = []string{testADSWorkspace}
				config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
				config.UI.Mask = tt.mask
				buf := withInsightsOutput(t, config, tt.format)

				require.NoError(t, RunADSWorkspaceQueryCredentials(config))
				assert.Contains(t, buf.String(), "reader")
				if tt.wantPassword {
					assert.Contains(t, buf.String(), "secret_value")
				} else {
					assert.NotContains(t, buf.String(), "secret_value")
					assert.Contains(t, buf.String(), secretMaskedValue)
				}
			})
		})
	}
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
		config.Doit.Set(config.NS, doctl.ArgADSSecretData, []string{"password=secret_value"})

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
		data    []string
		file    string
		wantErr string
	}{
		{name: "no properties", wantErr: "at least one property is required"},
		{name: "flag and file", data: []string{"a=b"}, file: nested, wantErr: "cannot be combined"},
		{name: "repeated key", data: []string{"password=secret_value", "password=secret_value"}, wantErr: `"password" is given more than once`},
		{name: "missing equals sign", data: []string{"secret_value"}, wantErr: "must be formatted as key=value"},
		{name: "empty key", data: []string{"=secret_value"}, wantErr: "names must not be empty"},
		{name: "whitespace key", data: []string{" =secret_value"}, wantErr: "names must not be empty"},
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

func TestADSSecretDataFlag(t *testing.T) {
	tests := []struct {
		name    string
		data    []string
		want    map[string]string
		wantErr string
	}{
		{
			name: "no repeats",
			data: []string{"user=loader", "password=hunter2"},
			want: map[string]string{"user": "loader", "password": "hunter2"},
		},
		{
			name:    "repeated key with different values",
			data:    []string{"password=first_value", "password=second_value"},
			wantErr: `secret property "password" is given more than once in --data; pass each property once`,
		},
		{
			name:    "repeated key with the same value",
			data:    []string{"password=same_value", "password=same_value"},
			wantErr: `secret property "password" is given more than once in --data; pass each property once`,
		},
		{
			name:    "key repeated three times",
			data:    []string{"password=first_value", "password=second_value", "password=third_value"},
			wantErr: `secret property "password" is given more than once in --data; pass each property once`,
		},
		{
			name:    "repeat separated by other keys",
			data:    []string{"password=first_value", "user=loader", "host=db", "password=second_value"},
			wantErr: `secret property "password" is given more than once in --data; pass each property once`,
		},
		{
			name:    "repeated key with an empty value",
			data:    []string{"password=", "password="},
			wantErr: `secret property "password" is given more than once in --data; pass each property once`,
		},
		{
			name: "same value under different keys",
			data: []string{"primary=shared_value", "replica=shared_value"},
			want: map[string]string{"primary": "shared_value", "replica": "shared_value"},
		},
		{
			name: "keys that differ only in case",
			data: []string{"password=lower_value", "Password=upper_value"},
			want: map[string]string{"password": "lower_value", "Password": "upper_value"},
		},
		{
			name:    "repeat alongside a key that differs only in case",
			data:    []string{"password=first_value", "Password=upper_value", "password=second_value"},
			wantErr: `secret property "password" is given more than once in --data; pass each property once`,
		},
		{
			name: "keys that differ only in case after trimming",
			data: []string{" Password=upper_value", "password=lower_value"},
			want: map[string]string{"Password": "upper_value", "password": "lower_value"},
		},
		{
			name: "keys trimmed, values kept as given",
			data: []string{" user =loader", "\tpassword= padded value "},
			want: map[string]string{"user": "loader", "password": " padded value "},
		},
		{
			name:    "repeated after trimming",
			data:    []string{"password=first_value", " password =second_value"},
			wantErr: `secret property "password" is given more than once in --data; pass each property once`,
		},
		{
			name: "brackets, quotes, commas and equals signs kept",
			data: []string{"[key=v", "csv=a,b", `quote=a"b`, "pad=a=b==", "password=abc]"},
			want: map[string]string{"[key": "v", "csv": "a,b", "quote": `a"b`, "pad": "a=b==", "password": "abc]"},
		},
		{
			name:    "missing equals sign",
			data:    []string{"user=loader", "hunter2_value"},
			wantErr: "--data must be formatted as key=value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgADSSecretData, tt.data)

				got, err := adsSecretDataFlag(config)
				if tt.wantErr == "" {
					require.NoError(t, err)
					assert.Equal(t, tt.want, got)
					return
				}
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				assert.NotContains(t, err.Error(), "_value", "the error echoed a secret value")
			})
		})
	}
}

func TestADSSecretDataFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
		wantErr string
	}{
		{
			name:    "json without repeats",
			content: `{"user":"loader","password":"hunter2"}`,
			want:    map[string]string{"user": "loader", "password": "hunter2"},
		},
		{
			name:    "yaml without repeats",
			content: "user: loader\npassword: hunter2\n",
			want:    map[string]string{"user": "loader", "password": "hunter2"},
		},
		{
			name:    "json repeated key",
			content: `{"password":"first_value","password":"second_value"}`,
			wantErr: `secret property "password" is given more than once in --data-file; pass each property once`,
		},
		{
			name:    "yaml repeated key",
			content: "password: first_value\npassword: second_value\n",
			wantErr: `secret property "password" is given more than once in --data-file; pass each property once`,
		},
		{
			name:    "repeated key with the same value",
			content: `{"password":"same_value","password":"same_value"}`,
			wantErr: `secret property "password" is given more than once in --data-file; pass each property once`,
		},
		{
			name:    "key repeated three times",
			content: `{"password":"first_value","user":"loader","password":"second_value","password":"third_value"}`,
			wantErr: `secret property "password" is given more than once in --data-file; pass each property once`,
		},
		{
			name:    "repeated after trimming",
			content: `{"password":"first_value"," password ":"second_value"}`,
			wantErr: `secret property "password" is given more than once in --data-file; pass each property once`,
		},
		{
			name:    "keys trimmed, values kept as given",
			content: `{" user ":"loader","password":" padded value "}`,
			want:    map[string]string{"user": "loader", "password": " padded value "},
		},
		{
			name:    "keys that differ only in case",
			content: `{"password":"lower_value","Password":"upper_value"}`,
			want:    map[string]string{"password": "lower_value", "Password": "upper_value"},
		},
		{
			name:    "empty file",
			content: "",
		},
		{
			name:    "not an object",
			content: `["password","secret_value"]`,
			wantErr: "must be a JSON or YAML object",
		},
		{
			name:    "non-string value",
			content: "password:\n  inner: secret_value\n",
			wantErr: "values are all strings",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "secret")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o600))

			got, err := adsSecretDataFile(path)
			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.NotContains(t, err.Error(), "_value", "the error echoed a secret value")
		})
	}
}

func TestRunADSSecretPutKeepsKeysThatDifferOnlyInCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"password":"a","Password":"b"}`), 0o600))

	tests := []struct {
		name string
		data []string
		file string
	}{
		{name: "from --data", data: []string{"password=a", "Password=b"}},
		{name: "from --data-file", file: path},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				tm.ads.EXPECT().PutWorkspaceSecret(&godo.ADSPutWorkspaceSecretRequest{
					Region:    testADSRegion,
					UUID:      testADSWorkspace,
					SecretKey: "warehouse-db",
					Data:      map[string]string{"password": "a", "Password": "b"},
				}).Return(&godo.ADSPutWorkspaceSecretResponse{}, nil)
				config.Args = []string{"warehouse-db"}
				config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
				config.Doit.Set(config.NS, doctl.ArgADSWorkspace, testADSWorkspace)
				if tt.data != nil {
					config.Doit.Set(config.NS, doctl.ArgADSSecretData, tt.data)
				}
				if tt.file != "" {
					config.Doit.Set(config.NS, doctl.ArgADSSecretDataFile, tt.file)
				}

				require.NoError(t, RunADSSecretPut(config))
			})
		})
	}
}

func TestRunADSSecretPutWarnsOnEmptyValue(t *testing.T) {
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "empty.json")
	require.NoError(t, os.WriteFile(emptyFile, []byte(`{"user":"loader","password":""}`), 0o600))

	tests := []struct {
		name string
		data []string
		file string
	}{
		{name: "from --data", data: []string{"user=loader", "password="}},
		{name: "from --data-file", file: emptyFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				var stderr bytes.Buffer
				withUIEnv(t, ui.Plain(&bytes.Buffer{}, &stderr))

				tm.ads.EXPECT().PutWorkspaceSecret(&godo.ADSPutWorkspaceSecretRequest{
					Region:    testADSRegion,
					UUID:      testADSWorkspace,
					SecretKey: "warehouse-db",
					Data:      map[string]string{"user": "loader", "password": ""},
				}).Return(&godo.ADSPutWorkspaceSecretResponse{}, nil)
				config.Args = []string{"warehouse-db"}
				config.Doit.Set(config.NS, doctl.ArgRegionSlug, testADSRegion)
				config.Doit.Set(config.NS, doctl.ArgADSWorkspace, testADSWorkspace)
				if tt.data != nil {
					config.Doit.Set(config.NS, doctl.ArgADSSecretData, tt.data)
				}
				if tt.file != "" {
					config.Doit.Set(config.NS, doctl.ArgADSSecretDataFile, tt.file)
				}

				require.NoError(t, RunADSSecretPut(config))
				assert.Contains(t, stderr.String(), `Warning: secret property "password" has an empty value`)
				assert.NotContains(t, stderr.String(), `"user"`, "only empty values should be reported")
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
