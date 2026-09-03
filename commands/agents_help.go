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

// Primary cobra command name and user-facing CLI prefix for Managed Agents
// Runtime Services (M.A.R.S). Viper keys stay under agents.* via agentsNS.
const (
	agentCmdName = "harness-runtime"
	agentCLI     = "doctl " + agentCmdName
)

const agentsRootHelpMD = `**Managed Agents Runtime Services (M.A.R.S)** — run a coding agent (Claude Code, OpenCode, Codex, …) in a DigitalOcean sandbox.

Two verbs cover the lifecycle, split on whether you want to watch:

- ` + "`launch`" + ` puts you in a session — creating it first if you gave it something to create from, or attaching to one you already have.
- ` + "`create`" + ` makes a session and leaves it alone, for scripts and agents.

Create a session and land in the chat:
` + "```bash\n" + agentCLI + ` launch \
  --harness claude-code \
  --gh-repo owner/repo \
  --prompt "Review the README"
` + "```\n\n" + `Or create without attaching, then open it later:
` + "```bash\n" + agentCLI + ` create \
  --harness claude-code \
  --gh-repo owner/repo \
  --prompt "Review the README"
` + agentCLI + ` launch my-session
` + "```\n\n" + `Session commands accept a session ID or an exact unique name.

**Renamed in recent betas.** ` + "`open-harness-runtime`" + ` is now ` + "`harness-runtime`" + ` (also ` + "`agent`" + `, ` + "`agents`" + `, ` + "`ohr`" + `). ` + "`start`" + ` / ` + "`run`" + ` are now ` + "`create`" + ` (both still work). ` + "`attach`" + ` is now ` + "`launch`" + ` (also ` + "`up`" + ` / ` + "`chat`" + ` / ` + "`attach`" + `; interactive only — scripts should use ` + "`create`" + `). ` + "`destroy`" + ` is now ` + "`remove`" + ` (still accepted as ` + "`destroy`" + ` / ` + "`rm`" + `).`

