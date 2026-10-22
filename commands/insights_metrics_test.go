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
	"errors"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testPromVectorJSON = `{"status":"success","data":{"resultType":"vector","result":[` +
		`{"metric":{"__name__":"up","instance":"10.0.0.1:9100","job":"node"},"value":[1756684800.5,"1"]},` +
		`{"metric":{},"value":[1756684800.5,"0"]}]}}`
	testPromMatrixJSON = `{"status":"success","data":{"resultType":"matrix","result":[` +
		`{"metric":{"__name__":"up","job":"node"},"values":[[1756684800,"1"],[1756684860,"0"]]}]}}`
	testPromScalarJSON = `{"status":"success","data":{"resultType":"scalar","result":[1756684800,"42"]}}`
)

func decodeTestJSON[T any](t *testing.T, s string) *T {
	t.Helper()
	v := new(T)
	require.NoError(t, json.Unmarshal([]byte(s), v))
	return v
}

func TestInsightsMetricsCommand(t *testing.T) {
	cmd := insightsMetrics()
	assert.Equal(t, "metrics", cmd.Name())
	assertCommandNames(t, cmd, "query", "query-range", "labels", "label-values", "series")
}

func TestRunInsightsMetricsQuery(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsQuery, " up ")
		config.Doit.Set(config.NS, doctl.ArgInsightsTime, "2026-09-01T00:00:00Z")
		config.Doit.Set(config.NS, doctl.ArgTimeout, "30s")

		tm.insights.EXPECT().PostQuery("nyc3", &godo.PromQueryOptions{
			Query:   "up",
			Time:    "2026-09-01T00:00:00Z",
			Timeout: "30s",
		}).Return(decodeTestJSON[godo.PromQueryResponse](t, testPromVectorJSON), nil)

		require.NoError(t, RunInsightsMetricsQuery(config))

		out := buf.String()
		assert.Contains(t, out, "Metric")
		assert.Contains(t, out, `up{instance="10.0.0.1:9100", job="node"}`)
		assert.Contains(t, out, "2025-09-01T00:00:00.5Z")
		assert.Contains(t, out, "{}")
	})
}

func TestRunInsightsMetricsQueryJSONPreservesPrometheusShape(t *testing.T) {
	for name, payload := range map[string]string{
		"vector": testPromVectorJSON,
		"matrix": testPromMatrixJSON,
		"scalar": testPromScalarJSON,
	} {
		t.Run(name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				buf := withInsightsOutput(t, config, "json")
				config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
				config.Doit.Set(config.NS, doctl.ArgInsightsQuery, "up")

				tm.insights.EXPECT().PostQuery("nyc3", &godo.PromQueryOptions{Query: "up"}).
					Return(decodeTestJSON[godo.PromQueryResponse](t, payload), nil)

				require.NoError(t, RunInsightsMetricsQuery(config))
				assert.JSONEq(t, payload, buf.String())
			})
		})
	}
}

func TestRunInsightsMetricsQueryEmptyResultJSON(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsQuery, "absent_metric")

		payload := `{"status":"success","data":{"resultType":"vector","result":[]}}`
		tm.insights.EXPECT().PostQuery("nyc3", &godo.PromQueryOptions{Query: "absent_metric"}).
			Return(decodeTestJSON[godo.PromQueryResponse](t, payload), nil)

		require.NoError(t, RunInsightsMetricsQuery(config))
		assert.JSONEq(t, payload, buf.String())
	})
}

func TestRunInsightsMetricsQueryRange(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsQuery, "up")
		config.Doit.Set(config.NS, doctl.ArgInsightsStart, "1756684800")
		config.Doit.Set(config.NS, doctl.ArgInsightsEnd, "2026-09-01T01:00:00Z")
		config.Doit.Set(config.NS, doctl.ArgInsightsStep, "1m")

		tm.insights.EXPECT().PostQueryRange("nyc3", &godo.PromQueryRangeOptions{
			Query: "up",
			Start: "1756684800",
			End:   "2026-09-01T01:00:00Z",
			Step:  "1m",
		}).Return(decodeTestJSON[godo.PromQueryRangeResponse](t, testPromMatrixJSON), nil)

		require.NoError(t, RunInsightsMetricsQueryRange(config))
		assert.JSONEq(t, testPromMatrixJSON, buf.String())
	})
}

