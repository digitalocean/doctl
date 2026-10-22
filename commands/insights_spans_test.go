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
	"encoding/json"
	"strings"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSpansFilterJSON = `{"condition":{"field":{"name":"service_name"},"operator":"FILTER_OPERATOR_EQ","value":{"string_value":"api"}}}`

func testSpansFilter() godo.SpansFilterExpression {
	api := "api"
	return godo.SpansFilterExpression{
		Condition: &godo.SpansFilterCondition{
			Field:    godo.SpansFilterField{Name: "service_name"},
			Operator: godo.InsightsSpansFilterOperatorEqual,
			Value:    &godo.SpansFilterValue{StringValue: &api},
		},
	}
}

func TestInsightsSpansCommand(t *testing.T) {
	cmd := insightsSpans()
	assert.Equal(t, "spans", cmd.Name())
	assertCommandNames(t, cmd, "search")
}

func TestSpansFilterContract(t *testing.T) {
	filters := []string{
		`{"condition":{"field":{"name":"service_name"},"operator":"FILTER_OPERATOR_EQ","value":{"string_value":"api"}}}`,
		`{"and":{"expressions":[{"condition":{"field":{"name":"http.method","scope":"FIELD_SCOPE_ATTRIBUTES"},"operator":"FILTER_OPERATOR_EQ","value":{"string_value":"GET"}}},{"condition":{"field":{"name":"duration_ns"},"operator":"FILTER_OPERATOR_GTE","value":{"number_value":10}}},{"text_search":{"query":"failed"}}]}}`,
		`{"or":{"expressions":[{"condition":{"field":{"name":"name"},"operator":"FILTER_OPERATOR_IN","value":{"string_array_value":{"values":["GET","POST"]}}}},{"condition":{"field":{"name":"duration_ns"},"operator":"FILTER_OPERATOR_IN","value":{"number_array_value":{"values":[10,20]}}}}]}}`,
		`{"not":{"condition":{"field":{"name":"status_code"},"operator":"FILTER_OPERATOR_EXISTS"}}}`,
		`{"text_search":{"query":"timeout"}}`,
	}
	for _, filterJSON := range filters {
		var filter godo.SpansFilterExpression
		require.NoError(t, decodeInsightsJSON(strings.NewReader(filterJSON), &filter))
		require.NoError(t, validateSpansFilterExpression(&filter))
	}
}

func TestSpansFilterGodoCompatibilityInput(t *testing.T) {
	const legacy = `{"type":"condition","field":{"name":"service_name"},"operator":"FILTER_OPERATOR_EQ","value":{"type":"string","value":"api"}}`
	var filter godo.SpansFilterExpression
	require.NoError(t, decodeInsightsJSON(strings.NewReader(legacy), &filter))
	require.NoError(t, validateSpansFilterExpression(&filter))

	encoded, err := json.Marshal(filter)
	require.NoError(t, err)
	assert.JSONEq(t, testSpansFilterJSON, string(encoded))
}

func TestRunInsightsSpansSearchDefaults(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		captureInsightsStderr(t)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		tm.insights.EXPECT().SearchSpans("nyc3", &godo.SpansSearchRequest{
			TimeRange: godo.SpansTimeRange{
				From: godo.SpansTimeBound{Relative: "1h"},
				To:   godo.SpansTimeBound{Relative: "now"},
			},
		}).Return(&godo.SpansSearchResponse{}, nil)

		require.NoError(t, RunInsightsSpansSearch(config))
	})
}

func TestRunInsightsSpansSearchFromFlags(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		stderr := captureInsightsStderr(t)
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingFrom, "2026-09-01T00:00:00Z")
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingTo, "2026-09-01T01:00:00Z")
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingFilter, testSpansFilterJSON)
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingOrderBy, []string{"start_time:DESC"})
		config.Doit.Set(config.NS, doctl.ArgInsightsTracingLimit, 50)

		filter := testSpansFilter()
		tm.insights.EXPECT().SearchSpans("nyc3", &godo.SpansSearchRequest{
			TimeRange: godo.SpansTimeRange{
				From: godo.SpansTimeBound{Absolute: "2026-09-01T00:00:00Z"},
				To:   godo.SpansTimeBound{Absolute: "2026-09-01T01:00:00Z"},
			},
			Filter:     &filter,
			OrderBy:    []godo.SpansOrderBy{{Field: godo.SpansFilterField{Name: "start_time"}, Direction: "desc"}},
			Pagination: &godo.SpansPaginationRequest{Limit: 50},
		}).Return(&godo.SpansSearchResponse{
			Data:       []godo.SpanRecord{{TraceID: "trace-1", SpanID: "span-1", ServiceName: "api"}},
			Pagination: &godo.SpansPaginationResponse{HasMore: true},
		}, nil)

		require.NoError(t, RunInsightsSpansSearch(config))
		assert.Contains(t, stderr.String(), "More spans matched")
	})
}

func TestRunInsightsSpansSearchRequestFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "json")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsRequestFile, writeInsightsTestFile(t, `{
			"time_range":{"from":{"relative":"1h"},"to":{"relative":"now"}},
			"filter":`+testSpansFilterJSON+`,
			"order_by":[{"field":{"name":"start_time"},"direction":"desc"}],
			"pagination":{"limit":10}
		}`))

		filter := testSpansFilter()
		tm.insights.EXPECT().SearchSpans("nyc3", &godo.SpansSearchRequest{
			TimeRange: godo.SpansTimeRange{
				From: godo.SpansTimeBound{Relative: "1h"},
				To:   godo.SpansTimeBound{Relative: "now"},
			},
			Filter:     &filter,
			OrderBy:    []godo.SpansOrderBy{{Field: godo.SpansFilterField{Name: "start_time"}, Direction: "desc"}},
			Pagination: &godo.SpansPaginationRequest{Limit: 10},
		}).Return(&godo.SpansSearchResponse{}, nil)

		require.NoError(t, RunInsightsSpansSearch(config))
	})
}

func TestRunInsightsSpansSearchValidation(t *testing.T) {
	validRequest := `{"time_range":{"from":{"relative":"1h"},"to":{"relative":"now"}}}`
	tests := []struct {
		name    string
		flags   map[string]any
		files   map[string]string
		wantErr string
	}{
		{name: "missing region", flags: map[string]any{doctl.ArgRegionSlug: ""}, wantErr: "--region is required"},
		{name: "zero limit", flags: map[string]any{doctl.ArgInsightsTracingLimit: 0}, wantErr: "--limit must be between 1 and 1000"},
		{name: "negative limit", flags: map[string]any{doctl.ArgInsightsTracingLimit: -1}, wantErr: "--limit must be between 1 and 1000"},
		{name: "limit too large", flags: map[string]any{doctl.ArgInsightsTracingLimit: 1001}, wantErr: "--limit must be between 1 and 1000"},
		{name: "invalid filter", flags: map[string]any{doctl.ArgInsightsTracingFilter: `{"condition":{"field":{"name":"service_name"},"operator":"FILTER_OPERATOR_EQ"}}`}, wantErr: "value is required"},
		{name: "deprecated operator", flags: map[string]any{doctl.ArgInsightsTracingFilter: `{"condition":{"field":{"name":"service_name"},"operator":"eq","value":{"string_value":"api"}}}`}, wantErr: `invalid condition operator "eq"`},
		{name: "deprecated scope", flags: map[string]any{doctl.ArgInsightsTracingFilter: `{"condition":{"field":{"name":"service.name","scope":"resource"},"operator":"FILTER_OPERATOR_EQ","value":{"string_value":"api"}}}`}, wantErr: `invalid field scope "resource"`},
		{name: "multiple filter nodes", flags: map[string]any{doctl.ArgInsightsTracingFilter: `{"condition":{},"text_search":{"query":"api"}}`}, wantErr: "must contain exactly one node"},
		{name: "invalid order", flags: map[string]any{doctl.ArgInsightsTracingOrderBy: []string{"start_time:newest"}}, wantErr: "direction must be `asc` or `desc`"},
		{name: "from after to", flags: map[string]any{doctl.ArgInsightsTracingFrom: "2026-09-02T00:00:00Z", doctl.ArgInsightsTracingTo: "2026-09-01T00:00:00Z"}, wantErr: "must not be after"},
		{name: "range over seven days", flags: map[string]any{doctl.ArgInsightsTracingFrom: "2026-09-01T00:00:00Z", doctl.ArgInsightsTracingTo: "2026-09-09T00:00:00Z"}, wantErr: "must not exceed 7 days"},
		{
			name:    "filter and filter file",
			flags:   map[string]any{doctl.ArgInsightsTracingFilter: testSpansFilterJSON},
			files:   map[string]string{doctl.ArgInsightsTracingFilterFile: testSpansFilterJSON},
			wantErr: "--filter and --filter-file cannot be combined",
		},
		{
			name:    "request file with from",
			flags:   map[string]any{doctl.ArgInsightsTracingFrom: "1h"},
			files:   map[string]string{doctl.ArgInsightsRequestFile: validRequest},
			wantErr: "--request-file cannot be combined with --from",
		},
		{
			name:    "request without time range",
			files:   map[string]string{doctl.ArgInsightsRequestFile: `{"pagination":{"limit":5}}`},
			wantErr: "time_range.from: time bound is required",
		},
		{
			name:    "request with zero limit",
			files:   map[string]string{doctl.ArgInsightsRequestFile: `{"time_range":{"from":{"relative":"1h"},"to":{"relative":"now"}},"pagination":{"limit":0}}`},
			wantErr: "--limit must be between 1 and 1000",
		},
		{
			name:    "request with camel case field",
			files:   map[string]string{doctl.ArgInsightsRequestFile: `{"timeRange":{"from":{"relative":"1h"},"to":{"relative":"now"}}}`},
			wantErr: `unknown field "timeRange"`,
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
				assert.ErrorContains(t, RunInsightsSpansSearch(config), tt.wantErr)
			})
		})
	}
}
