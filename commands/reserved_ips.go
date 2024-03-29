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
	"errors"
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// ReservedIP creates the command hierarchy for reserved ips.
func ReservedIP() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "reserved-ip",
			Short: "Display commands to manage reserved IP addresses",
			Long: `The sub-commands of ` + "`" + `doctl compute reserved-ip` + "`" + ` manage reserved IP addresses.
Reserved IPs are publicly-accessible static IP addresses that you can to one of your Droplets. They can be used to create highly available setups or other configurations requiring movable addresses. Reserved IPs are bound to the regions they are created in.`,
			Aliases: []string{"fip", "floating-ip", "floating-ips", "reserved-ips"},
		},
	}

	cmdReservedIPCreate := CmdBuilder(cmd, RunReservedIPCreate, "create", "Create a new reserved IP address", `Creates a new reserved IP address.

Reserved IP addresses can either be assigned to Droplets or held in the region they were created in on your account, but because of the IPv4 address shortage, unassigned reserved IP addresses incur charges.`, Writer,
		aliasOpt("c"), displayerType(&displayers.ReservedIP{}))
	AddStringFlag(cmdReservedIPCreate, doctl.ArgRegionSlug, "", "",
		fmt.Sprintf("The region where to create the reserved IP address. Cannot be used with the `--%s` flag.",
			doctl.ArgDropletID))
	AddStringFlag(cmdReservedIPCreate, doctl.ArgProjectID, "", "",
		fmt.Sprintf("The ID of the project to assign the IP address. When excluded, the address is assigned to your default project. When using the `--%s` flag, it is assigned to the project containing the Droplet.",
			doctl.ArgDropletID))
	AddIntFlag(cmdReservedIPCreate, doctl.ArgDropletID, "", 0,
		fmt.Sprintf("The ID of the Droplet to assign the reserved IP to. Cannot be used with the `--%s` flag.",
			doctl.ArgRegionSlug))
	cmdReservedIPCreate.Example = `The following example creates a reserved IP address in the ` + "`" + `nyc1` + "`" + ` region and assigns it to a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute reserved-ip create --region nyc1 --droplet-id 386734086`

	cmdReservedIPGet := CmdBuilder(cmd, RunReservedIPGet, "get <reserved-ip>", "Retrieve information about a reserved IP address", "Retrieves detailed information about a reserved IP address, including its region and the ID of the Droplet its assigned to.", Writer,
		aliasOpt("g"), displayerType(&displayers.ReservedIP{}))
	cmdReservedIPGet.Example = `The following example retrieves information about the reserved IP address ` + "`" + `203.0.113.25` + "`" + `: doctl compute reserved-ip get 203.0.113.25`

	cmdRunReservedIPDelete := CmdBuilder(cmd, RunReservedIPDelete, "delete <reserved-ip>", "Permanently delete a reserved IP address", "Permanently deletes a reserved IP address. This is irreversible.", Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunReservedIPDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the reserved IP address without confirmation")
	cmdRunReservedIPDelete.Example = `The following example deletes the reserved IP address ` + "`" + `203.0.113.25` + "`" + `: doctl compute reserved-ip delete 203.0.113.25`

	cmdReservedIPList := CmdBuilder(cmd, RunReservedIPList, "list", "List all reserved IP addresses on your account", "Retrieves a list of all the reserved IP addresses on your account.", Writer,
		aliasOpt("ls"), displayerType(&displayers.ReservedIP{}))
	AddStringFlag(cmdReservedIPList, doctl.ArgRegionSlug, "", "", "Retrieves a list of reserved IP addresses in the specified region")
	cmdReservedIPList.Example = `The following example lists all reserved IP addresses in the ` + "`" + `nyc1` + "`" + ` region: doctl compute reserved-ip list --region nyc1`

	return cmd
}

// RunReservedIPCreate runs reserved IP create.
func RunReservedIPCreate(c *CmdConfig) error {
	ris := c.ReservedIPs()

	// ignore errors since we don't know which one is valid
	region, _ := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	dropletID, _ := c.Doit.GetInt(c.NS, doctl.ArgDropletID)
	projectID, _ := c.Doit.GetString(c.NS, doctl.ArgProjectID)

	if region == "" && dropletID == 0 {
		return doctl.NewMissingArgsErr("Region and Droplet ID can't both be blank.")
	}

	if region != "" && dropletID != 0 {
		return fmt.Errorf("Only one of `--%s` or `--%s` may be specified when creating a reserved IP address.", doctl.ArgRegionSlug, doctl.ArgDropletID)
	}

	if projectID != "" && dropletID != 0 {
		return fmt.Errorf("Only one of `--%s` or `--%s` may be specified when creating a reserved IP address.", doctl.ArgProjectID, doctl.ArgDropletID)
	}

	req := &godo.ReservedIPCreateRequest{
		Region:    region,
		DropletID: dropletID,
		ProjectID: projectID,
	}

	ip, err := ris.Create(req)
	if err != nil {
		fmt.Println(err)
		return err
	}

	item := &displayers.ReservedIP{ReservedIPs: do.ReservedIPs{*ip}}
	return c.Display(item)
}

// RunReservedIPGet retrieves a reserved IP's details.
func RunReservedIPGet(c *CmdConfig) error {
	ris := c.ReservedIPs()

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

	item := &displayers.ReservedIP{ReservedIPs: do.ReservedIPs{*rip}}
	return c.Display(item)
}

// RunReservedIPDelete runs reserved IP delete.
func RunReservedIPDelete(c *CmdConfig) error {
	ris := c.ReservedIPs()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("reserved IP", 1) == nil {
		ip := c.Args[0]
		return ris.Delete(ip)
	}

	return errOperationAborted
}

// RunReservedIPList runs reserved IP create.
func RunReservedIPList(c *CmdConfig) error {
	ris := c.ReservedIPs()

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}

	list, err := ris.List()
	if err != nil {
		return err
	}

	rips := &displayers.ReservedIP{ReservedIPs: do.ReservedIPs{}}
	for _, rip := range list {
		var skip bool
		if region != "" && region != rip.Region.Slug {
			skip = true
		}

		if !skip {
			rips.ReservedIPs = append(rips.ReservedIPs, rip)
		}
	}

	item := rips
	return c.Display(item)
}
