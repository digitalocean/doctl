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
	"strconv"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

type volumeActionFn func(das do.VolumeActionsService) (*do.Action, error)

func performVolumeAction(c *CmdConfig, fn volumeActionFn) error {
	das := c.VolumeActions()

	a, err := fn(das)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		a, err = actionWait(c, a.ID, 5)
		if err != nil {
			return err
		}

	}

	item := &displayers.Action{Actions: do.Actions{*a}}
	return c.Display(item)
}

// VolumeAction creates the volume command
func VolumeAction() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "volume-action",
			Short: "Display commands to perform actions on a volume",
			Long:  "Block storage volume action commands allow you to attach, detach, and resize existing volumes.",
		},
	}

	cmdRunVolumeAttach := CmdBuilder(cmd, RunVolumeAttach, "attach <volume-id> <droplet-id>", "Attach a volume to a Droplet", `Use this command to attach a block storage volume to a Droplet.

Each volume can be attached to only one Droplet at a time. However, up to five volumes may be attached to a single Droplet.

When you attach a pre-formatted volume to Ubuntu, Debian, Fedora, Fedora Atomic, and CentOS Droplets created on or after April 26, 2018, the volume will be automatically mounted. On older Droplets, additional configuration is required. Visit https://www.digitalocean.com/docs/volumes/how-to/format-and-mount/#mounting-the-filesystems for details`, Writer,
		aliasOpt("a"))
	AddBoolFlag(cmdRunVolumeAttach, doctl.ArgCommandWait, "", false, "Wait for volume to attach")

	cmdRunVolumeDetach := CmdBuilder(cmd, RunVolumeDetach, "detach <volume-id> <droplet-id>", "Detach a volume from a Droplet", `Use this command to detach a block storage volume from a Droplet.`, Writer,
		aliasOpt("d"))
	AddBoolFlag(cmdRunVolumeDetach, doctl.ArgCommandWait, "", false, "Wait for volume to detach")

	CmdBuilder(cmd, RunVolumeDetach, "detach-by-droplet-id <volume-id> <droplet-id>", "(Deprecated) Detach a volume. Use `detach` instead.", "This command detaches a volume. This command is deprecated. Use `doctl compute volume-action detach` instead.",
		Writer)

	cmdRunVolumeResize := CmdBuilder(cmd, RunVolumeResize, "resize <volume-id>", "Resize the disk of a volume", `Use this command to resize a block storage volume.

Volumes may only be resized upwards. The maximum size for a volume is 16TiB.`, Writer,
		aliasOpt("r"))
	AddIntFlag(cmdRunVolumeResize, doctl.ArgSizeSlug, "", 0, "New size in GiB",
		requiredOpt())
	AddStringFlag(cmdRunVolumeResize, doctl.ArgRegionSlug, "", "", "Volume region",
		requiredOpt())
	AddBoolFlag(cmdRunVolumeResize, doctl.ArgCommandWait, "", false, "Wait for volume to resize")

	return cmd

}

// RunVolumeAttach attaches a volume to a droplet.
func RunVolumeAttach(c *CmdConfig) error {
	fn := func(das do.VolumeActionsService) (*do.Action, error) {
		if len(c.Args) != 2 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		volumeID := c.Args[0]
		dropletID, err := strconv.Atoi(c.Args[1])
		if err != nil {
			return nil, err

		}
		a, err := das.Attach(volumeID, dropletID)
		return a, err
	}
	return performVolumeAction(c, fn)
}

// RunVolumeDetach detaches a volume by droplet ID
func RunVolumeDetach(c *CmdConfig) error {
	fn := func(das do.VolumeActionsService) (*do.Action, error) {
		if len(c.Args) != 2 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		volumeID := c.Args[0]
		dropletID, err := strconv.Atoi(c.Args[1])
		if err != nil {
			return nil, err
		}
		a, err := das.Detach(volumeID, dropletID)
		return a, err
	}
	return performVolumeAction(c, fn)
}

// RunVolumeResize resizes a volume
func RunVolumeResize(c *CmdConfig) error {
	fn := func(das do.VolumeActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}

		volumeID := c.Args[0]

		size, err := c.Doit.GetInt(c.NS, doctl.ArgSizeSlug)
		if err != nil {
			return nil, err
		}

		region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
		if err != nil {
			return nil, err
		}

		a, err := das.Resize(volumeID, size, region)
		return a, err
	}
	return performVolumeAction(c, fn)
}
