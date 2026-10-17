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
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

const (
	signalsSessionTypeAgent     = "agent"
	signalsSessionTypeInference = "inference"

	// signalsInferencePlaceholderAgentID is the nil agent UUID stamped on
	// serverless-inference Signals data (no Mars Agent Config).
	signalsInferencePlaceholderAgentID = "00000000-0000-0000-0000-000000000000"
)

// SignalsSession creates the `doctl signals session` subcommand group.
func SignalsSession() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "session",
			Short: "Display commands for Signals sessions and dialogues",
			Long:  "The subcommands of `doctl signals session` list agent sessions and session dialogues.",
		},
	}

	cmdSessionList := CmdBuilder(
		cmd,
		RunSignalsSessionList,
		"list",
		"List Signals sessions",
		`Lists Signals sessions, including session ID, total turns, started-at time, duration, and signal count.

Use --type to select the session source:
- agent (default): list sessions for --agent-id (required)
- inference: list team serverless-inference sessions (uses the nil agent UUID; do not pass --agent-id)`,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.SignalsSession{}),
	)
	AddStringFlag(cmdSessionList, doctl.ArgSignalsType, "", signalsSessionTypeAgent, "Session source: agent or inference.")
	AddStringFlag(cmdSessionList, doctl.ArgSignalsAgentID, "", "", "The agent ID to list sessions for (required when --type=agent).")
	AddIntFlag(cmdSessionList, doctl.ArgSignalsExportLimit, "", 20, "Maximum number of sessions to return.")
	AddStringFlag(cmdSessionList, doctl.ArgSignalsExportAfter, "", "", "Opaque pagination cursor.")
	AddIntFlag(cmdSessionList, doctl.ArgSignalsExportStartTime, "", 0, "Start time filter (Unix epoch seconds).")
	AddIntFlag(cmdSessionList, doctl.ArgSignalsExportEndTime, "", 0, "End time filter (Unix epoch seconds).")
	AddStringSliceFlag(cmdSessionList, doctl.ArgSignalsExportSignalType, "", []string{}, "Signal types to filter by (repeatable).")

	cmdDialogueList := CmdBuilder(
		cmd,
		RunSignalsSessionDialogueList,
		"dialogues",
		"List dialogues for a session",
		"Lists Signals dialogues for the given session, including dialogue ID, run ID, sequence, user message, run status, and any detected signals.",
		Writer, aliasOpt("d"),
		displayerType(&displayers.SignalsSessionDialogue{}),
	)
	AddStringFlag(cmdDialogueList, doctl.ArgSignalsSessionID, "", "", "The session ID to list dialogues for.", requiredOpt())
	AddIntFlag(cmdDialogueList, doctl.ArgSignalsExportLimit, "", 20, "Maximum number of dialogues to return.")
	AddStringFlag(cmdDialogueList, doctl.ArgSignalsExportAfter, "", "", "Opaque pagination cursor.")
	AddIntFlag(cmdDialogueList, doctl.ArgSignalsExportStartTime, "", 0, "Start time filter (Unix epoch seconds).")
	AddIntFlag(cmdDialogueList, doctl.ArgSignalsExportEndTime, "", 0, "End time filter (Unix epoch seconds).")
	AddStringSliceFlag(cmdDialogueList, doctl.ArgSignalsExportSignalType, "", []string{}, "Signal types to filter by (repeatable).")

	return cmd
}

// RunSignalsSessionList lists sessions for an agent or inference.
func RunSignalsSessionList(c *CmdConfig) error {
	sessionType, err := c.Doit.GetString(c.NS, doctl.ArgSignalsType)
	if err != nil {
		return err
	}
	agentID, err := c.Doit.GetString(c.NS, doctl.ArgSignalsAgentID)
	if err != nil {
		return err
	}
	agentID, err = resolveSignalsAgentID(sessionType, agentID)
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
	startTime, err := c.Doit.GetInt(c.NS, doctl.ArgSignalsExportStartTime)
	if err != nil {
		return err
	}
	endTime, err := c.Doit.GetInt(c.NS, doctl.ArgSignalsExportEndTime)
	if err != nil {
		return err
	}
	signalType, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSignalsExportSignalType)
	if err != nil {
		return err
	}

	opts := &godo.SignalsListAgentSessionsOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
			Limit: limit,
			After: after,
		},
		SignalType: signalType,
	}
	if startTime > 0 {
		st := int64(startTime)
		opts.StartTime = &st
	}
	if endTime > 0 {
		et := int64(endTime)
		opts.EndTime = &et
	}

	sessions, err := c.Signals().ListAgentSessions(agentID, opts)
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsSession{Sessions: sessions})
}

// resolveSignalsAgentID maps --type / --agent-id to the agent UUID used by
// Signals APIs. Inference traffic is stamped with the nil agent UUID.
func resolveSignalsAgentID(sessionType, agentID string) (string, error) {
	switch sessionType {
	case "", signalsSessionTypeAgent:
		if agentID == "" {
			return "", fmt.Errorf("--agent-id is required when --type=%s", signalsSessionTypeAgent)
		}
		return agentID, nil
	case signalsSessionTypeInference:
		if agentID != "" {
			return "", fmt.Errorf("--agent-id must not be set when --type=%s", signalsSessionTypeInference)
		}
		return signalsInferencePlaceholderAgentID, nil
	default:
		return "", fmt.Errorf("invalid --type %q: must be %s or %s", sessionType, signalsSessionTypeAgent, signalsSessionTypeInference)
	}
}

// RunSignalsSessionDialogueList lists dialogues for a session.
func RunSignalsSessionDialogueList(c *CmdConfig) error {
	sessionID, err := c.Doit.GetString(c.NS, doctl.ArgSignalsSessionID)
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
	startTime, err := c.Doit.GetInt(c.NS, doctl.ArgSignalsExportStartTime)
	if err != nil {
		return err
	}
	endTime, err := c.Doit.GetInt(c.NS, doctl.ArgSignalsExportEndTime)
	if err != nil {
		return err
	}
	signalType, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSignalsExportSignalType)
	if err != nil {
		return err
	}

	opts := &godo.SignalsListDialoguesOptions{
		SignalsCursorPageOptions: godo.SignalsCursorPageOptions{
			Limit: limit,
			After: after,
		},
		SignalType: signalType,
	}
	if startTime > 0 {
		st := int64(startTime)
		opts.StartTime = &st
	}
	if endTime > 0 {
		et := int64(endTime)
		opts.EndTime = &et
	}

	dialogues, err := c.Signals().ListSessionDialogues(sessionID, opts)
	if err != nil {
		return err
	}
	return c.Display(&displayers.SignalsSessionDialogue{Dialogues: dialogues})
}
