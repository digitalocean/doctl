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

	"github.com/digitalocean/doctl/do"
)

type AlertPolicy struct {
	AlertPolicies do.AlertPolicies
}

var _ Displayable = &AlertPolicy{}

func (a *AlertPolicy) JSON(out io.Writer) error {
	return writeJSON(a.AlertPolicies, out)
}

func (a *AlertPolicy) Cols() []string {
	return []string{"UUID", "Type", "Description", "Compare",
		"Value", "Window", "Entities", "Tags", "Emails", "Slack Channels", "Enabled"}
}

func (a *AlertPolicy) ColMap() map[string]string {
	return map[string]string{
		"UUID":           "UUID",
		"Type":           "Type",
		"Description":    "Description",
		"Compare":        "Compare",
		"Value":          "Value",
		"Window":         "Window",
		"Entities":       "Entities",
		"Tags":           "Tags",
		"Emails":         "Emails",
		"Slack Channels": "Slack Channels",
		"Enabled":        "Enabled",
	}
}

func (a *AlertPolicy) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(a.AlertPolicies))

	for _, x := range a.AlertPolicies {
		emails := ""
		if x.Alerts.Email != nil {
			emails = strings.Join(x.Alerts.Email, ",")
		}
		slackChannels := make([]string, 0)
		if x.Alerts.Slack != nil {
			for _, v := range x.Alerts.Slack {
				slackChannels = append(slackChannels, v.Channel)
			}
		}
		slacks := strings.Join(slackChannels, ",")

		o := map[string]interface{}{
			"UUID":           x.UUID,
			"Type":           x.Type,
			"Description":    x.Description,
			"Compare":        x.Compare,
			"Value":          x.Value,
			"Window":         x.Window,
			"Entities":       x.Entities,
			"Tags":           x.Tags,
			"Emails":         emails,
			"Slack Channels": slacks,
			"Enabled":        x.Enabled,
		}
		out = append(out, o)
	}

	return out
}
