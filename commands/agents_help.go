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

package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const agentsRootHelpMD = `Quick start — create, wait for readiness, and attach in one step:
` + "```bash\n" + `doctl agents run \
  --harness opencode \
  --gh-repo owner/repo \
  --prompt "Review the README"
` + "```\n\n" + `For full control (custom manifests, config IDs, or attach later):
` + "```bash\n" + `doctl agents start --spec agent.yaml --name my-session
doctl agents attach my-session
` + "```\n\n" + `Commands that act on a single session accept either the session ID or its name. A name must match exactly one session; if it is ambiguous, pass the session ID instead.`

const agentsStartHelpMD = `Provide exactly one of ` + "`--spec`" + ` (manifest file) or ` + "`--config-id`" + ` (existing Agent Config).

For a one-step flow without a manifest file, use ` + "`doctl agents run --harness <name>`" + ` instead.

With ` + "`--config-id`" + `, ` + "`--name`" + ` is required (see ` + "`doctl agents config`" + `). The ` + "`--spec`" + ` flag accepts a flat agents.yaml with a top-level ` + "`agent:`" + ` key:

` + "```yaml\n" + `agent: opencode
` + "```\n\n" + `${VAR} in the manifest is expanded from your environment; ` + "`$${VAR}`" + ` is a literal. For ` + "`codex`" + `, set ` + "`$OPENAI_API_KEY`" + ` locally before starting.

` + "`--name`" + ` sets the session name (required with ` + "`--config-id`" + `). Names must be unique among active sessions; reference sessions by name in other commands.`

const agentsRunHelpMD = `Provide exactly one of ` + "`--harness`" + ` (opencode, claude-code, codex) or ` + "`--spec`" + ` (a manifest file). With ` + "`--harness`" + `, doctl builds the manifest for you — no spec file needed.

Use ` + "`--gh-repo`" + ` to clone a repository (` + "`owner/repo`" + ` or GitHub URL) and ` + "`--prompt`" + ` to send the first message. For ` + "`codex`" + `, set ` + "`$OPENAI_API_KEY`" + ` locally.

Pass ` + "`--no-attach`" + ` to wait for the session to become ready without opening the TUI. Use ` + "`--wait-timeout`" + ` to limit how long to wait (default 300 seconds).`

const agentsAttachHelpMD = `Open an interactive TUI on an existing session. Type messages and press Enter; Ctrl-D detaches without destroying the session.

If the connection drops, doctl reconnects automatically. For OpenAI sandbox sessions, set ` + "`$OPENAI_API_KEY`" + ` locally.

When approval is required: ` + "`y`" + `/` + "`a`" + ` approve, ` + "`n`" + `/` + "`r`" + ` reject, ` + "`d`" + ` defer. Type ` + "`/help`" + ` for slash commands.`

const agentsListHelpMD = `List sessions visible to your team. Filter with ` + "`--status`" + `, ` + "`--name`" + `, or ` + "`--parent-session-id`" + `. Paginate with ` + "`--page-size`" + ` and ` + "`--page-token`" + `.`

const agentsShowHelpMD = `Print details for one session. Pass the session ID or name.`

const agentsLogsHelpMD = `Replay the session's event history, then exit. Very old or long histories may show only recent events.`

const agentsApproveHelpMD = `Resolve a pending approval without attaching: ` + "`approve`" + `, ` + "`reject`" + `, or ` + "`defer`" + `.`

const agentsDestroyHelpMD = `Tear down a session and its workspace sandbox.`

const agentsPauseHelpMD = `Pause a running session. The workspace is preserved — resume with ` + "`doctl agents resume`" + `.`

const agentsResumeHelpMD = `Resume a previously paused session.`

const agentsUploadHelpMD = `Copy a local file into the session workspace at ` + "`--workspace-path`" + ` (under ` + "`/workspace`" + `).

Use ` + "`--archive`" + ` when uploading an uncompressed tar to extract at the destination. Maximum size 50 GiB.`

