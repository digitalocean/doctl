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
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsightsTraces(t *testing.T) {
	displayer := &InsightsTraces{Response: &godo.TracesSearchResponse{
		Data: []godo.TraceSummary{{
			TraceID: "trace-1", DurationNs: 2000000000, SpanNames: []string{"root", "child"},
		}},
		Pagination: &godo.TracesPaginationResponse{HasMore: false},
	}}

	rows := displayer.KV()
	require.Len(t, rows, 1)
	assert.Equal(t, "2s", rows[0]["Duration"])
	assert.Equal(t, "root, child", rows[0]["SpanNames"])

	var out bytes.Buffer
	require.NoError(t, displayer.JSON(&out))
	assert.JSONEq(t, `{
		"data":[{"traceId":"trace-1","started":"0001-01-01T00:00:00Z","spanNames":["root","child"],"durationNs":2000000000}],
		"pagination":{"hasMore":false}
	}`, out.String())
}

func TestInsightsTrace(t *testing.T) {
	displayer := &InsightsTrace{Trace: &godo.Trace{
		TraceID: "trace-1",
		Spans: []godo.TraceSpan{{
			TraceID: "trace-1", SpanID: "span-1", DurationNs: 750000,
		}},
		Truncated: true,
	}}

	rows := displayer.KV()
	require.Len(t, rows, 1)
	assert.Equal(t, "750µs", rows[0]["Duration"])

	var out bytes.Buffer
	require.NoError(t, displayer.JSON(&out))
	assert.JSONEq(t, `{
		"traceId":"trace-1",
		"spans":[{"traceId":"trace-1","spanId":"span-1","startTime":"0001-01-01T00:00:00Z","endTime":"0001-01-01T00:00:00Z","durationNs":750000}],
		"truncated":true
	}`, out.String())
}

func TestInsightsTraceEmptySpansJSON(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, (&InsightsTrace{Trace: &godo.Trace{TraceID: "trace-1"}}).JSON(&out))
	assert.JSONEq(t, `{"traceId":"trace-1","spans":[]}`, out.String())
}