const agentsCreateHelpMD = `Create a hosted session, wait until it is ready, optionally send ` + "`--prompt`" + `, and print a ready summary. It never opens the chat — that is ` + "`" + agentCLI + " launch`" + `, which is the only difference between the two commands. Open the session later with ` + "`" + agentCLI + " launch <session>`" + `.

Provide exactly one of:

- a **manifest** — as a positional path, or with ` + "`--spec`" + ` / ` + "`-f`" + ` / ` + "`--file`" + ` (` + "`-`" + ` reads stdin). With none of these, ` + "`./agents.yaml`" + ` is used when it exists.
- ` + "`--harness`" + ` — opencode, claude-code, or codex (builds the manifest for you)
- ` + "`--from-config`" + ` — an existing Agent Config, by name or ID (` + "`--name`" + ` required)

These four are equivalent:
` + "```bash\n" + agentCLI + ` create                       # uses ./agents.yaml
` + agentCLI + ` create agents.yaml
` + agentCLI + ` create -f agents.yaml
` + agentCLI + ` create --spec agents.yaml
` + "```\n\n" + `With a manifest or ` + "`--harness`" + `, optionally pass ` + "`--gh-repo`" + ` and ` + "`--prompt`" + `. Example:
` + "```bash\n" + agentCLI + ` create --harness claude-code --gh-repo owner/repo --prompt "Review the README"
` + "```\n\n" + `Flat manifests use a top-level ` + "`name`" + ` (or pass ` + "`--name`" + `, which writes that field). Minimal example:
` + "```yaml\n" + `name: my-session
agent: opencode
` + "```\n\n" + `${VAR} in a manifest is expanded from your environment (prompted in a terminal when missing). For ` + "`codex`" + `, doctl prompts for ` + "`$OPENAI_API_KEY`" + ` when unset.

**Secrets.** ` + "`--secret NAME=VALUE`" + ` fills a secret slot the manifest declares, overriding any value written in the file — so a checked-in ` + "`agents.yaml`" + ` can name the credentials it needs without carrying them. ` + "`NAME=@path`" + ` reads the value from a file and ` + "`NAME=-`" + ` from stdin, neither of which leaves it in shell history. Repeat the flag per secret. For ` + "`claude-code`" + `, ` + "`--secret ANTHROPIC_API_KEY=@path`" + ` (or ` + "`=-`" + `) is enough — the key need not also be in the process environment. ` + "`NAME=-`" + ` cannot be combined with a manifest on stdin (` + "`--spec -`" + `). Not needed with ` + "`--from-config`" + `: that config captured its values when it was created, and keeps them server-side.

**Inspect before creating.** ` + "`--dry-run`" + ` prints the fully-resolved manifest — name applied, ` + "`${VAR}`" + ` expanded, secret values redacted — and exits without calling the API. That makes a working ` + "`--harness`" + ` invocation reusable, with no file ever touching disk:
` + "```bash\n" + agentCLI + ` create --harness claude-code --gh-repo owner/repo --dry-run \
  | ` + agentCLI + ` config create --spec - --name reviewer --secret ANTHROPIC_API_KEY=@~/.secrets/anthropic.key
` + "```\n\n" + `**Unattended runs.** ` + "`--on-hitl approve|reject|defer`" + ` stays attached headlessly after the session is ready — no TUI, no keyboard input, one log line per event — and resolves every approval request that way until the run finishes. Ctrl-C detaches and leaves the session running. Combined with the global ` + "`--interactive=false`" + `, an agent can drive a whole run with no human present:
` + "```bash\n" + agentCLI + ` create --spec agents.yaml --prompt "Fix the failing test" \
  --on-hitl approve --interactive=false
` + "```\n\n" + `From another shell, poll with ` + "`show <session> -o json`" + `, move files with ` + "`upload`" + ` / ` + "`download`" + `, and tear down with ` + "`remove`" + ` (also ` + "`destroy`" + ` / ` + "`rm`" + `).

Creating from a manifest or ` + "`--harness`" + ` also persists an Agent Config named after the session (shown as Config on the ready card). Start later sessions from it with ` + "`create --from-config`" + ` or ` + "`config start-session`" + `.

Use ` + "`-o json`" + ` for machine-readable create output without waiting. Combined with ` + "`--prompt`" + `, doctl still waits until the session is ready and delivers the prompt before printing JSON — otherwise the prompt would be dropped.`

const agentsValidateHelpMD = `Check an agents.yaml / JSON manifest client-side without creating a session.

Catches missing ` + "`agent`" + `/` + "`spec.runtime.adapter`" + `, unknown adapters, reserved env keys, credentials placed in ` + "`env`" + ` instead of ` + "`secrets`" + `, and conflicting model env keys (` + "`MODEL`" + ` / ` + "`HARNESS_INFERENCE_MODEL`" + ` / ` + "`ANTHROPIC_MODEL`" + `). The API remains the authoritative validator for the full contract.

Name the manifest as a positional path or with ` + "`--spec`" + ` / ` + "`-f`" + ` / ` + "`--file`" + ` (` + "`-`" + ` reads stdin); with none of these, ` + "`./agents.yaml`" + ` is used when it exists.

Example:
` + "```bash\n" + agentCLI + ` validate            # checks ./agents.yaml
` + agentCLI + ` validate agent.yaml
` + "```"

