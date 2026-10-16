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
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTracesFilterJSON = `{"type":"condition","field":{"name":"serviceName"},"operator":"eq","value":{"type":"string","value":"api"}}`

func testTracesFilter() godo.TracesFilterExpression {
	api := "api"
	return godo.TracesFilterExpression{
		Type:     "condition",
		Field:    &godo.TracesFilterField{Name: "serviceName"},
		Operator: "eq",
		Value:    &godo.TracesFilterValue{String: &api},
	}
}

func TestInsightsTracesCommand(t *testing.T) {
	cmd := insightsTraces()
	assert.Equal(t, "traces", cmd.Name())
	assertCommandNames(t, cmd, "search", "get")
}

func TestRunInsightsTracesSearchFromFlags(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		stderr := captureInsightsStderr(t)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingFrom, "30m")
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingFilter, testTracesFilterJSON)
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingOrderBy, []string{"started"})
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingLimit, 25)

		filter := testTracesFilter()
		tm.insights.EXPECT().SearchTraces("nyc3", &godo.TracesSearchRequest{
			TimeRange: godo.TracesTimeRange{
				From: godo.TracesTimeBound{Relative: "30m"},
				To:   godo.TracesTimeBound{Relative: "now"},
			},
			Filter:     &filter,
			OrderBy:    []godo.TracesOrderBy{{Field: godo.TracesFilterField{Name: "started"}}},
			Pagination: &godo.TracesPaginationRequest{Limit: 25},
		}).Return(&godo.TracesSearchResponse{
			Data:       []godo.TraceSummary{{TraceID: "trace-1", ServiceName: "api"}},
			Pagination: &godo.TracesPaginationResponse{HasMore: true},
		}, nil)

		require.NoError(t, RunInsightsTracesSearch(config))
		assert.Contains(t, stderr.String(), "More traces matched")
	})
}

func TestRunInsightsTracesSearchRequestFileYAML(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "json")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsRequestFile, writeInsightsTestFile(t, `
timeRange:
  from:
    relative: 1h
  to:
    relative: now
pagination:
  limit: 10
`))
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingLimit, 20)

		tm.insights.EXPECT().SearchTraces("nyc3", &godo.TracesSearchRequest{
			TimeRange: godo.TracesTimeRange{
				From: godo.TracesTimeBound{Relative: "1h"},
				To:   godo.TracesTimeBound{Relative: "now"},
			},
			Pagination: &godo.TracesPaginationRequest{Limit: 20},
		}).Return(&godo.TracesSearchResponse{}, nil)

		require.NoError(t, RunInsightsTracesSearch(config))
	})
}

func TestRunInsightsTracesSearchFilterFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		captureInsightsStderr(t)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingFilterFile, writeInsightsTestFile(t, testTracesFilterJSON))

		filter := testTracesFilter()
		tm.insights.EXPECT().SearchTraces("nyc3", &godo.TracesSearchRequest{
			TimeRange: godo.TracesTimeRange{
				From: godo.TracesTimeBound{Relative: "1h"},
				To:   godo.TracesTimeBound{Relative: "now"},
			},
			Filter: &filter,
		}).Return(&godo.TracesSearchResponse{}, nil)

		require.NoError(t, RunInsightsTracesSearch(config))
	})
}

