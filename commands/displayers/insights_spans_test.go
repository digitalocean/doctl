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
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsightsSpans(t *testing.T) {
	started := time.Date(2026, 9, 1, 1, 2, 3, 400000000, time.UTC)
	displayer := &InsightsSpans{Response: &godo.SpansSearchResponse{
		Data: []godo.SpanRecord{{
			TraceID: "trace-1", SpanID: "span-1", StartTime: started,
			DurationNs: 1500000, Name: "GET /", Kind: "server",
			StatusCode: "OK", ServiceName: "api",
		}},
		Pagination: &godo.SpansPaginationResponse{HasMore: true},
	}}

	rows := displayer.KV()
	require.Len(t, rows, 1)
	assert.Equal(t, "2026-09-01T01:02:03.4Z", rows[0]["StartTime"])
	assert.Equal(t, "1.5ms", rows[0]["Duration"])

	var out bytes.Buffer
	require.NoError(t, displayer.JSON(&out))
	assert.JSONEq(t, `{
		"data":[{"trace_id":"trace-1","span_id":"span-1","start_time":"2026-09-01T01:02:03.4Z","end_time":"0001-01-01T00:00:00Z","duration_ns":1500000,"name":"GET /","kind":"server","status_code":"OK","service_name":"api"}],
		"pagination":{"has_more":true}
	}`, out.String())
}

func TestInsightsSpansEmptyJSON(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, (&InsightsSpans{Response: &godo.SpansSearchResponse{}}).JSON(&out))
	assert.JSONEq(t, `{"data":[]}`, out.String())
}
