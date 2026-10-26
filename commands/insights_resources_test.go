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
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testInsightsResource = godo.InsightsResource{
	ResourceURN: "do:droplet:12345",
	ProductType: "droplet",
	Region:      "nyc3",
	Metrics: &godo.InsightsResourceSignalState{
		Enabled:     true,
		EffectiveAt: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
	},
	Logs: &godo.InsightsResourceSignalState{
		Enabled:     false,
		EffectiveAt: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC),
	},
}

func TestInsightsResourcesCommand(t *testing.T) {
	cmd := insightsResources()
	assert.Equal(t, "resource", cmd.Name())
	assertCommandNames(t, cmd, "list", "get", "update")
}

func TestRunInsightsResourceList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		config.Doit.Set(config.NS, doctl.ArgInsightsResourceProductType, []string{" droplet ", "kubernetes"})
		config.Doit.Set(config.NS, doctl.ArgInsightsResourceRegion, []string{"nyc3", " sfo3 "})
		config.Doit.Set(config.NS, doctl.ArgInsightsPage, 2)
		config.Doit.Set(config.NS, doctl.ArgInsightsPerPage, 50)
		config.Doit.Set(config.NS, doctl.ArgInsightsResourceSort, godo.InsightsResourceSortEffectiveAt)
		config.Doit.Set(config.NS, doctl.ArgInsightsResourceSortDirection, "DESC")

		tm.insights.EXPECT().ListResources(&godo.InsightsResourceListOptions{
			ProductTypes:  []string{"droplet", "kubernetes"},
			Regions:       []string{"nyc3", "sfo3"},
			Page:          2,
			PerPage:       50,
			Sort:          godo.InsightsResourceSortEffectiveAt,
			SortDirection: godo.InsightsResourceSortDescending,
		}).Return(&godo.InsightsResourceList{
			Resources:  []godo.InsightsResource{testInsightsResource},
			Pagination: &godo.InsightsResourcesPagination{Page: 2, Pages: 3, PerPage: 50, Total: 120},
		}, nil)

		require.NoError(t, RunInsightsResourceList(config))
		assert.Contains(t, buf.String(), testInsightsResource.ResourceURN)
		assert.Contains(t, buf.String(), "true")
		assert.Contains(t, buf.String(), "false")
	})
}

func TestRunInsightsResourceListJSON(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		response := &godo.InsightsResourceList{
			Resources:  []godo.InsightsResource{testInsightsResource},
			Pagination: &godo.InsightsResourcesPagination{Page: 1, Pages: 1, PerPage: 20, Total: 1},
		}
		tm.insights.EXPECT().ListResources(&godo.InsightsResourceListOptions{}).Return(response, nil)

		require.NoError(t, RunInsightsResourceList(config))
		var got godo.InsightsResourceList
		require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
		assert.Equal(t, *response, got)
	})
}

func TestRunInsightsResourceListValidation(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		value   any
		wantErr string
	}{
		{name: "page zero", flag: doctl.ArgInsightsPage, value: 0, wantErr: "--page must be at least 1"},
		{name: "page negative", flag: doctl.ArgInsightsPage, value: -1, wantErr: "--page must be at least 1"},
		{name: "per page zero", flag: doctl.ArgInsightsPerPage, value: 0, wantErr: "--per-page must be between 1 and 200"},
		{name: "per page too large", flag: doctl.ArgInsightsPerPage, value: 201, wantErr: "--per-page must be between 1 and 200"},
		{name: "invalid sort", flag: doctl.ArgInsightsResourceSort, value: "region", wantErr: "invalid --sort"},
		{name: "invalid direction", flag: doctl.ArgInsightsResourceSortDirection, value: "sideways", wantErr: "invalid --sort-direction"},
		{name: "empty product type", flag: doctl.ArgInsightsResourceProductType, value: []string{""}, wantErr: "--product-type values must not be empty"},
		{name: "empty region", flag: doctl.ArgInsightsResourceRegion, value: []string{" "}, wantErr: "--region values must not be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				config.Doit.Set(config.NS, tt.flag, tt.value)
				assert.ErrorContains(t, RunInsightsResourceList(config), tt.wantErr)
			})
		})
	}
}

func TestRunInsightsResourceGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		config.Args = []string{" do:droplet:12345 "}
		tm.insights.EXPECT().GetResource("do:droplet:12345").Return(&testInsightsResource, nil)

		require.NoError(t, RunInsightsResourceGet(config))
		assert.JSONEq(t, `{
			"resource": {
				"resource_urn": "do:droplet:12345",
				"product_type": "droplet",
				"region": "nyc3",
				"metrics": {"enabled": true, "effective_at": "2026-10-01T12:00:00Z"},
				"logs": {"enabled": false, "effective_at": "2026-10-02T12:00:00Z"}
			}
		}`, buf.String())
	})
}

func TestRunInsightsResourceUpdatePreservesTriState(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		withInsightsOutput(t, config, "text")
		config.Args = []string{testInsightsResource.ResourceURN}
		config.Doit.Set(config.NS, doctl.ArgInsightsResourceRegion, " nyc3 ")
		config.Doit.Set(config.NS, doctl.ArgInsightsResourceMetrics, true)
		config.Doit.Set(config.NS, doctl.ArgInsightsResourceLogs, false)

		tm.insights.EXPECT().UpdateResourceSignals(testInsightsResource.ResourceURN, &godo.InsightsResourceSignalsRequest{
			Region:  "nyc3",
			Metrics: godo.Bool(true),
			Logs:    godo.Bool(false),
			Traces:  nil,
		}).Return(&testInsightsResource, nil)

		require.NoError(t, RunInsightsResourceUpdate(config))
	})
}

func TestRunInsightsResourceUpdateValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setFlag bool
		wantErr string
	}{
		{name: "missing resource urn", wantErr: "missing required arguments"},
		{name: "empty resource urn", args: []string{" "}, wantErr: "a resource URN is required"},
		{name: "no signal", args: []string{"do:droplet:12345"}, wantErr: "at least one of --metrics, --logs, or --traces must be set"},
		{name: "region is not a signal", args: []string{"do:droplet:12345"}, setFlag: true, wantErr: "at least one of --metrics, --logs, or --traces must be set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
				config.Args = tt.args
				if tt.setFlag {
					config.Doit.Set(config.NS, doctl.ArgInsightsResourceRegion, "nyc3")
				}
				assert.ErrorContains(t, RunInsightsResourceUpdate(config), tt.wantErr)
			})
		})
	}
}