const agentsLaunchHelpMD = `Open an interactive chat on a session, creating it first when you give it something to create from. Aliases: ` + "`up`" + `, ` + "`chat`" + `, ` + "`attach`" + `.

**Attach to a session that exists.** Pass its name or ID and nothing else:
` + "```bash\n" + agentCLI + ` launch my-session
` + agentCLI + ` launch sess_abc123
` + "```\n\n" + `A paused session is resumed on the way in, so waking one up and looking at it is a single command. A destroyed or failed session is an error rather than something to resume.

**Create a session, then attach.** Pass a manifest, ` + "`--harness`" + `, or ` + "`--from-config`" + ` — every creation flag works exactly as it does on ` + "`" + agentCLI + " create`" + `:
` + "```bash\n" + agentCLI + ` launch                       # uses ./agents.yaml
` + agentCLI + ` launch agents.yaml
` + agentCLI + ` launch --harness opencode --gh-repo owner/repo --prompt "Review the README"
` + agentCLI + ` launch --from-config reviewer --name my-session
` + "```\n\n" + `**How the positional argument is read.** One rule: *does a readable file exist at that path?* If yes it is a manifest and a new session is created; if no it is treated as a session name or ID. An argument that is neither reports both possibilities. Naming a creation flag settles it without consulting the filesystem.

Type messages and press Enter; Ctrl-D (or Ctrl-C) detaches without removing the session — reattach later, or run ` + "`" + agentCLI + " remove`" + ` to tear it down. If the connection drops, doctl reconnects automatically. For OpenAI sandbox sessions, doctl prompts for ` + "`$OPENAI_API_KEY`" + ` when it is unset.

When approval is required: ` + "`y`" + `/` + "`a`" + ` approve, ` + "`n`" + `/` + "`r`" + ` reject, ` + "`d`" + ` defer. Type ` + "`/help`" + ` for slash commands. To resolve approvals with no human at all, see ` + "`" + agentCLI + " create --on-hitl`" + `.

This command always ends in a terminal chat, so it has no ` + "`-o json`" + ` form and needs a TTY; use ` + "`" + agentCLI + " create`" + ` in scripts and pipelines. For the native Codex desktop/CLI UI instead of doctl chat, see ` + "`" + agentCLI + " start-proxy --help`" + `.`

const agentsListHelpMD = `List sessions visible to your team. Filter with ` + "`--status`" + `, ` + "`--name`" + `, or ` + "`--parent-session-id`" + `. Paginate with ` + "`--page-size`" + ` and ` + "`--page-token`" + `.`

const agentsShowHelpMD = `Print details for one session. Pass the session ID or name.`

const agentsLogsHelpMD = `Replay the session's event history, then exit. Very old or long histories may show only recent events.`

const agentsApproveHelpMD = `Resolve a pending approval without attaching: ` + "`approve`" + `, ` + "`reject`" + `, or ` + "`defer`" + `.`

const agentsRemoveHelpMD = `Remove a session and tear down its workspace sandbox. Aliases: ` + "`destroy`" + `, ` + "`rm`" + `.`

const agentsPauseHelpMD = `Pause a running session. The workspace is preserved — resume with ` + "`" + agentCLI + " resume`" + `.`

const agentsResumeHelpMD = `Resume a previously paused session.`

const agentsUploadHelpMD = `Copy a local file into the session workspace at ` + "`--workspace-path`" + ` (under ` + "`/workspace`" + `).

Use ` + "`--archive`" + ` when uploading an uncompressed tar to extract at the destination. Maximum size 50 GiB.`

const agentsDownloadHelpMD = `Copy a file from the session workspace to a local path (` + "`--save-to`" + `).

Use ` + "`--archive`" + ` to download a directory as a tar archive. Maximum size 50 GiB.`

const agentsAuthHelpMD = `Connect an external provider (e.g. GitHub) so sessions can clone and push to private repositories.

Opens a browser to authorize unless ` + "`--no-browser`" + ` is set. The connection is shared by your team. Use ` + "`--no-wait`" + ` to print the URL and exit without waiting.`

const agentsForkHelpMD = `Create up to 4 independent child sessions from a checkpoint, or from the current state if ` + "`--from-checkpoint`" + ` is omitted. Each child can be attached normally.`

const agentsRollbackHelpMD = `Rewind a session to a prior checkpoint. The session ID stays the same.`

