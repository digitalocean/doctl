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

const (
	// defaultSandboxTemplate is the template `sandbox create` uses without --template.
	defaultSandboxTemplate = "coding-base"
	// defaultSandboxSize is the size slug `sandbox create` uses without --size.
	defaultSandboxSize = "mars-2vcpu-4gb"
)

// Sandbox creates the sandbox command hierarchy.
func Sandbox() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "sandbox",
			Aliases: []string{"sandboxes", "sb"},
			Short:   "Manage Sandboxes",
			Long: "The subcommands of `doctl sandbox` manage DigitalOcean Sandboxes: isolated environments " +
				"created from a template. Use them to create, list, inspect, pause, resume and delete sandboxes.",
			GroupID: manageResourcesGroup,
		},
	}

	cmdSandboxCreate := CmdBuilder(cmd, RunSandboxCreate, "create",
		"Create a sandbox",
		"Creates a sandbox from a template and returns once it is running. "+
			"Without flags it uses the `"+defaultSandboxTemplate+"` template and the `"+defaultSandboxSize+"` size.",
		Writer, aliasOpt("c"), displayerType(&displayers.Sandbox{}))
	AddStringFlag(cmdSandboxCreate, doctl.ArgSandboxTemplate, "", defaultSandboxTemplate,
		"The template to create the sandbox from")
	AddStringFlag(cmdSandboxCreate, doctl.ArgSizeSlug, "", defaultSandboxSize,
		"The size slug of the sandbox. It must be an enabled size")
	AddBoolFlag(cmdSandboxCreate, doctl.ArgSandboxAllowInternetAccess, "", true,
		"Let the sandbox reach the internet. Pass `--allow-internet-access=false` to limit it to the platform file-transfer host")
	AddStringArrayFlag(cmdSandboxCreate, doctl.ArgSandboxEnv, "", nil,
		"An environment variable for the sandbox, in `KEY=VALUE` form. Repeatable. The values are secrets and are never shown")
	cmdSandboxCreate.Example = "The following example creates a sandbox from the default template and size: doctl sandbox create\n\n" +
		"The following example creates a sandbox with no internet access and one environment variable: " +
		"doctl sandbox create --template coding-base --size mars-2vcpu-4gb --allow-internet-access=false --env API_KEY=secret"

	cmdSandboxList := CmdBuilder(cmd, RunSandboxList, "list",
		"List the sandboxes on your account",
		"Lists the sandboxes on your account. Use `--state` to show only running, paused or creating sandboxes.",
		Writer, aliasOpt("ls"), displayerType(&displayers.Sandbox{}))
	AddStringFlag(cmdSandboxList, doctl.ArgSandboxState, "", "",
		"Show only sandboxes in this state: `running`, `paused` or `creating`")
	cmdSandboxList.Example = "The following example lists the running sandboxes: doctl sandbox list --state running"

	cmdSandboxGet := CmdBuilder(cmd, RunSandboxGet, "get <sandbox-id>",
		"Retrieve a sandbox",
		"Retrieves the details of a sandbox: its ID, template, state, size and creation time.",
		Writer, aliasOpt("g"), displayerType(&displayers.Sandbox{}))
	cmdSandboxGet.Example = "The following example retrieves a sandbox, showing only its ID and state: " +
		"doctl sandbox get 01a0ef1a-d873-793d-ae6a-5dd6239be62a --format ID,State"

	cmdSandboxPause := CmdBuilder(cmd, RunSandboxPause, "pause <sandbox-id>",
		"Pause a sandbox",
		"Pauses a running sandbox. Resume it with `doctl sandbox resume`. An ephemeral sandbox cannot be paused.",
		Writer, displayerType(&displayers.Sandbox{}))
	cmdSandboxPause.Example = "The following example pauses a sandbox: doctl sandbox pause 01a0ef1a-d873-793d-ae6a-5dd6239be62a"

	cmdSandboxResume := CmdBuilder(cmd, RunSandboxResume, "resume <sandbox-id>",
		"Resume a paused sandbox",
		"Resumes a paused sandbox.",
		Writer, displayerType(&displayers.Sandbox{}))
	cmdSandboxResume.Example = "The following example resumes a sandbox: doctl sandbox resume 01a0ef1a-d873-793d-ae6a-5dd6239be62a"

	cmdSandboxDelete := CmdBuilder(cmd, RunSandboxDelete, "delete <sandbox-id>...",
		"Permanently delete one or more sandboxes",
		"Permanently deletes the sandboxes you name. This is irreversible.",
		Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdSandboxDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the sandbox(es) without a confirmation prompt")
	cmdSandboxDelete.Example = "The following example deletes two sandboxes without asking: " +
		"doctl sandbox delete 01a0ef1a-d873-793d-ae6a-5dd6239be62a 01a0ef1b-0000-7000-8000-000000000001 --force"

	return cmd
}

