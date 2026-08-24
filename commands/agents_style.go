/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    15|See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
)

// relativeTimeAgo renders t as a short relative duration ("just now", "5m
// ago", "3h ago", "2d ago"). Falls back to a plain date once it's more than
// 30 days old, where day-level relative times stop being a useful measure.
func relativeTimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	default:
		return t.UTC().Format("2006-01-02")
	}
}

// formatCreatedAt renders a resource's creation time: a short relative
// duration by default ("5m ago"), or the full UTC timestamp with -v/--verbose.
func formatCreatedAt(t time.Time) string {
	if Verbose {
		return t.UTC().Format("2006-01-02 15:04")
	}
	return relativeTimeAgo(t)
}

// createdAgo renders "Created <formatCreatedAt(t)>" for inline meta lines
// that don't have their own "Created" row label.
func createdAgo(t time.Time) string {
	return "Created " + formatCreatedAt(t)
}

// printAgentSuccess prints a compact success line used by side-effect verbs.
func printAgentSuccess(w io.Writer, msg string) {
	fmt.Fprintf(w, "%s %s\n", colorize("✓", colSuccess), msg)
}

// printAgentNextPage prints a muted pagination hint when next is non-empty.
func printAgentNextPage(w io.Writer, next string) {
	if next == "" {
		return
	}
	fmt.Fprintf(w, "\n%s %s\n", colorize("Next page token:", colMuted), next)
}

// --- checkpoints ------------------------------------------------------------

func printCheckpointsList(w io.Writer, checkpoints []godo.HostedAgentCheckpoint) {
	if len(checkpoints) == 0 {
		fmt.Fprintln(w, colorize("No checkpoints", colMuted))
		return
	}
	noun := "checkpoints"
	if len(checkpoints) == 1 {
		noun = "checkpoint"
	}
	fmt.Fprintln(w, boldColor(fmt.Sprintf("%d %s", len(checkpoints), noun), colHighlight))
	fmt.Fprintln(w)

	for i, cp := range checkpoints {
		if i > 0 {
			fmt.Fprintln(w)
		}
		printCheckpointListItem(w, &cp)
	}
}

func printCheckpointListItem(w io.Writer, cp *godo.HostedAgentCheckpoint) {
	if cp == nil {
		return
	}
	title := strings.TrimSpace(cp.Label)
	if title == "" {
		title = cp.CheckpointID
	}
	fmt.Fprintf(w, "%s %s\n", checkpointStatusGlyph(cp.Status), boldColor(title, colHighlight))
	meta := []string{colorize(string(cp.Kind), colMuted), colorizeCheckpointStatus(cp.Status)}
	if cp.SizeBytes > 0 {
		meta = append(meta, colorize(humanBytes(cp.SizeBytes), colMuted))
	}
	if !cp.CreatedAt.Time.IsZero() {
		meta = append(meta, colorize(createdAgo(cp.CreatedAt.Time), colMuted))
	}
	fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
	if id := strings.TrimSpace(cp.CheckpointID); id != "" && id != title {
		fmt.Fprintf(w, "  %s\n", colorize(id, colMuted))
	}
}

func printCheckpointCard(w io.Writer, cp *godo.HostedAgentCheckpoint, created bool) {
	if cp == nil {
		fmt.Fprintln(w, colorize("No checkpoint", colMuted))
		return
	}
	var body strings.Builder
	if created {
		fmt.Fprintf(&body, "%s\n\n", boldColor("Checkpoint created", colSuccess))
	}
	title := strings.TrimSpace(cp.Label)
	if title == "" {
		title = cp.CheckpointID
	}
	body.WriteString(cardRow("Name", title))
	if id := strings.TrimSpace(cp.CheckpointID); id != "" && id != title {
		body.WriteString(cardRow("ID", colorize(id, colMuted)))
	}
	body.WriteString(cardRow("Status", checkpointStatusGlyph(cp.Status)+" "+colorizeCheckpointStatus(cp.Status)))
	if kind := strings.TrimSpace(string(cp.Kind)); kind != "" {
		body.WriteString(cardRow("Kind", colorize(kind, colMuted)))
	}
	if cp.SizeBytes > 0 {
		body.WriteString(cardRow("Size", colorize(humanBytes(cp.SizeBytes), colMuted)))
	}
	if !cp.CreatedAt.Time.IsZero() {
		body.WriteString(cardRow("Created", colorize(formatCreatedAt(cp.CreatedAt.Time), colMuted)))
	}
	if errMsg := strings.TrimSpace(cp.ErrorMessage); errMsg != "" {
		body.WriteString(cardRow("Error", colorize(errMsg, colError)))
	}
	if sess := strings.TrimSpace(cp.SessionID); sess != "" {
		fmt.Fprintln(&body)
		fmt.Fprintln(&body, colorize("Next step", colMuted))
		body.WriteString(cardRow("rollback", "doctl harness-runtime rollback "+sess+" "+cp.CheckpointID))
	}
	renderAgentCard(w, body.String())
}

