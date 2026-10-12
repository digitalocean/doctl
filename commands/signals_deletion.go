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

package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const (
	signalsDeletionMaxListLimit = 100
)

// SignalsDeletion creates the `doctl signals deletion` subcommand group.
func SignalsDeletion() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "deletion",
			Aliases: []string{"deletions"},
			Short:   "Display commands for Signals data deletion",
			Long: `The subcommands of ` + "`doctl signals deletion`" + ` permanently delete Signals conversation data on demand.

Deletion cannot be undone. When you create a deletion for a Harness Runtime agent (` + "`--type agent`" + `) or for Serverless Inference (` + "`--type inference`" + `), Signals turns collection off for that scope immediately, hides matching conversations from Signals reads immediately, and deletes that conversation data asynchronously in the background. Poll ` + "`doctl signals deletion get <deletion-id>`" + ` until the status is ` + "`complete`" + `.

For ` + "`--type agent`" + `, set ` + "`--agent-id`" + ` to the agent's Environment Config ID (the agent UUID). For ` + "`--type inference`" + `, Signals deletes Serverless Inference conversations for your whole team (team-wide; not limited to one endpoint or model) that were collected at or before the deletion request. Do not pass ` + "`--agent-id`" + ` for inference.

For an agent deletion, wait until the job is ` + "`complete`" + ` before enabling Signals for that agent again. Earlier export files are not deleted and remain available until they expire.

Status values: queued, running, complete, failed. DigitalOcean may retry transient processing errors while a job is still active. If a job ends as ` + "`failed`" + `, matching conversations stay hidden and collection stays off; run ` + "`create`" + ` again to retry, and use ` + "`--format DeletionID,Status,ErrorMessage`" + ` to see why it failed.

Note: Deletion removes Signals conversation data for the requested scope. Aggregated Insights or metrics charts may still show historical summary datapoints after deletion. Those summaries are not the same as recoverable conversation content.`,
		},
	}

	deletionDetails := `
- The deletion job ID
- The type (agent or inference)
- The agent ID (agent only)
- The status (queued, running, complete, failed)
- Created / started / completed timestamps (Unix seconds)

Extra columns are available with --format: TeamID, ErrorMessage.
`

	cmdCreate := CmdBuilder(
		cmd,
		RunSignalsDeletionCreate,
		"create",
		"Permanently delete Signals data",
		`Starts an asynchronous, permanent deletion of Signals conversation data. This cannot be undone.

- --type agent --agent-id <environment-config-id>: deletes Signals conversations for one Harness Runtime agent and turns off collection for that agent. --agent-id is the agent's Environment Config ID (the agent UUID).
- --type inference: deletes Serverless Inference Signals conversations for your whole team (team-wide; not limited to one endpoint or model) that were collected at or before the deletion request, and turns off team inference collection. Do not pass --agent-id.

On create, matching conversations are hidden from Signals reads immediately and collection for that scope is turned off; deletion continues in the background. Poll `+"`doctl signals deletion get <deletion-id>`"+` until the status is complete. For an agent deletion, wait until the job is complete before enabling Signals for that agent again.

You are asked to confirm unless you pass --force. Without a terminal, --force is required.

If an active deletion for the same target already exists, the API returns HTTP 200 with that job (doctl prints a note on stderr); a new job is HTTP 202. Safe to retry while queued or running. After the first job finishes, create starts a new one. Earlier export files are not deleted and remain available until they expire.

Note: Aggregated Insights or metrics charts may still show historical summary datapoints after deletion. Those summaries are not the same as recoverable conversation content.

Other API errors you may see: 400 (bad type/agent_id), 403 (team_id mismatch), 404 (unknown agent), 429 (too many active deletion jobs for the team).

--type is required (the API also requires it; omitting type is HTTP 400). Your team ID is detected automatically (this needs permission to read Signals consents). If you lack that permission, pass --team-id.

Each job contains:`+deletionDetails,
		Writer, aliasOpt("c"),
		displayerType(&displayers.SignalsDeletion{}),
	)
	AddStringFlag(cmdCreate, doctl.ArgSignalsDeletionType, "", "", `What to delete: "agent" or "inference".`, requiredOpt())
	AddStringFlag(cmdCreate, doctl.ArgSignalsAgentID, "", "", "Harness Runtime Environment Config ID (agent UUID / agent_id). Required for --type agent; must not be set for inference.")
	AddIntFlag(cmdCreate, doctl.ArgSignalsTeamID, "", 0, "Numeric team ID. Auto-detected when omitted.")
	AddBoolFlag(cmdCreate, doctl.ArgForce, doctl.ArgShortForce, false, "Skip the confirmation prompt")

	CmdBuilder(
		cmd,
		RunSignalsDeletionGet,
		"get <deletion-id>",
		"Get a Signals deletion job",
		"Retrieves one Signals deletion job by ID. Poll until the status is complete. Each job contains:"+deletionDetails,
		Writer, aliasOpt("g"),
		displayerType(&displayers.SignalsDeletion{}),
	)

	cmdList := CmdBuilder(
		cmd,
		RunSignalsDeletionList,
		"list",
		"List Signals deletion jobs",
		"Lists Signals deletion jobs for your team, newest first. Each job contains:"+deletionDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.SignalsDeletion{}),
	)
	AddIntFlag(cmdList, doctl.ArgSignalsExportLimit, "", 20, "Maximum number of jobs to return (1-100). Server default is also 20 when omitted.")
	AddStringFlag(cmdList, doctl.ArgSignalsExportAfter, "", "", "Opaque pagination cursor from page_info.end_cursor of a previous page (printed as the next page token on stderr).")

	return cmd
}

