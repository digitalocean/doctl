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

func testAlertRule() godo.AlertRule {
	critical := 90.0
	channels := []godo.NotificationChannelBinding{{
		NotificationChannelID: "channel-id",
		NotifyOn:              []string{godo.InsightsSeverityCritical},
	}}
	return godo.AlertRule{
		ID:     "rule-id",
		Status: godo.InsightsAlertRuleStatusActive,
		Spec: godo.AlertRuleSpec{
			Name: "High CPU",
			Query: godo.AlertMetricsQuery{
				Metric:       "do.droplets.cpu_utilization",
				ResourceURNs: []string{"do:droplet:123"},
				Filters: []godo.AlertQueryFilter{{
					Field:    "region",
					Operator: godo.InsightsFilterOperatorEqual,
					Value:    "nyc3",
				}},
			},
			Condition: &godo.AlertCondition{Window: godo.InsightsEvaluationWindow5m},
			Thresholds: godo.AlertThresholds{
				Critical: &critical,
				Operator: godo.InsightsThresholdOperatorGreaterThan,
			},
			NotificationChannels: &channels,
		},
	}
}

func TestInsightsAlertRuleCommand(t *testing.T) {
	assertCommandNames(t, InsightsAlertRule(), "create", "get", "list", "update", "delete")
}

func TestInsightsAlertRuleFilterFlagPreservesCommas(t *testing.T) {
	command, _, err := InsightsAlertRule().Find([]string{"create"})
	require.NoError(t, err)
	require.NoError(t, command.Flags().Set(doctl.ArgInsightsFilter, "region,eq,nyc3"))

	filters, err := command.Flags().GetStringArray(doctl.ArgInsightsFilter)
	require.NoError(t, err)
	assert.Equal(t, []string{"region,eq,nyc3"}, filters)
}

func TestRunInsightsAlertRuleCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		rule := testAlertRule()
		request := &godo.AlertRuleRequest{Spec: rule.Spec, Status: godo.InsightsAlertRuleStatusActive}
		tm.insights.EXPECT().CreateAlertRule(request).Return(&rule, nil)

		config.Doit.Set(config.NS, doctl.ArgInsightsName, "High CPU")
		config.Doit.Set(config.NS, doctl.ArgInsightsMetric, "do.droplets.cpu_utilization")
		config.Doit.Set(config.NS, doctl.ArgInsightsResourceURN, []string{"do:droplet:123"})
		config.Doit.Set(config.NS, doctl.ArgInsightsFilter, []string{"region,eq,nyc3"})
		config.Doit.Set(config.NS, doctl.ArgInsightsWindow, "5m")
		config.Doit.Set(config.NS, doctl.ArgInsightsOperator, "gt")
		config.Doit.Set(config.NS, doctl.ArgInsightsCriticalThreshold, 90.0)
		config.Doit.Set(config.NS, doctl.ArgInsightsNotificationChannel, []string{"channel-id"})
		config.Doit.Set(config.NS, doctl.ArgInsightsNotifyOn, []string{"critical"})
		config.Doit.Set(config.NS, doctl.ArgInsightsStatus, "active")

		require.NoError(t, RunInsightsAlertRuleCreate(config))
	})
}

func TestRunInsightsAlertRuleCreateRequiresThreshold(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgInsightsName, "High CPU")
		config.Doit.Set(config.NS, doctl.ArgInsightsMetric, "do.droplets.cpu_utilization")
		config.Doit.Set(config.NS, doctl.ArgInsightsOperator, "gt")
		config.Doit.Set(config.NS, doctl.ArgInsightsNotificationChannel, []string{"channel-id"})

		err := RunInsightsAlertRuleCreate(config)
		require.EqualError(t, err, "at least one warning or critical threshold is required")
	})
}

func TestRunInsightsAlertRuleGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		rule := testAlertRule()
		tm.insights.EXPECT().GetAlertRule("rule-id").Return(&rule, nil)
		config.Args = []string{"rule-id"}

		require.NoError(t, RunInsightsAlertRuleGet(config))
	})
}

func TestRunInsightsAlertRuleList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		rule := testAlertRule()
		options := &godo.AlertRuleListOptions{
			Page:        3,
			PerPage:     10,
			ResourceURN: "do:droplet:123",
		}
		tm.insights.EXPECT().ListAlertRules(options).Return([]godo.AlertRule{rule}, nil, nil)
		config.Doit.Set(config.NS, doctl.ArgInsightsPage, 3)
		config.Doit.Set(config.NS, doctl.ArgInsightsPerPage, 10)
		config.Doit.Set(config.NS, doctl.ArgInsightsResourceURN, "do:droplet:123")

		require.NoError(t, RunInsightsAlertRuleList(config))
	})
}

func TestRunInsightsAlertRuleUpdateOmitsUnchangedOptionalFields(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		rule := testAlertRule()
		updated := rule
		updated.Status = godo.InsightsAlertRuleStatusPaused
		expectedSpec := rule.Spec
		expectedSpec.NotificationChannels = nil
		expectedSpec.ReAlertDuration = ""
		request := &godo.AlertRuleRequest{
			Spec:   expectedSpec,
			Status: godo.InsightsAlertRuleStatusPaused,
		}
		tm.insights.EXPECT().GetAlertRule("rule-id").Return(&rule, nil)
		tm.insights.EXPECT().UpdateAlertRule("rule-id", request).Return(&updated, nil)

		config.Args = []string{"rule-id"}
		config.Doit.Set(config.NS, doctl.ArgInsightsStatus, "paused")

		require.NoError(t, RunInsightsAlertRuleUpdate(config))
	})
}

func TestRunInsightsAlertRuleDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.insights.EXPECT().DeleteAlertRule("rule-id").Return(nil)
		config.Args = []string{"rule-id"}

		require.NoError(t, RunInsightsAlertRuleDelete(config))
	})
}

func TestParseAlertRuleFilters(t *testing.T) {
	filters, err := parseAlertRuleFilters([]string{"region,eq,nyc3", "service,neq,api,worker"})
	require.NoError(t, err)
	assert.Equal(t, []godo.AlertQueryFilter{
		{Field: "region", Operator: godo.InsightsFilterOperatorEqual, Value: "nyc3"},
		{Field: "service", Operator: godo.InsightsFilterOperatorNotEqual, Value: "api,worker"},
	}, filters)
}
