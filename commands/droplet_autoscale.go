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
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func DropletAutoscale() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "droplet-autoscale",
			Aliases: []string{"das"},
			Short:   "Display commands to manage Droplet autoscale pools",
			Long: `Use the subcommands of ` + "`" + `doctl compute droplet-autoscale` + "`" + ` to perform actions on Droplet Autoscale Pools.

You can use droplet-autoscale to perform CRUD operations on a Droplet Autoscale Pools.`,
		},
	}
	cmdDropletAutoscaleCreate := CmdBuilder(cmd, RunDropletAutoscaleCreate, "create", "Create a new Droplet autoscale pool", "", Writer, displayerType(&displayers.DropletAutoscalePools{}))

	cmdDropletAutoscaleUpdate := CmdBuilder(cmd, RunDropletAutoscaleUpdate, "update <autoscale-pool-id>", "Update an active Droplet autoscale pool", "", Writer, displayerType(&displayers.DropletAutoscalePools{}))

	for _, c := range []*Command{
		cmdDropletAutoscaleCreate,
		cmdDropletAutoscaleUpdate,
	} {
		AddStringFlag(c, doctl.ArgAutoscaleName, "", "", "Name of the Droplet autoscale pool", requiredOpt())
		AddIntFlag(c, doctl.ArgAutoscaleMinInstances, "", 0, "Min number of members")
		AddIntFlag(c, doctl.ArgAutoscaleMaxInstances, "", 0, "Max number of members")
		AddStringFlag(c, doctl.ArgAutoscaleCpuTarget, "", "", "CPU target threshold")
		AddStringFlag(c, doctl.ArgAutoscaleMemTarget, "", "", "Memory target threshold")
		AddIntFlag(c, doctl.ArgAutoscaleCooldownMinutes, "", 0, "Cooldown duration")
		AddIntFlag(c, doctl.ArgAutoscaleTargetInstances, "", 0, "Target number of members")

		AddStringFlag(c, doctl.ArgSizeSlug, "", "", "Droplet size")
		AddStringFlag(c, doctl.ArgRegionSlug, "", "", "Droplet region")
		AddStringFlag(c, doctl.ArgImage, "", "", "Droplet image")
		AddStringSliceFlag(c, doctl.ArgTag, "", []string{}, "Droplet tags")
		AddStringSliceFlag(c, doctl.ArgSSHKeys, "", []string{}, "Droplet SSH keys")
		AddStringFlag(c, doctl.ArgVPCUUID, "", "", "Droplet VPC UUID")
		AddBoolFlag(c, doctl.ArgDropletAgent, "", true, "Enable droplet agent")
		AddStringFlag(c, doctl.ArgProjectID, "", "", "Droplet project ID")
		AddBoolFlag(c, doctl.ArgIPv6, "", true, "Enable droplet IPv6")
		AddStringFlag(c, doctl.ArgUserData, "", "", "Droplet user data")
	}

	CmdBuilder(cmd, RunDropletAutoscaleGet, "get <autoscale-pool-id>", "Get an active Droplet autoscale pool", "", Writer, displayerType(&displayers.DropletAutoscalePools{}))

	CmdBuilder(cmd, RunDropletAutoscaleList, "list", "List all active Droplet autoscale pools", "", Writer, displayerType(&displayers.DropletAutoscalePools{}), aliasOpt("ls"))

	CmdBuilder(cmd, RunDropletAutoscaleListMembers, "list-members <autoscale-pool-id>", "List all members of a Droplet autoscale pool", "", Writer, displayerType(&displayers.DropletAutoscaleResources{}))

	CmdBuilder(cmd, RunDropletAutoscaleListHistory, "list-history <autoscale-pool-id>", "List all history events for a Droplet autoscale pool", "", Writer, displayerType(&displayers.DropletAutoscaleHistoryEvents{}))

	cmdDropletAutoscaleDelete := CmdBuilder(cmd, RunDropletAutoscaleDelete, "delete <autoscale-pool-id>", "Delete an active Droplet autoscale pool", "", Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdDropletAutoscaleDelete, doctl.ArgForce, "", false, "Force delete without a confirmation prompt")

	cmdDropletAutoscaleDeleteDangerous := CmdBuilder(cmd, RunDropletAutoscaleDeleteDangerous, "delete-dangerous <autoscale-pool-id>", "Delete an active Droplet autoscale pool and all its members", "", Writer)
	AddBoolFlag(cmdDropletAutoscaleDeleteDangerous, doctl.ArgForce, "", false, "Force delete without a confirmation prompt")

	return cmd
}