func TestRunInsightsMetricsQueryRangeText(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsQuery, "up")
		config.Doit.Set(config.NS, doctl.ArgInsightsStart, "1756684800")
		config.Doit.Set(config.NS, doctl.ArgInsightsEnd, "1756684860")
		config.Doit.Set(config.NS, doctl.ArgInsightsStep, "60")

		tm.insights.EXPECT().PostQueryRange("nyc3", gomock.Any()).
			Return(decodeTestJSON[godo.PromQueryRangeResponse](t, testPromMatrixJSON), nil)

		require.NoError(t, RunInsightsMetricsQueryRange(config))

		out := buf.String()
		assert.Contains(t, out, `up{job="node"}`)
		assert.Contains(t, out, "2025-09-01T00:00:00Z")
		assert.Contains(t, out, "2025-09-01T00:01:00Z")
	})
}

func TestRunInsightsMetricsValidation(t *testing.T) {
	valid := map[string]any{
		doctl.ArgRegionSlug:    "nyc3",
		doctl.ArgInsightsQuery: "up",
		doctl.ArgInsightsStart: "2026-09-01T00:00:00Z",
		doctl.ArgInsightsEnd:   "2026-09-01T01:00:00Z",
		doctl.ArgInsightsStep:  "1m",
	}

	tests := []struct {
		name      string
		run       CmdRunner
		overrides map[string]any
		wantErr   string
	}{
		{name: "query missing region", run: RunInsightsMetricsQuery, overrides: map[string]any{doctl.ArgRegionSlug: ""}, wantErr: "--region is required"},
		{name: "query missing query", run: RunInsightsMetricsQuery, overrides: map[string]any{doctl.ArgInsightsQuery: "  "}, wantErr: "--query is required"},
		{name: "query invalid time", run: RunInsightsMetricsQuery, overrides: map[string]any{doctl.ArgInsightsTime: "yesterday"}, wantErr: `invalid --time "yesterday"`},
		{name: "query invalid timeout", run: RunInsightsMetricsQuery, overrides: map[string]any{doctl.ArgTimeout: "soon"}, wantErr: `invalid --timeout "soon"`},
		{name: "range missing start", run: RunInsightsMetricsQueryRange, overrides: map[string]any{doctl.ArgInsightsStart: ""}, wantErr: "--start is required"},
		{name: "range missing end", run: RunInsightsMetricsQueryRange, overrides: map[string]any{doctl.ArgInsightsEnd: ""}, wantErr: "--end is required"},
		{name: "range missing step", run: RunInsightsMetricsQueryRange, overrides: map[string]any{doctl.ArgInsightsStep: ""}, wantErr: "--step is required"},
		{name: "range invalid step", run: RunInsightsMetricsQueryRange, overrides: map[string]any{doctl.ArgInsightsStep: "1 minute"}, wantErr: `invalid --step "1 minute"`},
		{name: "range zero step", run: RunInsightsMetricsQueryRange, overrides: map[string]any{doctl.ArgInsightsStep: "0s"}, wantErr: "must be greater than zero"},
		{name: "range start after end", run: RunInsightsMetricsQueryRange, overrides: map[string]any{doctl.ArgInsightsStart: "2026-09-02T00:00:00Z"}, wantErr: "--start must not be after --end"},
		{name: "range invalid start", run: RunInsightsMetricsQueryRange, overrides: map[string]any{doctl.ArgInsightsStart: "2026-09-01"}, wantErr: `invalid --start "2026-09-01"`},
		{name: "labels missing region", run: RunInsightsMetricsLabels, overrides: map[string]any{doctl.ArgRegionSlug: ""}, wantErr: "--region is required"},
		{name: "labels start after end", run: RunInsightsMetricsLabels, overrides: map[string]any{doctl.ArgInsightsEnd: "1"}, wantErr: "--start must not be after --end"},
		{name: "series missing match", run: RunInsightsMetricsSeries, wantErr: "at least one --match selector is required"},
		{name: "series missing region", run: RunInsightsMetricsSeries, overrides: map[string]any{doctl.ArgRegionSlug: "", doctl.ArgInsightsMatch: []string{"up"}}, wantErr: "--region is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				for k, v := range valid {
					config.Doit.Set(config.NS, k, v)
				}
				for k, v := range tt.overrides {
					config.Doit.Set(config.NS, k, v)
				}

				assert.ErrorContains(t, tt.run(config), tt.wantErr)
			})
		})
	}
}

