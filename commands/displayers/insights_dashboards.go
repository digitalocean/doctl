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
	"encoding/json"
	"io"
	"time"

	"github.com/digitalocean/godo"
)

var insightsDashboardColumns = []string{"UUID", "Name", "Description", "CreatedAt", "UpdatedAt"}

var insightsDashboardColumnMap = map[string]string{
	"UUID":        "UUID",
	"Name":        "Name",
	"Description": "Description",
	"CreatedAt":   "Created At",
	"UpdatedAt":   "Updated At",
}

// InsightsDashboards displays a page of dashboards.
type InsightsDashboards struct {
	Response *godo.DashboardList
}

var _ Displayable = &InsightsDashboards{}

func (d *InsightsDashboards) JSON(out io.Writer) error {
	response := struct {
		Dashboards []insightsDashboardJSON           `json:"dashboards,omitempty"`
		Pagination *insightsDashboardsPaginationJSON `json:"pagination,omitempty"`
	}{}
	if d.Response != nil {
		for _, dashboard := range d.Response.Dashboards {
			response.Dashboards = append(response.Dashboards, insightsDashboardForJSON(dashboard))
		}
		if d.Response.Pagination != nil {
			response.Pagination = insightsDashboardsPaginationForJSON(d.Response.Pagination)
		}
	}
	return writeJSON(response, out)
}

func (d *InsightsDashboards) Cols() []string {
	return insightsDashboardColumns
}

func (d *InsightsDashboards) ColMap() map[string]string {
	return insightsDashboardColumnMap
}

func (d *InsightsDashboards) KV() []map[string]any {
	if d.Response == nil {
		return []map[string]any{}
	}
	return insightsDashboardRows(d.Response.Dashboards)
}

// InsightsDashboard displays one dashboard.
type InsightsDashboard struct {
	Dashboard *godo.Dashboard
}

var _ Displayable = &InsightsDashboard{}

func (d *InsightsDashboard) JSON(out io.Writer) error {
	response := struct {
		Dashboard *insightsDashboardJSON `json:"dashboard"`
	}{}
	if d.Dashboard != nil {
		dashboard := insightsDashboardForJSON(*d.Dashboard)
		response.Dashboard = &dashboard
	}
	return writeJSON(response, out)
}

func (d *InsightsDashboard) Cols() []string {
	return insightsDashboardColumns
}

func (d *InsightsDashboard) ColMap() map[string]string {
	return insightsDashboardColumnMap
}

func (d *InsightsDashboard) KV() []map[string]any {
	if d.Dashboard == nil {
		return []map[string]any{}
	}
	return insightsDashboardRows([]godo.Dashboard{*d.Dashboard})
}

func insightsDashboardRows(dashboards []godo.Dashboard) []map[string]any {
	rows := make([]map[string]any, 0, len(dashboards))
	for _, dashboard := range dashboards {
		rows = append(rows, map[string]any{
			"UUID":        dashboard.UUID,
			"Name":        dashboard.Name,
			"Description": dashboard.Description,
			"CreatedAt":   formatInsightsTime(dashboard.CreatedAt),
			"UpdatedAt":   formatInsightsTime(dashboard.UpdatedAt),
		})
	}
	return rows
}

// insightsDashboardJSON renders a dashboard with its spec embedded as a JSON
// object rather than the string-encoded form the API transports, so the
// output of `get` can be fed back to `--spec-file` unchanged. Name and
// Description are omitted when empty, matching the API.
type insightsDashboardJSON struct {
	UUID        string          `json:"uuid"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Spec        json.RawMessage `json:"spec"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func insightsDashboardForJSON(dashboard godo.Dashboard) insightsDashboardJSON {
	return insightsDashboardJSON{
		UUID:        dashboard.UUID,
		Name:        dashboard.Name,
		Description: dashboard.Description,
		Spec:        dashboard.Spec,
		CreatedAt:   dashboard.CreatedAt,
		UpdatedAt:   dashboard.UpdatedAt,
	}
}

type insightsDashboardsPaginationJSON struct {
	Page    int  `json:"page"`
	Pages   *int `json:"pages,omitempty"`
	PerPage int  `json:"per_page"`
	Total   *int `json:"total,omitempty"`
}

func insightsDashboardsPaginationForJSON(pagination *godo.DashboardsPagination) *insightsDashboardsPaginationJSON {
	result := &insightsDashboardsPaginationJSON{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}
	if pagination.Pages > 0 {
		result.Pages = &pagination.Pages
	}
	if pagination.Total > 0 {
		result.Total = &pagination.Total
	}
	return result
}
