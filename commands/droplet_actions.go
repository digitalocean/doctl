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

You can use Droplet actions to perform tasks on a Droplet, such as rebooting, resizing, or snapshotting it.`,
		},
	}

	cmdDropletActionGet := CmdBuilder(cmd, RunDropletActionGet, "get <droplet-id>", "Retrieve a specific Droplet action", `Retrieves information about an action performed on a Droplet, including its status, type, and completion time.`, Writer,
		aliasOpt("g"), displayerType(&displayers.Action{}))
	AddIntFlag(cmdDropletActionGet, doctl.ArgActionID, "", 0, "Action ID", requiredOpt())
	cmdDropletActionGet.Example = `The following example retrieves information about an action, with the ID ` + "`" + `1978716488` + "`" + `, performed on a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action get 1978716488 --action-id 386734086`

	cmdDropletActionEnableBackups := CmdBuilder(cmd, RunDropletActionEnableBackups,
		"enable-backups <droplet-id>", "Enable backups on a Droplet", `Enables backups on a Droplet. This automatically creates and stores a disk image of the Droplet at weekly intervals.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionEnableBackups, doctl.ArgCommandWait, "", false, "Wait for action to complete")
	cmdDropletActionEnableBackups.Example = `The following example enables backups on a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action enable-backups 386734086`

	cmdDropletActionDisableBackups := CmdBuilder(cmd, RunDropletActionDisableBackups,
		"disable-backups <droplet-id>", "Disable backups on a Droplet", `Disables backups on a Droplet. This does not delete existing backups.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionDisableBackups, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionDisableBackups.Example = `The following example disables backups on a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action disable-backups 386734086`

	cmdDropletActionReboot := CmdBuilder(cmd, RunDropletActionReboot,
		"reboot <droplet-id>", "Reboot a Droplet", `Reboots a Droplet. A reboot action is an attempt to reboot the Droplet in a graceful way, similar to using the reboot command from the Droplet's console.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionReboot, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionReboot.Example = `The following example reboots a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action reboot 386734086`

	cmdDropletActionPowerCycle := CmdBuilder(cmd, RunDropletActionPowerCycle,
		"power-cycle <droplet-id>", "Powercycle a Droplet", `Powercycles a Droplet. A powercycle action is similar to pushing the reset button on a physical machine.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPowerCycle, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionPowerCycle.Example = `The following example powercycles a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action power-cycle 386734086`

	cmdDropletActionShutdown := CmdBuilder(cmd, RunDropletActionShutdown,
		"shutdown <droplet-id>", "Shut down a Droplet", `Shuts down a Droplet. 
		
A shutdown action is an attempt to shutdown the Droplet in a graceful way, similar to using the shutdown command from the Droplet's console. Since a shutdown command can fail, this action guarantees that the command is issued, not that it succeeds. The preferred way to turn off a Droplet is to attempt a shutdown, with a reasonable timeout, followed by a `+"`"+`doctl compute droplet-action power_off`+"`"+` action to ensure the Droplet is off.
		
Droplets that are powered off are still billable. To stop incurring charges on a Droplet, destroy it.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionShutdown, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionShutdown.Example = `The following example shuts down a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action shutdown 386734086`

	cmdDropletActionPowerOff := CmdBuilder(cmd, RunDropletActionPowerOff,
		"power-off <droplet-id>", "Power off a Droplet", `Use this command to power off a Droplet.
		
A `+"`"+`power_off`+"`"+` event is a hard shutdown and should only be used if the shutdown action is not successful. It is similar to cutting the power on a server and could lead to complications.

Droplets that are powered off are still billable. To stop incurring charges on a Droplet, destroy it.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPowerOff, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionPowerOff.Example = `The following example powers off a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action power-off 386734086`

	cmdDropletActionPowerOn := CmdBuilder(cmd, RunDropletActionPowerOn,
		"power-on <droplet-id>", "Power on a Droplet", `Powers on a Droplet. This is similar to pressing the power button on a physical machine.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPowerOn, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionPowerOn.Example = `The following example powers on a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action power-on 386734086`

	cmdDropletActionPasswordReset := CmdBuilder(cmd, RunDropletActionPasswordReset,
		"password-reset <droplet-id>", "Reset the root password for a Droplet", `Initiates a root password reset on a Droplet. We provide a new password for the Droplet via the accounts email address. The password must be changed after first use. 

This also powercycles the Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionPasswordReset, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionPasswordReset.Example = `The following example resets the root password for a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action password-reset 386734086`

	cmdDropletActionEnableIPv6 := CmdBuilder(cmd, RunDropletActionEnableIPv6,
		"enable-ipv6 <droplet-id>", "Enable IPv6 on a Droplet", `Enables IPv6 networking on a Droplet. When executed, we automatically assign an IPv6 address to the Droplet. 

The Droplet may require additional network configuration to properly use the new IPv6 address. For more information, see: https://docs.digitalocean.com/products/networking/ipv6/how-to/enable`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionEnableIPv6, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionEnableIPv6.Example = `The following example enables IPv6 on a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action enable-ipv6 386734086`

	cmdDropletActionEnablePrivateNetworking := CmdBuilder(cmd, RunDropletActionEnablePrivateNetworking,
		"enable-private-networking <droplet-id>", "Enable private networking on a Droplet", `Enables VPC networking on a Droplet. This command adds a private IPv4 address to the Droplet that other resources inside the Droplet's VPC network can access. The Droplet is placed in the default VPC network for the region it resides in.

All Droplets created after 1 October 2020 are provided a private IP address and placed into a VPC network by default. You can use this command to enable private networking on a Droplet that was created before 1 October 2020 and was not already in a VPC network.

Once you have manually enabled private networking for a Droplet, the Droplet requires additional internal network configuration for it to become accessible through the VPC network. For more information, see: https://docs.digitalocean.com/products/networking/vpc/how-to/enable`, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionEnablePrivateNetworking, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionEnablePrivateNetworking.Example = `The following example enables private networking on a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute droplet-action enable-private-networking 386734086`

	cmdDropletActionRestore := CmdBuilder(cmd, RunDropletActionRestore,
		"restore <droplet-id>", "Restore a Droplet from a backup", `Restores a Droplet from a backup image. You must pass an image ID that is a backup of the current Droplet instance. The operation leaves any embedded SSH keys intact.
		
		To retrieve a list of backup images, use the `+"`"+`doctl compute image list`+"`"+` command.`, Writer,
		displayerType(&displayers.Action{}))
	AddIntFlag(cmdDropletActionRestore, doctl.ArgImageID, "", 0, "The ID of the image to restore the Droplet from", requiredOpt())
	AddBoolFlag(cmdDropletActionRestore, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionRestore.Example = `The following example restores a Droplet with the ID ` + "`" + `386734086` + "`" + ` from a backup image with the ID ` + "`" + `146288445` + "`" + `: doctl compute droplet-action restore 386734086 --image-id 146288445`

	dropletResizeDesc := `Resizes a Droplet to a different plan.

By default, this command only increases or decreases the CPU and RAM of the Droplet, not its disk size. Unlike increasing disk size, you can reverse this action.

To also increase the Droplet's disk size, choose a size slug with the desired amount of vCPUs, RAM, and disk space and then set the ` + "`" + `--resize-disk` + "`" + ` flag to ` + "`" + `true` + "`" + `. This is a permanent change and cannot be reversed as a Droplet's disk size cannot be decreased.

For a list of size slugs, use the ` + "`" + `doctl compute size list` + "`" + ` command.

This command automatically powers off the Droplet before resizing it.`
	cmdDropletActionResize := CmdBuilder(cmd, RunDropletActionResize,
		"resize <droplet-id>", "Resize a Droplet", dropletResizeDesc, Writer,
		displayerType(&displayers.Action{}))
	AddBoolFlag(cmdDropletActionResize, doctl.ArgResizeDisk, "", false, "Resize the Droplet's disk size in addition to its RAM and CPUs")
	AddStringFlag(cmdDropletActionResize, doctl.ArgSizeSlug, "", "", "A slug indicating the new size for the Droplet, for example `s-2vcpu-2gb`. Run `doctl compute size list` for a list of valid sizes.", requiredOpt())
	AddBoolFlag(cmdDropletActionResize, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionResize.Example = `The following example resizes a Droplet with the ID ` + "`" + `386734086` + "`" + ` to a Droplet with two CPUs, two GiB of RAM, and 60 GBs of disk space. The 60 GBs of disk space is the defined amount for the ` + "`" + `s-2vcpu-2gb` + "`" + ` plan: doctl compute droplet-action resize 386734086 --size s-2vcpu-2gb --resize-disk=true`

	cmdDropletActionRebuild := CmdBuilder(cmd, RunDropletActionRebuild,
		"rebuild <droplet-id>", "Rebuild a Droplet", `Rebuilds a Droplet from an image, such as an Ubuntu base image or a backup image of the Droplet. Set the image attribute to an image ID or slug.

To retrieve a list of images on your account, use the `+"`"+`doctl compute image list`+"`"+` command. To retrieve a list of base images, use the `+"`"+`doctl compute image list-distribution`+"`"+` command.`, Writer,
		displayerType(&displayers.Action{}))
	AddStringFlag(cmdDropletActionRebuild, doctl.ArgImage, "", "", "An image ID or slug", requiredOpt())
	AddBoolFlag(cmdDropletActionRebuild, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionRebuild.Example = `The following example rebuilds a Droplet with the ID ` + "`" + `386734086` + "`" + ` from the image with the ID ` + "`" + `146288445` + "`" + `: doctl compute droplet-action rebuild 386734086 --image 146288445`

	cmdDropletActionRename := CmdBuilder(cmd, RunDropletActionRename,
		"rename <droplet-id>", "Rename a Droplet", `Renames a Droplet. When using a Fully Qualified Domain Name (FQDN) this also updates the Droplet's pointer (PTR) record.`, Writer,
		displayerType(&displayers.Action{}))
	AddStringFlag(cmdDropletActionRename, doctl.ArgDropletName, "", "", "The new name for the Droplet", requiredOpt())
	AddBoolFlag(cmdDropletActionRename, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")
	cmdDropletActionRename.Example = `The following example renames a Droplet with the ID ` + "`" + `386734086` + "`" + ` to ` + "`" + `example.com` + "`" + ` an FQDN: doctl compute droplet-action rename 386734086 --droplet-name example.com`

	cmdDropletActionChangeKernel := CmdBuilder(cmd, RunDropletActionChangeKernel,
		"change-kernel <droplet-id>", "Change a Droplet's kernel", `Changes a Droplet's kernel. This is only available for externally managed kernels. All Droplets created after 17 March 2017 have internally managed kernels by default.
		
Use the `+"`"+`doctl compute droplet kernels <droplet-id>`+"`"+` command to retrieve a list of kernels for the Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	AddIntFlag(cmdDropletActionChangeKernel, doctl.ArgKernelID, "", 0, "Kernel ID", requiredOpt())
	AddBoolFlag(cmdDropletActionChangeKernel, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")

	cmdDropletActionSnapshot := CmdBuilder(cmd, RunDropletActionSnapshot,
		"snapshot <droplet-id>", "Take a Droplet snapshot", `Takes a snapshot of a Droplet. Snapshots are complete disk images that contain all of the data on a Droplet at the time of the snapshot. This can be useful for restoring and rebuilding Droplets.
		
We recommend that you power off the Droplet before taking a snapshot to ensure data consistency.`, Writer,
		displayerType(&displayers.Action{}))
	AddStringFlag(cmdDropletActionSnapshot, doctl.ArgSnapshotName, "", "", "The snapshot's name", requiredOpt())
	AddBoolFlag(cmdDropletActionSnapshot, doctl.ArgCommandWait, "", false, "Instruct the terminal to wait for the action to complete before returning access to the user")

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