const agentsDownloadHelpMD = `Copy a file from the session workspace to a local path (` + "`--save-to`" + `).

Use ` + "`--archive`" + ` to download a directory as a tar archive. Maximum size 50 GiB.`

const agentsAuthHelpMD = `Connect an external provider (e.g. GitHub) so agent sessions can clone and push to private repositories.

Opens a browser to authorize unless ` + "`--no-browser`" + ` is set. The connection is shared by your team. Use ` + "`--no-wait`" + ` to print the URL and exit without waiting.`

const agentsForkHelpMD = `Create up to 4 independent child sessions from a checkpoint, or from the current state if ` + "`--from-checkpoint`" + ` is omitted. Each child can be attached normally.`

const agentsRollbackHelpMD = `Rewind a session to a prior checkpoint. The session ID stays the same.`

const agentsStartProxyHelpMD = `Start a local WebSocket proxy so the Codex CLI can drive a hosted session:

` + "```bash\n" + `codex --remote ws://127.0.0.1:1144
` + "```\n\n" + `Run ` + "`doctl agents start-proxy`" + ` first, then connect with ` + "`--type codex`" + ` and ` + "`--session`" + `. Only one of attach and start-proxy can stream the same session from one machine at a time.`

const agentsConfigRootHelpMD = `Reusable agent manifests for your team. Create a config once, then start sessions from its ID with ` + "`doctl agents config start-session`" + ` or ` + "`doctl agents start --config-id`" + `.

Configs are immutable — create a new config to change a manifest. Delete is blocked while sessions from the config are still active.`

const agentsConfigCreateHelpMD = `Create an immutable config from an agents.yaml manifest (same format as ` + "`doctl agents start --spec`" + `). ` + "`--name`" + ` must be unique within your team.`

const agentsConfigListHelpMD = `List configs for your team. Paginate with ` + "`--page-size`" + ` and ` + "`--page-token`" + `.`

const agentsConfigGetHelpMD = `Print one config, including its manifest. Secret values are redacted.`

const agentsConfigDeleteHelpMD = `Delete a config and free its name. Destroy active sessions started from the config first (` + "`doctl agents config list-sessions`" + `).`

const agentsConfigListSessionsHelpMD = `List sessions started from a config. Filter with ` + "`--status`" + ` or ` + "`--name`" + `.`

const agentsConfigStartSessionHelpMD = `Start a new session from a config ID. ` + "`--name`" + ` is required and must be unique among active sessions.`

const agentsCheckpointRootHelpMD = `Save points for a hosted agent session. Fork into new sessions or rollback the same session in place.`

const agentsCheckpointCreateHelpMD = `Create a checkpoint for a session. Can only be taken between agent turns. Optional ` + "`--label`" + ` for a human-readable name.`

const agentsCheckpointListHelpMD = `List checkpoints for a session, newest first.`

const agentsCheckpointGetHelpMD = `Print details for one checkpoint.`

const agentsCheckpointDeleteHelpMD = `Delete a checkpoint.`

const agentsTriggersRootHelpMD = `Webhook and cron triggers that start agent runs on external events or a schedule.

Webhook triggers get a signed URL for your external system. Cron triggers run on ` + "`--cron-expr`" + ` in the given timezone. Each trigger either creates a fresh session per run or reuses a paused session.`

const agentsTriggersListHelpMD = `List triggers for your team. Filter with ` + "`--kind`" + ` (webhook|cron) or ` + "`--status`" + ` (active|paused).`

const agentsTriggersCreateHelpMD = `Create a webhook or cron trigger.

` + "`--session-mode fresh`" + ` needs ` + "`--spec`" + ` (agents.yaml). ` + "`--session-mode reuse`" + ` needs ` + "`--bound-session-id`" + ` (a paused session — see ` + "`list-reusable-sessions`" + `).

Webhook: optional ` + "`--provider`" + ` (github|gitlab|custom). Cron: requires ` + "`--cron-expr`" + ` and ` + "`--timezone`" + `. Set ` + "`--output-mode`" + ` to none, email, or slack for run-result delivery.`

