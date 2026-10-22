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
	"io"
	"strconv"

	"github.com/digitalocean/godo"
)

var insightsResourceColumns = []string{
	"ResourceURN", "ProductType", "Region",
	"Metrics", "MetricsEffectiveAt",
	"Logs", "LogsEffectiveAt",
	"Traces", "TracesEffectiveAt",
}

var insightsResourceColumnMap = map[string]string{
	"ResourceURN":        "Resource URN",
	"ProductType":        "Product Type",
	"Region":             "Region",
	"Metrics":            "Metrics",
	"MetricsEffectiveAt": "Metrics Effective At",
	"Logs":               "Logs",
	"LogsEffectiveAt":    "Logs Effective At",
	"Traces":             "Traces",
	"TracesEffectiveAt":  "Traces Effective At",
}

// InsightsResources displays a page of resource signal enablement.
type InsightsResources struct {
	Response *godo.InsightsResourceList
}

var _ Displayable = &InsightsResources{}

func (d *InsightsResources) JSON(out io.Writer) error {
	response := struct {
		Resources  []godo.InsightsResource          `json:"resources,omitempty"`
		Pagination *insightsResourcesPaginationJSON `json:"pagination,omitempty"`
	}{}
	if d.Response != nil {
		response.Resources = d.Response.Resources
		if d.Response.Pagination != nil {
			response.Pagination = insightsResourcesPaginationForJSON(d.Response.Pagination)
		}
	}
	return writeJSON(response, out)
}

func (d *InsightsResources) Cols() []string {
	return insightsResourceColumns
}

func (d *InsightsResources) ColMap() map[string]string {
	return insightsResourceColumnMap
}

func (d *InsightsResources) KV() []map[string]any {
	if d.Response == nil {
		return []map[string]any{}
	}
	return insightsResourceRows(d.Response.Resources)
}

// InsightsResource displays one resource's signal enablement.
type InsightsResource struct {
	Resource *godo.InsightsResource
}

var _ Displayable = &InsightsResource{}

func (d *InsightsResource) JSON(out io.Writer) error {
	response := struct {
		Resource *godo.InsightsResource `json:"resource"`
	}{Resource: d.Resource}
	return writeJSON(response, out)
}

func (d *InsightsResource) Cols() []string {
	return insightsResourceColumns
}

func (d *InsightsResource) ColMap() map[string]string {
	return insightsResourceColumnMap
}

func (d *InsightsResource) KV() []map[string]any {
	if d.Resource == nil {
		return []map[string]any{}
	}
	return insightsResourceRows([]godo.InsightsResource{*d.Resource})
}

func insightsResourceRows(resources []godo.InsightsResource) []map[string]any {
	rows := make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		metrics, metricsEffectiveAt := insightsSignalValues(resource.Metrics)
		logs, logsEffectiveAt := insightsSignalValues(resource.Logs)
		traces, tracesEffectiveAt := insightsSignalValues(resource.Traces)
		rows = append(rows, map[string]any{
			"ResourceURN":        resource.ResourceURN,
			"ProductType":        resource.ProductType,
			"Region":             resource.Region,
			"Metrics":            metrics,
			"MetricsEffectiveAt": metricsEffectiveAt,
			"Logs":               logs,
			"LogsEffectiveAt":    logsEffectiveAt,
			"Traces":             traces,
			"TracesEffectiveAt":  tracesEffectiveAt,
		})
	}
	return rows
}

func insightsSignalValues(signal *godo.InsightsResourceSignalState) (enabled, effectiveAt string) {
	if signal == nil {
		return "", ""
	}
	return strconv.FormatBool(signal.Enabled), formatInsightsTime(signal.EffectiveAt)
}

type insightsResourcesPaginationJSON struct {
	Page    int  `json:"page"`
	Pages   *int `json:"pages,omitempty"`
	PerPage int  `json:"per_page"`
	Total   *int `json:"total,omitempty"`
}

func insightsResourcesPaginationForJSON(pagination *godo.InsightsResourcesPagination) *insightsResourcesPaginationJSON {
	result := &insightsResourcesPaginationJSON{
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