// RunSignalsDeletionCreate starts a permanent Signals data deletion.
func RunSignalsDeletionCreate(c *CmdConfig) error {
	typ, err := c.Doit.GetString(c.NS, doctl.ArgSignalsDeletionType)
	if err != nil {
		return err
	}
	agentID, err := c.Doit.GetString(c.NS, doctl.ArgSignalsAgentID)
	if err != nil {
		return err
	}
	teamID, err := c.Doit.GetInt(c.NS, doctl.ArgSignalsTeamID)
	if err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	// Trim first, then validate the trimmed values.
	typ = strings.TrimSpace(typ)
	agentID = strings.TrimSpace(agentID)

	switch typ {
	case godo.SignalsDeletionTypeAgent:
		if agentID == "" {
			return fmt.Errorf("--agent-id is required when --type is agent")
		}
		parsed, err := uuid.Parse(agentID)
		if err != nil {
			return fmt.Errorf("--agent-id must be a valid UUID: %w", err)
		}
		// Send the canonical lowercase form so the server's duplicate-job check
		// (keyed on the raw string) sees one spelling per agent.
		agentID = parsed.String()
	case godo.SignalsDeletionTypeInference:
		if agentID != "" {
			return fmt.Errorf("--agent-id must not be set when --type is inference")
		}
	default:
		return fmt.Errorf(`--type must be "agent" or "inference"`)
	}
	if teamID < 0 {
		return fmt.Errorf("--team-id must not be negative")
	}

	resolvedTeamID := int64(teamID)
	if resolvedTeamID == 0 {
		resolvedTeamID, err = c.Signals().GetTeamID()
		if err != nil {
			return fmt.Errorf("could not detect your team ID: %w; pass --team-id to skip auto-detection", err)
		}
	}

	if !force {
		if err := AskForConfirm(signalsDeletionConfirmMessage(typ, agentID, resolvedTeamID)); err != nil {
			return err
		}
	}

	job, existing, err := c.Signals().CreateDeletion(&godo.SignalsCreateDeletionRequest{
		Type:    typ,
		TeamID:  resolvedTeamID,
		AgentID: agentID,
	})
	if err != nil {
		return err
	}
	if existing {
		// stderr, so `--output json` on stdout stays valid JSON.
		fmt.Fprintln(os.Stderr, "An active deletion for this target already exists; showing the existing job.")
	}
	return c.Display(&displayers.SignalsDeletion{Deletions: do.SignalsDeletions{*job}})
}

// signalsDeletionConfirmMessage builds the prompt text. AskForConfirm adds the
// "Are you sure you want to " prefix and the trailing "?", so the message must
// start with a verb and must not end with a question mark.
func signalsDeletionConfirmMessage(typ, agentID string, teamID int64) string {
	if typ == godo.SignalsDeletionTypeAgent {
		return fmt.Sprintf("permanently delete all Signals data for agent %s (team %d) and turn off collection for it (this cannot be undone)", agentID, teamID)
	}
	return fmt.Sprintf("permanently delete all Signals inference data for the whole team %d (this cannot be undone)", teamID)
}

// RunSignalsDeletionGet retrieves a Signals deletion job by ID.
func RunSignalsDeletionGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	id := strings.TrimSpace(c.Args[0])
	if id == "" {
		return fmt.Errorf("deletion ID must not be empty")
	}
	job, err := c.Signals().GetDeletion(id)
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsDeletion{Deletions: do.SignalsDeletions{*job}})
}

// RunSignalsDeletionList lists Signals deletion jobs.
func RunSignalsDeletionList(c *CmdConfig) error {
	limit, err := c.Doit.GetInt(c.NS, doctl.ArgSignalsExportLimit)
	if err != nil {
		return err
	}
	after, err := c.Doit.GetString(c.NS, doctl.ArgSignalsExportAfter)
	if err != nil {
		return err
	}
	if limit < 1 || limit > signalsDeletionMaxListLimit {
		return fmt.Errorf("--limit must be between 1 and %d", signalsDeletionMaxListLimit)
	}

	jobs, next, err := c.Signals().ListDeletions(&godo.SignalsListDeletionsOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
			Limit: limit,
			After: after,
		},
	})
	if err != nil {
		return err
	}
	if err := c.Display(&displayers.SignalsDeletion{Deletions: jobs}); err != nil {
		return err
	}
	if next != "" {
		// stderr, so stdout stays clean for scripts and `--output json`.
		fmt.Fprintf(os.Stderr, "Next page token: %s\n", next)
	}
	return nil
}
