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
	"time"

	"github.com/digitalocean/godo"
)

// InsightsNotificationChannel displays Insights notification channels.
type InsightsNotificationChannel struct {
	Channels []godo.NotificationChannel
}

var _ Displayable = &InsightsNotificationChannel{}

func (d *InsightsNotificationChannel) Cols() []string {
	return []string{"ID", "Name", "Type", "Destination", "RuleCount", "CreatedAt", "UpdatedAt"}
}

func (d *InsightsNotificationChannel) ColMap() map[string]string {
	return map[string]string{
		"ID":          "ID",
		"Name":        "Name",
		"Type":        "Type",
		"Destination": "Destination",
		"RuleCount":   "RuleCount",
		"CreatedAt":   "CreatedAt",
		"UpdatedAt":   "UpdatedAt",
	}
}

func (d *InsightsNotificationChannel) KV() []map[string]any {
	rows := make([]map[string]any, 0, len(d.Channels))
	for _, channel := range d.Channels {
		ruleCount := 0
		if channel.Usage != nil {
			ruleCount = channel.Usage.RuleCount
		}
		rows = append(rows, map[string]any{
			"ID":          channel.ID,
			"Name":        channel.Name,
			"Type":        notificationChannelType(channel.ChannelType),
			"Destination": notificationChannelDestination(channel),
			"RuleCount":   ruleCount,
			"CreatedAt":   formatInsightsTime(channel.CreatedAt),
			"UpdatedAt":   formatInsightsTime(channel.UpdatedAt),
		})
	}
	return rows
}

func (d *InsightsNotificationChannel) JSON(out io.Writer) error {
	return writeJSON(d.Channels, out)
}

func notificationChannelType(channelType string) string {
	switch channelType {
	case godo.InsightsChannelTypeEmail:
		return "email"
	case godo.InsightsChannelTypeSlack:
		return "slack"
	case godo.InsightsChannelTypeWebhook:
		return "webhook"
	default:
		return channelType
	}
}

func notificationChannelDestination(channel godo.NotificationChannel) string {
	switch {
	case channel.Email != nil:
		return channel.Email.To
	case channel.Slack != nil:
		return channel.Slack.Channel
	case channel.Webhook != nil:
		return channel.Webhook.URL
	default:
		return ""
	}
}

func formatInsightsTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