// RunSandboxCreate creates a sandbox.
func RunSandboxCreate(c *CmdConfig) error {
	if len(c.Args) != 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	template, err := c.Doit.GetString(c.NS, doctl.ArgSandboxTemplate)
	if err != nil {
		return err
	}
	if strings.TrimSpace(template) == "" {
		return doctl.NewMissingArgsErr("--template must not be empty")
	}
	size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return err
	}
	if strings.TrimSpace(size) == "" {
		return doctl.NewMissingArgsErr("--size must not be empty")
	}
	allowInternet, err := c.Doit.GetBool(c.NS, doctl.ArgSandboxAllowInternetAccess)
	if err != nil {
		return err
	}
	pairs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSandboxEnv)
	if err != nil {
		return err
	}
	env, err := parseEnvPairs(pairs)
	if err != nil {
		return err
	}

	req := &godo.SandboxCreateRequest{Template: template, SizeSlug: size}
	if !allowInternet {
		// Unset means unrestricted, so only the restricted case is sent.
		req.AllowInternetAccess = &allowInternet
	}
	if len(env) > 0 {
		req.EnvVars = env
	}

	sb, err := c.Sandboxes().Create(req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Sandbox{Sandboxes: do.Sandboxes{*sb}})
}

// RunSandboxList lists sandboxes.
func RunSandboxList(c *CmdConfig) error {
	if len(c.Args) != 0 {
		return doctl.NewTooManyArgsErr(c.NS)
	}
	flag, err := c.Doit.GetString(c.NS, doctl.ArgSandboxState)
	if err != nil {
		return err
	}
	state, err := parseSandboxState(flag)
	if err != nil {
		return err
	}
	list, err := c.Sandboxes().List(state)
	if err != nil {
		return err
	}
	return c.Display(&displayers.Sandbox{Sandboxes: list})
}

// RunSandboxGet retrieves a sandbox.
func RunSandboxGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sb, err := c.Sandboxes().Get(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.Sandbox{Sandboxes: do.Sandboxes{*sb}})
}

// RunSandboxPause pauses a sandbox.
func RunSandboxPause(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sb, err := c.Sandboxes().Pause(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.Sandbox{Sandboxes: do.Sandboxes{*sb}})
}

// RunSandboxResume resumes a sandbox.
func RunSandboxResume(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	sb, err := c.Sandboxes().Resume(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.Sandbox{Sandboxes: do.Sandboxes{*sb}})
}

// RunSandboxDelete deletes one or more sandboxes.
func RunSandboxDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if err := confirmDelete(force, "sandbox", len(c.Args)); err != nil {
		return err
	}
	svc := c.Sandboxes()
	for _, id := range c.Args {
		if err := svc.Delete(id); err != nil {
			return fmt.Errorf("deleting sandbox %s: %w", id, err)
		}
	}
	return nil
}

// parseSandboxState turns the --state flag into an API state. It accepts the
// short form (running) or the API value (SANDBOX_STATE_RUNNING), in any case.
// An empty value means any state.
func parseSandboxState(value string) (godo.SandboxState, error) {
	v := strings.ToUpper(strings.TrimSpace(value))
	if v == "" {
		return "", nil
	}
	v = strings.TrimPrefix(v, "SANDBOX_STATE_")
	switch v {
	case "RUNNING":
		return godo.SandboxStateRunning, nil
	case "PAUSED":
		return godo.SandboxStatePaused, nil
	case "CREATING":
		return godo.SandboxStateCreating, nil
	}
	return "", fmt.Errorf("invalid --state %q: expected running, paused or creating", value)
}