func buildDropletAutoscaleRequestFromArgs(c *CmdConfig, r *godo.DropletAutoscalePoolRequest) error {
	var hydrators = []func() error{
		func() error {
			name, err := c.Doit.GetString(c.NS, doctl.ArgAutoscaleName)
			if err != nil {
				return err
			}
			r.Name = name
			return nil
		},
		func() error {
			minCount, err := c.Doit.GetInt(c.NS, doctl.ArgAutoscaleMinInstances)
			if err != nil {
				return err
			}
			r.Config.MinInstances = uint64(minCount)
			return nil
		},
		func() error {
			maxCount, err := c.Doit.GetInt(c.NS, doctl.ArgAutoscaleMaxInstances)
			if err != nil {
				return err
			}
			r.Config.MaxInstances = uint64(maxCount)
			return nil
		},
		func() error {
			cpuStr, err := c.Doit.GetString(c.NS, doctl.ArgAutoscaleCpuTarget)
			if err != nil {
				return err
			}
			if cpuStr != "" {
				cpuTarget, err := strconv.ParseFloat(cpuStr, 64)
				if err != nil {
					return err
				}
				r.Config.TargetCPUUtilization = cpuTarget
			}
			return nil
		},
		func() error {
			memStr, err := c.Doit.GetString(c.NS, doctl.ArgAutoscaleMemTarget)
			if err != nil {
				return err
			}
			if memStr != "" {
				memTarget, err := strconv.ParseFloat(memStr, 64)
				if err != nil {
					return err
				}
				r.Config.TargetMemoryUtilization = memTarget
			}
			return nil
		},
		func() error {
			cooldown, err := c.Doit.GetInt(c.NS, doctl.ArgAutoscaleCooldownMinutes)
			if err != nil {
				return err
			}
			r.Config.CooldownMinutes = uint32(cooldown)
			return nil
		},
		func() error {
			targetCount, err := c.Doit.GetInt(c.NS, doctl.ArgAutoscaleTargetInstances)
			if err != nil {
				return err
			}
			r.Config.TargetNumberInstances = uint64(targetCount)
			return nil
		},
		func() error {
			size, err := c.Doit.GetString(c.NS, doctl.ArgSizeSlug)
			if err != nil {
				return err
			}
			r.DropletTemplate.Size = size
			return nil
		},
		func() error {
			region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
			if err != nil {
				return err
			}
			r.DropletTemplate.Region = region
			return nil
		},
		func() error {
			image, err := c.Doit.GetString(c.NS, doctl.ArgImage)
			if err != nil {
				return err
			}
			r.DropletTemplate.Image = image
			return nil
		},
		func() error {
			tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
			if err != nil {
				return err
			}
			r.DropletTemplate.Tags = tags
			return nil
		},
		func() error {
			sshKeys, err := c.Doit.GetStringSlice(c.NS, doctl.ArgSSHKeys)
			if err != nil {
				return err
			}
			r.DropletTemplate.SSHKeys = sshKeys
			return nil
		},
		func() error {
			vpcUUID, err := c.Doit.GetString(c.NS, doctl.ArgVPCUUID)
			if err != nil {
				return err
			}
			r.DropletTemplate.VpcUUID = vpcUUID
			return nil
		},
		func() error {
			enableAgent, err := c.Doit.GetBool(c.NS, doctl.ArgDropletAgent)
			if err != nil {
				return err
			}
			r.DropletTemplate.WithDropletAgent = enableAgent
			return nil
		},
		func() error {
			projectID, err := c.Doit.GetString(c.NS, doctl.ArgProjectID)
			if err != nil {
				return err
			}
			r.DropletTemplate.ProjectID = projectID
			return nil
		},
		func() error {
			enableIPv6, err := c.Doit.GetBool(c.NS, doctl.ArgIPv6)
			if err != nil {
				return err
			}
			r.DropletTemplate.IPV6 = enableIPv6
			return nil
		},
		func() error {
			userData, err := c.Doit.GetString(c.NS, doctl.ArgUserData)
			if err != nil {
				return err
			}
			r.DropletTemplate.UserData = userData
			return nil
		},
	}
	for _, h := range hydrators {
		if err := h(); err != nil {
			return err
		}
	}
	return nil
}

