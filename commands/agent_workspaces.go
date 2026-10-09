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
	"math"
	"os"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// errWorkspaceNeedsConfig is returned when --workspace is used without
// --from-config, before any request is made.
func errWorkspaceNeedsConfig() error {
	return fmt.Errorf("--%s needs --%s: save the manifest as an Agent Config first (`%s config create --%s <file> --%s <name>`), then create the session from it",
		doctl.ArgAgentWorkspace, doctl.ArgAgentFromConfig, agentCLI, doctl.ArgAgentSpec, doctl.ArgAgentName)
}

// AgentWorkspaces generates the `doctl harness-runtime workspace` subtree.
func AgentWorkspaces() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "workspace",
			Aliases: []string{"workspaces"},
			Short:   "Manage persistent workspaces (files kept between sessions)",
			Long:    agentsWorkspaceRootHelpMD,
		},
	}

	cmdCreate := CmdBuilder(cmd, RunAgentsWorkspaceCreate, "create",
		"Create a persistent workspace",
		agentsWorkspaceCreateHelpMD,
		Writer, agentPrettyErrors(),
		displayerType(&displayers.HostedAgentWorkspace{}))
	AddIntFlag(cmdCreate, doctl.ArgAgentWorkspaceSizeGiB, "", 0, "Size of the workspace in GiB, a whole number from 1 to 100 (required). Cannot be changed later.")
	AddStringFlag(cmdCreate, doctl.ArgAgentName, "", "", "Optional label for the workspace. Not unique.")
	AddStringFlag(cmdCreate, doctl.ArgAgentIdempotencyKey, "", "", "Key that makes a repeated create return the first workspace. Generated for you when omitted; pass your own when a script may re-run this command.")
	cmdCreate.Example = `doctl harness-runtime workspace create --size-gib 10 --name notes; doctl harness-runtime workspace create --size-gib 10 --idempotency-key nightly-notes-1`

	cmdList := CmdBuilder(cmd, RunAgentsWorkspaceList, "list",
		"List persistent workspaces",
		agentsWorkspaceListHelpMD,
		Writer, agentPrettyErrors(), aliasOpt("ls"),
		displayerType(&displayers.HostedAgentWorkspace{}))
	AddStringFlag(cmdList, doctl.ArgAgentWorkspaceState, "", "", "Only list workspaces in this state: AVAILABLE, ATTACHING, ATTACHED, RELEASING, FAILED or DELETING")
	AddIntFlag(cmdList, doctl.ArgAgentPageSize, "", 0, "Maximum number of workspaces to return per page")
	AddStringFlag(cmdList, doctl.ArgAgentPageToken, "", "", "Pagination cursor from a previous list response")
	cmdList.Example = `doctl harness-runtime workspace list --page-size 10; doctl harness-runtime workspace list --state AVAILABLE`

	CmdBuilder(cmd, RunAgentsWorkspaceGet, "get <workspace-id>",
		"Get a persistent workspace",
		agentsWorkspaceGetHelpMD,
		Writer, agentPrettyErrors(), aliasOpt("show"),
		displayerType(&displayers.HostedAgentWorkspace{}))

	cmdDelete := CmdBuilder(cmd, RunAgentsWorkspaceDelete, "delete <workspace-id>",
		"Permanently delete a persistent workspace",
		agentsWorkspaceDeleteHelpMD,
		Writer, agentPrettyErrors(), aliasOpt("rm"))
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete without asking first (a terminal asks by default; with no terminal it never asks)")
	cmdDelete.Example = `doctl harness-runtime workspace delete 018f6f2a-3c1e-7b6a-9d4e-5a7b8c9d0e1f`

	requireAgentSubcommand(cmd)
	return cmd
}

