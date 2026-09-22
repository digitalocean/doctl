/*
Copyright 2025 The Doctl Authors All rights reserved.
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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/doctl/pkg/terminal"
	"github.com/digitalocean/godo"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

// MicroVM creates the microvm command tree.
func MicroVM() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "microvm",
			Short: "Manage MicroVMs",
			Long: `The subcommands under ` + "`" + `doctl compute microvm` + "`" + ` manage MicroVMs — lightweight ` +
				`microVM sandboxes that pause when idle and resume on demand. Use these commands to ` +
				`create, inspect, pause, resume, delete, exec into, and console into MicroVMs, and to manage their checkpoints.`,
			Hidden: true, // public preview: keep out of --help and generated docs until GA
		},
	}

	cmdMicroVMList := CmdBuilder(cmd, RunMicroVMList, "list",
		"List MicroVMs on your account",
		"Retrieves a list of MicroVMs on your account. Filters combine: `--region`, `--name`, and `--tag-name` are ANDed.",
		Writer, aliasOpt("ls"), displayerType(&displayers.MicroVM{}))
	AddStringFlag(cmdMicroVMList, doctl.ArgRegionSlug, "", "",
		"Filter MicroVMs by region slug, such as `nyc1`")
	AddStringFlag(cmdMicroVMList, "name", "", "",
		"Filter MicroVMs by exact name")
	AddStringFlag(cmdMicroVMList, doctl.ArgTagName, "", "",
		"Filter MicroVMs by resource tag name. A tag that matches nothing returns an empty list")

	CmdBuilder(cmd, RunMicroVMGet, "get <microvm-id>",
		"Retrieve information about a MicroVM",
		"Retrieves information about a MicroVM by its UUID.",
		Writer, aliasOpt("g"), displayerType(&displayers.MicroVM{}))

	cmdMicroVMCreate := CmdBuilder(cmd, RunMicroVMCreate, "create <microvm-name>",
		"Create a new MicroVM",
		"Creates a new MicroVM. Provide exactly one of `--oci-ref` or `--checkpoint-id`. "+
			"When creating from an OCI ref, `--region`, `--cpu`, and `--memory` are required. "+
			"When restoring from a checkpoint, region/size/environment may be omitted to inherit from the checkpoint.",
		Writer, aliasOpt("c"), displayerType(&displayers.MicroVM{}))
	AddStringFlag(cmdMicroVMCreate, doctl.ArgRegionSlug, "", "",
		"A `slug` specifying the region to create the MicroVM in, such as `nyc1` (required for `--oci-ref`; optional for `--checkpoint-id`)")
	AddIntFlag(cmdMicroVMCreate, "cpu", "", 0,
		"Number of vCPUs (required for `--oci-ref`; optional for `--checkpoint-id`)")
	AddIntFlag(cmdMicroVMCreate, "memory", "", 0,
		"Memory in MiB (required for `--oci-ref`; optional for `--checkpoint-id`)")
	AddStringFlag(cmdMicroVMCreate, "oci-ref", "", "",
		"OCI reference for the workload container (mutually exclusive with `--checkpoint-id`)")
	AddStringFlag(cmdMicroVMCreate, "checkpoint-id", "", "",
		"Checkpoint UUID to restore (mutually exclusive with `--oci-ref`)")
	AddStringFlag(cmdMicroVMCreate, "networking", "", "",
		"Networking mode for the MicroVM: `public` or `vpc`")
	AddStringFlag(cmdMicroVMCreate, doctl.ArgVPCUUID, "", "",
		"The UUID of a non-default VPC to place the MicroVM in (only valid when `--networking=vpc`)")
	AddBoolFlag(cmdMicroVMCreate, "auto-pause", "", false,
		"Whether the MicroVM auto-pauses after the idle timeout. Omit to keep the product default (enabled). Pass `--auto-pause=false` to disable")
	AddStringFlag(cmdMicroVMCreate, "auto-pause-idle-timeout", "", "",
		"Idle duration before auto-pause (e.g. `5m`, `30s`). Can be set without `--auto-pause`")
	AddBoolFlag(cmdMicroVMCreate, "auto-resume", "", false,
		"Whether the MicroVM auto-resumes on incoming HTTP traffic. Omit to keep the default (enabled). Pass `--auto-resume=false` to require an explicit resume")
	AddIntFlag(cmdMicroVMCreate, "http-port", "", 0,
		"HTTP port exposed by the MicroVM workload")
	AddStringFlag(cmdMicroVMCreate, "http-protocol", "", "",
		"HTTP protocol served by the MicroVM: `http` or `http2`")
	AddStringSliceFlag(cmdMicroVMCreate, "ports", "", []string{},
		"Guest ports to open for ingress. Repeatable. Defaults to just `--http-port` when omitted.")
	AddStringSliceFlag(cmdMicroVMCreate, "env", "", []string{},
		"Environment variables to inject, in `KEY=VALUE` form. Repeatable.")
	AddStringSliceFlag(cmdMicroVMCreate, doctl.ArgTag, "", []string{},
		"Tags to apply to the MicroVM. Repeatable.")

	CmdBuilder(cmd, RunMicroVMPause, "pause <microvm-id>",
		"Pause a running MicroVM",
		"Requests that the MicroVM transition to the `paused` state. Returns the mutated MicroVM.",
		Writer, displayerType(&displayers.MicroVM{}))

	CmdBuilder(cmd, RunMicroVMResume, "resume <microvm-id>",
		"Resume a paused MicroVM",
		"Requests that the MicroVM transition to the `running` state. Returns the mutated MicroVM.",
		Writer, displayerType(&displayers.MicroVM{}))

	cmdMicroVMDelete := CmdBuilder(cmd, RunMicroVMDelete, "delete <microvm-id>...",
		"Permanently delete one or more MicroVMs",
		"Permanently deletes the specified MicroVMs. This is irreversible.",
		Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdMicroVMDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the MicroVM(s) without a confirmation prompt")

	CmdBuilder(cmd, RunMicroVMOptions, "options",
		"List MicroVM create options",
		"Retrieves the sizes (with available regions), features, and account limits available when creating a MicroVM.",
		Writer, displayerType(&displayers.MicroVMCreateOptions{}))

	cmdMicroVMExec := CmdBuilder(cmd, RunMicroVMExec, "exec <microvm-id> -- <command> [args...]",
		"Run a one-shot command in a MicroVM",
		"Runs a one-shot, non-PTY command in the MicroVM's workload container and prints stdout/stderr. "+
			"Requires the `exec_pty` feature (see `doctl compute microvm options`). A paused MicroVM is auto-resumed. "+
			"A non-zero guest exit code makes doctl exit non-zero.",
		Writer)
	AddStringFlag(cmdMicroVMExec, "cwd", "", "",
		"Working directory inside the workload container")

	CmdBuilder(cmd, RunMicroVMConsole, "console <microvm-id>",
		"Open an interactive console to a MicroVM",
		"Opens an interactive PTY console to the MicroVM's workload container over WebSocket. "+
			"Requires the `exec_pty` feature (see `doctl compute microvm options`). A paused MicroVM is auto-resumed.",
		Writer)

	cmd.AddCommand(microVMCheckpoints())

	return cmd
}

func microVMCheckpoints() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "checkpoint",
			Aliases: []string{"checkpoints", "cp"},
			Short:   "Manage MicroVM checkpoints",
			Long: `The subcommands under ` + "`" + `doctl compute microvm checkpoint` + "`" + ` manage ` +
				`checkpoints — persisted memory and disk state captured from a MicroVM.`,
		},
	}

	cmdList := CmdBuilder(cmd, RunMicroVMCheckpointList, "list",
		"List MicroVM checkpoints",
		"Retrieves checkpoints for your account. Optionally filter by the MicroVM they were captured from.",
		Writer, aliasOpt("ls"), displayerType(&displayers.MicroVMCheckpoint{}))
	AddStringFlag(cmdList, "microvm-id", "", "",
		"Filter checkpoints captured from this MicroVM UUID")

	CmdBuilder(cmd, RunMicroVMCheckpointGet, "get <checkpoint-id>",
		"Retrieve a MicroVM checkpoint",
		"Retrieves information about a checkpoint by its UUID.",
		Writer, aliasOpt("g"), displayerType(&displayers.MicroVMCheckpoint{}))

	cmdCreate := CmdBuilder(cmd, RunMicroVMCheckpointCreate, "create <microvm-id>",
		"Create a checkpoint of a MicroVM",
		"Starts an asynchronous checkpoint of a running MicroVM.",
		Writer, aliasOpt("c"), displayerType(&displayers.MicroVMCheckpoint{}))
	AddStringFlag(cmdCreate, "name", "", "",
		"Optional human-readable name for the checkpoint")

	cmdDelete := CmdBuilder(cmd, RunMicroVMCheckpointDelete, "delete <checkpoint-id>...",
		"Delete one or more MicroVM checkpoints",
		"Releases the state stored by the specified checkpoints. This is irreversible.",
		Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the checkpoint(s) without a confirmation prompt")

	return cmd
}

// RunMicroVMList lists MicroVMs, optionally filtered by region, name, and tag.
func RunMicroVMList(c *CmdConfig) error {
	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, "name")
	if err != nil {
		return err
	}
	tagName, err := c.Doit.GetString(c.NS, doctl.ArgTagName)
	if err != nil {
		return err
	}

	list, err := c.MicroVMs().List(do.MicroVMListFilter{
		Region:  region,
		Name:    name,
		TagName: tagName,
	})
	if err != nil {
		return err
	}

	return c.Display(&displayers.MicroVM{MicroVMs: list})
}

// RunMicroVMGet retrieves a MicroVM by its UUID.
func RunMicroVMGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	md, err := c.MicroVMs().Get(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroVM{MicroVMs: do.MicroVMs{*md}})
}

// RunMicroVMCreate creates a MicroVM with the provided configuration.
func RunMicroVMCreate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	name := c.Args[0]

	ociRef, err := c.Doit.GetString(c.NS, "oci-ref")
	if err != nil {
		return err
	}
	checkpointID, err := c.Doit.GetString(c.NS, "checkpoint-id")
	if err != nil {
		return err
	}
	if (ociRef == "") == (checkpointID == "") {
		return fmt.Errorf("exactly one of --oci-ref or --checkpoint-id is required")
	}

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	cpu, err := c.Doit.GetInt(c.NS, "cpu")
	if err != nil {
		return err
	}
	memory, err := c.Doit.GetInt(c.NS, "memory")
	if err != nil {
		return err
	}
	networking, err := c.Doit.GetString(c.NS, "networking")
	if err != nil {
		return err
	}
	vpcUUID, err := c.Doit.GetString(c.NS, doctl.ArgVPCUUID)
	if err != nil {
		return err
	}
	autoPause, err := c.Doit.GetBoolPtr(c.NS, "auto-pause")
	if err != nil {
		return err
	}
	autoPauseIdle, err := c.Doit.GetString(c.NS, "auto-pause-idle-timeout")
	if err != nil {
		return err
	}
	autoResume, err := c.Doit.GetBoolPtr(c.NS, "auto-resume")
	if err != nil {
		return err
	}
	httpPort, err := c.Doit.GetInt(c.NS, "http-port")
	if err != nil {
		return err
	}
	httpProtocol, err := c.Doit.GetString(c.NS, "http-protocol")
	if err != nil {
		return err
	}
	portStrs, err := c.Doit.GetStringSlice(c.NS, "ports")
	if err != nil {
		return err
	}
	envPairs, err := c.Doit.GetStringSlice(c.NS, "env")
	if err != nil {
		return err
	}
	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return err
	}

	req := &godo.MicroVMCreateRequest{Name: name}
	if ociRef != "" {
		if region == "" {
			return fmt.Errorf("--region is required when creating from --oci-ref")
		}
		if cpu <= 0 || memory <= 0 {
			return fmt.Errorf("--cpu and --memory are required when creating from --oci-ref")
		}
		req.Region = region
		req.Size = &godo.MicroVMSizeRequest{CPU: uint32(cpu), Memory: uint32(memory)}
		req.Source = &godo.MicroVMSource{OCIRef: ociRef}
	} else {
		// Checkpoint restore: leave region/size/environment unset so the API
		// inherits them (api-v2 C6), unless the caller overrides.
		req.Source = &godo.MicroVMSource{CheckpointID: checkpointID}
		if region != "" {
			req.Region = region
		}
		if cpu > 0 || memory > 0 {
			if cpu <= 0 || memory <= 0 {
				return fmt.Errorf("--cpu and --memory must both be set when overriding size")
			}
			req.Size = &godo.MicroVMSizeRequest{CPU: uint32(cpu), Memory: uint32(memory)}
		}
	}

	if networking != "" {
		req.Networking = godo.MicroVMNetworking(networking)
	}
	if vpcUUID != "" {
		req.VPCUUID = vpcUUID
	}
	// Omit auto_pause entirely unless the caller set enabled or a timeout.
	// An explicit false disables auto-pause; a timeout alone leaves the
	// product default (enabled) in place.
	if autoPause != nil || autoPauseIdle != "" {
		req.AutoPause = &godo.AutoPauseConfig{
			Enabled:     autoPause,
			IdleTimeout: autoPauseIdle,
		}
	}
	if autoResume != nil {
		req.AutoResume = autoResume
	}
	if httpPort > 0 {
		req.HTTPPort = uint32(httpPort)
	}
	if httpProtocol != "" {
		req.HTTPProtocol = godo.MicroVMHTTPProtocol(httpProtocol)
	}
	if len(portStrs) > 0 {
		ports, err := parsePorts(portStrs)
		if err != nil {
			return err
		}
		req.Ports = ports
	}
	if len(envPairs) > 0 {
		env, err := parseEnvPairs(envPairs)
		if err != nil {
			return err
		}
		req.Environment = env
	}
	if len(tags) > 0 {
		req.Tags = tags
	}

	md, err := c.MicroVMs().Create(req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroVM{MicroVMs: do.MicroVMs{*md}})
}

// RunMicroVMPause transitions a MicroVM to the paused state.
func RunMicroVMPause(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	md, err := c.MicroVMs().Pause(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroVM{MicroVMs: do.MicroVMs{*md}})
}

// RunMicroVMResume transitions a MicroVM to the running state.
func RunMicroVMResume(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	md, err := c.MicroVMs().Resume(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroVM{MicroVMs: do.MicroVMs{*md}})
}

// RunMicroVMDelete deletes one or more MicroVMs by UUID.
func RunMicroVMDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if !(force || AskForConfirmDelete("MicroVM", len(c.Args)) == nil) {
		return errOperationAborted
	}

	svc := c.MicroVMs()
	for _, id := range c.Args {
		if err := svc.Delete(id); err != nil {
			return fmt.Errorf("Unable to delete MicroVM %s: %v", id, err)
		}
	}
	return nil
}

// RunMicroVMOptions retrieves create options for MicroVMs.
func RunMicroVMOptions(c *CmdConfig) error {
	opts, err := c.MicroVMs().GetCreateOptions()
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroVMCreateOptions{Options: opts})
}

// RunMicroVMCheckpointList lists checkpoints, optionally filtered by MicroVM.
func RunMicroVMCheckpointList(c *CmdConfig) error {
	microVMID, err := c.Doit.GetString(c.NS, "microvm-id")
	if err != nil {
		return err
	}
	checkpoints, err := c.MicroVMs().ListCheckpoints(microVMID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroVMCheckpoint{Checkpoints: checkpoints})
}

// RunMicroVMCheckpointGet retrieves a checkpoint by UUID.
func RunMicroVMCheckpointGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	cp, err := c.MicroVMs().GetCheckpoint(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroVMCheckpoint{Checkpoints: do.MicroVMCheckpoints{*cp}})
}

// RunMicroVMCheckpointCreate starts a checkpoint of a MicroVM.
func RunMicroVMCheckpointCreate(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, "name")
	if err != nil {
		return err
	}
	req := &godo.MicroVMCheckpointCreateRequest{}
	if name != "" {
		req.Name = name
	}
	cp, err := c.MicroVMs().CreateCheckpoint(c.Args[0], req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroVMCheckpoint{Checkpoints: do.MicroVMCheckpoints{*cp}})
}

// RunMicroVMCheckpointDelete deletes one or more checkpoints.
func RunMicroVMCheckpointDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if !(force || AskForConfirmDelete("MicroVM checkpoint", len(c.Args)) == nil) {
		return errOperationAborted
	}

	svc := c.MicroVMs()
	for _, id := range c.Args {
		if err := svc.DeleteCheckpoint(id); err != nil {
			return fmt.Errorf("Unable to delete checkpoint %s: %v", id, err)
		}
	}
	return nil
}

// RunMicroVMExec runs a one-shot command in a MicroVM workload container.
func RunMicroVMExec(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]
	argv := c.Args[1:]

	cwd, err := c.Doit.GetString(c.NS, "cwd")
	if err != nil {
		return err
	}

	result, err := c.MicroVMs().Exec(id, &godo.MicroVMExecRequest{Argv: argv, Cwd: cwd})
	if err != nil {
		return err
	}

	if result.Stdout != "" {
		fmt.Fprint(c.Out, result.Stdout)
	}
	if result.Stderr != "" {
		fmt.Fprint(os.Stderr, result.Stderr)
	}
	if result.Truncated {
		fmt.Fprintln(os.Stderr, "warning: exec output was truncated by the server")
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("command exited with code %d", result.ExitCode)
	}
	return nil
}

// RunMicroVMConsole opens an interactive PTY console to a MicroVM.
func RunMicroVMConsole(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	id := c.Args[0]

	opt := &godo.MicroVMConsoleOptions{}
	if size := terminalSize(); size != nil {
		opt.Rows = uint32(size.Height)
		opt.Cols = uint32(size.Width)
	}

	wsURL, err := c.MicroVMs().ConsoleURL(id, opt)
	if err != nil {
		return err
	}
	u, err := url.Parse(wsURL)
	if err != nil {
		return err
	}

	token := c.getContextAccessToken()
	if token == "" {
		return fmt.Errorf("access token is required for MicroVM console")
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return fmt.Errorf("error creating websocket connection: %w", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	term := c.Doit.Terminal()
	stdinCh := make(chan string)
	restoreTerminal, err := term.ReadRawStdin(ctx, stdinCh)
	if err != nil {
		return err
	}
	defer restoreTerminal()

	resizeEvents := make(chan terminal.TerminalSize)
	grp, ctx := errgroup.WithContext(ctx)

	grp.Go(func() error {
		return term.MonitorResizeEvents(ctx, resizeEvents)
	})

	grp.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case in := <-stdinCh:
				if err := conn.WriteMessage(websocket.BinaryMessage, []byte(in)); err != nil {
					return fmt.Errorf("error writing stdin: %w", err)
				}
			case ev := <-resizeEvents:
				payload, err := godo.MarshalMicroVMConsoleResize(uint32(ev.Height), uint32(ev.Width))
				if err != nil {
					return fmt.Errorf("error encoding resize: %w", err)
				}
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
					return fmt.Errorf("error writing resize: %w", err)
				}
			}
		}
	})

	grp.Go(func() error {
		defer cancel()
		for {
			msgType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return nil
				}
				return fmt.Errorf("error reading from websocket: %w", err)
			}
			switch msgType {
			case websocket.BinaryMessage:
				if _, err := c.Out.Write(message); err != nil {
					return err
				}
			case websocket.TextMessage:
				ctrl, err := godo.ParseMicroVMConsoleControl(message)
				if err != nil {
					// ignore unrecognized control frames
					continue
				}
				if ctrl.Error != nil {
					return fmt.Errorf("console error (%s): %s", ctrl.Error.Code, ctrl.Error.Message)
				}
				if ctrl.Exit != nil {
					if ctrl.Exit.Code != 0 {
						return fmt.Errorf("console exited with code %d", ctrl.Exit.Code)
					}
					return nil
				}
				// status frames (e.g. resuming) are informational; keep the session open
			}
		}
	})

	return grp.Wait()
}

func terminalSize() *terminal.TerminalSize {
	// Best-effort initial size for ConsoleURL ?rows=&cols=. Resize events
	// update the PTY after connect; failing here just uses server defaults.
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		return nil
	}
	return &terminal.TerminalSize{Width: w, Height: h}
}

// parseEnvPairs turns "KEY=VALUE" pairs into a map, returning an error on
// malformed entries.
func parseEnvPairs(pairs []string) (map[string]string, error) {
	out := make(map[string]string, len(pairs))
	for _, p := range pairs {
		key, value, ok := strings.Cut(p, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --env value %q: expected KEY=VALUE", p)
		}
		out[key] = value
	}
	return out, nil
}

func parsePorts(portStrs []string) ([]uint32, error) {
	ports := make([]uint32, 0, len(portStrs))
	for _, s := range portStrs {
		n, err := strconv.ParseUint(s, 10, 32)
		if err != nil || n == 0 || n > 65535 {
			return nil, fmt.Errorf("invalid --ports value %q: expected an integer 1-65535", s)
		}
		ports = append(ports, uint32(n))
	}
	return ports, nil
}
