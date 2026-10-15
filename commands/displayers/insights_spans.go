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
	"time"

	"github.com/digitalocean/godo"
)

// InsightsSpans displays spans returned by an Insights search.
type InsightsSpans struct {
	Response *godo.SpansSearchResponse
}

var _ Displayable = &InsightsSpans{}

func (d *InsightsSpans) JSON(out io.Writer) error {
	response := struct {
		Data       []godo.SpanRecord             `json:"data"`
		Pagination *godo.SpansPaginationResponse `json:"pagination,omitempty"`
	}{Data: []godo.SpanRecord{}}
	if d.Response != nil {
		response.Pagination = d.Response.Pagination
		if d.Response.Data != nil {
			response.Data = d.Response.Data
		}
	}
	return writeJSON(response, out)
}

func (d *InsightsSpans) Cols() []string {
	return []string{"TraceID", "SpanID", "ParentSpanID", "StartTime", "Duration", "Name", "Kind", "Status", "Service"}
}

func (d *InsightsSpans) ColMap() map[string]string {
	return map[string]string{
		"TraceID":      "Trace ID",
		"SpanID":       "Span ID",
		"ParentSpanID": "Parent Span ID",
		"StartTime":    "Start Time",
		"Duration":     "Duration",
		"Name":         "Name",
		"Kind":         "Kind",
		"Status":       "Status",
		"Service":      "Service",
	}
}

func (d *InsightsSpans) KV() []map[string]any {
	rows := []map[string]any{}
	if d.Response == nil {
		return rows
	}
	for _, span := range d.Response.Data {
		rows = append(rows, map[string]any{
			"TraceID":      span.TraceID,
			"SpanID":       span.SpanID,
			"ParentSpanID": span.ParentSpanID,
			"StartTime":    formatInsightsTimestamp(span.StartTime),
			"Duration":     formatInsightsDuration(span.DurationNs),
			"Name":         span.Name,
			"Kind":         span.Kind,
			"Status":       span.StatusCode,
			"Service":      span.ServiceName,
		})
	}
	return rows
}

func formatInsightsTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func formatInsightsDuration(nanoseconds int64) string {
	return time.Duration(nanoseconds).String()
}
