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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// AgentConfigs generates the `doctl harness-runtime config` subtree, which wraps the
// godo Agent Configs API: immutable, team-scoped agent manifests that sessions
// can be launched from by ID.
func AgentConfigs() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "config",
			Aliases: []string{"configs", "cfg"},
			Short:   "Manage reusable agent configs",
			Long:    agentsConfigRootHelpMD,
		},
	}

	// The subcommand group is named "config", which collides with doctl's
	// reserved top-level `--config` viper key (flag namespaces are only two
	// levels deep: parent.child). Force each subcommand's flags to nest under
	// "agents.config" instead, mirroring how registry.repository does it. The
	// override must be passed as a build option so it is in effect when the
	// displayer's --format / --no-header flags are bound inside CmdBuilder.
	ns := agentSubNS("agents.config")

	cmdCreate := CmdBuilder(cmd, RunAgentsConfigCreate, "create",
		"Create an agent config from a manifest",
		agentsConfigCreateHelpMD,
		Writer, append(ns, aliasOpt("c"),
			displayerType(&displayers.HostedAgentConfig{}))...)
	AddStringFlag(cmdCreate, doctl.ArgAgentSpec, "", "", `Path to an agent manifest in YAML or JSON. Prefer flat format (top-level name + agent), e.g. "name: my-config\nagent: opencode". Set to "-" to read from stdin. ${VAR} references are resolved from the local environment.`, requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgAgentName, "", "", "Team-unique name for the config", requiredOpt())
	AddStringSliceFlag(cmdCreate, doctl.ArgAgentSecret, "", nil, agentSecretFlagDesc)
	cmdCreate.Example = agentCLI + ` config create --spec agent-spec.yaml --name my-config; ` + agentCLI + ` config create --spec agent-spec.yaml --name my-config --secret ANTHROPIC_API_KEY=@~/.secrets/anthropic.key`

	cmdList := CmdBuilder(cmd, RunAgentsConfigList, "list",
		"List agent configs",
		agentsConfigListHelpMD,
		Writer, append(ns, aliasOpt("ls"),
			displayerType(&displayers.HostedAgentConfigSummary{}))...)
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of configs to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	cmdList.Example = `doctl harness-runtime config list --page-size 10`

	CmdBuilder(cmd, RunAgentsConfigGet, "get <config-id>",
		"Get an agent config",
		agentsConfigGetHelpMD,
		Writer, append(ns, aliasOpt("show"),
			displayerType(&displayers.HostedAgentConfig{}))...)

	CmdBuilder(cmd, RunAgentsConfigDelete, "delete <config-id>",
		"Delete an agent config",
		agentsConfigDeleteHelpMD,
		Writer, append(ns, aliasOpt("rm"))...)

	cmdSessions := CmdBuilder(cmd, RunAgentsConfigListSessions, "list-sessions <config-id>",
		"List sessions started from a config",
		agentsConfigListSessionsHelpMD,
		Writer, append(ns, aliasOpt("sessions"),
			displayerType(&displayers.HostedAgentSession{}))...)
	AddIntFlag(cmdSessions, doctl.ArgAgentPageSize, "", 0, "Maximum number of sessions to return per page")
	AddStringFlag(cmdSessions, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	AddStringFlag(cmdSessions, doctl.ArgAgentStatus, "", "", "Filter by session status (e.g. SESSION_STATUS_READY)")
	AddStringFlag(cmdSessions, doctl.ArgAgentName, "", "", "Filter by session name")
	cmdSessions.Example = `doctl harness-runtime config list-sessions cfg_abc123 --status SESSION_STATUS_READY`

	cmdStartSession := CmdBuilder(cmd, RunAgentsConfigStartSession, "start-session <config-id>",
		"Start a session from a config",
		agentsConfigStartSessionHelpMD,
		Writer, append(ns, aliasOpt("start"),
			displayerType(&displayers.HostedAgentSession{}))...)
	AddStringFlag(cmdStartSession, doctl.ArgAgentName, "", "", "Name for the new session", requiredOpt())
	cmdStartSession.Example = `doctl harness-runtime config start-session cfg_abc123 --name my-session`

	requireAgentSubcommand(cmd)
	return cmd
}