func checkpointStatusGlyph(status godo.HostedAgentCheckpointStatus) string {
	switch status {
	case godo.HostedAgentCheckpointStatusReady:
		return colorize("●", colSuccess)
	case godo.HostedAgentCheckpointStatusPending:
		return colorize("…", colWarning)
	case godo.HostedAgentCheckpointStatusFailed:
		return colorize("✗", colError)
	default:
		return colorize("·", colMuted)
	}
}

func colorizeCheckpointStatus(status godo.HostedAgentCheckpointStatus) string {
	label := strings.ToLower(string(status))
	switch status {
	case godo.HostedAgentCheckpointStatusReady:
		return colorize(label, colSuccess)
	case godo.HostedAgentCheckpointStatusPending:
		return colorize(label, colWarning)
	case godo.HostedAgentCheckpointStatusFailed:
		return colorize(label, colError)
	default:
		return colorize(label, colMuted)
	}
}

// --- triggers ---------------------------------------------------------------

func printTriggersList(w io.Writer, triggers []do.HostedAgentTrigger) {
	if len(triggers) == 0 {
		fmt.Fprintln(w, colorize("No triggers", colMuted))
		return
	}
	noun := "triggers"
	if len(triggers) == 1 {
		noun = "trigger"
	}
	fmt.Fprintln(w, boldColor(fmt.Sprintf("%d %s", len(triggers), noun), colHighlight))
	fmt.Fprintln(w)

	for i, t := range triggers {
		if i > 0 {
			fmt.Fprintln(w)
		}
		printTriggerListItem(w, &t)
	}
}

func printTriggerListItem(w io.Writer, t *do.HostedAgentTrigger) {
	if t == nil || t.HostedAgentTrigger == nil {
		return
	}
	name := strings.TrimSpace(t.Name)
	if name == "" {
		name = t.TriggerID
	}
	fmt.Fprintf(w, "%s %s\n", triggerStatusGlyph(t.Status), boldColor(name, colHighlight))
	meta := []string{
		colorize(string(t.Kind), colMuted),
		colorize(string(t.SessionMode), colMuted),
		colorizeTriggerStatus(t.Status),
	}
	if agent := prettyAgentKind(t.AgentKind); agent != "" && agent != "agent" {
		meta = append(meta, colorize(agent, colMuted))
	}
	if !t.CreatedAt.Time.IsZero() {
		meta = append(meta, colorize(createdAgo(t.CreatedAt.Time), colMuted))
	}
	fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
	if id := strings.TrimSpace(t.TriggerID); id != "" && id != name {
		fmt.Fprintf(w, "  %s\n", colorize(id, colMuted))
	}
}

