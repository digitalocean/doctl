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

package displayers

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDisplayDashboard = godo.Dashboard{
	UUID:        "7c9e6679-7425-40de-944b-e07fc1f90ae7",
	Name:        "Droplet CPU",
	Description: "CPU utilization across production Droplets.",
	Spec:        json.RawMessage(`{"kind":"Dashboard","spec":{"display":{"name":"Droplet CPU","description":"CPU utilization across production Droplets."}}}`),
	CreatedAt:   time.Date(2026, 10, 29, 10, 21, 44, 0, time.UTC),
	UpdatedAt:   time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC),
}

func TestInsightsDashboardsKV(t *testing.T) {
	displayer := &InsightsDashboards{Response: &godo.DashboardList{
		Dashboards: []godo.Dashboard{
			testDisplayDashboard,
			{UUID: "3f2504e0-4f89-41d3-9a0c-0305e82c3301", Name: "LB traffic", Spec: json.RawMessage(`{}`)},
		},
		Pagination: &godo.DashboardsPagination{Page: 1, Pages: 1, PerPage: 20, Total: 2},
	}}

	assert.Equal(t, []string{"UUID", "Name", "Description", "CreatedAt", "UpdatedAt"}, displayer.Cols())
	assert.Equal(t, "Created At", displayer.ColMap()["CreatedAt"])

	rows := displayer.KV()
	require.Len(t, rows, 2)
	assert.Equal(t, map[string]any{
		"UUID":        "7c9e6679-7425-40de-944b-e07fc1f90ae7",
		"Name":        "Droplet CPU",
		"Description": "CPU utilization across production Droplets.",
		"CreatedAt":   "2026-10-29T10:21:44Z",
		"UpdatedAt":   "2026-10-30T08:00:00Z",
	}, rows[0])
	assert.Equal(t, "", rows[1]["Description"])
	assert.Equal(t, "", rows[1]["CreatedAt"])
}

func TestInsightsDashboardsJSONEmbedsSpecAsObject(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, (&InsightsDashboards{Response: &godo.DashboardList{
		Dashboards: []godo.Dashboard{testDisplayDashboard},
		Pagination: &godo.DashboardsPagination{Page: 1, Pages: 1, PerPage: 20, Total: 1},
	}}).JSON(&out))

	assert.JSONEq(t, `{
		"dashboards": [{
			"uuid": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
			"name": "Droplet CPU",
			"description": "CPU utilization across production Droplets.",
			"spec": {"kind":"Dashboard","spec":{"display":{"name":"Droplet CPU","description":"CPU utilization across production Droplets."}}},
			"created_at": "2026-10-29T10:21:44Z",
			"updated_at": "2026-10-30T08:00:00Z"
		}],
		"pagination": {"page": 1, "pages": 1, "per_page": 20, "total": 1}
	}`, out.String())
}

func TestInsightsDashboardJSONOmitsEmptyName(t *testing.T) {
	// The API omits name when spec.display.name is an empty string.
	var out bytes.Buffer
	require.NoError(t, (&InsightsDashboard{Dashboard: &godo.Dashboard{
		UUID:      "3f2504e0-4f89-41d3-9a0c-0305e82c3301",
		Spec:      json.RawMessage(`{"kind":"Dashboard","spec":{"display":{"name":""}}}`),
		CreatedAt: time.Date(2026, 10, 27, 16, 9, 31, 0, time.UTC),
		UpdatedAt: time.Date(2026, 10, 27, 16, 9, 31, 0, time.UTC),
	}}).JSON(&out))

	assert.JSONEq(t, `{"dashboard":{
		"uuid":"3f2504e0-4f89-41d3-9a0c-0305e82c3301",
		"spec":{"kind":"Dashboard","spec":{"display":{"name":""}}},
		"created_at":"2026-10-27T16:09:31Z",
		"updated_at":"2026-10-27T16:09:31Z"
	}}`, out.String())
}

func TestInsightsDashboardsEmptyJSON(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, (&InsightsDashboards{Response: &godo.DashboardList{
		Pagination: &godo.DashboardsPagination{Page: 1, PerPage: 20},
	}}).JSON(&out))
	assert.JSONEq(t, `{"pagination":{"page":1,"per_page":20}}`, out.String())
}

func TestInsightsDashboardsNilResponse(t *testing.T) {
	displayer := &InsightsDashboards{}
	assert.Empty(t, displayer.KV())

	var out bytes.Buffer
	require.NoError(t, displayer.JSON(&out))
	assert.JSONEq(t, `{}`, out.String())
}

func TestInsightsDashboardJSON(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, (&InsightsDashboard{Dashboard: &godo.Dashboard{
		UUID:      "3f2504e0-4f89-41d3-9a0c-0305e82c3301",
		Name:      "LB traffic",
		Spec:      json.RawMessage(`{"kind":"Dashboard"}`),
		CreatedAt: time.Date(2026, 10, 27, 16, 9, 31, 0, time.UTC),
		UpdatedAt: time.Date(2026, 10, 27, 16, 9, 31, 0, time.UTC),
	}}).JSON(&out))

	assert.JSONEq(t, `{"dashboard":{
		"uuid":"3f2504e0-4f89-41d3-9a0c-0305e82c3301",
		"name":"LB traffic",
		"spec":{"kind":"Dashboard"},
		"created_at":"2026-10-27T16:09:31Z",
		"updated_at":"2026-10-27T16:09:31Z"
	}}`, out.String())
}

func TestInsightsDashboardKV(t *testing.T) {
	rows := (&InsightsDashboard{Dashboard: &testDisplayDashboard}).KV()
	require.Len(t, rows, 1)
	assert.Equal(t, "Droplet CPU", rows[0]["Name"])

	assert.Empty(t, (&InsightsDashboard{}).KV())
}