// RunAgentsConfigCreate creates an immutable Agent Config from a manifest file.
func RunAgentsConfigCreate(c *CmdConfig) error {
	specPath, err := c.Doit.GetString(c.NS, doctl.ArgAgentSpec)
	if err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	if err != nil {
		return err
	}
	secrets, err := agentSecretFlags(c)
	if err != nil {
		return err
	}
	manifest, err := readManifest(os.Stdin, specPath)
	if err != nil {
		return err
	}
	manifest, err = injectManifestSecrets(manifest, secrets)
	if err != nil {
		return err
	}
	if err := rejectRedactedSecrets(manifest); err != nil {
		return err
	}
	if err := reportDurableAgentManifestValidation(validateAgentManifest(manifest)); err != nil {
		return err
	}
	cfg, err := c.HostedAgents().CreateAgentConfig(&godo.HostedAgentConfigCreateRequest{
		Name:         name,
		ManifestYAML: string(manifest),
	})
	if err != nil {
		return err
	}
	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentConfig{Configs: []godo.HostedAgentConfig{*cfg}, Single: true}); err != nil {
			return err
		}
	} else {
		stylingEnabled = detectStyling()
		printAgentConfigCard(c.Out, cfg, true)
	}
	for _, w := range cfg.Warnings {
		fmt.Fprintf(c.Out, "%s %s\n", colorize("Warning:", colWarning), w)
	}
	return nil
}

// RunAgentsConfigList lists Agent Configs for the caller's team.
func RunAgentsConfigList(c *CmdConfig) error {
	opt := &godo.HostedAgentConfigListOptions{}
	pageSize, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPageSize)
	if err != nil {
		return err
	}
	opt.PageSize = pageSize
	pageToken, err := c.Doit.GetString(c.NS, doctl.ArgAgentPageToken)
	if err != nil {
		return err
	}
	opt.PageToken = pageToken

	configs, next, err := c.HostedAgents().ListAgentConfigs(opt)
	if err != nil {
		return err
	}
	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentConfigSummary{Configs: configs}); err != nil {
			return err
		}
		if next != "" {
			fmt.Fprintf(os.Stderr, "Next page token: %s\n", next)
		}
		return nil
	}

	stylingEnabled = detectStyling()
	printAgentConfigsList(c.Out, configs)
	if next != "" {
		fmt.Fprintf(c.Out, "\n%s %s\n", colorize("Next page token:", colMuted), next)
	}
	return nil
}

// RunAgentsConfigGet fetches one Agent Config.
func RunAgentsConfigGet(c *CmdConfig) error {
	configID, err := configIDArg(c)
	if err != nil {
		return err
	}
	cfg, err := c.HostedAgents().GetAgentConfig(configID)
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentConfig{Configs: []godo.HostedAgentConfig{*cfg}, Single: true})
	}
	stylingEnabled = detectStyling()
	printAgentConfigCard(c.Out, cfg, false)
	return nil
}

// RunAgentsConfigDelete soft-deletes an Agent Config.
func RunAgentsConfigDelete(c *CmdConfig) error {
	configID, err := configIDArg(c)
	if err != nil {
		return err
	}
	if err := c.HostedAgents().DeleteAgentConfig(configID); err != nil {
		if agentConfigHasActiveSessionsErr(err) {
			msg, _, _ := agentAPIError(err)
			return fmt.Errorf("%s. List them with `%s config list-sessions %s`, remove each with `%s remove`, then retry", strings.TrimRight(msg, "."), agentCLI, configID, agentCLI)
		}
		return err
	}
	stylingEnabled = detectStyling()
	printAgentSuccess(c.Out, fmt.Sprintf("Deleted agent config %s", configID))
	return nil
}

