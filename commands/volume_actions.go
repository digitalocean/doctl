/*
Copyright 2016 The Doctl Authors All rights reserved.
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

	item := &action{actions: do.Actions{*a}}
	return c.Display(item)
}

// VolumeAction creates the volume command
func VolumeAction() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "volume-action",
			Short: "volume action commands",
			Long:  "volume-action is used to access volume action commands",
		},
	}

	CmdBuilder(cmd, RunVolumeAttach, "attach <volume-id> <droplet-id>", "attach a volume", Writer,
		aliasOpt("a"))

	CmdBuilder(cmd, RunVolumeDetachByDropletID, "detach-by-droplet-id <volume-id> <droplet-id>", "detach a volume by droplet ID", Writer,
		aliasOpt("dd"))

	cmdRunVolumeResize := CmdBuilder(cmd, RunVolumeResize, "resize <volume-id>", "resize a volume", Writer,
		aliasOpt("r"))
	AddIntFlag(cmdRunVolumeResize, doctl.ArgSizeSlug, "", 0, "New size",
		requiredOpt())
	AddStringFlag(cmdRunVolumeResize, doctl.ArgRegionSlug, "", "", "Volume region",
		requiredOpt())

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

// RunVolumeDetachByDropletID detaches a volume by droplet ID
func RunVolumeDetachByDropletID(c *CmdConfig) error {
	fn := func(das do.VolumeActionsService) (*do.Action, error) {
		if len(c.Args) != 2 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		volumeID := c.Args[0]
		dropletID, err := strconv.Atoi(c.Args[1])
		if err != nil {
			return nil, err
		}
		a, err := das.DetachByDropletID(volumeID, dropletID)
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
