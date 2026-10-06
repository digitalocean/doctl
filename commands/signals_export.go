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
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

// SignalsExport creates the `doctl signals export` subcommand group.
func SignalsExport() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "export",
			Short: "Display commands for Signals bulk exports",
			Long:  "The subcommands of `doctl signals export` manage async bulk export jobs for Signals data.",
		},
	}

	exportDetails := `
		- The export job ID
		- The agent ID
		- The export status
		- The signal types included
		- When the export was created
		- When the export completed (if applicable)
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
	AddStringFlag(cmdExportList, doctl.ArgSignalsExportAfter, "", "", "Opaque pagination cursor from a previous response.")

	cmdExportCreate := CmdBuilder(
		cmd,
		RunSignalsExportCreate,
		"create",
		"Create a Signals export",
		"Starts a bulk Signals export job. Idempotent: if an active job with the same fingerprint exists, returns it.",
		Writer, aliasOpt("c"),
		displayerType(&displayers.SignalsExport{}),
	)
	AddStringFlag(cmdExportCreate, doctl.ArgSignalsAgentID, "", "", "The agent ID to export data for.", requiredOpt())
	AddStringSliceFlag(cmdExportCreate, doctl.ArgSignalsExportSignalTypes, "", []string{}, "Signal types to include in the export.")
	AddIntFlag(cmdExportCreate, doctl.ArgSignalsExportStartTime, "", 0, "Start time filter (Unix epoch seconds, inclusive).")
	AddIntFlag(cmdExportCreate, doctl.ArgSignalsExportEndTime, "", 0, "End time filter (Unix epoch seconds, exclusive).")

	CmdBuilder(
		cmd,
		RunSignalsExportGet,
		"get <export-id>",
		"Get a Signals export",
		"Retrieves a Signals export job by ID. Poll this endpoint until status is complete to get the download URL.",
		Writer, aliasOpt("g"),
		displayerType(&displayers.SignalsExport{}),
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
	signalTypes, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSignalsExportSignalTypes)
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

	req := &do.SignalsCreateExportRequest{
		AgentID:     agentID,
		SignalTypes: signalTypes,
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
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	exportID := c.Args[0]

	export, err := c.Signals().GetExport(exportID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsExport{Exports: do.SignalsExports{*export}})
}