// RunAgentsConfigListSessions lists sessions started from an Agent Config.
func RunAgentsConfigListSessions(c *CmdConfig) error {
	configID, err := configIDArg(c)
	if err != nil {
		return err
	}
	opt := &godo.HostedAgentSessionListOptions{}
	pageSize, err := c.Doit.GetInt(c.NS, doctl.ArgAgentPageSize)
	if err != nil {
		return err
	}
	opt.PageSize = pageSize
	pageToken, err := c.Doit.GetString(c.NS, doctl.ArgAgentPageToken)
	if err != nil {
		return err
	}
	opt.PageToken = pageToken
	status, err := c.Doit.GetString(c.NS, doctl.ArgAgentStatus)
	if err != nil {
		return err
	}
	if status != "" {
		opt.Status = godo.HostedAgentSessionStatus(status)
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	if err != nil {
		return err
	}
	opt.Name = name

	sessions, next, err := c.HostedAgents().ListAgentConfigSessions(configID, opt)
	if err != nil {
		return err
	}
	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentSession{Sessions: sessions}); err != nil {
			return err
		}
		if next != "" {
			fmt.Fprintf(os.Stderr, "Next page token: %s\n", next)
		}
		return nil
	}

	stylingEnabled = detectStyling()
	printSessionsList(c.Out, sessions)
	if next != "" {
		fmt.Fprintf(c.Out, "\n%s %s\n", colorize("Next page token:", colMuted), next)
	}
	return nil
}

// RunAgentsConfigStartSession creates a session from an existing Agent Config.
func RunAgentsConfigStartSession(c *CmdConfig) error {
	configID, err := configIDArg(c)
	if err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	if err != nil {
		return err
	}
	sess, err := c.HostedAgents().CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
		Name:     name,
		ConfigID: configID,
	})
	if err != nil {
		return err
	}
	if Output == "json" {
		return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}, Single: true})
	}
	stylingEnabled = detectStyling()
	printSessionShowCard(c.Out, sess)
	if sess != nil && sess.HostedAgentSession != nil {
		printSessionCreateWarnings(sess.Warnings)
	}
	return nil
}

// configIDPrefix is the Agent Config ID prefix used in API responses and docs.
const configIDPrefix = "cfg_"

// configRefPageSize pages the name scan below. Agent Configs are a small,
// deliberately-created resource, but a team can still have more than one page
// of them, and a name that exists must never resolve to "not found" because it
// sat on page two.
const configRefPageSize = 200

// looksLikeConfigID reports whether ref is already a config ID rather than a
// name, so it can be used directly with no lookup. Mirrors looksLikeSessionID.
func looksLikeConfigID(ref string) bool {
	return sessionUUIDRe.MatchString(ref) || strings.HasPrefix(ref, configIDPrefix)
}

// resolveConfigRef turns a user-supplied Agent Config reference into a config
// ID, accepting either the ID or the config's team-unique name — the same
// courtesy resolveSessionRef extends to sessions, so a name printed by
// `config list` can be pasted straight back into any command.
//
// Unlike sessions, HostedAgentConfigListOptions has no server-side name
// filter, so matching is a client-side scan. A server-side filter would make
// this exact and cheap; until then the scan is bounded by how few configs a
// team realistically has.
func resolveConfigRef(svc do.HostedAgentsService, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("an agent config ID or name is required")
	}
	if looksLikeConfigID(ref) {
		return ref, nil
	}

	var matches []godo.HostedAgentConfigSummary
	pageToken := ""
	for {
		configs, next, err := svc.ListAgentConfigs(&godo.HostedAgentConfigListOptions{
			PageSize:  configRefPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			return "", fmt.Errorf("resolving agent config name %q: %w", ref, err)
		}
		for _, cfg := range configs {
			if strings.EqualFold(strings.TrimSpace(cfg.Name), ref) {
				matches = append(matches, cfg)
			}
		}
		if next == "" || len(configs) == 0 {
			break
		}
		pageToken = next
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no agent config goes by the name %q; pass a config ID or run `%s config list` to see available configs", ref, agentCLI)
	case 1:
		return matches[0].ID, nil
	default:
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, m.ID)
		}
		return "", fmt.Errorf("many agent configs go by the name %q, they have the following IDs: %s", ref, strings.Join(ids, ", "))
	}
}

