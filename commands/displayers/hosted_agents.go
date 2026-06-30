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

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// HostedAgentSession wraps one or more hosted-agent sessions for display. The
// same struct backs `doctl agents start`, `... show`, and `... list` — a slice
// of len 1 vs len N is the only difference between them.
type HostedAgentSession struct {
	Sessions []do.HostedAgentSession
}

var _ Displayable = &HostedAgentSession{}

func (h *HostedAgentSession) JSON(out io.Writer) error {
	// Unwrap to the godo type so the JSON output stays flat (matches the
	// server wire format, not doctl's nested do.* wrapper).
	raw := make([]any, 0, len(h.Sessions))
	for _, s := range h.Sessions {
		raw = append(raw, s.HostedAgentSession)
	}
	if len(raw) == 1 {
		return writeJSON(raw[0], out)
	}
	return writeJSON(raw, out)
}

func (h *HostedAgentSession) Cols() []string {
	return []string{"SessionID", "AgentKind", "Status", "RepoHint", "CreatedAt"}
}

func (h *HostedAgentSession) ColMap() map[string]string {
	return map[string]string{
		"SessionID": "Session",
		"AgentKind": "Agent",
		"Status":    "Status",
		"RepoHint":  "Repo",
		"CreatedAt": "Created",
	}
}

func (h *HostedAgentSession) KV() []map[string]any {
	if h == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(h.Sessions))
	for _, s := range h.Sessions {
		if s.HostedAgentSession == nil {
			continue
		}
		out = append(out, map[string]any{
			"SessionID": s.SessionID,
			"AgentKind": s.AgentKind,
			"Status":    s.Status,
			"RepoHint":  s.RepoHint,
			"CreatedAt": s.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}

// HostedAgentWorkspaceUpload renders the result of `doctl agents upload`.
type HostedAgentWorkspaceUpload struct {
	Uploads []*godo.HostedAgentWorkspaceUploadResponse
}

var _ Displayable = &HostedAgentWorkspaceUpload{}

func (h *HostedAgentWorkspaceUpload) JSON(out io.Writer) error {
	if len(h.Uploads) == 1 {
		return writeJSON(h.Uploads[0], out)
	}
	return writeJSON(h.Uploads, out)
}

func (h *HostedAgentWorkspaceUpload) Cols() []string {
	return []string{"Path", "BytesWritten"}
}

func (h *HostedAgentWorkspaceUpload) ColMap() map[string]string {
	return map[string]string{
		"Path":         "Path",
		"BytesWritten": "BytesWritten",
	}
}

func (h *HostedAgentWorkspaceUpload) KV() []map[string]any {
	if h == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(h.Uploads))
	for _, u := range h.Uploads {
		if u == nil {
			continue
		}
		out = append(out, map[string]any{
			"Path":         u.Path,
			"BytesWritten": u.BytesWritten,
		})
	}
	return out
}