func TestRunInsightsMetricsLabels(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsMatch, []string{"up", " "})
		config.Doit.Set(config.NS, doctl.ArgInsightsStart, "1756684800")

		tm.insights.EXPECT().PostLabels("nyc3", &godo.PromSelectorOptions{Match: []string{"up"}, Start: "1756684800"}).
			Return(&godo.PromLabelsResponse{Status: "success", Data: []string{"__name__", "job"}}, nil)

		require.NoError(t, RunInsightsMetricsLabels(config))
		assert.Equal(t, "Name\n__name__\njob\n", buf.String())
	})
}

func TestRunInsightsMetricsLabelValues(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		config.Args = append(config.Args, "job")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		tm.insights.EXPECT().LabelValues("nyc3", "job", &godo.PromSelectorOptions{Match: []string{}}).
			Return(&godo.PromLabelsResponse{Status: "success"}, nil)

		require.NoError(t, RunInsightsMetricsLabelValues(config))
		assert.JSONEq(t, `{"status":"success","data":[]}`, buf.String())
	})
}

func TestRunInsightsMetricsLabelValuesRequiresLabelName(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")

		var missing *doctl.MissingArgsErr
		assert.ErrorAs(t, RunInsightsMetricsLabelValues(config), &missing)
	})
}

func TestRunInsightsMetricsSeries(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsMatch, []string{`up{job="api"}`, `up{job="worker"}`})

		tm.insights.EXPECT().PostSeries("nyc3", &godo.PromSelectorOptions{Match: []string{`up{job="api"}`, `up{job="worker"}`}}).
			Return(&godo.PromSeriesResponse{Status: "success", Data: []map[string]string{
				{"__name__": "up", "job": "api"},
				{"__name__": "up", "job": "worker"},
			}}, nil)

		require.NoError(t, RunInsightsMetricsSeries(config))
		assert.Equal(t, "Series\nup{job=\"api\"}\nup{job=\"worker\"}\n", buf.String())
	})
}

func TestRunInsightsMetricsError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgRegionSlug, "nyc3")
		config.Doit.Set(config.NS, doctl.ArgInsightsQuery, "up")
		tm.insights.EXPECT().PostQuery("nyc3", &godo.PromQueryOptions{Query: "up"}).Return(nil, errors.New("bad_data: parse error"))

		assert.EqualError(t, RunInsightsMetricsQuery(config), "bad_data: parse error")
	})
}

func TestInsightsMatchFlagPreservesCommasAndQuotes(t *testing.T) {
	series, _, err := insightsMetrics().Find([]string{"series"})
	require.NoError(t, err)
	require.NoError(t, series.ParseFlags([]string{
		"--match", `up{job="api",instance="a,b"}`,
		"--match", `node_load1`,
	}))

	got, err := series.Flags().GetStringArray(doctl.ArgInsightsMatch)
	require.NoError(t, err)
	assert.Equal(t, []string{`up{job="api",instance="a,b"}`, "node_load1"}, got)
}

func TestParsePromDuration(t *testing.T) {
	tests := map[string]time.Duration{
		"30":      30 * time.Second,
		"1.5":     1500 * time.Millisecond,
		"30s":     30 * time.Second,
		"1h30m":   90 * time.Minute,
		"1d":      24 * time.Hour,
		"1w":      7 * 24 * time.Hour,
		"250ms":   250 * time.Millisecond,
		"1y":      365 * 24 * time.Hour,
		"2h500ms": 2*time.Hour + 500*time.Millisecond,
	}
	for in, want := range tests {
		got, err := parsePromDuration(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}

	for _, in := range []string{"", "-1", "1 m", "m", "1M", "NaN"} {
		_, err := parsePromDuration(in)
		assert.Error(t, err, in)
	}
}
