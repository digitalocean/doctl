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

	"github.com/digitalocean/doctl/do"
)

var _ Displayable = &UptimeAlert{}

type UptimeAlert struct {
	UptimeAlerts []do.UptimeAlert
}

func (ua *UptimeAlert) JSON(out io.Writer) error {
	return writeJSON(ua.UptimeAlerts, out)
}

func (ua *UptimeAlert) Cols() []string {
	return []string{
		"ID", "Name", "Type", "Threshold", "Comparison", "Notifications", "Period",
	}
}

func (ua *UptimeAlert) ColMap() map[string]string {
	return map[string]string{
		"ID":            "ID",
		"Name":          "Name",
		"Type":          "Type",
		"Threshold":     "Threshold",
		"Comparison":    "Comparison",
		"Notifications": "Notifications",
		"Period":        "Period",
	}
}

func (ua *UptimeAlert) KV() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(ua.UptimeAlerts))
	for _, uptimeAlert := range ua.UptimeAlerts {
		m := map[string]interface{}{
			"ID":           uptimeAlert.ID,
			"Name":         uptimeAlert.Name,
			"Type":         uptimeAlert.Type,
			"Threshold":    uptimeAlert.Threshold,
			"Comparison":   uptimeAlert.Comparison,
			"Notification": uptimeAlert.Notifications,
			"Period":       uptimeAlert.Period,
		}
		out = append(out, m)
	}
	return out
}
