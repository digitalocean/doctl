/*
Copyright 2026 The Doctl Authors All rights reserved.
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

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// SandboxWorkspace displays sandbox workspaces.
type SandboxWorkspace struct {
	Workspaces do.SandboxWorkspaces
}

var _ Displayable = &SandboxWorkspace{}

func (s *SandboxWorkspace) JSON(out io.Writer) error {
	return writeJSON(s.Workspaces, out)
}

func (s *SandboxWorkspace) Cols() []string {
	return []string{"ID", "Name", "State", "SizeLimitGiB", "AttachedSandbox", "BytesUsed", "Created"}
}

func (s *SandboxWorkspace) ColMap() map[string]string {
	return map[string]string{
		"ID":              "ID",
		"Name":            "Name",
		"State":           "State",
		"SizeLimitGiB":    "Size Limit (GiB)",
		"AttachedSandbox": "Attached Sandbox",
		"BytesUsed":       "Bytes Used",
		"Created":         "Created At",
	}
}

func (s *SandboxWorkspace) KV() []map[string]any {
	out := make([]map[string]any, 0, len(s.Workspaces))
	for _, w := range s.Workspaces {
		out = append(out, map[string]any{
			"ID":              w.ID,
			"Name":            w.Name,
			"State":           FormatSandboxWorkspaceState(w.State),
			"SizeLimitGiB":    w.SizeLimitGiB,
			"AttachedSandbox": w.AttachedSandboxID,
			"BytesUsed":       w.Used(),
			"Created":         w.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

// FormatSandboxWorkspaceState turns an API state such as
// WORKSPACE_STATE_AVAILABLE into the short form shown in tables: available.
// JSON output keeps the API value.
func FormatSandboxWorkspaceState(state godo.SandboxWorkspaceState) string {
	return formatSandboxStatus(string(state), "WORKSPACE_STATE_")
}
