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
	"fmt"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// MicroDroplet creates the microdroplet command tree.
func MicroDroplet() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "microdroplet",
			Short: "Manage MicroDroplets",
			Long: `The subcommands under ` + "`" + `doctl compute microdroplet` + "`" + ` manage MicroDroplets — lightweight ` +
				`microVM sandboxes that pause when idle and resume on demand. Use these commands to ` +
				`create, inspect, pause, resume, and delete MicroDroplets, and to manage their checkpoints.`,
			Hidden: true, // public preview: keep out of --help and generated docs until GA
		},
	}

	cmdMicroDropletList := CmdBuilder(cmd, RunMicroDropletList, "list",
		"List MicroDroplets on your account",
		"Retrieves a list of MicroDroplets on your account, optionally filtered by region.",
		Writer, aliasOpt("ls"), displayerType(&displayers.MicroDroplet{}))
	AddStringFlag(cmdMicroDropletList, doctl.ArgRegionSlug, "", "",
		"Filter MicroDroplets by region slug, such as `nyc1`")

	CmdBuilder(cmd, RunMicroDropletGet, "get <microdroplet-id>",
		"Retrieve information about a MicroDroplet",
		"Retrieves information about a MicroDroplet by its UUID.",
		Writer, aliasOpt("g"), displayerType(&displayers.MicroDroplet{}))

	cmdMicroDropletCreate := CmdBuilder(cmd, RunMicroDropletCreate, "create <microdroplet-name>",
		"Create a new MicroDroplet",
		"Creates a new MicroDroplet. Provide exactly one of `--oci-ref` or `--checkpoint-id`. "+
			"When creating from an OCI ref, `--region`, `--cpu`, and `--memory` are required. "+
			"When restoring from a checkpoint, region/size/environment may be omitted to inherit from the checkpoint.",
		Writer, aliasOpt("c"), displayerType(&displayers.MicroDroplet{}))
	AddStringFlag(cmdMicroDropletCreate, doctl.ArgRegionSlug, "", "",
		"A `slug` specifying the region to create the MicroDroplet in, such as `nyc1` (required for `--oci-ref`; optional for `--checkpoint-id`)")
	AddIntFlag(cmdMicroDropletCreate, "cpu", "", 0,
		"Number of vCPUs (required for `--oci-ref`; optional for `--checkpoint-id`)")
	AddIntFlag(cmdMicroDropletCreate, "memory", "", 0,
		"Memory in MiB (required for `--oci-ref`; optional for `--checkpoint-id`)")
	AddStringFlag(cmdMicroDropletCreate, "oci-ref", "", "",
		"OCI reference for the workload container (mutually exclusive with `--checkpoint-id`)")
	AddStringFlag(cmdMicroDropletCreate, "checkpoint-id", "", "",
		"Checkpoint UUID to restore (mutually exclusive with `--oci-ref`)")
	AddStringFlag(cmdMicroDropletCreate, "networking", "", "",
		"Networking mode for the MicroDroplet: `public` or `vpc`")
	AddStringFlag(cmdMicroDropletCreate, doctl.ArgVPCUUID, "", "",
		"The UUID of a non-default VPC to place the MicroDroplet in (only valid when `--networking=vpc`)")
	AddBoolFlag(cmdMicroDropletCreate, "auto-pause", "", false,
		"Enable auto-pause when the MicroDroplet is idle")
	AddStringFlag(cmdMicroDropletCreate, "auto-pause-idle-timeout", "", "",
		"Idle duration before auto-pause (e.g. `5m`, `30s`); requires `--auto-pause`")
	AddBoolFlag(cmdMicroDropletCreate, "auto-resume", "", false,
		"Enable auto-resume when the MicroDroplet is paused and receives traffic")
	AddIntFlag(cmdMicroDropletCreate, "http-port", "", 0,
		"HTTP port exposed by the MicroDroplet workload")
	AddStringFlag(cmdMicroDropletCreate, "http-protocol", "", "",
		"HTTP protocol served by the MicroDroplet: `http` or `http2`")
	AddStringSliceFlag(cmdMicroDropletCreate, "ports", "", []string{},
		"Guest ports to open for ingress. Repeatable. Defaults to just `--http-port` when omitted.")
	AddStringSliceFlag(cmdMicroDropletCreate, "env", "", []string{},
		"Environment variables to inject, in `KEY=VALUE` form. Repeatable.")
	AddStringSliceFlag(cmdMicroDropletCreate, doctl.ArgTag, "", []string{},
		"Tags to apply to the MicroDroplet. Repeatable.")

	CmdBuilder(cmd, RunMicroDropletPause, "pause <microdroplet-id>",
		"Pause a running MicroDroplet",
		"Requests that the MicroDroplet transition to the `paused` state. Returns the mutated MicroDroplet.",
		Writer, displayerType(&displayers.MicroDroplet{}))

	CmdBuilder(cmd, RunMicroDropletResume, "resume <microdroplet-id>",
		"Resume a paused MicroDroplet",
		"Requests that the MicroDroplet transition to the `running` state. Returns the mutated MicroDroplet.",
		Writer, displayerType(&displayers.MicroDroplet{}))

	cmdMicroDropletDelete := CmdBuilder(cmd, RunMicroDropletDelete, "delete <microdroplet-id>...",
		"Permanently delete one or more MicroDroplets",
		"Permanently deletes the specified MicroDroplets. This is irreversible.",
		Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdMicroDropletDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the MicroDroplet(s) without a confirmation prompt")

	CmdBuilder(cmd, RunMicroDropletOptions, "options",
		"List MicroDroplet create options",
		"Retrieves the regions, sizes, features, and account limits available when creating a MicroDroplet.",
		Writer, displayerType(&displayers.MicroDropletCreateOptions{}))

	cmd.AddCommand(microDropletCheckpoints())

	return cmd
}

