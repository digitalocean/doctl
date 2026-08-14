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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// AgentCheckpoints generates the `doctl agents checkpoint` subtree.
func AgentCheckpoints() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "checkpoint",
			Aliases: []string{"checkpoints", "cp"},
			Short:   "Manage session checkpoints (save points)",
			Long: `The ` + "`" + `doctl agents checkpoint` + "`" + ` commands create and manage save points for a hosted agent session.

A checkpoint captures the session's microVM (files and live process memory) between turns. Use forks to branch into independent sessions from a checkpoint, or rollback to rewind the same session in place.`,
		},
	}

	cmdCreate := CmdBuilder(cmd, RunAgentsCheckpointCreate, "create <session>",
		"Create a checkpoint for a session",
		`Creates an explicit checkpoint for a session. The call blocks until the checkpoint is READY.

Checkpoints can only be taken between turns (after run.completed). Optional `+"`"+`--label`+"`"+` stores a human-readable name.`,
		Writer, aliasOpt("save"),
		displayerType(&displayers.HostedAgentCheckpoint{}))
	AddStringFlag(cmdCreate, doctl.ArgAgentCheckpointLabel, "", "", "Optional label for the checkpoint")
	cmdCreate.Example = `doctl agents checkpoint create sess_abc123 --label before-refactor`

	cmdList := CmdBuilder(cmd, RunAgentsCheckpointList, "list <session>",
		"List checkpoints for a session",
		`Lists checkpoints for a session, newest first. Supports `+"`"+`--page-size`+"`"+` and `+"`"+`--page-token`+"`"+`.`,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HostedAgentCheckpoint{}))
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of checkpoints to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	cmdList.Example = `doctl agents checkpoint list sess_abc123`

	CmdBuilder(cmd, RunAgentsCheckpointGet, "get <session> <checkpoint-id>",
		"Get a checkpoint",
		"Prints details for one checkpoint.",
		Writer, aliasOpt("show"),
		displayerType(&displayers.HostedAgentCheckpoint{}))

	CmdBuilder(cmd, RunAgentsCheckpointDelete, "delete <session> <checkpoint-id>",
		"Delete a checkpoint",
		"Deletes a checkpoint (control-plane row and substrate capture). Idempotent.",
		Writer, aliasOpt("rm"))

	return cmd
}

// RunAgentsCheckpointCreate creates an explicit checkpoint.
func RunAgentsCheckpointCreate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	label, err := c.Doit.GetString(c.NS, doctl.ArgAgentCheckpointLabel)
	if err != nil {
		return err
	}
	cp, err := c.HostedAgents().CreateCheckpoint(sessionID, &godo.HostedAgentCheckpointCreateRequest{Label: label})
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentCheckpoint{Checkpoints: []godo.HostedAgentCheckpoint{*cp}, Single: true})
}

// RunAgentsCheckpointList lists checkpoints for a session.
func RunAgentsCheckpointList(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	opt := &godo.HostedAgentCheckpointListOptions{}
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

	checkpoints, next, err := c.HostedAgents().ListCheckpoints(sessionID, opt)
	if err != nil {
		return err
	}
	if err := c.Display(&displayers.HostedAgentCheckpoint{Checkpoints: checkpoints}); err != nil {
		return err
	}
	if next != "" {
		fmt.Fprintf(c.Out, "\nNext page token: %s\n", next)
	}
	return nil
}

// RunAgentsCheckpointGet fetches one checkpoint.
func RunAgentsCheckpointGet(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	cp, err := c.HostedAgents().GetCheckpoint(sessionID, c.Args[1])
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentCheckpoint{Checkpoints: []godo.HostedAgentCheckpoint{*cp}, Single: true})
}

// RunAgentsCheckpointDelete deletes a checkpoint.
func RunAgentsCheckpointDelete(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	resp, err := c.HostedAgents().DeleteCheckpoint(sessionID, c.Args[1])
	if err != nil {
		return err
	}
	fmt.Fprintf(c.Out, "Deleted checkpoint %s (deleted=%v)\n", resp.CheckpointID, resp.Deleted)
	return nil
}

// RunAgentsFork forks a session into N independent children.
func RunAgentsFork(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	fromCP, err := c.Doit.GetString(c.NS, doctl.ArgAgentFromCheckpoint)
	if err != nil {
		return err
	}
	count, err := c.Doit.GetInt(c.NS, doctl.ArgAgentForkCount)
	if err != nil {
		return err
	}
	if count < 0 || count > godo.HostedAgentForkMaxCount {
		return fmt.Errorf("--count must be between 1 and %d (got %d); omit or use 0 for server default of 1", godo.HostedAgentForkMaxCount, count)
	}
	sessions, err := c.HostedAgents().ForkSession(sessionID, &godo.HostedAgentForkSessionRequest{
		FromCheckpointID: fromCP,
		Count:            count,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentSession{Sessions: sessions})
}

// RunAgentsRollback rolls a session back to a checkpoint in place.
func RunAgentsRollback(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	sessionID, err := resolveSessionRef(c.HostedAgents(), c.Args[0])
	if err != nil {
		return err
	}
	sess, err := c.HostedAgents().RollbackToCheckpoint(sessionID, c.Args[1])
	if err != nil {
		return err
	}
	return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}, Single: true})
}