const agentsTriggersUpdateHelpMD = `Update a trigger. Only flags you pass are changed. Kind and webhook provider cannot be changed.`

const agentsTriggersDeleteHelpMD = `Delete a trigger. Does not destroy a bound reuse session.`

const agentsTriggersPauseHelpMD = `Pause a trigger. New events or cron ticks are ignored until resumed.`

const agentsTriggersResumeHelpMD = `Resume a paused trigger.`

const agentsTriggersRotateSecretHelpMD = `Issue a new webhook secret (shown once). Webhook triggers only.`

const agentsTriggersListExecutionsHelpMD = `List firings for a trigger. Use ` + "`get-execution`" + ` for full payload and output.`

const agentsTriggersGetExecutionHelpMD = `Print one trigger execution, including payload and output when available.`

const agentsTriggersGetBySessionHelpMD = `Find the trigger that created or binds a session.`

const agentsTriggersListReusableHelpMD = `List paused sessions available for reuse triggers.`

const agentsTriggersListProvidersHelpMD = `List supported webhook providers and their signature schemes.`

var (
	agentsHelpCodeFence    = regexp.MustCompile("(?s)```(\\w*)\n(.*?)```")
	agentsHelpExtraNewline = regexp.MustCompile(`\n{3,}`)
)

// renderAgentsHelpLong renders help Long text: prose stays plain; fenced code
// blocks get a compact lipgloss border (same family as attach/run cards).
func renderAgentsHelpLong(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	styled := detectStyling()
	var b strings.Builder
	last := 0
	for _, loc := range agentsHelpCodeFence.FindAllStringSubmatchIndex(text, -1) {
		b.WriteString(normalizeHelpProse(text[last:loc[0]], styled))
		code := strings.TrimRight(text[loc[4]:loc[5]], "\n")
		b.WriteString(renderHelpCodeBlock(code, styled))
		b.WriteByte('\n')
		last = loc[1]
	}
	b.WriteString(normalizeHelpProse(text[last:], styled))
	out := agentsHelpExtraNewline.ReplaceAllString(strings.TrimRight(b.String(), "\n"), "\n\n")
	return out
}

func normalizeHelpProse(s string, styled bool) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if styled {
		s = highlightInlineCode(s)
	}
	return s + "\n"
}

var agentsHelpInlineCode = regexp.MustCompile("`([^`]+)`")

func highlightInlineCode(s string) string {
	return agentsHelpInlineCode.ReplaceAllStringFunc(s, func(match string) string {
		inner := match[1 : len(match)-1]
		return colorize(inner, colHighlight)
	})
}

func renderHelpCodeBlock(code string, styled bool) string {
	code = strings.TrimRight(code, "\n")
	if code == "" {
		return ""
	}
	if !styled {
		var b strings.Builder
		for _, line := range strings.Split(code, "\n") {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
		return b.String()
	}
	body := colorize(code, colHighlight)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colMuted).
		Padding(0, 1).
		Render(body)
	return box + "\n"
}

// agentsStyledHelpFunc prints a compact colored header on TTYs, plain prose with
// bordered code examples, then the standard cobra usage section (always plain).
func agentsStyledHelpFunc(cmd *cobra.Command, _ []string) {
	w := cmd.OutOrStdout()
	stylingEnabled = detectStyling()

	if long := strings.TrimSpace(cmd.Long); long != "" {
		if stylingEnabled {
			fmt.Fprintln(w, boldColor(cmd.CommandPath(), colHighlight))
			if short := strings.TrimSpace(cmd.Short); short != "" {
				fmt.Fprintln(w, colorize(short, colMuted))
			}
			fmt.Fprintln(w)
		}
		fmt.Fprint(w, renderAgentsHelpLong(long))
		fmt.Fprintln(w)
	} else if short := strings.TrimSpace(cmd.Short); short != "" {
		fmt.Fprintln(w, short)
		fmt.Fprintln(w)
	}

	if cmd.Runnable() || cmd.HasAvailableSubCommands() {
		fmt.Fprint(w, cmd.UsageString())
	}
}
