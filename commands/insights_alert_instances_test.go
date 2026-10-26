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
	"errors"
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testInsightsResolvedAt    = time.Date(2026, 9, 1, 0, 20, 0, 0, time.UTC)
	testInsightsAlertInstance = godo.AlertInstance{
		ID:              "9c1f8e2a-3b4d-4c5e-9f6a-7b8c9d0e1f2a",
		RuleID:          "4b6f2a7e-1c2d-4e5f-8a9b-0c1d2e3f4a5b",
		Severity:        godo.InsightsAlertInstanceSeverityCritical,
		Status:          godo.InsightsAlertInstanceStatusResolved,
		ResourceURN:     "do:droplet:12345",
		Value:           92.5,
		TriggeredAt:     time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		LastTriggeredAt: time.Date(2026, 9, 1, 0, 10, 0, 0, time.UTC),
		ResolvedAt:      &testInsightsResolvedAt,
	}
)

// withInsightsOutput captures command output in the given format.
func withInsightsOutput(t *testing.T, config *CmdConfig, format string) *bytes.Buffer {
	t.Helper()
	prev := Output
	Output = format
	t.Cleanup(func() { Output = prev })

	buf := &bytes.Buffer{}
	config.Out = buf
	return buf
}

func TestInsightsAlertInstanceCommand(t *testing.T) {
	cmd := insightsAlertInstances()
	assert.Equal(t, "alert-instance", cmd.Name())
	assertCommandNames(t, cmd, "get", "list")
}

func TestRunInsightsAlertInstanceListAllPages(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		config.Doit.Set(config.NS, doctl.ArgInsightsStatus, "Active")
		config.Doit.Set(config.NS, doctl.ArgInsightsAlertRuleID, testInsightsAlertInstance.RuleID)
		config.Doit.Set(config.NS, doctl.ArgInsightsResourceURN, testInsightsAlertInstance.ResourceURN)

		second := testInsightsAlertInstance
		second.ID = "second-instance"

		base := godo.AlertInstanceListOptions{
			Status:      godo.InsightsAlertInstanceStatusActive,
			RuleID:      testInsightsAlertInstance.RuleID,
			ResourceURN: testInsightsAlertInstance.ResourceURN,
			PerPage:     200,
		}
		page1, page2 := base, base
		page1.Page, page2.Page = 1, 2

		resp := &godo.Response{Links: &godo.Links{Pages: &godo.Pages{
			Last: "https://api.digitalocean.com/v2/insights/alert-instances?page=2&per_page=200",
		}}}
		tm.insights.EXPECT().ListAlertInstances(&page1).Return([]godo.AlertInstance{testInsightsAlertInstance}, resp, nil)
		tm.insights.EXPECT().ListAlertInstances(&page2).Return([]godo.AlertInstance{second}, &godo.Response{}, nil)

		require.NoError(t, RunInsightsAlertInstanceList(config))

		out := buf.String()
		assert.Contains(t, out, "Rule ID")
		assert.Contains(t, out, testInsightsAlertInstance.ID)
		assert.Contains(t, out, "second-instance")
		assert.Contains(t, out, "92.5")
		assert.Contains(t, out, "2026-09-01T00:20:00Z")
	})
}

func TestRunInsightsAlertInstanceListSinglePage(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		config.Doit.Set(config.NS, doctl.ArgInsightsPage, 3)
		config.Doit.Set(config.NS, doctl.ArgInsightsPerPage, 25)

		tm.insights.EXPECT().ListAlertInstances(&godo.AlertInstanceListOptions{Page: 3, PerPage: 25}).
			Return([]godo.AlertInstance{testInsightsAlertInstance}, &godo.Response{}, nil)

		require.NoError(t, RunInsightsAlertInstanceList(config))

		var got []godo.AlertInstance
		require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
		assert.Equal(t, []godo.AlertInstance{testInsightsAlertInstance}, got)
	})
}

func TestRunInsightsAlertInstanceListEmptyJSON(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "json")
		config.Doit.Set(config.NS, doctl.ArgInsightsPage, 1)

		tm.insights.EXPECT().ListAlertInstances(&godo.AlertInstanceListOptions{Page: 1}).Return(nil, &godo.Response{}, nil)

		require.NoError(t, RunInsightsAlertInstanceList(config))
		assert.Equal(t, "[]", buf.String())
	})
}

func TestRunInsightsAlertInstanceListValidation(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   any
		wantErr string
	}{
		{name: "invalid status", key: doctl.ArgInsightsStatus, value: "firing", wantErr: `invalid --status "firing"`},
		{name: "negative page", key: doctl.ArgInsightsPage, value: -1, wantErr: "must not be negative"},
		{name: "negative per page", key: doctl.ArgInsightsPerPage, value: -5, wantErr: "must not be negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
				config.Doit.Set(config.NS, tt.key, tt.value)

				err := RunInsightsAlertInstanceList(config)
				assert.ErrorContains(t, err, tt.wantErr)
			})
		})
	}
}

func TestRunInsightsAlertInstanceListError(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInsightsPage, 1)
		tm.insights.EXPECT().ListAlertInstances(&godo.AlertInstanceListOptions{Page: 1}).Return(nil, nil, errors.New("boom"))

		assert.EqualError(t, RunInsightsAlertInstanceList(config), "boom")
	})
}

func TestRunInsightsAlertInstanceGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		buf := withInsightsOutput(t, config, "text")
		config.Args = append(config.Args, testInsightsAlertInstance.ID)

		tm.insights.EXPECT().GetAlertInstance(testInsightsAlertInstance.ID).Return(&testInsightsAlertInstance, nil)

		require.NoError(t, RunInsightsAlertInstanceGet(config))
		assert.Contains(t, buf.String(), testInsightsAlertInstance.ID)
		assert.Contains(t, buf.String(), "do:droplet:12345")
	})
}

func TestRunInsightsAlertInstanceGetRequiresOneArg(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		var missing *doctl.MissingArgsErr
		assert.ErrorAs(t, RunInsightsAlertInstanceGet(config), &missing)

		config.Args = append(config.Args, "a", "b")
		var tooMany *doctl.TooManyArgsErr
		assert.ErrorAs(t, RunInsightsAlertInstanceGet(config), &tooMany)
	})
}
