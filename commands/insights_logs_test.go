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
	"go.uber.org/mock/gomock"
)

const testLogsServiceFilterJSON = `{"condition":{"field":{"name":"service.name"},"operator":"FILTER_OPERATOR_EQ","value":{"string_value":"api"}}}`

var testLogsResponse = &godo.LogsSearchResponse{
	Data: []godo.InsightsLogRecord{{
		Timestamp:    time.Date(2026, 9, 1, 0, 0, 1, 500000000, time.UTC),
		SeverityText: "ERROR",
		ServiceName:  "api",
		Body:         "request timeout\nretrying",
		TraceID:      "4bf92f3577b34da6a3ce929d0e0e4736",
	}},
	Pagination: &godo.LogsPaginationResponse{HasMore: true, NextCursor: "next-page"},
}

func testLogsServiceFilter() godo.LogsFilterExpression {
	api := "api"
	return godo.LogsFilterExpression{Condition: &godo.LogsFilterCondition{
		Field:    godo.LogsFieldRef{Name: "service.name"},
		Operator: "FILTER_OPERATOR_EQ",
		Value:    &godo.LogsFilterValue{StringValue: &api},
	}}
}

func writeInsightsTestFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.json")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func captureInsightsStderr(t *testing.T) *bytes.Buffer {
	t.Helper()
	prev := insightsStderr
	buf := &bytes.Buffer{}
	insightsStderr = buf
	t.Cleanup(func() { insightsStderr = prev })
	return buf
}

func TestInsightsLogsCommand(t *testing.T) {
	cmd := insightsLogs()
	assert.Equal(t, "logs", cmd.Name())
	assertCommandNames(t, cmd, "search")
}

func TestRunInsightsLogsSearchDefaults(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		captureInsightsStderr(t)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		tm.insights.EXPECT().SearchLogs("nyc3", &godo.LogsSearchRequest{
			TimeRange: godo.LogsTimeRange{
				From: godo.LogsTimeInstant{Relative: "1h"},
				To:   godo.LogsTimeInstant{Relative: "now"},
			},
		}).Return(&godo.LogsSearchResponse{}, nil)

		require.NoError(t, RunInsightsLogsSearch(config))
	})
}

func TestRunInsightsLogsSearchFromFlags(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		stderr := captureInsightsStderr(t)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsFrom, "2026-09-01T00:00:00Z")
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsTo, "now-5m")
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsFilter, testLogsServiceFilterJSON)
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsText, "timeout")
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsOrderBy, []string{"timestamp:DESC"})
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsLimit, 50)
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsCursor, "abc")

		tm.insights.EXPECT().SearchLogs("nyc3", &godo.LogsSearchRequest{
			TimeRange: godo.LogsTimeRange{
				From: godo.LogsTimeInstant{Absolute: "2026-09-01T00:00:00Z"},
				To:   godo.LogsTimeInstant{Relative: "now-5m"},
			},
			Filter: &godo.LogsFilterExpression{And: &godo.LogsFilterGroup{Expressions: []godo.LogsFilterExpression{
				testLogsServiceFilter(),
				{TextSearch: &godo.LogsTextSearch{Query: "timeout"}},
			}}},
			OrderBy:    []godo.LogsOrderBy{{Field: godo.LogsFieldRef{Name: "timestamp"}, Direction: "SORT_DIRECTION_DESC"}},
			Pagination: &godo.LogsPaginationRequest{Limit: 50, Cursor: "abc"},
		}).Return(testLogsResponse, nil)

		require.NoError(t, RunInsightsLogsSearch(config))

		out := buf.String()
		assert.Contains(t, out, "Timestamp")
		assert.Contains(t, out, "2026-09-01T00:00:01.5Z")
		assert.Contains(t, out, "ERROR")
		assert.Contains(t, out, `request timeout\nretrying`)
		assert.NotContains(t, out, "next-page")
		assert.Contains(t, stderr.String(), "--cursor next-page")
	})
}

