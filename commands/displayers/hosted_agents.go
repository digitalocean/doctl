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
// same struct backs `doctl open-harness-runtime start`, `... show`, and `... list` — a slice
// of len 1 vs len N is the only difference between them.
// Set Single=true for get/create verbs so JSON output is a bare object;
// leave false (default) for list so JSON output is always an array.
type HostedAgentSession struct {
	Sessions []do.HostedAgentSession
	Single   bool
}

var _ Displayable = &HostedAgentSession{}

func (h *HostedAgentSession) JSON(out io.Writer) error {
	// Unwrap to the godo type so the JSON output stays flat (matches the
	// server wire format, not doctl's nested do.* wrapper).
	raw := make([]any, 0, len(h.Sessions))
	for _, s := range h.Sessions {
		raw = append(raw, s.HostedAgentSession)
	}
	if h.Single && len(raw) == 1 {
		return writeJSON(raw[0], out)
	}
	return writeJSON(raw, out)
}

func (h *HostedAgentSession) Cols() []string {
	return []string{"SessionID", "Name", "AgentKind", "Status", "ConfigID", "ParentSessionID", "ForkID", "RepoHint", "CreatedAt"}
}

func (h *HostedAgentSession) ColMap() map[string]string {
	return map[string]string{
		"SessionID":       "Session",
		"Name":            "Name",
		"AgentKind":       "Agent",
		"Status":          "Status",
		"ConfigID":        "Config",
		"ParentSessionID": "Parent",
		"ForkID":          "Fork",
		"RepoHint":        "Repo",
		"CreatedAt":       "Created",
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
			"SessionID":       s.SessionID,
			"Name":            s.Name,
			"AgentKind":       s.AgentKind,
			"Status":          s.Status,
			"ConfigID":        s.ConfigID,
			"ParentSessionID": s.ParentSessionID,
			"ForkID":          s.ForkID,
			"RepoHint":        s.RepoHint,
			"CreatedAt":       s.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}

// HostedAgentCheckpoint wraps one or more session checkpoints for display.
type HostedAgentCheckpoint struct {
	Checkpoints []godo.HostedAgentCheckpoint
	Single      bool
}

var _ Displayable = &HostedAgentCheckpoint{}

func (h *HostedAgentCheckpoint) JSON(out io.Writer) error {
	if h.Single && len(h.Checkpoints) == 1 {
		return writeJSON(h.Checkpoints[0], out)
	}
	return writeJSON(h.Checkpoints, out)
}

func (h *HostedAgentCheckpoint) Cols() []string {
	return []string{"CheckpointID", "SessionID", "Status", "Kind", "Label", "SizeBytes", "CreatedAt"}
}

func (h *HostedAgentCheckpoint) ColMap() map[string]string {
	return map[string]string{
		"CheckpointID": "Checkpoint",
		"SessionID":    "Session",
		"Status":       "Status",
		"Kind":         "Kind",
		"Label":        "Label",
		"SizeBytes":    "SizeBytes",
		"CreatedAt":    "Created",
	}
}

func (h *HostedAgentCheckpoint) KV() []map[string]any {
	if h == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(h.Checkpoints))
	for _, cp := range h.Checkpoints {
		out = append(out, map[string]any{
			"CheckpointID": cp.CheckpointID,
			"SessionID":    cp.SessionID,
			"Status":       cp.Status,
			"Kind":         cp.Kind,
			"Label":        cp.Label,
			"SizeBytes":    cp.SizeBytes,
			"CreatedAt":    cp.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}

// HostedAgentWorkspaceUpload renders the result of `doctl open-harness-runtime upload`.
// Upload is single-file-only today (Uploads always has exactly one element),
// so JSON() defaults to list semantics (always an array) like every other
// list-shaped displayer; set Single=true if a future bare-object verb needs it
// (see MARSOHS-869/887 — a bare len==1 unwrap silently changes the container
// type based on row count, breaking callers that range or count the result).
type HostedAgentWorkspaceUpload struct {
	Uploads []*godo.HostedAgentWorkspaceUploadResponse
	Single  bool
}

var _ Displayable = &HostedAgentWorkspaceUpload{}

func (h *HostedAgentWorkspaceUpload) JSON(out io.Writer) error {
	if h.Single && len(h.Uploads) == 1 {
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

// HostedAgentSandboxExec renders the result of `doctl open-harness-runtime exec`
// under `-o json`. Text output never reaches this displayer: the command passes
// the guest's stdout and stderr straight through instead, so it composes in a
// pipeline.
//
// Single is always set by the exec verb (one command, one result), but the field
// is kept explicit rather than unwrapping on len==1, so the JSON container type
// never changes with the row count (MARSOHS-869/887).
type HostedAgentSandboxExec struct {
	Execs  []*godo.HostedAgentSandboxExecResponse
	Single bool
}

var _ Displayable = &HostedAgentSandboxExec{}

func (h *HostedAgentSandboxExec) JSON(out io.Writer) error {
	if h.Single && len(h.Execs) == 1 {
		return writeJSON(h.Execs[0], out)
	}
	return writeJSON(h.Execs, out)
}

func (h *HostedAgentSandboxExec) Cols() []string {
	return []string{"ExitCode", "Stdout", "Stderr"}
}

func (h *HostedAgentSandboxExec) ColMap() map[string]string {
	return map[string]string{
		"ExitCode": "ExitCode",
		"Stdout":   "Stdout",
		"Stderr":   "Stderr",
	}
}

func (h *HostedAgentSandboxExec) KV() []map[string]any {
	if h == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(h.Execs))
	for _, e := range h.Execs {
		if e == nil {
			continue
		}
		out = append(out, map[string]any{
			"ExitCode": e.ExitCode,
			"Stdout":   e.Stdout,
			"Stderr":   e.Stderr,
		})
	}
	return out
}

// HostedAgentConfig renders full Agent Configs, backing `doctl open-harness-runtime config
// get` and `... create`. Set Single=true so those verbs emit a bare JSON
// object; the list verb uses HostedAgentConfigSummary instead.
type HostedAgentConfig struct {
	Configs []godo.HostedAgentConfig
	Single  bool
}

var _ Displayable = &HostedAgentConfig{}

func (h *HostedAgentConfig) JSON(out io.Writer) error {
	if h.Single && len(h.Configs) == 1 {
		return writeJSON(h.Configs[0], out)
	}
	return writeJSON(h.Configs, out)
}

func (h *HostedAgentConfig) Cols() []string {
	return []string{"ID", "Name", "SchemaVersion", "ContentHash", "CreatedBy", "CreatedAt"}
}

func (h *HostedAgentConfig) ColMap() map[string]string {
	return map[string]string{
		"ID":            "ID",
		"Name":          "Name",
		"SchemaVersion": "SchemaVersion",
		"ContentHash":   "ContentHash",
		"CreatedBy":     "CreatedBy",
		"CreatedAt":     "Created",
	}
}

func (h *HostedAgentConfig) KV() []map[string]any {
	if h == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(h.Configs))
	for _, c := range h.Configs {
		out = append(out, map[string]any{
			"ID":            c.ID,
			"Name":          c.Name,
			"SchemaVersion": c.AgentSpecSchemaVersion,
			"ContentHash":   c.ContentHash,
			"CreatedBy":     c.CreatedBy,
			"CreatedAt":     c.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}

// HostedAgentConfigSummary renders the list view of Agent Configs (no manifest
// or credential slots), backing `doctl open-harness-runtime config list`.
type HostedAgentConfigSummary struct {
	Configs []godo.HostedAgentConfigSummary
}

var _ Displayable = &HostedAgentConfigSummary{}

func (h *HostedAgentConfigSummary) JSON(out io.Writer) error {
	return writeJSON(h.Configs, out)
}

func (h *HostedAgentConfigSummary) Cols() []string {
	return []string{"ID", "Name", "SchemaVersion", "ContentHash", "CreatedBy", "CreatedAt"}
}

func (h *HostedAgentConfigSummary) ColMap() map[string]string {
	return map[string]string{
		"ID":            "ID",
		"Name":          "Name",
		"SchemaVersion": "SchemaVersion",
		"ContentHash":   "ContentHash",
		"CreatedBy":     "CreatedBy",
		"CreatedAt":     "Created",
	}
}

func (h *HostedAgentConfigSummary) KV() []map[string]any {
	if h == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(h.Configs))
	for _, c := range h.Configs {
		out = append(out, map[string]any{
			"ID":            c.ID,
			"Name":          c.Name,
			"SchemaVersion": c.AgentSpecSchemaVersion,
			"ContentHash":   c.ContentHash,
			"CreatedBy":     c.CreatedBy,
			"CreatedAt":     c.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}
