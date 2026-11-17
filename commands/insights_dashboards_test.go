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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testInsightsDashboardSpec = `{"kind": "Dashboard", "spec": {"display": {"name": "Droplet CPU"}}}`

var testInsightsDashboard = godo.Dashboard{
	UUID:        "7c9e6679-7425-40de-944b-e07fc1f90ae7",
	Name:        "Droplet CPU",
	Description: "CPU utilization across production Droplets.",
	Spec:        json.RawMessage(testInsightsDashboardSpec),
	CreatedAt:   time.Date(2026, 10, 29, 10, 21, 44, 0, time.UTC),
	UpdatedAt:   time.Date(2026, 10, 29, 10, 21, 44, 0, time.UTC),
}

func TestInsightsDashboardsCommand(t *testing.T) {
	cmd := insightsDashboards()
	assert.Equal(t, "dashboard", cmd.Name())
	assert.Contains(t, cmd.Aliases, "dashboards")
	assertCommandNames(t, cmd, "list", "get", "create", "update", "delete", "schema")
}

func TestRunInsightsDashboardList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		config.Doit.Set(config.NS, doctl.ArgInsightsPage, 2)
		config.Doit.Set(config.NS, doctl.ArgInsightsPerPage, 50)
		config.Doit.Set(config.NS, doctl.ArgInsightsDashboardSort, " Updated_At ")
		config.Doit.Set(config.NS, doctl.ArgInsightsDashboardSortDirection, "ASC")

		tm.insights.EXPECT().ListDashboards(&godo.DashboardListOptions{
			Page:          2,
			PerPage:       50,
			Sort:          godo.InsightsDashboardSortUpdatedAt,
			SortDirection: godo.InsightsDashboardSortAscending,
		}).Return(&godo.DashboardList{
			Dashboards: []godo.Dashboard{testInsightsDashboard},
			Pagination: &godo.DashboardsPagination{Page: 2, Pages: 3, PerPage: 50, Total: 120},
		}, nil)

		require.NoError(t, RunInsightsDashboardList(config))
		assert.Contains(t, buf.String(), testInsightsDashboard.UUID)
		assert.Contains(t, buf.String(), "Droplet CPU")
		assert.Contains(t, buf.String(), "2026-10-29T10:21:44Z")
	})
}

func TestRunInsightsDashboardListJSON(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		tm.insights.EXPECT().ListDashboards(&godo.DashboardListOptions{}).Return(&godo.DashboardList{
			Dashboards: []godo.Dashboard{testInsightsDashboard},
			Pagination: &godo.DashboardsPagination{Page: 1, Pages: 1, PerPage: 20, Total: 1},
		}, nil)

		require.NoError(t, RunInsightsDashboardList(config))
		assert.JSONEq(t, `{
			"dashboards": [{
				"uuid": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
				"name": "Droplet CPU",
				"description": "CPU utilization across production Droplets.",
				"spec": {"kind": "Dashboard", "spec": {"display": {"name": "Droplet CPU"}}},
				"created_at": "2026-10-29T10:21:44Z",
				"updated_at": "2026-10-29T10:21:44Z"
			}],
			"pagination": {"page": 1, "pages": 1, "per_page": 20, "total": 1}
		}`, buf.String())
	})
}

func TestRunInsightsDashboardListEmptyJSON(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		tm.insights.EXPECT().ListDashboards(&godo.DashboardListOptions{}).Return(&godo.DashboardList{
			Pagination: &godo.DashboardsPagination{Page: 1, PerPage: 20},
		}, nil)

		require.NoError(t, RunInsightsDashboardList(config))
		assert.JSONEq(t, `{"pagination": {"page": 1, "per_page": 20}}`, buf.String())
	})
}

