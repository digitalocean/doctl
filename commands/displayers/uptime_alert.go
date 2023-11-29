/*
Copyright 2023 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/doctl/do"
)

type UptimeAlert struct {
	UptimeAlerts []do.UptimeAlert
}

func (uc *UptimeAlert) JSON(out io.Writer) error {
	return writeJSON(uc.UptimeAlerts, out)
}

func (uc *UptimeAlert) Cols() []string {
	return []string{
		"ID", "Name", "Type", "Threshold", "Comparison", "Period", "Emails", "Slack Channels",
	}
}

func (ua *UptimeAlert) ColMap() map[string]string {
	return map[string]string{
		"ID":             "ID",
		"Name":           "Name",
		"Type":           "Type",
		"Threshold":      "Threshold",
		"Comparison":     "Comparison",
		"Period":         "Period",
		"Emails":         "Emails",
		"Slack Channels": "Slack Channels",
	}
}

func (ua *UptimeAlert) KV() []map[string]any {
	out := make([]map[string]any, 0, len(ua.UptimeAlerts))
	for _, uptimeAlert := range ua.UptimeAlerts {
		emails := ""
		if uptimeAlert.Notifications.Email != nil {
			emails = strings.Join(uptimeAlert.Notifications.Email, ",")
		}
		slackChannels := make([]string, 0)
		if uptimeAlert.Notifications.Slack != nil {
			for _, v := range uptimeAlert.Notifications.Slack {
				slackChannels = append(slackChannels, v.Channel)
			}
		}
		slacks := strings.Join(slackChannels, ",")

		m := map[string]any{
			"ID":             uptimeAlert.ID,
			"Name":           uptimeAlert.Name,
			"Type":           uptimeAlert.Type,
			"Threshold":      uptimeAlert.Threshold,
			"Comparison":     uptimeAlert.Comparison,
			"Period":         uptimeAlert.Period,
			"Emails":         emails,
			"Slack Channels": slacks,
		}
		out = append(out, m)
	}
	return out
}
