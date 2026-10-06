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

Optional filters match signals-api POST /v1/signals/exports:
- --signal-type (repeatable) filters instances in the artifact; JSON field is signal_type, not signal_types
- --session-ids limits the cohort
- --start-time / --end-time are Unix epoch seconds (inclusive / exclusive)
- --concerning, --signal-category, --signal-layer

Cohorts over 2000 segments are rejected. Empty cohorts are allowed.
Use GET .../download after status is complete; the job JSON has no download_url.`,
		Writer, aliasOpt("c"),
		displayerType(&displayers.SignalsExport{}),
	)
	AddStringFlag(cmdExportCreate, doctl.ArgSignalsAgentID, "", "", "The agent UUID to export data for.", requiredOpt())
	AddStringSliceFlag(cmdExportCreate, doctl.ArgSignalsExportSignalType, "", []string{}, "Signal types to include (JSON: signal_type). Repeatable. Catalog: doctl signals export options.")
	AddStringSliceFlag(cmdExportCreate, doctl.ArgSignalsExportSessionIDs, "", []string{}, "Optional session IDs to include (OR). Omit for all sessions.")
	AddIntFlag(cmdExportCreate, doctl.ArgSignalsExportStartTime, "", 0, "Start time filter (Unix epoch seconds, inclusive).")
	AddIntFlag(cmdExportCreate, doctl.ArgSignalsExportEndTime, "", 0, "End time filter (Unix epoch seconds, exclusive).")
	AddBoolFlag(cmdExportCreate, doctl.ArgSignalsExportConcerning, "", false, "If set, only export concerning segments.")
	AddStringFlag(cmdExportCreate, doctl.ArgSignalsExportSignalCategory, "", "", "Optional signal category filter.")
	AddStringFlag(cmdExportCreate, doctl.ArgSignalsExportSignalLayer, "", "", "Optional signal layer filter.")

	CmdBuilder(
		cmd,
		RunSignalsExportGet,
		"get",
		"Get a Signals export",
		"Retrieves a Signals export job by ID. Poll until status is complete, then run `download`. The job body does not include a download URL.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.SignalsExport{}),
	)

	CmdBuilder(
		cmd,
		RunSignalsExportDownload,
		"download",
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
		"Returns the static signal_type catalog used by create and by weekly export-trigger.",
		Writer,
		displayerType(&displayers.SignalsExportOptions{}),
	)

	return cmd
}

// SignalsExportTrigger creates `doctl signals export-trigger`.
func SignalsExportTrigger() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "export-trigger",
			Short: "Manage weekly Signals scheduled exports",
			Long:  "Team-level weekly export opt-in (GET/PUT /v1/signals/export-trigger). Separate from per-agent collection consent. This API uses signal_types (plural).",
		},
	}

	CmdBuilder(
		cmd,
		RunSignalsExportTriggerGet,
		"get",
		"Get the team's weekly export trigger",
		"Returns the stored trigger, or a synthetic disabled trigger if none has been saved.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.SignalsExportTrigger{}),
	)

	cmdSet := CmdBuilder(
		cmd,
		RunSignalsExportTriggerSet,
		"set",
		"Create or update the team's weekly export trigger",
		"Upserts the weekly trigger. enabled=true requires at least one catalog signal type (this request or previously stored). enabled=false with empty signal-types preserves stored types.",
		Writer, aliasOpt("s"),
		displayerType(&displayers.SignalsExportTrigger{}),
	)
	AddBoolFlag(cmdSet, doctl.ArgSignalsEnabled, "", false, "Enable (true) or disable (false) weekly scheduled exports.", requiredOpt())
	AddStringSliceFlag(cmdSet, doctl.ArgSignalsExportTriggerSignalTypes, "", []string{}, "Signal types for the weekly cron (JSON: signal_types). Catalog: doctl signals export options.")
	AddStringFlag(cmdSet, doctl.ArgSignalsExportCadence, "", "weekly", "Cadence. Only weekly is accepted.")

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

	exports, err := c.Signals().ListExports(&do.SignalsExportListOptions{
		AgentID: agentID,
		Limit:   limit,
		After:   after,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsExport{Exports: exports})
}

// RunSignalsExportCreate creates a Signals export job.
func RunSignalsExportCreate(c *CmdConfig) error {
	agentID, err := c.Doit.GetString(c.NS, doctl.ArgSignalsAgentID)
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
	sessionIDs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSignalsExportSessionIDs)
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
	concerning, err := c.Doit.GetBool(c.NS, doctl.ArgSignalsExportConcerning)
	if err != nil {
		return err
	}
	category, err := c.Doit.GetString(c.NS, doctl.ArgSignalsExportSignalCategory)
	if err != nil {
		return err
	}
	layer, err := c.Doit.GetString(c.NS, doctl.ArgSignalsExportSignalLayer)
	if err != nil {
		return err
	}

	req := &do.SignalsCreateExportRequest{
		AgentID:    agentID,
		SignalType: signalType,
		SessionIDs: sessionIDs,
	}
	if startTime > 0 {
		st := int64(startTime)
		req.StartTime = &st
	}
	if endTime > 0 {
		et := int64(endTime)
		req.EndTime = &et
	}
	if concerning {
		req.Concerning = &concerning
	}
	if category != "" {
		req.SignalCategory = &category
	}
	if layer != "" {
		req.SignalLayer = &layer
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

// RunSignalsExportTriggerGet fetches the weekly trigger.
func RunSignalsExportTriggerGet(c *CmdConfig) error {
	trig, err := c.Signals().GetExportTrigger()
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsExportTrigger{Triggers: []do.SignalsExportTrigger{*trig}})
}

// RunSignalsExportTriggerSet upserts the weekly trigger.
func RunSignalsExportTriggerSet(c *CmdConfig) error {
	enabled, err := c.Doit.GetBool(c.NS, doctl.ArgSignalsEnabled)
	if err != nil {
		return err
	}
	types, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSignalsExportTriggerSignalTypes)
	if err != nil {
		return err
	}
	cadence, err := c.Doit.GetString(c.NS, doctl.ArgSignalsExportCadence)
	if err != nil {
		return err
	}

	trig, err := c.Signals().UpsertExportTrigger(&do.SignalsExportTriggerUpsertRequest{
		Enabled:     enabled,
		Cadence:     cadence,
		SignalTypes: types,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsExportTrigger{Triggers: []do.SignalsExportTrigger{*trig}})
}