func TestRunInsightsTracesSearchValidation(t *testing.T) {
	validRequest := `{"timeRange":{"from":{"relative":"1h"},"to":{"relative":"now"}}}`
	tests := []struct {
		name    string
		flags   map[string]any
		files   map[string]string
		wantErr string
	}{
		{name: "missing region", flags: map[string]any{doctl.ArgRegionSlug: ""}, wantErr: "--region is required"},
		{name: "zero limit", flags: map[string]any{doctl.ArgInsightsTracingLimit: 0}, wantErr: "--limit must be between 1 and 1000"},
		{name: "limit too large", flags: map[string]any{doctl.ArgInsightsTracingLimit: 1001}, wantErr: "--limit must be between 1 and 1000"},
		{name: "invalid filter", flags: map[string]any{doctl.ArgInsightsTracingFilter: `{"field":{"name":"serviceName"},"operator":"eq"}`}, wantErr: "value is required"},
		{name: "unknown filter field", flags: map[string]any{doctl.ArgInsightsTracingFilter: `{"condition":{}}`}, wantErr: `unknown field "condition"`},
		{name: "invalid order", flags: map[string]any{doctl.ArgInsightsTracingOrderBy: []string{"started:newest"}}, wantErr: "direction must be `asc` or `desc`"},
		{name: "from after to", flags: map[string]any{doctl.ArgInsightsTracingFrom: "2026-09-02T00:00:00Z", doctl.ArgInsightsTracingTo: "2026-09-01T00:00:00Z"}, wantErr: "must not be after"},
		{name: "range over seven days", flags: map[string]any{doctl.ArgInsightsTracingFrom: "2026-09-01T00:00:00Z", doctl.ArgInsightsTracingTo: "2026-09-09T00:00:00Z"}, wantErr: "must not exceed 7 days"},
		{
			name:    "filter and filter file",
			flags:   map[string]any{doctl.ArgInsightsTracingFilter: testTracesFilterJSON},
			files:   map[string]string{doctl.ArgInsightsTracingFilterFile: testTracesFilterJSON},
			wantErr: "--filter and --filter-file cannot be combined",
		},
		{
			name:    "request file with order",
			flags:   map[string]any{doctl.ArgInsightsTracingOrderBy: []string{"started:desc"}},
			files:   map[string]string{doctl.ArgInsightsRequestFile: validRequest},
			wantErr: "--request-file cannot be combined with --order-by",
		},
		{
			name:    "request with zero limit",
			files:   map[string]string{doctl.ArgInsightsRequestFile: `{"timeRange":{"from":{"relative":"1h"},"to":{"relative":"now"}},"pagination":{"limit":0}}`},
			wantErr: "--limit must be between 1 and 1000",
		},
		{
			name:    "request with unknown field",
			files:   map[string]string{doctl.ArgInsightsRequestFile: `{"timeRange":{"from":{"relative":"1h"},"to":{"relative":"now"}},"cursor":"x"}`},
			wantErr: `unknown field "cursor"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
				for key, value := range tt.flags {
					config.Doit.Set(config.NS, key, value)
				}
				for key, contents := range tt.files {
					config.Doit.Set(config.NS, key, writeInsightsTestFile(t, contents))
				}
				assert.ErrorContains(t, RunInsightsTracesSearch(config), tt.wantErr)
			})
		})
	}
}

func TestRunInsightsTracesGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		stderr := captureInsightsStderr(t)
		config.Args = []string{"trace-1"}
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingFrom, "2h")
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingTo, "now-1h")

		tm.insights.EXPECT().GetTrace("nyc3", "trace-1", &godo.TraceGetOptions{
			From: "2h",
			To:   "now-1h",
		}).Return(&godo.Trace{TraceID: "trace-1", Truncated: true}, nil)

		require.NoError(t, RunInsightsTraceGet(config))
		assert.Contains(t, stderr.String(), "trace contains additional spans")
	})
}

func TestRunInsightsTracesGetValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		from    string
		to      string
		wantErr string
	}{
		{name: "missing trace id", wantErr: "missing required arguments"},
		{name: "too many arguments", args: []string{"one", "two"}, wantErr: "missing required arguments"},
		{name: "empty trace id", args: []string{" "}, wantErr: "trace ID is required"},
		{name: "from after to", args: []string{"trace-1"}, from: "2026-09-02T00:00:00Z", to: "2026-09-01T00:00:00Z", wantErr: "must not be after"},
		{name: "range over seven days", args: []string{"trace-1"}, from: "2026-09-01T00:00:00Z", to: "2026-09-09T00:00:00Z", wantErr: "must not exceed 7 days"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				config.Args = tt.args
				config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
				if tt.from != "" {
					config.Doit.Set(config.NS, doctl.ArgInsightsTracingFrom, tt.from)
				}
				if tt.to != "" {
					config.Doit.Set(config.NS, doctl.ArgInsightsTracingTo, tt.to)
				}
				assert.ErrorContains(t, RunInsightsTraceGet(config), tt.wantErr)
			})
		})
	}
}
