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
	"strconv"
	"time"

	"github.com/digitalocean/godo"
)

// InsightsAlertInstances displays Insights alert instances.
type InsightsAlertInstances struct {
	AlertInstances []godo.AlertInstance
}

var _ Displayable = &InsightsAlertInstances{}

func (a *InsightsAlertInstances) JSON(out io.Writer) error {
	return writeJSON(a.AlertInstances, out)
}

func (a *InsightsAlertInstances) Cols() []string {
	return []string{"ID", "RuleID", "Severity", "Status", "ResourceURN", "Value", "TriggeredAt", "LastTriggeredAt", "ResolvedAt"}
}

func (a *InsightsAlertInstances) ColMap() map[string]string {
	return map[string]string{
		"ID":              "ID",
		"RuleID":          "Rule ID",
		"Severity":        "Severity",
		"Status":          "Status",
		"ResourceURN":     "Resource URN",
		"Value":           "Value",
		"TriggeredAt":     "Triggered At",
		"LastTriggeredAt": "Last Triggered At",
		"ResolvedAt":      "Resolved At",
		"LastNotifiedAt":  "Last Notified At",
	}
}

func (a *InsightsAlertInstances) KV() []map[string]any {
	out := make([]map[string]any, 0, len(a.AlertInstances))
	for _, ai := range a.AlertInstances {
		out = append(out, map[string]any{
			"ID":              ai.ID,
			"RuleID":          ai.RuleID,
			"Severity":        ai.Severity,
			"Status":          ai.Status,
			"ResourceURN":     ai.ResourceURN,
			"Value":           strconv.FormatFloat(ai.Value, 'f', -1, 64),
			"TriggeredAt":     formatInsightsOptionalTime(&ai.TriggeredAt),
			"LastTriggeredAt": formatInsightsOptionalTime(&ai.LastTriggeredAt),
			"ResolvedAt":      formatInsightsOptionalTime(ai.ResolvedAt),
			"LastNotifiedAt":  formatInsightsOptionalTime(ai.LastNotifiedAt),
		})
	}
	return out
}

func formatInsightsOptionalTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
