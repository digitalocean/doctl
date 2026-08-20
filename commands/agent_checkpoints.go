/*
Copyright 2026 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    15|See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"fmt"
	"os"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// AgentCheckpoints generates the `doctl agent checkpoint` subtree.
func AgentCheckpoints() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "checkpoint",
			Aliases: []string{"checkpoints", "cp"},
			Short:   "Manage session checkpoints (save points)",
			Long:    agentsCheckpointRootHelpMD,
		},
	}

	cmdCreate := CmdBuilder(cmd, RunAgentsCheckpointCreate, "create <session>",
		"Create a checkpoint for a session",
		agentsCheckpointCreateHelpMD,
		Writer, aliasOpt("save"),
		displayerType(&displayers.HostedAgentCheckpoint{}))
	AddStringFlag(cmdCreate, doctl.ArgAgentCheckpointLabel, "", "", "Optional label for the checkpoint")
	cmdCreate.Example = `doctl agent checkpoint create sess_abc123 --label before-refactor`

	cmdList := CmdBuilder(cmd, RunAgentsCheckpointList, "list <session>",
		"List checkpoints for a session",
		agentsCheckpointListHelpMD,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.HostedAgentCheckpoint{}))
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of checkpoints to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	cmdList.Example = `doctl agent checkpoint list sess_abc123`

	CmdBuilder(cmd, RunAgentsCheckpointGet, "get <session> <checkpoint-id>",
		"Get a checkpoint",
		agentsCheckpointGetHelpMD,
		Writer, aliasOpt("show"),
		displayerType(&displayers.HostedAgentCheckpoint{}))

	CmdBuilder(cmd, RunAgentsCheckpointDelete, "delete <session> <checkpoint-id>",
		"Delete a checkpoint",
		agentsCheckpointDeleteHelpMD,
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
	if Output == "json" {
		return c.Display(&displayers.HostedAgentCheckpoint{Checkpoints: []godo.HostedAgentCheckpoint{*cp}, Single: true})
	}
	stylingEnabled = detectStyling()
	printCheckpointCard(c.Out, cp, true)
	return nil
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
	if Output == "json" {
		if err := c.Display(&displayers.HostedAgentCheckpoint{Checkpoints: checkpoints}); err != nil {
			return err
		}
		if next != "" {
			fmt.Fprintf(os.Stderr, "Next page token: %s\n", next)
		}
		return nil
	}
	stylingEnabled = detectStyling()
	printCheckpointsList(c.Out, checkpoints)
	printAgentNextPage(c.Out, next)
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
	if Output == "json" {
		return c.Display(&displayers.HostedAgentCheckpoint{Checkpoints: []godo.HostedAgentCheckpoint{*cp}, Single: true})
	}
	stylingEnabled = detectStyling()
	printCheckpointCard(c.Out, cp, false)
	return nil
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
	stylingEnabled = detectStyling()
	printAgentSuccess(c.Out, fmt.Sprintf("Deleted checkpoint %s", resp.CheckpointID))
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
	if Output == "json" {
		return c.Display(&displayers.HostedAgentSession{Sessions: sessions})
	}
	stylingEnabled = detectStyling()
	printSessionsList(c.Out, sessions)
	return nil
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
	if Output == "json" {
		return c.Display(&displayers.HostedAgentSession{Sessions: []do.HostedAgentSession{*sess}, Single: true})
	}
	stylingEnabled = detectStyling()
	printSessionShowCard(c.Out, sess)
	return nil
}
