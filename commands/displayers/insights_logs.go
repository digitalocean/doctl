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
	"time"

	"github.com/digitalocean/godo"
)

// InsightsLogs displays the records returned by an Insights logs search.
type InsightsLogs struct {
	Response *godo.LogsSearchResponse
}

var _ Displayable = &InsightsLogs{}

func (l *InsightsLogs) JSON(out io.Writer) error {
	r := struct {
		Data       []godo.InsightsLogRecord     `json:"data"`
		Pagination *godo.LogsPaginationResponse `json:"pagination,omitempty"`
	}{Data: []godo.InsightsLogRecord{}}
	if l.Response != nil {
		r.Pagination = l.Response.Pagination
		if l.Response.Data != nil {
			r.Data = l.Response.Data
		}
	}
	return writeJSON(r, out)
}

func (l *InsightsLogs) Cols() []string {
	return []string{"Timestamp", "Severity", "Service", "Body"}
}

func (l *InsightsLogs) ColMap() map[string]string {
	return map[string]string{
		"Timestamp": "Timestamp",
		"Severity":  "Severity",
		"Service":   "Service",
		"Body":      "Body",
		"TraceID":   "Trace ID",
		"SpanID":    "Span ID",
	}
}

var logBodyReplacer = strings.NewReplacer("\r\n", `\n`, "\n", `\n`, "\r", `\r`, "\t", " ")

func (l *InsightsLogs) KV() []map[string]any {
	out := []map[string]any{}
	if l.Response == nil {
		return out
	}
	for _, r := range l.Response.Data {
		ts := ""
		if !r.Timestamp.IsZero() {
			ts = r.Timestamp.UTC().Format(time.RFC3339Nano)
		}
		out = append(out, map[string]any{
			"Timestamp": ts,
			"Severity":  r.SeverityText,
			"Service":   r.ServiceName,
			"Body":      logBodyReplacer.Replace(r.Body),
			"TraceID":   r.TraceID,
			"SpanID":    r.SpanID,
		})
	}
	return out
}
