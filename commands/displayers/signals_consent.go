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

	"github.com/digitalocean/doctl/do"
)

// SignalsConsent displays Signals consent records.
type SignalsConsent struct {
	Consents do.SignalsConsents
}

var _ Displayable = &SignalsConsent{}

func (d *SignalsConsent) JSON(out io.Writer) error {
	return writeJSON(d.Consents, out)
}

func (d *SignalsConsent) Cols() []string {
	return []string{
		"ID",
		"TeamID",
		"AgentID",
		"Enabled",
		"UpdatedAt",
	}
}

func (d *SignalsConsent) ColMap() map[string]string {
	return map[string]string{
		"ID":        "ID",
		"TeamID":    "Team ID",
		"AgentID":   "Agent ID",
		"Enabled":   "Enabled",
		"UpdatedAt": "Updated At",
	}
}

func (d *SignalsConsent) KV() []map[string]any {
	out := make([]map[string]any, len(d.Consents))
	for i, c := range d.Consents {
		out[i] = map[string]any{
			"ID":        c.ID,
			"TeamID":    c.TeamID,
			"AgentID":   c.AgentID,
			"Enabled":   strconv.FormatBool(c.Enabled),
			"UpdatedAt": c.UpdatedAt,
		}
	}
	return out
}
