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
		"ExportID",
		"AgentID",
		"Status",
		"SignalType",
		"CreatedAt",
		"CompletedAt",
	}
}

func (d *SignalsExport) ColMap() map[string]string {
	return map[string]string{
		"ExportID":    "Export ID",
		"AgentID":     "Agent ID",
		"Status":      "Status",
		"SignalType":  "Signal Type",
		"CreatedAt":   "Created At",
		"CompletedAt": "Completed At",
	}
}

func (d *SignalsExport) KV() []map[string]any {
	out := make([]map[string]any, len(d.Exports))
	for i, e := range d.Exports {
		agentID := ""
		if e.AgentID != nil {
			agentID = *e.AgentID
		}
		completedAt := ""
		if e.CompletedAt != nil {
			completedAt = strconv.FormatInt(*e.CompletedAt, 10)
		}
		out[i] = map[string]any{
			"ExportID":    e.ExportID,
			"AgentID":     agentID,
			"Status":      e.Status,
			"SignalType":  strings.Join(e.Filters.SignalType, ", "),
			"CreatedAt":   e.CreatedAt,
			"CompletedAt": completedAt,
		}
	}
	return out
}

// SignalsExportDownload displays a pre-signed download URL.
type SignalsExportDownload struct {
	Download do.SignalsExportDownload
}

var _ Displayable = &SignalsExportDownload{}

func (d *SignalsExportDownload) JSON(out io.Writer) error {
	return writeJSON(d.Download, out)
}

func (d *SignalsExportDownload) Cols() []string {
	return []string{"DownloadURL", "ExpiresAt"}
}

func (d *SignalsExportDownload) ColMap() map[string]string {
	return map[string]string{
		"DownloadURL": "Download URL",
		"ExpiresAt":   "Expires At",
	}
}

func (d *SignalsExportDownload) KV() []map[string]any {
	return []map[string]any{{
		"DownloadURL": d.Download.DownloadURL,
		"ExpiresAt":   d.Download.ExpiresAt,
	}}
}

// SignalsExportOptions displays GET /exports/options.
type SignalsExportOptions struct {
	Options do.SignalsExportOptions
}

var _ Displayable = &SignalsExportOptions{}

func (d *SignalsExportOptions) JSON(out io.Writer) error {
	return writeJSON(d.Options, out)
}

func (d *SignalsExportOptions) Cols() []string {
	return []string{"SignalType"}
}

func (d *SignalsExportOptions) ColMap() map[string]string {
	return map[string]string{"SignalType": "Signal Type"}
}

func (d *SignalsExportOptions) KV() []map[string]any {
	out := make([]map[string]any, 0, len(d.Options.Filters.SignalType))
	for _, t := range d.Options.Filters.SignalType {
		out = append(out, map[string]any{"SignalType": t})
	}
	return out
}

// SignalsExportTrigger displays the weekly trigger.
type SignalsExportTrigger struct {
	Triggers []do.SignalsExportTrigger
}

var _ Displayable = &SignalsExportTrigger{}

func (d *SignalsExportTrigger) JSON(out io.Writer) error {
	return writeJSON(d.Triggers, out)
}

func (d *SignalsExportTrigger) Cols() []string {
	return []string{"Enabled", "Cadence", "SignalTypes", "UpdatedAt"}
}

func (d *SignalsExportTrigger) ColMap() map[string]string {
	return map[string]string{
		"Enabled":     "Enabled",
		"Cadence":     "Cadence",
		"SignalTypes": "Signal Types",
		"UpdatedAt":   "Updated At",
	}
}

func (d *SignalsExportTrigger) KV() []map[string]any {
	out := make([]map[string]any, len(d.Triggers))
	for i, t := range d.Triggers {
		updated := ""
		if t.UpdatedAt != nil {
			updated = strconv.FormatInt(*t.UpdatedAt, 10)
		}
		out[i] = map[string]any{
			"Enabled":     strconv.FormatBool(t.Enabled),
			"Cadence":     t.Cadence,
			"SignalTypes": strings.Join(t.SignalTypes, ", "),
			"UpdatedAt":   updated,
		}
	}
	return out
}
