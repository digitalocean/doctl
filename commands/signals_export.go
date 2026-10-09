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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// SignalsExport creates the `doctl signals export` subcommand group.
func SignalsExport() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "export",
			Short: "Display commands for Signals bulk exports",
			Long:  "The subcommands of `doctl signals export` manage async bulk export jobs for Signals data. Create is idempotent. Poll `get` until status is `complete`, then call `download` for a short-lived URL. Status values: queued, running, complete, failed, expired.",
		},
	}

	exportDetails := `
- The export job ID
- The agent ID
- The export status (queued, running, complete, failed, expired)
- Stored filters (signal_type, session_ids, time window, …)
- Created / completed / expires timestamps (Unix seconds)
`

	cmdExportList := CmdBuilder(
		cmd,
		RunSignalsExportList,
		"list",
		"List Signals export jobs",
		"Lists Signals export jobs for the authenticated team, where each job contains:"+exportDetails,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.SignalsExport{}),
	)
	AddStringFlag(cmdExportList, doctl.ArgSignalsAgentID, "", "", "Filter exports by agent ID.")
	AddIntFlag(cmdExportList, doctl.ArgSignalsExportLimit, "", 20, "Maximum number of exports to return.")
	AddStringFlag(cmdExportList, doctl.ArgSignalsExportAfter, "", "", "Opaque pagination cursor from a previous page_info.end_cursor.")

	cmdExportCreate := CmdBuilder(
		cmd,
		RunSignalsExportCreate,
		"create",
		"Create a Signals export",
		`Starts a bulk Signals export job (201 Created, or 200 if an active job with the same fingerprint already exists).

Use --type to select the export source:
- agent (default): export for --agent-id (required)
- inference: export team serverless-inference data (uses the nil agent UUID; do not pass --agent-id)

Optional filters match signals-api POST /v1/signals/exports:
- --signal-type (repeatable) filters instances in the artifact; JSON field is signal_type, not signal_types
- --start-time / --end-time are Unix epoch seconds (inclusive / exclusive)

Cohorts over 2000 segments are rejected. Empty cohorts are allowed.
Use GET .../download after status is complete; the job JSON has no download_url.`,
		Writer, aliasOpt("c"),
		displayerType(&displayers.SignalsExport{}),
	)
	AddStringFlag(cmdExportCreate, doctl.ArgSignalsType, "", signalsSessionTypeAgent, "Export source: agent or inference.")
	AddStringFlag(cmdExportCreate, doctl.ArgSignalsAgentID, "", "", "The agent UUID to export data for (required when --type=agent).")
	AddStringSliceFlag(cmdExportCreate, doctl.ArgSignalsExportSignalType, "", []string{}, "Signal types to include (JSON: signal_type). Repeatable. Catalog: doctl signals export options.")
	AddIntFlag(cmdExportCreate, doctl.ArgSignalsExportStartTime, "", 0, "Start time filter (Unix epoch seconds, inclusive).")
	AddIntFlag(cmdExportCreate, doctl.ArgSignalsExportEndTime, "", 0, "End time filter (Unix epoch seconds, exclusive).")

	CmdBuilder(
		cmd,
		RunSignalsExportGet,
		"get <export-id>",
		"Get a Signals export",
		"Retrieves a Signals export job by ID. Poll until status is complete, then run `download`. The job body does not include a download URL.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.SignalsExport{}),
	)

	CmdBuilder(
		cmd,
		RunSignalsExportDownload,
		"download <export-id>",
		"Get a Signals export download URL",
		"Mints a ~15 minute pre-signed URL for a completed export. 404 if not complete; 410 if the artifact expired.",
		Writer, aliasOpt("dl"),
		displayerType(&displayers.SignalsExportDownload{}),
	)

	CmdBuilder(
		cmd,
		RunSignalsExportOptions,
		"options",
		"List Signals export filter options",
		"Returns the static signal_type catalog used by create.",
		Writer,
		displayerType(&displayers.SignalsExportOptions{}),
	)

	return cmd
}

// RunSignalsExportList lists Signals export jobs.
func RunSignalsExportList(c *CmdConfig) error {
	agentID, err := c.Doit.GetString(c.NS, doctl.ArgSignalsAgentID)
	if err != nil {
		return err
	}
	limit, err := c.Doit.GetInt(c.NS, doctl.ArgSignalsExportLimit)
	if err != nil {
		return err
	}
	after, err := c.Doit.GetString(c.NS, doctl.ArgSignalsExportAfter)
	if err != nil {
		return err
	}

	exports, err := c.Signals().ListExports(&godo.SignalsListExportsOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
			Limit: limit,
			After: after,
		},
		AgentID: agentID,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsExport{Exports: exports})
}

// RunSignalsExportCreate creates a Signals export job.
func RunSignalsExportCreate(c *CmdConfig) error {
	exportType, err := c.Doit.GetString(c.NS, doctl.ArgSignalsType)
	if err != nil {
		return err
	}
	agentID, err := c.Doit.GetString(c.NS, doctl.ArgSignalsAgentID)
	if err != nil {
		return err
	}
	agentID, err = resolveSignalsAgentID(exportType, agentID)
	if err != nil {
		return err
	}
	if _, err := uuid.Parse(agentID); err != nil {
		return fmt.Errorf("agent-id must be a valid UUID: %w", err)
	}
	signalType, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSignalsExportSignalType)
	if err != nil {
		return err
	}
	startTime, err := c.Doit.GetInt(c.NS, doctl.ArgSignalsExportStartTime)
	if err != nil {
		return err
	}
	endTime, err := c.Doit.GetInt(c.NS, doctl.ArgSignalsExportEndTime)
	if err != nil {
		return err
	}

	req := &godo.SignalsCreateExportRequest{
		AgentID:    agentID,
		SignalType: signalType,
	}
	if startTime > 0 {
		st := int64(startTime)
		req.StartTime = &st
	}
	if endTime > 0 {
		et := int64(endTime)
		req.EndTime = &et
	}

	export, err := c.Signals().CreateExport(req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsExport{Exports: do.SignalsExports{*export}})
}

// RunSignalsExportGet retrieves a Signals export job by ID.
func RunSignalsExportGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	export, err := c.Signals().GetExport(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsExport{Exports: do.SignalsExports{*export}})
}

// RunSignalsExportDownload mints a pre-signed download URL.
func RunSignalsExportDownload(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	dl, err := c.Signals().GetExportDownload(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsExportDownload{Download: *dl})
}

// RunSignalsExportOptions returns the signal_type catalog.
func RunSignalsExportOptions(c *CmdConfig) error {
	opts, err := c.Signals().GetExportOptions()
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsExportOptions{Options: *opts})
}