func TestRunInsightsDashboardListValidation(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		value   any
		wantErr string
	}{
		{name: "page zero", flag: doctl.ArgInsightsPage, value: 0, wantErr: "--page must be at least 1"},
		{name: "per page too large", flag: doctl.ArgInsightsPerPage, value: 201, wantErr: "--per-page must be between 1 and 200"},
		{name: "invalid sort", flag: doctl.ArgInsightsDashboardSort, value: "description", wantErr: `invalid --sort "description": must be "name", "created_at", or "updated_at"`},
		{name: "invalid direction", flag: doctl.ArgInsightsDashboardSortDirection, value: "sideways", wantErr: `invalid --sort-direction "sideways": must be "asc" or "desc"`},
		// The server ignores sort_direction unless sort is set; fail loudly
		// instead of silently returning created_at desc.
		{name: "direction without sort", flag: doctl.ArgInsightsDashboardSortDirection, value: "asc", wantErr: "--sort-direction requires --sort"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				config.Doit.Set(config.NS, tt.flag, tt.value)
				assert.ErrorContains(t, RunInsightsDashboardList(config), tt.wantErr)
			})
		})
	}
}

func TestRunInsightsDashboardGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		config.Args = []string{" " + testInsightsDashboard.UUID + " "}
		tm.insights.EXPECT().GetDashboard(testInsightsDashboard.UUID).Return(&testInsightsDashboard, nil)

		require.NoError(t, RunInsightsDashboardGet(config))
		assert.JSONEq(t, `{"dashboard": {
			"uuid": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
			"name": "Droplet CPU",
			"description": "CPU utilization across production Droplets.",
			"spec": {"kind": "Dashboard", "spec": {"display": {"name": "Droplet CPU"}}},
			"created_at": "2026-10-29T10:21:44Z",
			"updated_at": "2026-10-29T10:21:44Z"
		}}`, buf.String())
	})
}

func TestInsightsDashboardUUIDValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "missing", wantErr: "missing required arguments"},
		{name: "too many", args: []string{"a", "b"}, wantErr: "contains unsupported arguments"},
		{name: "blank", args: []string{" "}, wantErr: "a dashboard UUID is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				config.Args = tt.args
				assert.ErrorContains(t, RunInsightsDashboardGet(config), tt.wantErr)
				assert.ErrorContains(t, RunInsightsDashboardUpdate(config), tt.wantErr)
				assert.ErrorContains(t, RunInsightsDashboardDelete(config), tt.wantErr)
			})
		})
	}
}

func TestRunInsightsDashboardCreateFromFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		path := filepath.Join(t.TempDir(), "dashboard.json")
		require.NoError(t, os.WriteFile(path, []byte(testInsightsDashboardSpec+"\n"), 0o600))
		config.Doit.Set(config.NS, doctl.ArgInsightsDashboardSpecFile, path)

		tm.insights.EXPECT().CreateDashboard(&godo.DashboardRequest{
			Spec: json.RawMessage(testInsightsDashboardSpec),
		}).Return(&testInsightsDashboard, nil)

		require.NoError(t, RunInsightsDashboardCreate(config))
		assert.Contains(t, buf.String(), testInsightsDashboard.UUID)
	})
}

func TestRunInsightsDashboardCreateFromStdin(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		prev := insightsStdin
		insightsStdin = strings.NewReader(testInsightsDashboardSpec)
		t.Cleanup(func() { insightsStdin = prev })
		config.Doit.Set(config.NS, doctl.ArgInsightsDashboardSpecFile, "-")

		tm.insights.EXPECT().CreateDashboard(&godo.DashboardRequest{
			Spec: json.RawMessage(testInsightsDashboardSpec),
		}).Return(&testInsightsDashboard, nil)

		require.NoError(t, RunInsightsDashboardCreate(config))
	})
}

func TestRunInsightsDashboardCreateFromYAML(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		path := filepath.Join(t.TempDir(), "dashboard.yaml")
		require.NoError(t, os.WriteFile(path, []byte("kind: Dashboard\nspec:\n  display:\n    name: Droplet CPU\n"), 0o600))
		config.Doit.Set(config.NS, doctl.ArgInsightsDashboardSpecFile, path)

		tm.insights.EXPECT().CreateDashboard(&godo.DashboardRequest{
			Spec: json.RawMessage(`{"kind":"Dashboard","spec":{"display":{"name":"Droplet CPU"}}}`),
		}).Return(&testInsightsDashboard, nil)

		require.NoError(t, RunInsightsDashboardCreate(config))
	})
}

