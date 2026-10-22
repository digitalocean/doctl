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

func TestInsightsNotificationChannelDisplayer(t *testing.T) {
	updatedAt := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	displayer := &InsightsNotificationChannel{Channels: []godo.NotificationChannel{{
		ID:          "channel-id",
		Name:        "Operations",
		ChannelType: godo.InsightsChannelTypeSlack,
		Slack:       &godo.SlackNotificationConfig{Channel: "#alerts", WebhookURL: "masked"},
		Usage:       &godo.NotificationChannelUsage{RuleCount: 2},
		UpdatedAt:   updatedAt,
	}}}

	require.Len(t, displayer.KV(), 1)
	assert.Equal(t, "slack", displayer.KV()[0]["Type"])
	assert.Equal(t, "#alerts", displayer.KV()[0]["Destination"])
	assert.Equal(t, 2, displayer.KV()[0]["RuleCount"])
	assert.Equal(t, updatedAt.Format(time.RFC3339), displayer.KV()[0]["UpdatedAt"])

	var output bytes.Buffer
	require.NoError(t, displayer.JSON(&output))
	assert.Contains(t, output.String(), `"channel_type": "CHANNEL_TYPE_SLACK"`)
}

func TestInsightsAlertRuleDisplayer(t *testing.T) {
	warning := 75.0
	channels := []godo.NotificationChannelBinding{{NotificationChannelID: "channel-id"}}
	displayer := &InsightsAlertRule{Rules: []godo.AlertRule{{
		ID:     "rule-id",
		Status: godo.InsightsAlertRuleStatusActive,
		Spec: godo.AlertRuleSpec{
			Name:      "High CPU",
			Query:     godo.AlertMetricsQuery{Metric: "do.droplets.cpu_utilization"},
			Condition: &godo.AlertCondition{Window: godo.InsightsEvaluationWindow5m},
			Thresholds: godo.AlertThresholds{
				Warning:  &warning,
				Operator: godo.InsightsThresholdOperatorGreaterThan,
			},
			NotificationChannels: &channels,
		},
	}}}

	require.Len(t, displayer.KV(), 1)
	assert.Equal(t, "active", displayer.KV()[0]["Status"])
	assert.Equal(t, "gt", displayer.KV()[0]["Operator"])
	assert.Equal(t, "5m", displayer.KV()[0]["Window"])
	assert.Equal(t, "channel-id", displayer.KV()[0]["NotificationChannels"])
}