func microDropletCheckpoints() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "checkpoint",
			Aliases: []string{"checkpoints", "cp"},
			Short:   "Manage MicroDroplet checkpoints",
			Long: `The subcommands under ` + "`" + `doctl compute microdroplet checkpoint` + "`" + ` manage ` +
				`checkpoints — persisted memory and disk state captured from a MicroDroplet.`,
		},
	}

	cmdList := CmdBuilder(cmd, RunMicroDropletCheckpointList, "list",
		"List MicroDroplet checkpoints",
		"Retrieves checkpoints for your account. Optionally filter by the MicroDroplet they were captured from.",
		Writer, aliasOpt("ls"), displayerType(&displayers.MicroDropletCheckpoint{}))
	AddStringFlag(cmdList, "microdroplet-id", "", "",
		"Filter checkpoints captured from this MicroDroplet UUID")

	CmdBuilder(cmd, RunMicroDropletCheckpointGet, "get <checkpoint-id>",
		"Retrieve a MicroDroplet checkpoint",
		"Retrieves information about a checkpoint by its UUID.",
		Writer, aliasOpt("g"), displayerType(&displayers.MicroDropletCheckpoint{}))

	cmdCreate := CmdBuilder(cmd, RunMicroDropletCheckpointCreate, "create <microdroplet-id>",
		"Create a checkpoint of a MicroDroplet",
		"Starts an asynchronous checkpoint of a running MicroDroplet.",
		Writer, aliasOpt("c"), displayerType(&displayers.MicroDropletCheckpoint{}))
	AddStringFlag(cmdCreate, "name", "", "",
		"Optional human-readable name for the checkpoint")

	cmdDelete := CmdBuilder(cmd, RunMicroDropletCheckpointDelete, "delete <checkpoint-id>...",
		"Delete one or more MicroDroplet checkpoints",
		"Releases the state stored by the specified checkpoints. This is irreversible.",
		Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the checkpoint(s) without a confirmation prompt")

	return cmd
}

// RunMicroDropletList lists MicroDroplets, optionally filtered by region.
func RunMicroDropletList(c *CmdConfig) error {
	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}

	svc := c.MicroDroplets()

	var list do.MicroDroplets
	if region != "" {
		list, err = svc.ListByRegion(region)
	} else {
		list, err = svc.List()
	}
	if err != nil {
		return err
	}

	return c.Display(&displayers.MicroDroplet{MicroDroplets: list})
}

// RunMicroDropletGet retrieves a MicroDroplet by its UUID.
func RunMicroDropletGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	md, err := c.MicroDroplets().Get(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDroplet{MicroDroplets: do.MicroDroplets{*md}})
}