func printTriggerCard(w io.Writer, t *do.HostedAgentTrigger, created bool) {
	if t == nil || t.HostedAgentTrigger == nil {
		fmt.Fprintln(w, colorize("No trigger", colMuted))
		return
	}
	var body strings.Builder
	if created {
		fmt.Fprintf(&body, "%s\n\n", boldColor("Trigger created", colSuccess))
	}
	name := strings.TrimSpace(t.Name)
	if name == "" {
		name = t.TriggerID
	}
	body.WriteString(cardRow("Name", name))
	if id := strings.TrimSpace(t.TriggerID); id != "" && id != name {
		body.WriteString(cardRow("ID", colorize(id, colMuted)))
	}
	if kind := strings.TrimSpace(string(t.Kind)); kind != "" {
		body.WriteString(cardRow("Kind", colorize(kind, colMuted)))
	}
	if status := strings.TrimSpace(string(t.Status)); status != "" {
		body.WriteString(cardRow("Status", triggerStatusGlyph(t.Status)+" "+colorizeTriggerStatus(t.Status)))
	}
	if mode := strings.TrimSpace(string(t.SessionMode)); mode != "" {
		body.WriteString(cardRow("Mode", colorize(mode, colMuted)))
	}
	if agent := prettyAgentKind(t.AgentKind); agent != "" && agent != "agent" {
		body.WriteString(cardRow("Agent", agent))
	}
	if bound := strings.TrimSpace(t.BoundSessionID); bound != "" {
		body.WriteString(cardRow("Bound", colorize(bound, colMuted)))
	}
	if t.Webhook != nil {
		if provider := strings.TrimSpace(string(t.Webhook.Provider)); provider != "" {
			body.WriteString(cardRow("Provider", colorize(provider, colMuted)))
		}
		if url := strings.TrimSpace(t.Webhook.WebhookURL); url != "" {
			body.WriteString(cardRow("Webhook", colorize(truncateMiddle(url, 48), colMuted)))
		}
	}
	if t.Cron != nil {
		if expr := strings.TrimSpace(t.Cron.CronExpr); expr != "" {
			cron := expr
			if tz := strings.TrimSpace(t.Cron.Timezone); tz != "" {
				cron += " (" + tz + ")"
			}
			body.WriteString(cardRow("Cron", colorize(cron, colMuted)))
		}
		if !t.Cron.NextRunAt.Time.IsZero() {
			body.WriteString(cardRow("Next", colorize(t.Cron.NextRunAt.Time.UTC().Format("2006-01-02 15:04 UTC"), colMuted)))
		}
	}
	if !t.CreatedAt.Time.IsZero() {
		body.WriteString(cardRow("Created", colorize(formatCreatedAt(t.CreatedAt.Time), colMuted)))
	}
	renderAgentCard(w, body.String())
}

func printWebhookSecretCard(w io.Writer, secret, webhookURL string) {
	if strings.TrimSpace(secret) == "" {
		return
	}
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Webhook secret", colWarning))
	fmt.Fprintln(&body, colorize("Shown once — store it now", colMuted))
	fmt.Fprintln(&body)
	body.WriteString(cardRow("Secret", secret))
	if url := strings.TrimSpace(webhookURL); url != "" {
		body.WriteString(cardRow("URL", colorize(url, colMuted)))
	}
	renderAgentCard(w, body.String())
	fmt.Fprintln(w)
}

func triggerStatusGlyph(status godo.HostedAgentTriggerStatus) string {
	switch status {
	case godo.HostedAgentTriggerStatusActive:
		return colorize("●", colSuccess)
	case godo.HostedAgentTriggerStatusPaused:
		return colorize("○", colWarning)
	default:
		return colorize("·", colMuted)
	}
}

func colorizeTriggerStatus(status godo.HostedAgentTriggerStatus) string {
	label := strings.ToLower(string(status))
	switch status {
	case godo.HostedAgentTriggerStatusActive:
		return colorize(label, colSuccess)
	case godo.HostedAgentTriggerStatusPaused:
		return colorize(label, colWarning)
	default:
		return colorize(label, colMuted)
	}
}

// --- trigger executions -----------------------------------------------------

func printTriggerExecutionsList(w io.Writer, execs []do.HostedAgentTriggerExecution) {
	if len(execs) == 0 {
		fmt.Fprintln(w, colorize("No executions", colMuted))
		return
	}
	noun := "executions"
	if len(execs) == 1 {
		noun = "execution"
	}
	fmt.Fprintln(w, boldColor(fmt.Sprintf("%d %s", len(execs), noun), colHighlight))
	fmt.Fprintln(w)

	for i, e := range execs {
		if i > 0 {
			fmt.Fprintln(w)
		}
		printTriggerExecutionListItem(w, &e)
	}
}