// RunAgentsWorkspaceCreate creates a persistent workspace.
func RunAgentsWorkspaceCreate(c *CmdConfig) error {
	if !agentFlagChanged(c, doctl.ArgAgentWorkspaceSizeGiB) {
		return fmt.Errorf("--%s is required: the workspace size in GiB, a whole number from 1 to 100", doctl.ArgAgentWorkspaceSizeGiB)
	}
	size, err := c.Doit.GetInt(c.NS, doctl.ArgAgentWorkspaceSizeGiB)
	if err != nil {
		return err
	}
	// The server owns the upper bound, which a team limit can lower. This only
	// refuses what can never be valid, and what would not fit the wire type.
	if size < 1 || size > math.MaxInt32 {
		return fmt.Errorf("invalid --%s %d: use a whole number from 1 to 100", doctl.ArgAgentWorkspaceSizeGiB, size)
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgAgentName)
	if err != nil {
		return err
	}
	key, err := c.Doit.GetString(c.NS, doctl.ArgAgentIdempotencyKey)
	if err != nil {
		return err
	}
	// One key per invocation, so a request the client repeats cannot create a
	// second workspace. A script that re-runs the command passes its own.
	if key == "" {
		key = uuid.NewString()
	}

	ws, err := c.HostedAgents().CreateWorkspace(&godo.HostedAgentWorkspaceCreateRequest{
		SizeGibibytes:  int32(size),
		Name:           name,
		IdempotencyKey: key,
	})
	if err != nil {
		return err
	}
	if agentStructuredOutput(c) {
		return c.Display(&displayers.HostedAgentWorkspace{Workspaces: []godo.HostedAgentWorkspace{*ws}, Single: true})
	}
	stylingEnabled = detectStyling()
	printWorkspaceCard(c.Out, ws, true)
	return nil
}

// RunAgentsWorkspaceList lists persistent workspaces.
func RunAgentsWorkspaceList(c *CmdConfig) error {
	opt := &godo.HostedAgentWorkspaceListOptions{}
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
	state, err := c.Doit.GetString(c.NS, doctl.ArgAgentWorkspaceState)
	if err != nil {
		return err
	}
	opt.State = godo.HostedAgentWorkspaceState(state)

	workspaces, next, err := c.HostedAgents().ListWorkspaces(opt)
	if err != nil {
		return err
	}
	if agentStructuredOutput(c) {
		if err := c.Display(&displayers.HostedAgentWorkspace{Workspaces: workspaces}); err != nil {
			return err
		}
		if next != "" {
			fmt.Fprintf(os.Stderr, "Next page token: %s\n", next)
		}
		return nil
	}
	stylingEnabled = detectStyling()
	if len(workspaces) == 0 && next != "" {
		// A page can be empty while more follow, for example when a state filter
		// skips every workspace the server looked at.
		fmt.Fprintln(c.Out, colorize("No workspaces on this page", colMuted))
	} else {
		printWorkspacesList(c.Out, workspaces)
	}
	printAgentNextPage(c.Out, next)
	return nil
}

// RunAgentsWorkspaceGet fetches one persistent workspace.
func RunAgentsWorkspaceGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	ws, err := c.HostedAgents().GetWorkspace(c.Args[0])
	if err != nil {
		return err
	}
	if agentStructuredOutput(c) {
		return c.Display(&displayers.HostedAgentWorkspace{Workspaces: []godo.HostedAgentWorkspace{*ws}, Single: true})
	}
	stylingEnabled = detectStyling()
	printWorkspaceCard(c.Out, ws, false)
	return nil
}

// RunAgentsWorkspaceDelete permanently deletes a persistent workspace.
func RunAgentsWorkspaceDelete(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	// A terminal asks first, since the files cannot be brought back. With no terminal
	// (a script, --interactive=false) there is nobody to ask, and the workspace id was
	// given explicitly, so the delete goes ahead without --force.
	if !force && Interactive && AskForConfirm("permanently delete this workspace and everything saved in it?") != nil {
		return fmt.Errorf("operation aborted")
	}
	if err := c.HostedAgents().DeleteWorkspace(c.Args[0]); err != nil {
		return err
	}
	stylingEnabled = detectStyling()
	printAgentSuccess(c.Out, fmt.Sprintf("Deleted workspace %s", c.Args[0]))
	return nil
}