// RunMicroDropletCreate creates a MicroDroplet with the provided configuration.
func RunMicroDropletCreate(c *CmdConfig) error {
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
	autoPauseEnabled, err := c.Doit.GetBool(c.NS, "auto-pause")
	if err != nil {
		return err
	}
	autoPauseIdle, err := c.Doit.GetString(c.NS, "auto-pause-idle-timeout")
	if err != nil {
		return err
	}
	autoResumeEnabled, err := c.Doit.GetBool(c.NS, "auto-resume")
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

	req := &godo.MicroDropletCreateRequest{Name: name}
	if ociRef != "" {
		if region == "" {
			return fmt.Errorf("--region is required when creating from --oci-ref")
		}
		if cpu <= 0 || memory <= 0 {
			return fmt.Errorf("--cpu and --memory are required when creating from --oci-ref")
		}
		req.Region = region
		req.Size = &godo.MicroDropletSizeRequest{CPU: uint32(cpu), Memory: uint32(memory)}
		req.Source = &godo.MicroDropletSource{OCIRef: ociRef}
	} else {
		// Checkpoint restore: leave region/size/environment unset so the API
		// inherits them (api-v2 C6), unless the caller overrides.
		req.Source = &godo.MicroDropletSource{CheckpointID: checkpointID}
		if region != "" {
			req.Region = region
		}
		if cpu > 0 || memory > 0 {
			if cpu <= 0 || memory <= 0 {
				return fmt.Errorf("--cpu and --memory must both be set when overriding size")
			}
			req.Size = &godo.MicroDropletSizeRequest{CPU: uint32(cpu), Memory: uint32(memory)}
		}
	}

	if networking != "" {
		req.Networking = godo.MicroDropletNetworking(networking)
	}
	if vpcUUID != "" {
		req.VPCUUID = vpcUUID
	}
	if autoPauseIdle != "" && !autoPauseEnabled {
		return fmt.Errorf("--auto-pause-idle-timeout requires --auto-pause")
	}
	if autoPauseEnabled {
		enabled := true
		req.AutoPause = &godo.AutoPauseConfig{
			Enabled:     &enabled,
			IdleTimeout: autoPauseIdle,
		}
	}
	if autoResumeEnabled {
		enabled := true
		req.AutoResume = &enabled
	}
	if httpPort > 0 {
		req.HTTPPort = uint32(httpPort)
	}
	if httpProtocol != "" {
		req.HTTPProtocol = godo.MicroDropletHTTPProtocol(httpProtocol)
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

	md, err := c.MicroDroplets().Create(req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDroplet{MicroDroplets: do.MicroDroplets{*md}})
}

// RunMicroDropletPause transitions a MicroDroplet to the paused state.
func RunMicroDropletPause(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	md, err := c.MicroDroplets().Pause(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDroplet{MicroDroplets: do.MicroDroplets{*md}})
}

// RunMicroDropletResume transitions a MicroDroplet to the running state.
func RunMicroDropletResume(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	md, err := c.MicroDroplets().Resume(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDroplet{MicroDroplets: do.MicroDroplets{*md}})
}

// RunMicroDropletDelete deletes one or more MicroDroplets by UUID.
func RunMicroDropletDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if !(force || AskForConfirmDelete("MicroDroplet", len(c.Args)) == nil) {
		return errOperationAborted
	}

	svc := c.MicroDroplets()
	for _, id := range c.Args {
		if err := svc.Delete(id); err != nil {
			return fmt.Errorf("Unable to delete MicroDroplet %s: %v", id, err)
		}
	}
	return nil
}

// RunMicroDropletOptions retrieves create options for MicroDroplets.
func RunMicroDropletOptions(c *CmdConfig) error {
	opts, err := c.MicroDroplets().GetCreateOptions()
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDropletCreateOptions{Options: opts})
}

// RunMicroDropletCheckpointList lists checkpoints, optionally filtered by MicroDroplet.
func RunMicroDropletCheckpointList(c *CmdConfig) error {
	microDropletID, err := c.Doit.GetString(c.NS, "microdroplet-id")
	if err != nil {
		return err
	}
	checkpoints, err := c.MicroDroplets().ListCheckpoints(microDropletID)
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDropletCheckpoint{Checkpoints: checkpoints})
}

// RunMicroDropletCheckpointGet retrieves a checkpoint by UUID.
func RunMicroDropletCheckpointGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	cp, err := c.MicroDroplets().GetCheckpoint(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDropletCheckpoint{Checkpoints: do.MicroDropletCheckpoints{*cp}})
}

// RunMicroDropletCheckpointCreate starts a checkpoint of a MicroDroplet.
func RunMicroDropletCheckpointCreate(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	name, err := c.Doit.GetString(c.NS, "name")
	if err != nil {
		return err
	}
	req := &godo.MicroDropletCheckpointCreateRequest{}
	if name != "" {
		req.Name = name
	}
	cp, err := c.MicroDroplets().CreateCheckpoint(c.Args[0], req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDropletCheckpoint{Checkpoints: do.MicroDropletCheckpoints{*cp}})
}

// RunMicroDropletCheckpointDelete deletes one or more checkpoints.
func RunMicroDropletCheckpointDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if !(force || AskForConfirmDelete("MicroDroplet checkpoint", len(c.Args)) == nil) {
		return errOperationAborted
	}

	svc := c.MicroDroplets()
	for _, id := range c.Args {
		if err := svc.DeleteCheckpoint(id); err != nil {
			return fmt.Errorf("Unable to delete checkpoint %s: %v", id, err)
		}
	}
	return nil
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