const agentsStartProxyHelpMD = `Bridge a local coding-agent CLI to a hosted M.A.R.S session.

` + "`launch`" + ` is doctl's built-in chat. ` + "`start-proxy`" + ` is different: it runs a small local WebSocket server that speaks the agent CLI's own protocol (today: Codex), so you can keep using the native Codex UI while the sandbox lives on DigitalOcean.

Typical flow:
` + "```bash\n" + `# terminal 1 — keep this running
` + agentCLI + ` start-proxy --type codex --session my-session --port 1144

# terminal 2 — connect Codex to the proxy
codex --remote ws://127.0.0.1:1144
` + "```\n\n" + `Only one of ` + "`launch`" + ` and ` + "`start-proxy`" + ` can stream the same session from one machine at a time.`

const agentsConfigRootHelpMD = `Reusable agent manifests for your team. Create a config once, then start sessions from it — by name or by ID — with ` + "`" + agentCLI + " create --from-config`" + `, ` + "`" + agentCLI + " launch --from-config`" + `, or ` + "`" + agentCLI + " config start-session`" + `.

Configs are immutable — create a new config to change a manifest. Delete is blocked while sessions from the config are still active.`

const agentsSizesRootHelpMD = `List sandbox (microVM) sizes you can set as ` + "`spec.sandbox.sizeSlug`" + ` when creating a session.`

const agentsSizesListHelpMD = `Print the customer-selectable sandbox size catalog (slug, vCPUs, memory). Ordered smallest to largest. Every returned slug is accepted by CreateSession as ` + "`spec.sandbox.sizeSlug`" + `.`

const agentsConfigCreateHelpMD = `Create an immutable config from an agents.yaml manifest (same format as ` + "`" + agentCLI + " create --spec`" + `). ` + "`--name`" + ` must be unique within your team.

` + "`--secret NAME=VALUE`" + ` (also ` + "`NAME=@path`" + ` / ` + "`NAME=-`" + `, repeatable) fills a secret slot the manifest declares. For ` + "`claude-code`" + `, that also satisfies ` + "`${ANTHROPIC_API_KEY}`" + ` in the manifest — the process environment is not required. ` + "`NAME=-`" + ` cannot be combined with ` + "`--spec -`" + `. The value is captured server-side against this config and never returned, so sessions started from it need no secrets of their own.`

const agentsConfigListHelpMD = `List configs for your team. Paginate with ` + "`--page-size`" + ` and ` + "`--page-token`" + `.`

const agentsConfigGetHelpMD = `Print one config, including its manifest. Secret values are redacted.`

const agentsConfigDeleteHelpMD = `Delete a config and free its name. Remove active sessions started from the config first (` + "`" + agentCLI + " config list-sessions`" + `).`

const agentsConfigListSessionsHelpMD = `List sessions started from a config. Filter with ` + "`--status`" + ` or ` + "`--name`" + `.`

const agentsConfigStartSessionHelpMD = `Start a new session from a config ID. ` + "`--name`" + ` is required and must be unique among active sessions.`

const agentsExecHelpMD = `Run one command in a session's sandbox and print its output.

Drives the sandbox directly rather than through its agent, so it works on a bare sandbox started with ` + "`--agent none`" + ` as well as on a managed-agent session.

Separate the guest command from doctl's own flags with ` + "`--`" + `, or flags meant for the guest (` + "`ls -la`" + `) are parsed as doctl flags.

Guest stdout and stderr pass through unchanged and the guest's exit code becomes doctl's, so this composes in pipelines and ` + "`&&`" + ` chains. Each call is independent: there is no shell session between calls, so ` + "`cd`" + ` does not carry over — use ` + "`--workdir`" + `, or run a shell explicitly (` + "`-- sh -c 'cd src && make'`" + `).

Output is buffered until the command finishes, and capped at 1 MiB per stream. With ` + "`-o json`" + ` the full response is emitted instead of the raw streams.`

const agentsCheckpointRootHelpMD = `Save points for a hosted agent session. Fork into new sessions or rollback the same session in place.`

const agentsCheckpointCreateHelpMD = `Create a checkpoint for a session. Can only be taken between agent turns. Optional ` + "`--label`" + ` for a human-readable name.`

const agentsCheckpointListHelpMD = `List checkpoints for a session, newest first.`

const agentsCheckpointGetHelpMD = `Print details for one checkpoint.`

