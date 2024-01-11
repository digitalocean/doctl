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

	actionDetail := `

- The unique numeric ID used to identify and reference a volume action.
- The status of the volume action. Possible values: ` + "`" + `in-progress` + "`" + `, ` + "`" + `completed` + "`" + `, ` + "`" + `errored` + "`" + `.
- When the action was initiated, in ISO8601 combined date and time format
- When the action was completed, in ISO8601 combined date and time format
- The resource ID, which is a unique identifier for the resource that the action is associated with.
- The type of resource that the action is associated with.
- The region where the action occurred.
- The slug for the region where the action occurred.
	`

	cmdVolumeActionsGet := CmdBuilder(cmd, RunVolumeActionsGet, "get <volume-id>", "Retrieve the status of a volume action", `Retrieves the status of a volume action, including the following details:`+actionDetail, Writer,
		displayerType(&displayers.Action{}))
	AddIntFlag(cmdVolumeActionsGet, doctl.ArgActionID, "", 0, "action id", requiredOpt())
	cmdVolumeActionsGet.Example = `The following example retrieves the status of an action taken on a volume with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute volume-action get f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --action-id 191669331`

	cmdVolumeActionsList := CmdBuilder(cmd, RunVolumeActionsList, "list <volume-id>", "Retrieve a list of actions taken on a volume", `Retrieves a list of actions taken on a volume. The following details are provided:`+actionDetail, Writer,
		aliasOpt("ls"), displayerType(&displayers.Action{}))
	cmdVolumeActionsList.Example = `The following example retrieves a list of actions taken on a volume with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `. The command also uses the ` + "`" + `--format` + "`" + ` flag to return ony the resource ID and status for each action listed: doctl compute volume-action list f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --format ResourceID,Status`

	cmdRunVolumeAttach := CmdBuilder(cmd, RunVolumeAttach, "attach <volume-id> <droplet-id>", "Attach a volume to a Droplet", `Attaches a block storage volume to a Droplet.

You can only attach one Droplet to a volume at a time. However, you can attach up to five different volumes to a Droplet at a time.

When you attach a pre-formatted volume to Ubuntu, Debian, Fedora, Fedora Atomic, and CentOS Droplets created on or after April 26, 2018, the volume automatically mounts. On older Droplets, additional configuration is required. Visit https://docs.digitalocean.com/products/volumes/how-to/mount/ for details`, Writer,
		aliasOpt("a"))
	AddBoolFlag(cmdRunVolumeAttach, doctl.ArgCommandWait, "", false, "Instructs the terminal to wait for the volume to attach before returning control to the user")
	cmdRunVolumeAttach.Example = `The following example attaches a volume with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute volume-action attach f81d4fae-7dec-11d0-a765-00a0c91e6bf6 386734086`

	cmdRunVolumeDetach := CmdBuilder(cmd, RunVolumeDetach, "detach <volume-id> <droplet-id>", "Detach a volume from a Droplet", `Detaches a block storage volume from a Droplet.`, Writer,
		aliasOpt("d"))
	AddBoolFlag(cmdRunVolumeDetach, doctl.ArgCommandWait, "", false, "Instructs the terminal to wait for the volume to detach before returning control to the user")
	cmdRunVolumeDetach.Example = `The following example detaches a volume with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` from a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute volume-action detach f81d4fae-7dec-11d0-a765-00a0c91e6bf6 386734086`

	CmdBuilder(cmd, RunVolumeDetach, "detach-by-droplet-id <volume-id> <droplet-id>", "(Deprecated) Detach a volume. Use `detach` instead.", "This command detaches a volume. This command is deprecated. Use `doctl compute volume-action detach` instead.",
		Writer)

	cmdRunVolumeResize := CmdBuilder(cmd, RunVolumeResize, "resize <volume-id>", "Resize the disk of a volume", `Resizes a block storage volume.

Volumes may only be resized upwards. The maximum size for a volume is 16TiB.`, Writer,
		aliasOpt("r"))
	AddIntFlag(cmdRunVolumeResize, doctl.ArgSizeSlug, "", 0, "The volume's new size, in GiB",
		requiredOpt())
	AddStringFlag(cmdRunVolumeResize, doctl.ArgRegionSlug, "", "", "The volume's current region",
		requiredOpt())
	AddBoolFlag(cmdRunVolumeResize, doctl.ArgCommandWait, "", false, "Instructs the terminal to wait for the volume to complete resizing before returning control to the user")
	cmdRunVolumeResize.Example = `The following example resizes a volume with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to 120 GiB in the ` + "`" + `nyc1` + "`" + ` region: doctl compute volume-action resize f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --size 120 --region nyc1`

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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
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

// RunVolumeActionsGet returns a Volume Action
func RunVolumeActionsGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	volumeID := c.Args[0]
	actionID, err := c.Doit.GetInt(c.NS, doctl.ArgActionID)
	if err != nil {
		return err

	}

	vas := c.VolumeActions()
	volumeA, err := vas.Get(volumeID, actionID)
	if err != nil {
		return err
	}

	item := &displayers.Action{Actions: do.Actions{*volumeA}}
	return c.Display(item)
}

// RunVolumeActionsList returns a Volume Action
func RunVolumeActionsList(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	volumeID := c.Args[0]
	vList, err := c.VolumeActions().List(volumeID)
	if err != nil {
		return err
	}

	item := &displayers.Action{Actions: vList}
	return c.Display(item)

}