// configIDArg resolves the positional <config-id> argument, which may be a
// config name.
func configIDArg(c *CmdConfig) (string, error) {
	if len(c.Args) < 1 {
		return "", doctl.NewMissingArgsErr(c.NS)
	}
	return resolveConfigRef(c.HostedAgents(), c.Args[0])
}

// agentConfigHasActiveSessionsErr reports the DELETE /configs/{id} 409 returned
// when any session created from the config is still active.
func agentConfigHasActiveSessionsErr(err error) bool {
	msg, status, ok := agentAPIError(err)
	if !ok || status != http.StatusConflict {
		return false
	}
	return strings.Contains(strings.ToLower(msg), "active sessions")
}

// printAgentConfigsList renders a compact styled list for `agent config list`.
func printAgentConfigsList(w io.Writer, configs []godo.HostedAgentConfigSummary) {
	if len(configs) == 0 {
		fmt.Fprintln(w, colorize("No configs", colMuted))
		return
	}
	noun := "configs"
	if len(configs) == 1 {
		noun = "config"
	}
	fmt.Fprintln(w, boldColor(fmt.Sprintf("%d %s", len(configs), noun), colHighlight))
	fmt.Fprintln(w)

	for i, cfg := range configs {
		if i > 0 {
			fmt.Fprintln(w)
		}
		name := strings.TrimSpace(cfg.Name)
		if name == "" {
			name = cfg.ID
		}
		fmt.Fprintf(w, "%s %s\n", colorize("●", colSuccess), boldColor(name, colHighlight))
		meta := make([]string, 0, 2)
		if id := strings.TrimSpace(cfg.ID); id != "" && id != name {
			meta = append(meta, colorize(id, colMuted))
		}
		if !cfg.CreatedAt.Time.IsZero() {
			meta = append(meta, colorize(createdAgo(cfg.CreatedAt.Time), colMuted))
		}
		if len(meta) > 0 {
			fmt.Fprintf(w, "  %s\n", strings.Join(meta, colorize(" · ", colMuted)))
		}
	}
}

// printAgentConfigCard renders a single config detail card for get/create.
func printAgentConfigCard(w io.Writer, cfg *godo.HostedAgentConfig, created bool) {
	if cfg == nil {
		fmt.Fprintln(w, colorize("No config", colMuted))
		return
	}
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = cfg.ID
	}

	var body strings.Builder
	if created {
		fmt.Fprintf(&body, "%s\n\n", boldColor("Config created", colSuccess))
	}
	body.WriteString(cardRow("Name", name))
	if id := strings.TrimSpace(cfg.ID); id != "" && id != name {
		body.WriteString(cardRow("ID", colorize(id, colMuted)))
	}
	if schema := shortAgentSchema(cfg.AgentSpecSchemaVersion); schema != "" {
		body.WriteString(cardRow("Schema", colorize(schema, colMuted)))
	}
	if hash := truncateMiddle(cfg.ContentHash, 20); hash != "" {
		body.WriteString(cardRow("Hash", colorize(hash, colMuted)))
	}
	if !cfg.CreatedAt.Time.IsZero() {
		body.WriteString(cardRow("Created", colorize(formatCreatedAt(cfg.CreatedAt.Time), colMuted)))
	}

	// Prefer the name in the hint: --from-config resolves either, and a name is
	// what the user just chose and will remember.
	if ref := name; strings.TrimSpace(ref) != "" {
		fmt.Fprintln(&body)
		fmt.Fprintln(&body, colorize("Next step", colMuted))
		body.WriteString(cardRow("launch", agentCLI+" launch --from-config "+ref+" --name my-session"))
	}

	renderAgentCard(w, body.String())
}

func shortAgentSchema(schema string) string {
	schema = strings.TrimSpace(schema)
	if schema == "" {
		return ""
	}
	if i := strings.LastIndex(schema, "/"); i >= 0 && i+1 < len(schema) {
		return schema[i+1:]
	}
	return schema
}

func truncateMiddle(s string, keep int) string {
	s = strings.TrimSpace(s)
	if keep < 8 || len(s) <= keep {
		return s
	}
	half := keep/2 - 1
	if half < 2 {
		half = 2
	}
	return s[:half] + "…" + s[len(s)-half:]
}
