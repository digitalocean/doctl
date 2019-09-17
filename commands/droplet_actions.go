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
	"fmt"
	"strconv"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

type actionFn func(das do.DropletActionsService) (*do.Action, error)

func performAction(c *CmdConfig, fn actionFn) error {
	das := c.DropletActions()

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

// DropletAction creates the droplet-action command.
func DropletAction() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "droplet-action",
			Aliases: []string{"da"},
			Short:   "droplet action commands",
			Long:    "droplet-action is used to access droplet action commands",
		},
	}

	cmdDropletActionGet := CmdBuilder(cmd, RunDropletActionGet, "get <droplet-id>", "get droplet action", Writer,
		aliasOpt("g"), displayerType(&displayers.Action{}))
	AddIntFlag(cmdDropletActionGet, doctl.ArgActionID, "", 0, "Action ID", requiredOpt())

	cmdDropletActionEnableBackups := CmdBuilder(cmd, RunDropletActionEnableBackups,
		"enable-backups <droplet-id>", "enable backups", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionEnableBackups, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionDisableBackups := CmdBuilder(cmd, RunDropletActionDisableBackups,
		"disable-backups <droplet-id>", "disable backups", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionDisableBackups, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionReboot := CmdBuilder(cmd, RunDropletActionReboot,
		"reboot <droplet-id>", "reboot droplet", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionReboot, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionPowerCycle := CmdBuilder(cmd, RunDropletActionPowerCycle,
		"power-cycle <droplet-id>", "power cycle droplet", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPowerCycle, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionShutdown := CmdBuilder(cmd, RunDropletActionShutdown,
		"shutdown <droplet-id>", "shutdown droplet", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionShutdown, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionPowerOff := CmdBuilder(cmd, RunDropletActionPowerOff,
		"power-off <droplet-id>", "power off droplet", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPowerOff, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionPowerOn := CmdBuilder(cmd, RunDropletActionPowerOn,
		"power-on <droplet-id>", "power on droplet", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPowerOn, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionPasswordReset := CmdBuilder(cmd, RunDropletActionPasswordReset,
		"password-reset <droplet-id>", "password reset droplet", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPasswordReset, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionEnableIPv6 := CmdBuilder(cmd, RunDropletActionEnableIPv6,
		"enable-ipv6 <droplet-id>", "enable ipv6", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionEnableIPv6, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionEnablePrivateNetworking := CmdBuilder(cmd, RunDropletActionEnablePrivateNetworking,
		"enable-private-networking <droplet-id>", "enable private networking", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionEnablePrivateNetworking, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionRestore := CmdBuilder(cmd, RunDropletActionRestore,
		"restore <droplet-id>", "restore backup", Writer,
		displayerType(&displayers.Action{}))
	AddIntFlag(cmdDropletActionRestore, doctl.ArgImageID, "", 0, "Image ID", requiredOpt())
	AddBoolFlag(cmdDropletActionRestore, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionResize := CmdBuilder(cmd, RunDropletActionResize,
		"resize <droplet-id>", "resize droplet", Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionResize, doctl.ArgResizeDisk, "", false, "Resize disk")
	AddStringFlag(cmdDropletActionResize, doctl.ArgSizeSlug, "", "", "New size")
	AddBoolFlag(cmdDropletActionResize, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionRebuild := CmdBuilder(cmd, RunDropletActionRebuild,
		"rebuild <droplet-id>", "rebuild droplet", Writer,
		displayerType(&displayers.Action{}))
	AddStringFlag(cmdDropletActionRebuild, doctl.ArgImage, "", "", "Image ID or Slug", requiredOpt())
	AddBoolFlag(cmdDropletActionRebuild, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionRename := CmdBuilder(cmd, RunDropletActionRename,
		"rename <droplet-id>", "rename droplet", Writer,
		displayerType(&displayers.Action{}))
	AddStringFlag(cmdDropletActionRename, doctl.ArgDropletName, "", "", "Droplet name", requiredOpt())
	AddBoolFlag(cmdDropletActionRename, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionChangeKernel := CmdBuilder(cmd, RunDropletActionChangeKernel,
		"change-kernel <droplet-id>", "change kernel", Writer,
		displayerType(&displayers.Action{}))
	AddIntFlag(cmdDropletActionChangeKernel, doctl.ArgKernelID, "", 0, "Kernel ID", requiredOpt())
	AddBoolFlag(cmdDropletActionChangeKernel, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionSnapshot := CmdBuilder(cmd, RunDropletActionSnapshot,
		"snapshot <droplet-id>", "snapshot droplet", Writer,
		displayerType(&displayers.Action{}))
	AddStringFlag(cmdDropletActionSnapshot, doctl.ArgSnapshotName, "", "", "Snapshot name", requiredOpt())
	AddBoolFlag(cmdDropletActionSnapshot, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	return cmd
}

// RunDropletActionGet returns a droplet action by id.
func RunDropletActionGet(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		dropletID, err := strconv.Atoi(c.Args[0])
		if err != nil {
			return nil, err
		}

		actionID, err := c.Doit.GetInt(c.NS, doctl.ArgActionID)
		if err != nil {
			return nil, err
		}

		a, err := das.Get(dropletID, actionID)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionEnableBackups disables backups for a droplet.
func RunDropletActionEnableBackups(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])
		if err != nil {
			return nil, err
		}

		a, err := das.EnableBackups(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionDisableBackups disables backups for a droplet.
func RunDropletActionDisableBackups(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])
		if err != nil {
			return nil, err
		}

		a, err := das.DisableBackups(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionReboot reboots a droplet.
func RunDropletActionReboot(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])
		if err != nil {
			return nil, err
		}

		a, err := das.Reboot(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionPowerCycle power cycles a droplet.
func RunDropletActionPowerCycle(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PowerCycle(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionShutdown shuts a droplet down.
func RunDropletActionShutdown(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])
		if err != nil {
			return nil, fmt.Errorf("Could not convert args into integer")
		}

		a, err := das.Shutdown(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionPowerOff turns droplet power off.
func RunDropletActionPowerOff(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PowerOff(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionPowerOn turns droplet power on.
func RunDropletActionPowerOn(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PowerOn(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionPasswordReset resets the droplet root password.
func RunDropletActionPasswordReset(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.PasswordReset(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionEnableIPv6 enables IPv6 for a droplet.
func RunDropletActionEnableIPv6(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.EnableIPv6(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionEnablePrivateNetworking enables private networking for a droplet.
func RunDropletActionEnablePrivateNetworking(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		a, err := das.EnablePrivateNetworking(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionRestore restores a droplet using an image id.
func RunDropletActionRestore(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		image, err := c.Doit.GetInt(c.NS, doctl.ArgImageID)
		if err != nil {
			return nil, err
		}

		a, err := das.Restore(id, image)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionResize resizesx a droplet giving a size slug and
// optionally expands the disk.
func RunDropletActionResize(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
		if err != nil {
			return nil, err
		}

		disk, err := c.Doit.GetBool(c.NS, doctl.ArgResizeDisk)
		if err != nil {
			return nil, err
		}

		a, err := das.Resize(id, size, disk)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionRebuild rebuilds a droplet using an image id or slug.
func RunDropletActionRebuild(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		image, err := c.Doit.GetString(c.NS, doctl.ArgImage)
		if err != nil {
			return nil, err
		}

		var a *do.Action
		if i, aerr := strconv.Atoi(image); aerr == nil {
			a, err = das.RebuildByImageID(id, i)
		} else {
			a, err = das.RebuildByImageSlug(id, image)
		}
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionRename renames a droplet.
func RunDropletActionRename(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		name, err := c.Doit.GetString(c.NS, doctl.ArgDropletName)
		if err != nil {
			return nil, err
		}

		a, err := das.Rename(id, name)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionChangeKernel changes the kernel for a droplet.
func RunDropletActionChangeKernel(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		kernel, err := c.Doit.GetInt(c.NS, doctl.ArgKernelID)
		if err != nil {
			return nil, err
		}

		a, err := das.ChangeKernel(id, kernel)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionSnapshot creates a snapshot for a droplet.
func RunDropletActionSnapshot(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		if len(c.Args) != 1 {
			return nil, doctl.NewMissingArgsErr(c.NS)
		}
		id, err := strconv.Atoi(c.Args[0])

		if err != nil {
			return nil, err
		}

		name, err := c.Doit.GetString(c.NS, doctl.ArgSnapshotName)
		if err != nil {
			return nil, err
		}

		a, err := das.Snapshot(id, name)
		return a, err
	}

	return performAction(c, fn)
}
