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
	"net/http"
	"os"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// AgentConfigs generates the `doctl agents config` subtree, which wraps the
// godo Agent Configs API: immutable, team-scoped agent manifests that sessions
// can be launched from by ID.
func AgentConfigs() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "config",
			Aliases: []string{"configs", "cfg"},
			Short:   "Manage reusable agent configs",
			Long: `The ` + "`" + `doctl agents config` + "`" + ` commands manage Agent Configs: immutable, team-scoped agent manifests.

An Agent Config captures a validated agent manifest (and any secret slots it declares) under a stable ID. Anyone on your team can then start sessions from that ID without re-supplying the manifest, so a config is the reusable, auditable unit behind repeated agent runs.

Configs are immutable: to change a manifest, create a new config. Deleting a config is a soft delete (the name is freed, the row is retained), but the API rejects the delete while any session created from the config is still active. Destroy those sessions first.`,
		},
	}

	// The subcommand group is named "config", which collides with doctl's
	// reserved top-level `--config` viper key (flag namespaces are only two
	// levels deep: parent.child). Force each subcommand's flags to nest under
	// "agents.config" instead, mirroring how registry.repository does it. The
	// override must be passed as a build option so it is in effect when the
	// displayer's --format / --no-header flags are bound inside CmdBuilder.
	ns := overrideCmdNS("agents.config")

	cmdCreate := CmdBuilder(cmd, RunAgentsConfigCreate, "create",
		"Create an agent config from a manifest",
		`Creates an immutable Agent Config from an agent manifest file.

The `+"`"+`--spec`+"`"+` flag accepts the same flat agents.yaml manifest as `+"`"+`doctl agents start`+"`"+`. `+"`"+`${VAR}`+"`"+` references are resolved from your local environment before upload, so secret values declared under `+"`"+`spec.secrets[].value`+"`"+` can be injected from the environment; the server extracts them into Secrets Manager and never returns them.

The `+"`"+`--name`+"`"+` is the team-unique handle for the config and is required.`,
		Writer, ns, aliasOpt("c"),
		displayerType(&displayers.HostedAgentConfig{}))
	AddStringFlag(cmdCreate, doctl.ArgAgentSpec, "", "", `Path to an agent manifest in YAML or JSON (flat format; minimal: "agent: opencode"). Set to "-" to read from stdin. ${VAR} references are resolved from the local environment.`, requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgAgentName, "", "", "Team-unique name for the config", requiredOpt())
	cmdCreate.Example = `doctl agents config create --spec agent-spec.yaml --name my-config`

	cmdList := CmdBuilder(cmd, RunAgentsConfigList, "list",
		"List agent configs",
		`Lists Agent Configs visible to your team. Supports pagination via `+"`"+`--page-size`+"`"+` and `+"`"+`--page-token`+"`"+`. When more pages exist, the next page token is printed after the table.`,
		Writer, ns, aliasOpt("ls"),
		displayerType(&displayers.HostedAgentConfigSummary{}))
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of configs to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	cmdList.Example = `doctl agents config list --page-size 10`

	CmdBuilder(cmd, RunAgentsConfigGet, "get <config-id>",
		"Get an agent config",
		"Prints one Agent Config, including its manifest and redacted credential slots.",
		Writer, ns, aliasOpt("show"),
		displayerType(&displayers.HostedAgentConfig{}))

	CmdBuilder(cmd, RunAgentsConfigDelete, "delete <config-id>",
		"Delete an agent config",
		`Soft-deletes an Agent Config and frees its team-unique name. The API returns an error if any session created from the config is still active (provisioning, ready, detached, or paused). List those sessions with `+"`"+`doctl agents config list-sessions`+"`"+`, destroy them (`+"`"+`doctl agents destroy <session>`+"`"+`), then retry.`,
		Writer, ns, aliasOpt("rm"))

	cmdSessions := CmdBuilder(cmd, RunAgentsConfigListSessions, "list-sessions <config-id>",
		"List sessions started from a config",
		`Lists sessions created from an Agent Config. Supports `+"`"+`--status`+"`"+`, `+"`"+`--name`+"`"+`, `+"`"+`--page-size`+"`"+`, and `+"`"+`--page-token`+"`"+`.`,
		Writer, ns, aliasOpt("sessions"),
		displayerType(&displayers.HostedAgentSession{}))
	AddIntFlag(cmdSessions, doctl.ArgAgentPageSize, "", 0, "Maximum number of sessions to return per page")
	AddStringFlag(cmdSessions, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	AddStringFlag(cmdSessions, doctl.ArgAgentStatus, "", "", "Filter by session status (e.g. SESSION_STATUS_READY)")
	AddStringFlag(cmdSessions, doctl.ArgAgentName, "", "", "Filter by session name")
	cmdSessions.Example = `doctl agents config list-sessions cfg_abc123 --status SESSION_STATUS_READY`

	cmdStartSession := CmdBuilder(cmd, RunAgentsConfigStartSession, "start-session <config-id>",
		"Start a session from a config",
		`Creates a new agent session from an existing Agent Config ID, without re-supplying the manifest. The `+"`"+`--name`+"`"+` for the new session is required and must be unique among your team's active sessions.`,
		Writer, ns, aliasOpt("start"),
		displayerType(&displayers.HostedAgentSession{}))
	AddStringFlag(cmdStartSession, doctl.ArgAgentName, "", "", "Name for the new session", requiredOpt())
	cmdStartSession.Example = `doctl agents config start-session cfg_abc123 --name my-session`

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
	manifest, err := readManifest(os.Stdin, specPath)
	if err != nil {
		return err
	}
	cfg, err := c.HostedAgents().CreateAgentConfig(&godo.HostedAgentConfigCreateRequest{
		Name:         name,
		ManifestYAML: string(manifest),
	})
	if err != nil {
		return err
	}
	if err := c.Display(&displayers.HostedAgentConfig{Configs: []godo.HostedAgentConfig{*cfg}, Single: true}); err != nil {
		return err
	}
	for _, w := range cfg.Warnings {
		fmt.Fprintf(c.Out, "Warning: %s\n", w)
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
	if err := c.Display(&displayers.HostedAgentConfigSummary{Configs: configs}); err != nil {
		return err
	}
	if next != "" {
		fmt.Fprintf(c.Out, "\nNext page token: %s\n", next)
	}
	return nil
}

// RunAgentsConfigGet fetches one Agent Config.
func RunAgentsConfigGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	cfg, err := c.HostedAgents().GetAgentConfig(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentConfig{Configs: []godo.HostedAgentConfig{*cfg}, Single: true})
}

