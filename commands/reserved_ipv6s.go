/*
Copyright 2024 The Doctl Authors All rights reserved.
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
	"errors"
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// ReservedIPv6 creates the command hierarchy for reserved IPv6s.
func ReservedIPv6() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "reserved-ipv6",
			Short: "Display commands to manage reserved IPv6 addresses",
			Long: `The sub-commands of ` + "`" + `doctl compute reserved-ipv6` + "`" + ` manage reserved IPv6 addresses.
Reserved IPv6s are publicly-accessible static IPv6 addresses that you can assign to one of your Droplets. They can be used to create highly available setups or other configurations requiring movable addresses. Reserved IPv6s are bound to the regions they are created in.`,
			Aliases: []string{"reserved-ipv6s"},
			Hidden:  true,
		},
	}

	cmdReservedIPv6Create := CmdBuilder(cmd, RunReservedIPv6Create, "create", "Create a new reserved IPv6 address", `Creates a new reserved IPv6 address.
Reserved IPv6 addresses can be held in the region they were created in on your account.`, Writer,
		aliasOpt("c"), displayerType(&displayers.ReservedIPv6{}))
	AddStringFlag(cmdReservedIPv6Create, doctl.ArgRegionSlug, "", "", "The region where to create the reserved IPv6 address.")
	cmdReservedIPv6Create.Example = `The following example creates a reserved IPv6 address in the ` + "`" + `nyc1` + "`" + ` region: doctl compute reserved-ipv6 create --region nyc1`

	cmdReservedIPv6Get := CmdBuilder(cmd, RunReservedIPv6Get, "get <reserved-ipv6>", "Retrieve information about a reserved IPv6 address", "Retrieves detailed information about a reserved IPv6 address, including its region and the ID of the Droplet its assigned to.", Writer,
		aliasOpt("g"), displayerType(&displayers.ReservedIPv6{}))
	cmdReservedIPv6Get.Example = `The following example retrieves information about the reserved IPv6 address ` + "`" + `5a11:a:b0a7` + "`" + `: doctl compute reserved-ip get 5a11:a:b0a7`

	cmdRunReservedIPv6Delete := CmdBuilder(cmd, RunReservedIPv6Delete, "delete <reserved-ipv6>", "Permanently delete a reserved IPv6 address", "Permanently deletes a reserved IPv6 address. This is irreversible.", Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunReservedIPv6Delete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the reserved IPv6 address without confirmation")
	cmdRunReservedIPv6Delete.Example = `The following example deletes the reserved IPv6 address ` + "`" + `5a11:a:b0a7` + "`" + `: doctl compute reserved-ip delete 5a11:a:b0a7`

	cmdReservedIPv6List := CmdBuilder(cmd, RunReservedIPv6List, "list", "List all reserved IPv6 addresses on your account", "Retrieves a list of all the reserved IPv6 addresses on your account.", Writer,
		aliasOpt("ls"), displayerType(&displayers.ReservedIPv6{}))
	AddStringFlag(cmdReservedIPv6List, doctl.ArgRegionSlug, "", "", "Retrieves a list of reserved IPv6 addresses in the specified region")
	cmdReservedIPv6List.Example = `The following example lists all reserved IPv6 addresses in the ` + "`" + `nyc1` + "`" + ` region: doctl compute reserved-ip list --region nyc1`

	return cmd
}

// RunReservedIPv6Create runs reserved IP create.
func RunReservedIPv6Create(c *CmdConfig) error {
	ris := c.ReservedIPv6s()

	// ignore errors since we don't know which one is valid
	region, _ := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)

	if region == "" {
		return doctl.NewMissingArgsErr("Region cannot be empty")
	}

	req := &godo.ReservedIPV6CreateRequest{
		Region: region,
	}

	ip, err := ris.Create(req)
	if err != nil {
		fmt.Println(err)
		return err
	}

	item := &displayers.ReservedIPv6{ReservedIPv6s: do.ReservedIPv6s{*ip}}
	return c.Display(item)
}

// RunReservedIPv6Get retrieves a reserved IP's details.
func RunReservedIPv6Get(c *CmdConfig) error {
	ris := c.ReservedIPv6s()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	ip := c.Args[0]

	if len(ip) < 1 {
		return errors.New("Invalid IP address")
	}

	rip, err := ris.Get(ip)
	if err != nil {
		return err
	}

	item := &displayers.ReservedIPv6{ReservedIPv6s: do.ReservedIPv6s{*rip}}
	return c.Display(item)
}

// RunReservedIPv6Delete runs reserved IP delete.
func RunReservedIPv6Delete(c *CmdConfig) error {
	ris := c.ReservedIPv6s()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("reserved IPv6", 1) == nil {
		ip := c.Args[0]
		return ris.Delete(ip)
	}

	return errOperationAborted
}

// RunReservedIPv6List runs reserved IP list.
func RunReservedIPv6List(c *CmdConfig) error {
	ris := c.ReservedIPv6s()

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}

	list, err := ris.List()
	if err != nil {
		return err
	}

	rips := &displayers.ReservedIPv6{ReservedIPv6s: do.ReservedIPv6s{}}
	for _, rip := range list {
		var skip bool
		if region != "" && region != rip.RegionSlug {
			skip = true
		}

		if !skip {
			rips.ReservedIPv6s = append(rips.ReservedIPv6s, rip)
		}
	}

	item := rips
	return c.Display(item)
}
