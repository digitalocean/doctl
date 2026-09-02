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
				`create, inspect, pause, resume, and delete MicroDroplets, and to list their checkpoints.`,
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
		"Creates a new MicroDroplet in the specified region from the specified image. "+
			"`--region`, `--size`, and `--image` are required. The MicroDroplet UUID and current state are returned on success.",
		Writer, aliasOpt("c"), displayerType(&displayers.MicroDroplet{}))
	AddStringFlag(cmdMicroDropletCreate, doctl.ArgRegionSlug, "", "",
		"A `slug` specifying the region to create the MicroDroplet in, such as `nyc1`", requiredOpt())
	AddStringFlag(cmdMicroDropletCreate, doctl.ArgSizeSlug, "", "",
		"A `slug` indicating the MicroDroplet's size", requiredOpt())
	AddStringFlag(cmdMicroDropletCreate, doctl.ArgImage, "", "",
		"The URN or UUID of a MicroDroplet image to launch", requiredOpt())
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

	CmdBuilder(cmd, RunMicroDropletCheckpoints, "checkpoints <microdroplet-id>",
		"List checkpoints for a MicroDroplet",
		"Retrieves a list of checkpoints for the specified MicroDroplet. "+
			"Checkpoints are captured automatically by DigitalOcean when a MicroDroplet is paused; "+
			"each one preserves the memory and disk state required to resume.",
		Writer, aliasOpt("cp"), displayerType(&displayers.MicroDropletCheckpoint{}))

	cmd.AddCommand(microDropletImages())

	return cmd
}

func microDropletImages() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "image",
			Short: "Manage MicroDroplet images",
			Long: `The subcommands under ` + "`" + `doctl compute microdroplet image` + "`" + ` manage OCI images ` +
				`imported for use with MicroDroplets.`,
		},
	}

	CmdBuilder(cmd, RunMicroDropletImageList, "list",
		"List MicroDroplet images on your account",
		"Retrieves a list of MicroDroplet images on your account.",
		Writer, aliasOpt("ls"), displayerType(&displayers.MicroDropletImage{}))

	CmdBuilder(cmd, RunMicroDropletImageGet, "get <image-id>",
		"Retrieve information about a MicroDroplet image",
		"Retrieves information about a MicroDroplet image by its UUID.",
		Writer, aliasOpt("g"), displayerType(&displayers.MicroDropletImage{}))

	cmdImageCreate := CmdBuilder(cmd, RunMicroDropletImageCreate, "create <image-name>",
		"Import a new MicroDroplet image",
		"Imports a new MicroDroplet image from a public OCI ref or a DOCR ref. Import is asynchronous.",
		Writer, aliasOpt("c"), displayerType(&displayers.MicroDropletImage{}))
	AddStringFlag(cmdImageCreate, "source", "", "",
		"The OCI or DOCR source ref for the image", requiredOpt())

	cmdImageDelete := CmdBuilder(cmd, RunMicroDropletImageDelete, "delete <image-id>...",
		"Permanently delete one or more MicroDroplet images",
		"Permanently deletes the specified MicroDroplet images. This is irreversible.",
		Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdImageDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the image(s) without a confirmation prompt")

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

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}
	size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
	if err != nil {
		return err
	}
	image, err := c.Doit.GetString(c.NS, doctl.ArgImage)
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
	envPairs, err := c.Doit.GetStringSlice(c.NS, "env")
	if err != nil {
		return err
	}
	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return err
	}

	req := &godo.MicroDropletCreateRequest{
		Name:   name,
		Region: region,
		Size:   size,
		Image:  image,
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

// RunMicroDropletCheckpoints lists checkpoints for a MicroDroplet.
func RunMicroDropletCheckpoints(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	checkpoints, err := c.MicroDroplets().ListCheckpoints(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDropletCheckpoint{Checkpoints: checkpoints})
}

// RunMicroDropletImageList lists MicroDroplet images.
func RunMicroDropletImageList(c *CmdConfig) error {
	imgs, err := c.MicroDropletImages().List()
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDropletImage{Images: imgs})
}

// RunMicroDropletImageGet retrieves a MicroDroplet image by UUID.
func RunMicroDropletImageGet(c *CmdConfig) error {
	if err := ensureOneArg(c); err != nil {
		return err
	}
	img, err := c.MicroDropletImages().Get(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDropletImage{Images: do.MicroDropletImages{*img}})
}

// RunMicroDropletImageCreate imports a MicroDroplet image from an OCI ref.
func RunMicroDropletImageCreate(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	source, err := c.Doit.GetString(c.NS, "source")
	if err != nil {
		return err
	}
	img, err := c.MicroDropletImages().Create(&godo.MicroDropletImageCreateRequest{
		Name:   c.Args[0],
		Source: source,
	})
	if err != nil {
		return err
	}
	return c.Display(&displayers.MicroDropletImage{Images: do.MicroDropletImages{*img}})
}

// RunMicroDropletImageDelete deletes one or more MicroDroplet images.
func RunMicroDropletImageDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if !(force || AskForConfirmDelete("MicroDroplet image", len(c.Args)) == nil) {
		return errOperationAborted
	}

	svc := c.MicroDropletImages()
	for _, id := range c.Args {
		if err := svc.Delete(id); err != nil {
			return fmt.Errorf("Unable to delete MicroDroplet image %s: %v", id, err)
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
