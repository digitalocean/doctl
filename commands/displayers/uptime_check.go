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
	"fmt"
	"io"

	"github.com/digitalocean/doctl/do"
)

type UptimeCheck struct {
	UptimeChecks []do.UptimeCheck
}

var _ Displayable = &UptimeCheck{}

func (uc *UptimeCheck) JSON(out io.Writer) error {
	return writeJSON(uc.UptimeChecks, out)

}

func (uc *UptimeCheck) Cols() []string {
	return []string{
		"ID", "Name", "Type", "Target", "Regions", "Enabled",
	}
}

func (uc *UptimeCheck) ColMap() map[string]string {
	return map[string]string{
		"ID":      "ID",
		"Name":    "Name",
		"Type":    "Type",
		"Target":  "Target",
		"Regions": "Regions",
		"Enabled": "Enabled",
	}
}

func (uc *UptimeCheck) KV() []map[string]any {
	out := make([]map[string]any, 0, len(uc.UptimeChecks))
	for _, uptimeCheck := range uc.UptimeChecks {
		m := map[string]any{
			"ID":      uptimeCheck.ID,
			"Name":    uptimeCheck.Name,
			"Type":    uptimeCheck.Type,
			"Target":  uptimeCheck.Target,
			"Enabled": uptimeCheck.Enabled,
		}
		m["Regions"] = ""
		if len(uptimeCheck.Regions) > 0 {
			m["Regions"] = fmt.Sprintf("%v", uptimeCheck.Regions)
		}
		out = append(out, m)
	}
	return out
}
