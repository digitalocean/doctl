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
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

const (
	dropletIDResource = "<droplet-id>"
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
			Short:   "Display Droplet action commands",
			Long: `Use the subcommands of ` + "`" + `doctl compute droplet-action` + "`" + ` to perform actions on Droplets.

Droplet actions are tasks that can be executed on a Droplet, such as rebooting, resizing, or snapshotting a Droplet.`,
		},
	}

	cmdDropletActionGet := CmdBuilder(cmd, RunDropletActionGet, "get <droplet-id>", "Retrieve a specific Droplet action", `Use this command to retrieve a Droplet action.`, Writer,
		aliasOpt("g"), displayerType(&displayers.Action{}))
	AddIntFlag(cmdDropletActionGet, doctl.ArgActionID, "", 0, "Action ID", requiredOpt())

	cmdDropletActionEnableBackups := CmdBuilder(cmd, RunDropletActionEnableBackups,
		"enable-backups <droplet-id>", "Enable backups on a Droplet", `Use this command to enable backups on a Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionEnableBackups, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionDisableBackups := CmdBuilder(cmd, RunDropletActionDisableBackups,
		"disable-backups <droplet-id>", "Disable backups on a Droplet", `Use this command to disable backups on a Droplet. This does not delete existing backups.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionDisableBackups, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionReboot := CmdBuilder(cmd, RunDropletActionReboot,
		"reboot <droplet-id>", "Reboot a Droplet", `Use this command to reboot a Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionReboot, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionPowerCycle := CmdBuilder(cmd, RunDropletActionPowerCycle,
		"power-cycle <droplet-id>", "Powercycle a Droplet", `Use this command to powercycle a Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPowerCycle, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionShutdown := CmdBuilder(cmd, RunDropletActionShutdown,
		"shutdown <droplet-id>", "Shut down a Droplet", `Use this command to shut down a Droplet. Droplets that are powered off are still billable. To stop billing, destroy the Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionShutdown, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionPowerOff := CmdBuilder(cmd, RunDropletActionPowerOff,
		"power-off <droplet-id>", "Power off a Droplet", `Use this command to power off a Droplet. Droplets that are powered off are still billable. To stop billing, destroy the Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPowerOff, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionPowerOn := CmdBuilder(cmd, RunDropletActionPowerOn,
		"power-on <droplet-id>", "Power on a Droplet", `Use this command to power on a Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPowerOn, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionPasswordReset := CmdBuilder(cmd, RunDropletActionPasswordReset,
		"password-reset <droplet-id>", "Reset the root password for a Droplet", `Use this command to initiate a root password reset on a Droplet. This also powercycles the Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPasswordReset, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionEnableIPv6 := CmdBuilder(cmd, RunDropletActionEnableIPv6,
		"enable-ipv6 <droplet-id>", "Enable IPv6 on a Droplet", `Use this command to enable IPv6 networking on a Droplet. DigitalOcean will automatically assign an IPv6 address to the Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionEnableIPv6, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionEnablePrivateNetworking := CmdBuilder(cmd, RunDropletActionEnablePrivateNetworking,
		"enable-private-networking <droplet-id>", "Enable private networking on a Droplet", `Use this command to enable private networking on a Droplet. This adds a private IPv4 address to the Droplet that other Droplets inside the network can access. The Droplet will require additional internal network configuration for it to become accessible through the private network.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionEnablePrivateNetworking, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionRestore := CmdBuilder(cmd, RunDropletActionRestore,
		"restore <droplet-id>", "Restore a Droplet from a backup", `Use this command to restore a Droplet from a backup.`, Writer,
		displayerType(&displayers.Action{}))
	AddIntFlag(cmdDropletActionRestore, doctl.ArgImageID, "", 0, "Image ID", requiredOpt())
	AddBoolFlag(cmdDropletActionRestore, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	dropletResizeDesc := `Use this command to resize a Droplet to a different plan.

By default, this command will only increase or decrease the CPU and RAM of the Droplet, not its disk size. This can be reversed.

To also increase the Droplet's disk size, pass the ` + "`--resize-disk`" + ` flag. This is a permanent change and cannot be reversed as a Droplet's disk size cannot be decreased.

In order to resize a Droplet, it must first be powered off.`
	cmdDropletActionResize := CmdBuilder(cmd, RunDropletActionResize,
		"resize <droplet-id>", "Resize a Droplet", dropletResizeDesc, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionResize, doctl.ArgResizeDisk, "", false, "Resize the Droplet's disk size in addition to its RAM and CPU.")
	AddStringFlag(cmdDropletActionResize, doctl.ArgSizeSlug, "", "", "A slug indicating the new size for the Droplet (e.g. `s-2vcpu-2gb`). Run `doctl compute size list` for a list of valid sizes.", requiredOpt())
	AddBoolFlag(cmdDropletActionResize, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionRebuild := CmdBuilder(cmd, RunDropletActionRebuild,
		"rebuild <droplet-id>", "Rebuild a Droplet", `Use this command to rebuild a Droplet from an image.`, Writer,
		displayerType(&displayers.Action{}))
	AddStringFlag(cmdDropletActionRebuild, doctl.ArgImage, "", "", "Image ID or Slug", requiredOpt())
	AddBoolFlag(cmdDropletActionRebuild, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionRename := CmdBuilder(cmd, RunDropletActionRename,
		"rename <droplet-id>", "Rename a Droplet", `Use this command to rename a Droplet. When using a fully qualified domain name (FQDN) this also updates the pointer (PTR) record.`, Writer,
		displayerType(&displayers.Action{}))
	AddStringFlag(cmdDropletActionRename, doctl.ArgDropletName, "", "", "Droplet name", requiredOpt())
	AddBoolFlag(cmdDropletActionRename, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionChangeKernel := CmdBuilder(cmd, RunDropletActionChangeKernel,
		"change-kernel <droplet-id>", "Change a Droplet's kernel", `Use this command to change a Droplet's kernel.`, Writer,
		displayerType(&displayers.Action{}))
	AddIntFlag(cmdDropletActionChangeKernel, doctl.ArgKernelID, "", 0, "Kernel ID", requiredOpt())
	AddBoolFlag(cmdDropletActionChangeKernel, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdDropletActionSnapshot := CmdBuilder(cmd, RunDropletActionSnapshot,
		"snapshot <droplet-id>", "Take a Droplet snapshot", `Use this command to take a snapshot of a Droplet. We recommend that you power off the Droplet before taking a snapshot to ensure data consistency.`, Writer,
		displayerType(&displayers.Action{}))
	AddStringFlag(cmdDropletActionSnapshot, doctl.ArgSnapshotName, "", "", "Snapshot name", requiredOpt())
	AddBoolFlag(cmdDropletActionSnapshot, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	return cmd
}

// RunDropletActionGet returns a droplet action by id.
func RunDropletActionGet(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		dropletID, err := ContextualAtoi(c.Args[0], dropletIDResource)
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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)
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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)
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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)
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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)
		if err != nil {
			return nil, err
		}

		a, err := das.Shutdown(id)
		return a, err
	}

	return performAction(c, fn)
}

// RunDropletActionPowerOff turns droplet power off.
func RunDropletActionPowerOff(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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

// RunDropletActionResize resizes a droplet giving a size slug and
// optionally expands the disk.
func RunDropletActionResize(c *CmdConfig) error {
	fn := func(das do.DropletActionsService) (*do.Action, error) {
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

		if err != nil {
			return nil, err
		}

		image, err := c.Doit.GetString(c.NS, doctl.ArgImage)
		if err != nil {
			return nil, err
		}

		var a *do.Action
		if i, aerr := ContextualAtoi(image, dropletIDResource); aerr == nil {
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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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
		err := ensureOneArg(c)
		if err != nil {
			return nil, err
		}
		id, err := ContextualAtoi(c.Args[0], dropletIDResource)

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