func printTriggerExecutionListItem(w io.Writer, e *do.HostedAgentTriggerExecution) {
	if e == nil || e.HostedAgentTriggerExecution == nil {
		return
	}
	id := strings.TrimSpace(e.ExecutionID)
	if id == "" {
		id = "execution"
	}
	fmt.Fprintf(w, "%s %s\n", executionStatusGlyph(e.Status), boldColor(id, colHighlight))
	meta := []string{colorizeExecutionStatus(e.Status)}
	if sess := strings.TrimSpace(e.SessionID); sess != "" {
		meta = append(meta, colorize(sess, colMuted))
	}
	if !e.CreatedAt.Time.IsZero() {
		meta = append(meta, colorize(createdAgo(e.CreatedAt.Time), colMuted))
	}
	fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
	if reason := strings.TrimSpace(e.FailureReason); reason != "" {
		fmt.Fprintf(w, "  %s %s\n", colorize("fail", colError), colorize(reason, colMuted))
	}
}

func printTriggerExecutionCard(w io.Writer, e *do.HostedAgentTriggerExecution) {
	if e == nil || e.HostedAgentTriggerExecution == nil {
		fmt.Fprintln(w, colorize("No execution", colMuted))
		return
	}
	var body strings.Builder
	body.WriteString(cardRow("Execution", e.ExecutionID))
	if trig := strings.TrimSpace(e.TriggerID); trig != "" {
		body.WriteString(cardRow("Trigger", colorize(trig, colMuted)))
	}
	body.WriteString(cardRow("Status", executionStatusGlyph(e.Status)+" "+colorizeExecutionStatus(e.Status)))
	if sess := strings.TrimSpace(e.SessionID); sess != "" {
		body.WriteString(cardRow("Session", colorize(sess, colMuted)))
	}
	if run := strings.TrimSpace(e.RunID); run != "" {
		body.WriteString(cardRow("Run", colorize(run, colMuted)))
	}
	if reason := strings.TrimSpace(e.FailureReason); reason != "" {
		body.WriteString(cardRow("Failure", colorize(reason, colError)))
	}
	if !e.CreatedAt.Time.IsZero() {
		body.WriteString(cardRow("Created", colorize(formatCreatedAt(e.CreatedAt.Time), colMuted)))
	}
	renderAgentCard(w, body.String())

	if out := strings.TrimSpace(e.OutputText); out != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, boldColor("Output", colHighlight))
		fmt.Fprintln(w, out)
		if e.OutputTruncated {
			fmt.Fprintln(w, colorize("(output truncated)", colMuted))
		}
	}
}

func executionStatusGlyph(status godo.HostedAgentTriggerExecutionStatus) string {
	switch status {
	case godo.HostedAgentTriggerExecutionStatusSucceeded:
		return colorize("●", colSuccess)
	case godo.HostedAgentTriggerExecutionStatusRunning, godo.HostedAgentTriggerExecutionStatusPending:
		return colorize("…", colWarning)
	case godo.HostedAgentTriggerExecutionStatusFailed:
		return colorize("✗", colError)
	default:
		return colorize("·", colMuted)
	}
}

func colorizeExecutionStatus(status godo.HostedAgentTriggerExecutionStatus) string {
	label := strings.ToLower(string(status))
	switch status {
	case godo.HostedAgentTriggerExecutionStatusSucceeded:
		return colorize(label, colSuccess)
	case godo.HostedAgentTriggerExecutionStatusRunning, godo.HostedAgentTriggerExecutionStatusPending:
		return colorize(label, colWarning)
	case godo.HostedAgentTriggerExecutionStatusFailed:
		return colorize(label, colError)
	default:
		return colorize(label, colMuted)
	}
}

// --- reusable sessions / providers / upload --------------------------------