func TestRunInsightsLogsSearchTextOnly(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		captureInsightsStderr(t)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsFrom, "15m")
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsText, "panic")

		tm.insights.EXPECT().SearchLogs("nyc3", &godo.LogsSearchRequest{
			TimeRange: godo.LogsTimeRange{
				From: godo.LogsTimeInstant{Relative: "15m"},
				To:   godo.LogsTimeInstant{Relative: "now"},
			},
			Filter: &godo.LogsFilterExpression{TextSearch: &godo.LogsTextSearch{Query: "panic"}},
		}).Return(&godo.LogsSearchResponse{}, nil)

		require.NoError(t, RunInsightsLogsSearch(config))
	})
}

func TestRunInsightsLogsSearchFilterFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		captureInsightsStderr(t)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsFilterFile, writeInsightsTestFile(t, testLogsServiceFilterJSON))

		filter := testLogsServiceFilter()
		tm.insights.EXPECT().SearchLogs("nyc3", &godo.LogsSearchRequest{
			TimeRange: godo.LogsTimeRange{
				From: godo.LogsTimeInstant{Relative: "1h"},
				To:   godo.LogsTimeInstant{Relative: "now"},
			},
			Filter: &filter,
		}).Return(&godo.LogsSearchResponse{}, nil)

		require.NoError(t, RunInsightsLogsSearch(config))
	})
}

func TestRunInsightsLogsSearchRequestFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		stderr := captureInsightsStderr(t)

		body := `{
			"time_range": {"from": {"unix_nano": "1756684800000000000"}, "to": {"relative": "now"}},
			"filter": ` + testLogsServiceFilterJSON + `,
			"order_by": [{"field": {"name": "timestamp"}, "direction": "SORT_DIRECTION_ASC"}],
			"pagination": {"limit": 10}
		}`
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsRequestFile, writeInsightsTestFile(t, body))
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsCursor, "next-page")

		filter := testLogsServiceFilter()
		tm.insights.EXPECT().SearchLogs("nyc3", &godo.LogsSearchRequest{
			TimeRange: godo.LogsTimeRange{
				From: godo.LogsTimeInstant{UnixNano: "1756684800000000000"},
				To:   godo.LogsTimeInstant{Relative: "now"},
			},
			Filter:     &filter,
			OrderBy:    []godo.LogsOrderBy{{Field: godo.LogsFieldRef{Name: "timestamp"}, Direction: "SORT_DIRECTION_ASC"}},
			Pagination: &godo.LogsPaginationRequest{Limit: 10, Cursor: "next-page"},
		}).Return(testLogsResponse, nil)

		require.NoError(t, RunInsightsLogsSearch(config))

		var got godo.LogsSearchResponse
		require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
		assert.Equal(t, *testLogsResponse, got)
		assert.Empty(t, stderr.String())
	})
}

func TestRunInsightsLogsSearchRequestFileFromStdin(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		captureInsightsStderr(t)
		prev := insightsStdin
		insightsStdin = strings.NewReader(`{"time_range":{"from":{"relative":"1d"},"to":{"relative":"now"}}}`)
		t.Cleanup(func() { insightsStdin = prev })

		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsLogsRequestFile, "-")

		tm.insights.EXPECT().SearchLogs("nyc3", &godo.LogsSearchRequest{
			TimeRange: godo.LogsTimeRange{
				From: godo.LogsTimeInstant{Relative: "1d"},
				To:   godo.LogsTimeInstant{Relative: "now"},
			},
		}).Return(&godo.LogsSearchResponse{}, nil)

		require.NoError(t, RunInsightsLogsSearch(config))
	})
}

func TestRunInsightsLogsSearchEmptyJSON(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		tm.insights.EXPECT().SearchLogs("nyc3", gomock.Any()).
			Return(&godo.LogsSearchResponse{Pagination: &godo.LogsPaginationResponse{}}, nil)

		require.NoError(t, RunInsightsLogsSearch(config))
		assert.JSONEq(t, `{"data":[],"pagination":{"has_more":false}}`, buf.String())
	})
}