// RunAgentsConfigDelete soft-deletes an Agent Config.
func RunAgentsConfigDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	configID := c.Args[0]
	if err := c.HostedAgents().DeleteAgentConfig(configID); err != nil {
		if agentConfigHasActiveSessionsErr(err) {
			msg, _, _ := agentAPIError(err)
			return fmt.Errorf("%s. List them with `doctl agents config list-sessions %s`, destroy each with `doctl agents destroy`, then retry", strings.TrimRight(msg, "."), configID)
		}
		return err
	}
	fmt.Fprintf(c.Out, "Deleted agent config %s\n", configID)
	return nil
}

// RunAgentsConfigListSessions lists sessions started from an Agent Config.
func RunAgentsConfigListSessions(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
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

	sessions, next, err := c.HostedAgents().ListAgentConfigSessions(c.Args[0], opt)
	if err != nil {
		return err
	}
	if err := c.Display(&displayers.HostedAgentSession{Sessions: sessions}); err != nil {
		return err
	}
	if next != "" {
		fmt.Fprintf(c.Out, "\nNext page token: %s\n", next)
	}
	return nil
}

// RunAgentsConfigStartSession creates a session from an existing Agent Config.
func RunAgentsConfigStartSession(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	if err != nil {
		return err
	}
	sess, err := c.HostedAgents().CreateSessionFromConfig(&godo.HostedAgentSessionFromConfigRequest{
		Name:     name,
		ConfigID: c.Args[0],
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}, Single: true})
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
