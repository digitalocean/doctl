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
	"fmt"
	"io"
	"strings"

	"github.com/digitalocean/doctl/do"
)

// HostedAgentTrigger wraps one or more hosted-agent triggers for display.
type HostedAgentTrigger struct {
	Triggers []do.HostedAgentTrigger
}

var _ Displayable = &HostedAgentTrigger{}

func (h *HostedAgentTrigger) JSON(out io.Writer) error {
	raw := make([]any, 0, len(h.Triggers))
	for _, t := range h.Triggers {
		raw = append(raw, t.HostedAgentTrigger)
	}
	if len(raw) == 1 {
		return writeJSON(raw[0], out)
	}
	return writeJSON(raw, out)
}

func (h *HostedAgentTrigger) Cols() []string {
	return []string{
		"TriggerID", "Name", "Kind", "Status", "SessionMode", "AgentKind",
		"BoundSessionID", "WebhookURL", "NextRunAt", "CreatedAt",
	}
}

func (h *HostedAgentTrigger) ColMap() map[string]string {
	return map[string]string{
		"TriggerID":      "ID",
		"Name":           "Name",
		"Kind":           "Kind",
		"Status":         "Status",
		"SessionMode":    "Session Mode",
		"AgentKind":      "Agent",
		"BoundSessionID": "Bound Session",
		"WebhookURL":     "Webhook URL",
		"NextRunAt":      "Next Run",
		"CreatedAt":      "Created",
	}
}

func (h *HostedAgentTrigger) KV() []map[string]any {
	if h == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(h.Triggers))
	for _, t := range h.Triggers {
		if t.HostedAgentTrigger == nil {
			continue
		}
		row := map[string]any{
			"TriggerID":      t.TriggerID,
			"Name":           t.Name,
			"Kind":           t.Kind,
			"Status":         t.Status,
			"SessionMode":    t.SessionMode,
			"AgentKind":      t.AgentKind,
			"BoundSessionID": t.BoundSessionID,
			"WebhookURL":     "",
			"NextRunAt":      "",
			"CreatedAt":      t.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if t.Webhook != nil {
			row["WebhookURL"] = t.Webhook.WebhookURL
		}
		if t.Cron != nil && !t.Cron.NextRunAt.Time.IsZero() {
			row["NextRunAt"] = t.Cron.NextRunAt.Time.UTC().Format("2006-01-02T15:04:05Z")
		}
		out = append(out, row)
	}
	return out
}

// HostedAgentTriggerExecution wraps trigger execution rows for display.
type HostedAgentTriggerExecution struct {
	Executions []do.HostedAgentTriggerExecution
}

var _ Displayable = &HostedAgentTriggerExecution{}

func (h *HostedAgentTriggerExecution) JSON(out io.Writer) error {
	raw := make([]any, 0, len(h.Executions))
	for _, e := range h.Executions {
		raw = append(raw, e.HostedAgentTriggerExecution)
	}
	if len(raw) == 1 {
		return writeJSON(raw[0], out)
	}
	return writeJSON(raw, out)
}

func (h *HostedAgentTriggerExecution) Cols() []string {
	return []string{"ExecutionID", "TriggerID", "Status", "SessionID", "RunID", "FailureReason", "CreatedAt"}
}

func (h *HostedAgentTriggerExecution) ColMap() map[string]string {
	return map[string]string{
		"ExecutionID":   "Execution",
		"TriggerID":     "Trigger",
		"Status":        "Status",
		"SessionID":     "Session",
		"RunID":         "Run",
		"FailureReason": "Failure",
		"CreatedAt":     "Created",
	}
}