const agentsCheckpointDeleteHelpMD = `Delete a checkpoint.`

const agentsTriggersRootHelpMD = `Webhook and cron triggers that start agent runs on external events or a schedule.

Webhook triggers get a signed URL for your external system. Cron triggers run on ` + "`--cron-expr`" + ` in the given timezone. Each trigger either creates a fresh session per run or reuses a paused session.`

const agentsTriggersListHelpMD = `List triggers for your team. Filter with ` + "`--kind`" + ` (webhook|cron) or ` + "`--status`" + ` (active|paused).`

const agentsTriggersCreateHelpMD = `Create a webhook or cron trigger.

` + "`--session-mode fresh`" + ` needs a session template, either ` + "`--spec`" + ` (agents.yaml) or ` + "`--from-config`" + ` (an existing Agent Config, by name or ID). ` + "`--session-mode reuse`" + ` needs ` + "`--bound-session-id`" + ` (a paused session — see ` + "`list-reusable-sessions`" + `).

Webhook: optional ` + "`--provider`" + ` (github|gitlab|custom). Cron: requires ` + "`--cron-expr`" + ` and ` + "`--timezone`" + `. Set ` + "`--output-mode`" + ` to none, email, or slack for run-result delivery.

**` + "`--from-config`" + ` copies, it does not link.** The config's manifest is read once, now, and stored on the trigger — replacing or editing the config later does not change triggers already created from it. A config's stored secret values are never returned to a client, so supply them again here with ` + "`--secret NAME=VALUE`" + ` (also ` + "`NAME=@path`" + ` / ` + "`NAME=-`" + `, repeatable), which works with ` + "`--spec`" + ` too.`

const agentsTriggersUpdateHelpMD = `Update a trigger. Only flags you pass are changed. Kind and webhook provider cannot be changed.

` + "`--spec`" + ` or ` + "`--from-config`" + ` replaces the session template for fresh triggers, with the same copy-not-link caveat as ` + "`create`" + `: re-supply any secret values with ` + "`--secret`" + `.`

const agentsTriggersDeleteHelpMD = `Delete a trigger. Does not remove a bound reuse session.`

const agentsTriggersPauseHelpMD = `Pause a trigger. New events or cron ticks are ignored until resumed.`

const agentsTriggersResumeHelpMD = `Resume a paused trigger.`

const agentsTriggersRotateSecretHelpMD = `Issue a new webhook secret (shown once). Webhook triggers only. By default the old secret keeps verifying deliveries for 5 minutes, so deliveries already in flight still succeed while you paste the new secret into your external system. Pass ` + "`--grace-period 0`" + ` to retire the old secret immediately — deliveries still signed with it start failing at once, so use it when the old secret is compromised. Pass a positive value (seconds, up to the server max of 1 hour) for a custom handoff window.`

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
		s = highlightHelpBold(s)
	}
	return s + "\n"
}

var (
	agentsHelpInlineCode = regexp.MustCompile("`([^`]+)`")
	agentsHelpBold       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
)

func highlightInlineCode(s string) string {
	return agentsHelpInlineCode.ReplaceAllStringFunc(s, func(match string) string {
		inner := match[1 : len(match)-1]
		return colorize(inner, colHighlight)
	})
}

func highlightHelpBold(s string) string {
	return agentsHelpBold.ReplaceAllStringFunc(s, func(match string) string {
		inner := match[2 : len(match)-2]
		return boldColor(inner, colHighlight)
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

// requireAgentSubcommand configures a parent-only agents cobra node so unknown
// positional args (e.g. "frobnicate") fail with a non-zero exit instead of
// silently printing help. Cobra validates unknown commands only at the root;
// nested parents with Args unset accept arbitrary args and return flag.ErrHelp
// (exit 0). See MARSOHS-1075.
func requireAgentSubcommand(cmd *Command) {
	cc := cmd.Command
	cc.Args = cobra.NoArgs
	cc.RunE = func(c *cobra.Command, args []string) error {
		return c.Help()
	}
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