// RunDropletAutoscaleCreate creates an autoscale pool
func RunDropletAutoscaleCreate(c *CmdConfig) error {
	createReq := new(godo.DropletAutoscalePoolRequest)
	createReq.Config = new(godo.DropletAutoscaleConfiguration)
	createReq.DropletTemplate = new(godo.DropletAutoscaleResourceTemplate)
	if err := buildDropletAutoscaleRequestFromArgs(c, createReq); err != nil {
		return err
	}
	pool, err := c.DropletAutoscale().Create(createReq)
	if err != nil {
		return err
	}
	item := &displayers.DropletAutoscalePools{AutoscalePools: []*godo.DropletAutoscalePool{pool}}
	return c.Display(item)
}

// RunDropletAutoscaleUpdate updates an autoscale pool
func RunDropletAutoscaleUpdate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	updateReq := new(godo.DropletAutoscalePoolRequest)
	updateReq.Config = new(godo.DropletAutoscaleConfiguration)
	updateReq.DropletTemplate = new(godo.DropletAutoscaleResourceTemplate)
	if err := buildDropletAutoscaleRequestFromArgs(c, updateReq); err != nil {
		return err
	}
	pool, err := c.DropletAutoscale().Update(id, updateReq)
	if err != nil {
		return err
	}
	item := &displayers.DropletAutoscalePools{AutoscalePools: []*godo.DropletAutoscalePool{pool}}
	return c.Display(item)
}

// RunDropletAutoscaleGet retrieves an autoscale pool
func RunDropletAutoscaleGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	pool, err := c.DropletAutoscale().Get(id)
	if err != nil {
		return err
	}
	item := &displayers.DropletAutoscalePools{AutoscalePools: []*godo.DropletAutoscalePool{pool}}
	return c.Display(item)
}

// RunDropletAutoscaleList lists all autoscale pools
func RunDropletAutoscaleList(c *CmdConfig) error {
	pools, err := c.DropletAutoscale().List()
	if err != nil {
		return err
	}
	item := &displayers.DropletAutoscalePools{AutoscalePools: pools}
	return c.Display(item)
}

// RunDropletAutoscaleListMembers lists autoscale pool members
func RunDropletAutoscaleListMembers(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	members, err := c.DropletAutoscale().ListMembers(id)
	if err != nil {
		return err
	}
	item := &displayers.DropletAutoscaleResources{Droplets: members}
	return c.Display(item)
}

// RunDropletAutoscaleListHistory lists autoscale pool history events
func RunDropletAutoscaleListHistory(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	history, err := c.DropletAutoscale().ListHistory(id)
	if err != nil {
		return err
	}
	item := &displayers.DropletAutoscaleHistoryEvents{History: history}
	return c.Display(item)
}

// RunDropletAutoscaleDelete deletes an autoscale pool
func RunDropletAutoscaleDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if force || AskForConfirmDelete("droplet autoscale pool", 1) == nil {
		if err = c.DropletAutoscale().Delete(id); err != nil {
			return err
		}
	} else {
		return errOperationAborted
	}
	return nil
}

// RunDropletAutoscaleDeleteDangerous deletes an autoscale pool and all underlying members
func RunDropletAutoscaleDeleteDangerous(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if force || AskForConfirmDelete("droplet autoscale pool", 1) == nil {
		if err = c.DropletAutoscale().DeleteDangerous(id); err != nil {
			return err
		}
	} else {
		return errOperationAborted
	}
	return nil
}
