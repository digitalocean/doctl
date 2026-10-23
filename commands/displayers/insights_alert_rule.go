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
	"io"
	"strings"

	"github.com/digitalocean/godo"
)

// InsightsAlertRule displays Insights alert rules.
type InsightsAlertRule struct {
	Rules []godo.AlertRule
}

var _ Displayable = &InsightsAlertRule{}

func (d *InsightsAlertRule) Cols() []string {
	return []string{
		"ID", "Name", "Metric", "Status", "Operator", "Warning", "Critical",
		"Window", "NotificationChannels", "UpdatedAt",
	}
}

func (d *InsightsAlertRule) ColMap() map[string]string {
	return map[string]string{
		"ID":                   "ID",
		"Name":                 "Name",
		"Metric":               "Metric",
		"Status":               "Status",
		"Operator":             "Operator",
		"Warning":              "Warning",
		"Critical":             "Critical",
		"Window":               "Window",
		"NotificationChannels": "NotificationChannels",
		"UpdatedAt":            "UpdatedAt",
	}
}

func (d *InsightsAlertRule) KV() []map[string]any {
	rows := make([]map[string]any, 0, len(d.Rules))
	for _, rule := range d.Rules {
		var warning, critical any = "", ""
		if rule.Spec.Thresholds.Warning != nil {
			warning = *rule.Spec.Thresholds.Warning
		}
		if rule.Spec.Thresholds.Critical != nil {
			critical = *rule.Spec.Thresholds.Critical
		}

		window := ""
		if rule.Spec.Condition != nil {
			window = insightsAlertWindow(rule.Spec.Condition.Window)
		}

		channelIDs := []string{}
		if rule.Spec.NotificationChannels != nil {
			for _, binding := range *rule.Spec.NotificationChannels {
				channelIDs = append(channelIDs, binding.NotificationChannelID)
			}
		}

		rows = append(rows, map[string]any{
			"ID":                   rule.ID,
			"Name":                 rule.Spec.Name,
			"Metric":               rule.Spec.Query.Metric,
			"Status":               insightsAlertStatus(rule.Status),
			"Operator":             insightsAlertOperator(rule.Spec.Thresholds.Operator),
			"Warning":              warning,
			"Critical":             critical,
			"Window":               window,
			"NotificationChannels": strings.Join(channelIDs, ","),
			"UpdatedAt":            formatInsightsTime(rule.UpdatedAt),
		})
	}
	return rows
}

func (d *InsightsAlertRule) JSON(out io.Writer) error {
	return writeJSON(d.Rules, out)
}

func insightsAlertStatus(status string) string {
	switch status {
	case godo.InsightsAlertRuleStatusActive:
		return "active"
	case godo.InsightsAlertRuleStatusPaused:
		return "paused"
	default:
		return status
	}
}

func insightsAlertOperator(operator string) string {
	values := map[string]string{
		godo.InsightsThresholdOperatorEqual:              "eq",
		godo.InsightsThresholdOperatorNotEqual:           "neq",
		godo.InsightsThresholdOperatorLessThan:           "lt",
		godo.InsightsThresholdOperatorLessThanOrEqual:    "lte",
		godo.InsightsThresholdOperatorGreaterThan:        "gt",
		godo.InsightsThresholdOperatorGreaterThanOrEqual: "gte",
	}
	if value, ok := values[operator]; ok {
		return value
	}
	return operator
}

func insightsAlertWindow(window string) string {
	values := map[string]string{
		godo.InsightsEvaluationWindow1m:  "1m",
		godo.InsightsEvaluationWindow5m:  "5m",
		godo.InsightsEvaluationWindow10m: "10m",
		godo.InsightsEvaluationWindow15m: "15m",
		godo.InsightsEvaluationWindow30m: "30m",
		godo.InsightsEvaluationWindow1h:  "1h",
	}
	if value, ok := values[window]; ok {
		return value
	}
	return window
}
