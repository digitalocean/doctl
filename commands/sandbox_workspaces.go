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
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// addSandboxWorkspaceCmds adds the `workspace` group to `doctl sandbox`.
func addSandboxWorkspaceCmds(parent *Command) {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "workspace",
			Aliases: []string{"workspaces"},
			Short:   "Manage sandbox workspaces",
			Long: "The subcommands of `doctl sandbox workspace` manage persistent workspaces. " +
				"A workspace is a disk that outlives any one sandbox. " +
				"Once it is available, pass its ID to `doctl sandbox create --workspace-id` to attach it to a new sandbox.",
		},
	}
	parent.AddCommand(cmd)

	cmdCreate := CmdBuilder(cmd, RunSandboxWorkspaceCreate, "create",
		"Create a workspace",
		"Creates a persistent workspace. It can still be creating when this returns. "+
			"Attach it to a sandbox once its state is available.",
		Writer, aliasOpt("c"), displayerType(&displayers.SandboxWorkspace{}))
	AddStringFlag(cmdCreate, doctl.ArgSandboxWorkspaceName, "", "", "A name for the workspace")
	AddIntFlag(cmdCreate, doctl.ArgSandboxWorkspaceSizeLimitGiB, "", 0,
		"The size limit of the workspace, in GiB", requiredOpt())
	cmdCreate.Example = "The following example creates a 10 GiB workspace: " +
		"doctl sandbox workspace create --name my-workspace --size-limit-gib 10"

	cmdList := CmdBuilder(cmd, RunSandboxWorkspaceList, "list",
		"List your workspaces",
		"Lists your team's persistent workspaces. Use `--state` to show only the workspaces in one state.",
		Writer, aliasOpt("ls"), displayerType(&displayers.SandboxWorkspace{}))
	AddStringFlag(cmdList, doctl.ArgSandboxWorkspaceState, "", "",
		"Show only workspaces in this state: "+sandboxWorkspaceStateNames)
	cmdList.Example = "The following example lists the workspaces that are ready to attach: " +
		"doctl sandbox workspace list --state available"

	cmdGet := CmdBuilder(cmd, RunSandboxWorkspaceGet, "get <workspace-id>",
		"Retrieve a workspace",
		"Retrieves a workspace: its name, state, size limit, attached sandbox and usage.",
		Writer, aliasOpt("g"), displayerType(&displayers.SandboxWorkspace{}))
	cmdGet.Example = "The following example shows only the state of a workspace: " +
		"doctl sandbox workspace get 11111111-2222-4333-8444-555555555555 --format Name,State"

	cmdDelete := CmdBuilder(cmd, RunSandboxWorkspaceDelete, "delete <workspace-id>...",
		"Permanently delete one or more workspaces",
		"Permanently deletes the workspaces you name and their data. This is irreversible.",
		Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the workspace(s) without a confirmation prompt")
	cmdDelete.Example = "The following example deletes a workspace without asking: " +
		"doctl sandbox workspace delete 11111111-2222-4333-8444-555555555555 --force"
}

// RunSandboxWorkspaceCreate creates a workspace.
func RunSandboxWorkspaceCreate(c *CmdConfig) error {
	if len(c.Args) != 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	name, err := c.Doit.GetString(c.NS, doctl.ArgSandboxWorkspaceName)
	if err != nil {
		return err
	}
	size, err := c.Doit.GetInt(c.NS, doctl.ArgSandboxWorkspaceSizeLimitGiB)
	if err != nil {
		return err
	}
	if size <= 0 {
		return doctl.NewMissingArgsErr("--size-limit-gib must be greater than 0")
	}

	w, err := c.Sandboxes().CreateWorkspace(&godo.SandboxWorkspaceCreateRequest{
		Name:         strings.TrimSpace(name),
		SizeLimitGiB: int64(size),
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.SandboxWorkspace{Workspaces: do.SandboxWorkspaces{*w}})
}

// RunSandboxWorkspaceList lists workspaces.
func RunSandboxWorkspaceList(c *CmdConfig) error {
	if len(c.Args) != 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	flag, err := c.Doit.GetString(c.NS, doctl.ArgSandboxWorkspaceState)
	if err != nil {
		return err
	}
	state, err := parseSandboxWorkspaceState(flag)
	if err != nil {
		return err
	}
	list, err := c.Sandboxes().ListWorkspaces(state)
	if err != nil {
		return err
	}
	return c.Display(&displayers.SandboxWorkspace{Workspaces: list})
}

// RunSandboxWorkspaceGet retrieves a workspace.
func RunSandboxWorkspaceGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	w, err := c.Sandboxes().GetWorkspace(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.SandboxWorkspace{Workspaces: do.SandboxWorkspaces{*w}})
}

// RunSandboxWorkspaceDelete deletes one or more workspaces.
func RunSandboxWorkspaceDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if err := confirmDelete(force, "sandbox workspace", len(c.Args)); err != nil {
		return err
	}
	svc := c.Sandboxes()
	for _, id := range c.Args {
		if err := svc.DeleteWorkspace(id); err != nil {
			return fmt.Errorf("deleting workspace %s: %w", id, err)
		}
	}
	return nil
}

const sandboxWorkspaceStateNames = "`creating`, `available`, `attaching`, `attached`, `releasing`, `deleting` or `failed`"

// parseSandboxWorkspaceState turns the --state flag into an API state. It
// accepts the short form (available) or the API value
// (WORKSPACE_STATE_AVAILABLE), in any case. An empty value means any state.
func parseSandboxWorkspaceState(value string) (godo.SandboxWorkspaceState, error) {
	v := strings.ToUpper(strings.TrimSpace(value))
	if v == "" {
		return "", nil
	}
	v = strings.TrimPrefix(v, "WORKSPACE_STATE_")
	switch state := godo.SandboxWorkspaceState("WORKSPACE_STATE_" + v); state {
	case godo.SandboxWorkspaceStateCreating,
		godo.SandboxWorkspaceStateAvailable,
		godo.SandboxWorkspaceStateAttaching,
		godo.SandboxWorkspaceStateAttached,
		godo.SandboxWorkspaceStateReleasing,
		godo.SandboxWorkspaceStateDeleting,
		godo.SandboxWorkspaceStateFailed:
		return state, nil
	}
	return "", fmt.Errorf("invalid --state %q: expected %s", value, strings.ReplaceAll(sandboxWorkspaceStateNames, "`", ""))
}