func TestRunInsightsDashboardCreateValidation(t *testing.T) {
	tests := []struct {
		name    string
		spec    *string
		wantErr string
	}{
		{name: "spec file unset", wantErr: "--spec-file is required"},
		{name: "spec file blank", spec: godo.PtrTo("  "), wantErr: "--spec-file is required"},
		{name: "array", spec: godo.PtrTo(`[]`), wantErr: "--spec-file: spec file must contain a single JSON object"},
		{name: "double encoded", spec: godo.PtrTo(`"{\"kind\":\"Dashboard\"}"`), wantErr: "--spec-file: spec file must contain a single JSON object"},
		{name: "invalid JSON", spec: godo.PtrTo(`{"kind":`), wantErr: "--spec-file: parsing spec file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				if tt.spec != nil {
					path := filepath.Join(t.TempDir(), "dashboard.json")
					require.NoError(t, os.WriteFile(path, []byte(*tt.spec), 0o600))
					if strings.TrimSpace(*tt.spec) == "" {
						path = *tt.spec
					}
					config.Doit.Set(config.NS, doctl.ArgInsightsDashboardSpecFile, path)
				}
				assert.ErrorContains(t, RunInsightsDashboardCreate(config), tt.wantErr)
			})
		})
	}
}

func TestRunInsightsDashboardCreateMissingFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInsightsDashboardSpecFile, filepath.Join(t.TempDir(), "missing.json"))
		assert.ErrorContains(t, RunInsightsDashboardCreate(config), "--spec-file: reading spec file")
	})
}

func TestRunInsightsDashboardUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		config.Args = []string{testInsightsDashboard.UUID}
		path := filepath.Join(t.TempDir(), "dashboard.json")
		require.NoError(t, os.WriteFile(path, []byte(testInsightsDashboardSpec), 0o600))
		config.Doit.Set(config.NS, doctl.ArgInsightsDashboardSpecFile, path)

		tm.insights.EXPECT().UpdateDashboard(testInsightsDashboard.UUID, &godo.DashboardRequest{
			Spec: json.RawMessage(testInsightsDashboardSpec),
		}).Return(&testInsightsDashboard, nil)

		require.NoError(t, RunInsightsDashboardUpdate(config))
		assert.Contains(t, buf.String(), `"uuid": "7c9e6679-7425-40de-944b-e07fc1f90ae7"`)
	})
}

func TestRunInsightsDashboardUpdateRequiresSpecFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Args = []string{testInsightsDashboard.UUID}
		assert.ErrorContains(t, RunInsightsDashboardUpdate(config), "--spec-file is required")
	})
}

func TestRunInsightsDashboardDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = []string{testInsightsDashboard.UUID}
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		tm.insights.EXPECT().DeleteDashboard(testInsightsDashboard.UUID).Return(nil)

		require.NoError(t, RunInsightsDashboardDelete(config))
	})
}

func TestRunInsightsDashboardDeleteRequiresConfirmation(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Args = []string{testInsightsDashboard.UUID}

		require.ErrorIs(t, RunInsightsDashboardDelete(config), errConfirmationRequired)
	})
}

func TestRunInsightsDashboardSchema(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		tm.insights.EXPECT().GetDashboardSchema().Return(json.RawMessage(
			`{"$schema":"https://json-schema.org/draft/2020-12/schema","title":"Dashboard","type":"object"}`,
		), nil)

		require.NoError(t, RunInsightsDashboardSchema(config))
		assert.Equal(t, `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Dashboard",
  "type": "object"
}
`, buf.String())
	})
}

func TestRunInsightsDashboardSchemaRejectsArgs(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Args = []string{"extra"}
		assert.ErrorContains(t, RunInsightsDashboardSchema(config), "contains unsupported arguments")
	})
}
