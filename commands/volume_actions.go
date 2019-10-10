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
			Short: "Provides commands to perform actions on Block Storage volumes",
			Long:  `Block Storage volume actions are commands that can be given to a DigitalOcean Block Storage volume. 

An example would be detaching or attaching a volume from a Droplet.`,
		},
	}

	cmdRunVolumeAttach := CmdBuilderWithDocs(cmd, RunVolumeAttach, "attach <volume-id> <droplet-id>", "attach a volume", `Use this command to attach a Block Storage volume to a Droplet. 

Each volume may only be attached to a single Droplet. However, up to five volumes may be attached to a Droplet at a time. 
Pre-formatted volumes will be automatically mounted to Ubuntu, Debian, Fedora, Fedora Atomic, and CentOS Droplets created on or after April 26, 2018 when attached. On older Droplets, additional configuration is required. Visit https://www.digitalocean.com/docs/volumes/how-to/format-and-mount/#mounting-the-filesystems for details`, Writer,
		aliasOpt("a"))
	AddBoolFlag(cmdRunVolumeAttach, doctl.ArgCommandWait, "", false, "Wait for volume to attach")

	cmdRunVolumeDetach := CmdBuilderWithDocs(cmd, RunVolumeDetach, "detach <volume-id> <droplet-id>", "detach a volume", `Use this command to detach a Block Storage volume from a Droplet.`, Writer,
		aliasOpt("d"))
	AddBoolFlag(cmdRunVolumeDetach, doctl.ArgCommandWait, "", false, "Wait for volume to detach")

	CmdBuilder(cmd, RunVolumeDetach, "detach-by-droplet-id <volume-id> <droplet-id>", "detach a volume (deprecated - use detach instead)",
		Writer)

	cmdRunVolumeResize := CmdBuilderWithDocs(cmd, RunVolumeResize, "resize <volume-id>", "resize a volume",`Use this command to resize a Block Storage volume. 
 
Volumes may only be resized upwards. The maximum size for a volume is 16TiB.`, Writer,
		aliasOpt("r"))
	AddIntFlag(cmdRunVolumeResize, doctl.ArgSizeSlug, "", 0, "New size",
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