func (h *HostedAgentTriggerExecution) KV() []map[string]any {
	if h == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(h.Executions))
	for _, e := range h.Executions {
		if e.HostedAgentTriggerExecution == nil {
			continue
		}
		out = append(out, map[string]any{
			"ExecutionID":   e.ExecutionID,
			"TriggerID":     e.TriggerID,
			"Status":        e.Status,
			"SessionID":     e.SessionID,
			"RunID":         e.RunID,
			"FailureReason": e.FailureReason,
			"CreatedAt":     e.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}

// HostedAgentWebhookProvider wraps webhook provider registry entries.
type HostedAgentWebhookProvider struct {
	Providers []do.HostedAgentWebhookProvider
}

var _ Displayable = &HostedAgentWebhookProvider{}

func (h *HostedAgentWebhookProvider) JSON(out io.Writer) error {
	raw := make([]any, 0, len(h.Providers))
	for _, p := range h.Providers {
		raw = append(raw, p.HostedAgentWebhookProvider)
	}
	if len(raw) == 1 {
		return writeJSON(raw[0], out)
	}
	return writeJSON(raw, out)
}

func (h *HostedAgentWebhookProvider) Cols() []string {
	return []string{"Key", "DisplayName", "SignatureHeader", "SignatureScheme", "DocsURL"}
}

func (h *HostedAgentWebhookProvider) ColMap() map[string]string {
	return map[string]string{
		"Key":             "Key",
		"DisplayName":     "Name",
		"SignatureHeader": "Header",
		"SignatureScheme": "Scheme",
		"DocsURL":         "Docs",
	}
}

func (h *HostedAgentWebhookProvider) KV() []map[string]any {
	if h == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(h.Providers))
	for _, p := range h.Providers {
		if p.HostedAgentWebhookProvider == nil {
			continue
		}
		row := map[string]any{
			"Key":             p.Key,
			"DisplayName":     p.DisplayName,
			"SignatureHeader": "",
			"SignatureScheme": "",
			"DocsURL":         p.DocsURL,
		}
		if p.Signature != nil {
			row["SignatureHeader"] = p.Signature.Header
			row["SignatureScheme"] = p.Signature.Scheme
		}
		out = append(out, row)
	}
	return out
}

// HostedAgentReusableSession wraps paused sessions for the reuse picker.
type HostedAgentReusableSession struct {
	Sessions []do.HostedAgentReusableSession
}

var _ Displayable = &HostedAgentReusableSession{}

func (h *HostedAgentReusableSession) JSON(out io.Writer) error {
	raw := make([]any, 0, len(h.Sessions))
	for _, s := range h.Sessions {
		raw = append(raw, s.HostedAgentReusableSession)
	}
	if len(raw) == 1 {
		return writeJSON(raw[0], out)
	}
	return writeJSON(raw, out)
}

func (h *HostedAgentReusableSession) Cols() []string {
	return []string{"SessionID", "Name", "AgentKind", "Status", "CreatedAt", "LastEventAt"}
}

func (h *HostedAgentReusableSession) ColMap() map[string]string {
	return map[string]string{
		"SessionID":   "Session",
		"Name":        "Name",
		"AgentKind":   "Agent",
		"Status":      "Status",
		"CreatedAt":   "Created",
		"LastEventAt": "Last Event",
	}
}

func (h *HostedAgentReusableSession) KV() []map[string]any {
	if h == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(h.Sessions))
	for _, s := range h.Sessions {
		if s.HostedAgentReusableSession == nil {
			continue
		}
		row := map[string]any{
			"SessionID":   s.SessionID,
			"Name":        s.Name,
			"AgentKind":   s.AgentKind,
			"Status":      s.Status,
			"CreatedAt":   s.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z"),
			"LastEventAt": "",
		}
		if !s.LastEventAt.Time.IsZero() {
			row["LastEventAt"] = s.LastEventAt.Time.UTC().Format("2006-01-02T15:04:05Z")
		}
		out = append(out, row)
	}
	return out
}

// FormatWebhookSecretNotice returns a one-time-secret warning for create/rotate.
func FormatWebhookSecretNotice(secret string) string {
	if secret == "" {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Webhook secret (shown once — store it now):\n%s\n", secret)
	return b.String()
}