func TestRunInsightsLogsSearchValidation(t *testing.T) {
	validRequestFile := `{"time_range":{"from":{"relative":"1h"},"to":{"relative":"now"}}}`

	tests := []struct {
		name    string
		flags   map[string]any
		files   map[string]string
		wantErr string
	}{
		{name: "missing region", flags: map[string]any{doctl.ArgRegionSlug: ""}, wantErr: "--region is required"},
		{name: "limit too large", flags: map[string]any{doctl.ArgInsightsLogsLimit: 1001}, wantErr: "--limit must be between 1 and 1000"},
		{name: "negative limit", flags: map[string]any{doctl.ArgInsightsLogsLimit: -1}, wantErr: "--limit must be between 1 and 1000"},
		{name: "invalid filter json", flags: map[string]any{doctl.ArgInsightsLogsFilter: `{"condition":`}, wantErr: "parsing --filter"},
		{name: "unknown filter field", flags: map[string]any{doctl.ArgInsightsLogsFilter: `{"conditon":{}}`}, wantErr: `unknown field "conditon"`},
		{
			name:    "filter and filter file",
			flags:   map[string]any{doctl.ArgInsightsLogsFilter: testLogsServiceFilterJSON},
			files:   map[string]string{doctl.ArgInsightsLogsFilterFile: testLogsServiceFilterJSON},
			wantErr: "--filter and --filter-file cannot be combined",
		},
		{name: "missing filter file", flags: map[string]any{doctl.ArgInsightsLogsFilterFile: "/does/not/exist.json"}, wantErr: "reading --filter-file"},
		{name: "invalid order by direction", flags: map[string]any{doctl.ArgInsightsLogsOrderBy: []string{"timestamp:newest"}}, wantErr: "direction must be `asc` or `desc`"},
		{name: "empty order by field", flags: map[string]any{doctl.ArgInsightsLogsOrderBy: []string{":desc"}}, wantErr: "a field name is required"},
		{
			name:    "cursor with non-timestamp ordering",
			flags:   map[string]any{doctl.ArgInsightsLogsCursor: "abc", doctl.ArgInsightsLogsOrderBy: []string{"severity_number:desc"}},
			wantErr: "--cursor requires results to be ordered by timestamp alone",
		},
		{
			name:    "request file with text",
			flags:   map[string]any{doctl.ArgInsightsLogsText: "timeout"},
			files:   map[string]string{doctl.ArgInsightsLogsRequestFile: validRequestFile},
			wantErr: "--text cannot be combined with --request-file",
		},
		{
			name:    "request file with from",
			flags:   map[string]any{doctl.ArgInsightsLogsFrom: "1h"},
			files:   map[string]string{doctl.ArgInsightsLogsRequestFile: validRequestFile},
			wantErr: "--from cannot be combined with --request-file",
		},
		{
			name:    "request file without time range",
			files:   map[string]string{doctl.ArgInsightsLogsRequestFile: `{"pagination":{"limit":5}}`},
			wantErr: "--request-file must set time_range.from and time_range.to",
		},
		{
			name:    "request file with unknown field",
			files:   map[string]string{doctl.ArgInsightsLogsRequestFile: `{"time_range":{"from":{"relative":"1h"},"to":{"relative":"now"}},"limit":5}`},
			wantErr: `unknown field "limit"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
				for k, v := range tt.flags {
					config.Doit.Set(config.NS, k, v)
				}
				for k, contents := range tt.files {
					config.Doit.Set(config.NS, k, writeInsightsTestFile(t, contents))
				}

				assert.ErrorContains(t, RunInsightsLogsSearch(config), tt.wantErr)
			})
		})
	}
}

func TestParseInsightsLogsTime(t *testing.T) {
	relative := []string{"now", "now-1h", "now+30m", "15m", "7d", "2w", "10s"}
	for _, in := range relative {
		assert.Equal(t, godo.LogsTimeInstant{Relative: in}, parseInsightsLogsTime(in), in)
	}

	absolute := []string{"2026-09-01T00:00:00Z", "1756684800", "1756684800000000000", "now-1x"}
	for _, in := range absolute {
		assert.Equal(t, godo.LogsTimeInstant{Absolute: in}, parseInsightsLogsTime(in), in)
	}
}
