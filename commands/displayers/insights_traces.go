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
	"strings"

	"github.com/digitalocean/godo"
)

// InsightsTraces displays trace summaries returned by an Insights search.
type InsightsTraces struct {
	Response *godo.TracesSearchResponse
}

var _ Displayable = &InsightsTraces{}

func (d *InsightsTraces) JSON(out io.Writer) error {
	response := struct {
		Data       []godo.TraceSummary            `json:"data"`
		Pagination *godo.TracesPaginationResponse `json:"pagination,omitempty"`
	}{Data: []godo.TraceSummary{}}
	if d.Response != nil {
		response.Pagination = d.Response.Pagination
		if d.Response.Data != nil {
			response.Data = d.Response.Data
		}
	}
	return writeJSON(response, out)
}

func (d *InsightsTraces) Cols() []string {
	return []string{"TraceID", "Started", "Service", "Duration", "ErrorCount", "Status", "SpanCount", "SpanNames"}
}

func (d *InsightsTraces) ColMap() map[string]string {
	return map[string]string{
		"TraceID":    "Trace ID",
		"Started":    "Started",
		"Service":    "Service",
		"Duration":   "Duration",
		"ErrorCount": "Error Count",
		"Status":     "Status",
		"SpanCount":  "Span Count",
		"SpanNames":  "Span Names",
	}
}

func (d *InsightsTraces) KV() []map[string]any {
	rows := []map[string]any{}
	if d.Response == nil {
		return rows
	}
	for _, trace := range d.Response.Data {
		rows = append(rows, map[string]any{
			"TraceID":    trace.TraceID,
			"Started":    formatInsightsTimestamp(trace.Started),
			"Service":    trace.ServiceName,
			"Duration":   formatInsightsDuration(trace.DurationNs),
			"ErrorCount": trace.ErrorCount,
			"Status":     trace.Status,
			"SpanCount":  trace.SpanCount,
			"SpanNames":  strings.Join(trace.SpanNames, ", "),
		})
	}
	return rows
}

// InsightsTrace displays all spans belonging to a trace.
type InsightsTrace struct {
	Trace *godo.Trace
}

var _ Displayable = &InsightsTrace{}

func (d *InsightsTrace) JSON(out io.Writer) error {
	trace := struct {
		TraceID   string           `json:"traceId"`
		Spans     []godo.TraceSpan `json:"spans"`
		Truncated bool             `json:"truncated,omitempty"`
	}{Spans: []godo.TraceSpan{}}
	if d.Trace != nil {
		trace.TraceID = d.Trace.TraceID
		trace.Truncated = d.Trace.Truncated
		if d.Trace.Spans != nil {
			trace.Spans = d.Trace.Spans
		}
	}
	return writeJSON(trace, out)
}

func (d *InsightsTrace) Cols() []string {
	return []string{"SpanID", "ParentSpanID", "StartTime", "Duration", "Name", "Kind", "Status", "Service"}
}

func (d *InsightsTrace) ColMap() map[string]string {
	return map[string]string{
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

func (d *InsightsTrace) KV() []map[string]any {
	rows := []map[string]any{}
	if d.Trace == nil {
		return rows
	}
	for _, span := range d.Trace.Spans {
		rows = append(rows, map[string]any{
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