func printReusableSessionsList(w io.Writer, sessions []do.HostedAgentReusableSession) {
	if len(sessions) == 0 {
		fmt.Fprintln(w, colorize("No reusable sessions", colMuted))
		return
	}
	noun := "reusable sessions"
	if len(sessions) == 1 {
		noun = "reusable session"
	}
	fmt.Fprintln(w, boldColor(fmt.Sprintf("%d %s", len(sessions), noun), colHighlight))
	fmt.Fprintln(w)

	for i, s := range sessions {
		if i > 0 {
			fmt.Fprintln(w)
		}
		if s.HostedAgentReusableSession == nil {
			continue
		}
		ref := strings.TrimSpace(s.Name)
		if ref == "" {
			ref = s.SessionID
		}
		fmt.Fprintf(w, "%s %s\n", sessionStatusGlyph(s.Status), boldColor(ref, colHighlight))
		meta := []string{prettyAgentKind(s.AgentKind), colorizeSessionStatus(s.Status)}
		if !s.CreatedAt.Time.IsZero() {
			meta = append(meta, colorize(createdAgo(s.CreatedAt.Time), colMuted))
		}
		fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
		if Verbose {
			if id := strings.TrimSpace(s.SessionID); id != "" && id != ref {
				fmt.Fprintf(w, "  %s\n", colorize(id, colMuted))
			}
		}
	}
}

func printWebhookProvidersList(w io.Writer, providers []do.HostedAgentWebhookProvider) {
	if len(providers) == 0 {
		fmt.Fprintln(w, colorize("No providers", colMuted))
		return
	}
	noun := "providers"
	if len(providers) == 1 {
		noun = "provider"
	}
	fmt.Fprintln(w, boldColor(fmt.Sprintf("%d %s", len(providers), noun), colHighlight))
	fmt.Fprintln(w)

	for i, p := range providers {
		if i > 0 {
			fmt.Fprintln(w)
		}
		if p.HostedAgentWebhookProvider == nil {
			continue
		}
		name := strings.TrimSpace(p.DisplayName)
		if name == "" {
			name = string(p.Key)
		}
		fmt.Fprintf(w, "%s %s\n", colorize("●", colSuccess), boldColor(name, colHighlight))
		meta := []string{colorize(string(p.Key), colMuted)}
		if p.Signature != nil {
			if scheme := strings.TrimSpace(string(p.Signature.Scheme)); scheme != "" {
				meta = append(meta, colorize(scheme, colMuted))
			}
			if header := strings.TrimSpace(p.Signature.Header); header != "" {
				meta = append(meta, colorize(header, colMuted))
			}
		}
		fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
		if docs := strings.TrimSpace(p.DocsURL); docs != "" {
			fmt.Fprintf(w, "  %s\n", colorize(docs, colMuted))
		}
	}
}

func printWorkspaceUploadCard(w io.Writer, path string, bytesWritten int64) {
	var body strings.Builder
	fmt.Fprintf(&body, "%s\n\n", boldColor("Upload complete", colSuccess))
	body.WriteString(cardRow("Path", path))
	body.WriteString(cardRow("Bytes", colorize(fmt.Sprintf("%d", bytesWritten), colMuted)))
	renderAgentCard(w, body.String())
}

func humanBytes(n uint64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// printDetachNotice tells the user that Ctrl-C/Ctrl-D only closed the local
// connection — the hosted session keeps running until explicitly removed.
func printDetachNotice(w io.Writer, sessionRef string) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", boldColor("✓ Disconnected from session locally", colSuccess))
	fmt.Fprintln(w, colorize("Your session is still active in the cloud.", colMuted))
}

// printSessionEndedNotice is shown when the remote run can no longer accept input.
func printSessionEndedNotice(w io.Writer, sessionRef string) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, colorize("This session's run has ended and can't accept new input.", colWarning))
	ref := strings.TrimSpace(sessionRef)
	if ref == "" {
		ref = "<session>"
	}
	fmt.Fprintf(w, "  %s %s\n", colorize("remove", colMuted), "doctl harness-runtime remove "+ref)
	fmt.Fprintf(w, "  %s %s\n", colorize("start ", colMuted), "doctl harness-runtime run --harness opencode --name new-session")
}
