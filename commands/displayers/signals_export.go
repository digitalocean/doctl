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

// SignalsExport displays Signals export records.
type SignalsExport struct {
	Exports do.SignalsExports
}

var _ Displayable = &SignalsExport{}

func (d *SignalsExport) JSON(out io.Writer) error {
	return writeJSON(d.Exports, out)
}

func (d *SignalsExport) Cols() []string {
	return []string{
		"ID",
		"AgentID",
		"Status",
		"SignalTypes",
		"CreatedAt",
		"CompletedAt",
	}
}

func (d *SignalsExport) ColMap() map[string]string {
	return map[string]string{
		"ID":          "ID",
		"AgentID":     "Agent ID",
		"Status":      "Status",
		"SignalTypes":  "Signal Types",
		"CreatedAt":   "Created At",
		"CompletedAt": "Completed At",
	}
}

func (d *SignalsExport) KV() []map[string]any {
	out := make([]map[string]any, len(d.Exports))
	for i, e := range d.Exports {
		completedAt := ""
		if e.CompletedAt != nil {
			completedAt = *e.CompletedAt
		}
		out[i] = map[string]any{
			"ID":          e.ID,
			"AgentID":     e.AgentID,
			"Status":      e.Status,
			"SignalTypes":  strings.Join(e.SignalTypes, ", "),
			"CreatedAt":   e.CreatedAt,
			"CompletedAt": completedAt,
		}
	}
	return out
}
