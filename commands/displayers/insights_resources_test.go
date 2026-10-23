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

func TestInsightsResources(t *testing.T) {
	effectiveAt := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	resource := godo.InsightsResource{
		ResourceURN: "do:droplet:12345",
		ProductType: "droplet",
		Region:      "nyc3",
		Metrics:     &godo.InsightsResourceSignalState{Enabled: true, EffectiveAt: effectiveAt},
	}
	displayer := &InsightsResources{Response: &godo.InsightsResourceList{
		Resources:  []godo.InsightsResource{resource},
		Pagination: &godo.InsightsResourcesPagination{Page: 1, Pages: 1, PerPage: 20, Total: 1},
	}}

	rows := displayer.KV()
	require.Len(t, rows, 1)
	assert.Equal(t, "true", rows[0]["Metrics"])
	assert.Equal(t, "2026-10-01T12:00:00Z", rows[0]["MetricsEffectiveAt"])
	assert.Equal(t, "", rows[0]["Logs"])
	assert.Equal(t, "", rows[0]["Traces"])

	var out bytes.Buffer
	require.NoError(t, displayer.JSON(&out))
	assert.JSONEq(t, `{
		"resources":[{
			"resource_urn":"do:droplet:12345",
			"product_type":"droplet",
			"region":"nyc3",
			"metrics":{"enabled":true,"effective_at":"2026-10-01T12:00:00Z"}
		}],
		"pagination":{"page":1,"pages":1,"per_page":20,"total":1}
	}`, out.String())
}

func TestInsightsResourcesEmptyJSON(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, (&InsightsResources{Response: &godo.InsightsResourceList{
		Pagination: &godo.InsightsResourcesPagination{Page: 1, PerPage: 20},
	}}).JSON(&out))
	assert.JSONEq(t, `{"pagination":{"page":1,"per_page":20}}`, out.String())
}

func TestInsightsResourceJSON(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, (&InsightsResource{Resource: &godo.InsightsResource{
		ResourceURN: "do:droplet:12345",
		ProductType: "droplet",
	}}).JSON(&out))
	assert.JSONEq(t, `{"resource":{"resource_urn":"do:droplet:12345","product_type":"droplet"}}`, out.String())
}
