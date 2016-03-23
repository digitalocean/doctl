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
	"errors"
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// FloatingIP creates the command heirarchy for floating ips.
func FloatingIP() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "floating-ip",
			Short:   "floating IP commands",
			Long:    "floating-ip is used to access commands on floating IPs",
			Aliases: []string{"fip"},
		},
		DocCategories: []string{"floatingip"},
		IsIndex:       true,
	}

	cmdFloatingIPCreate := CmdBuilder(cmd, RunFloatingIPCreate, "create", "create a floating IP", Writer,
		aliasOpt("c"), displayerType(&floatingIP{}), docCategories("floatingip"))
	AddStringFlag(cmdFloatingIPCreate, doit.ArgRegionSlug, "",
		fmt.Sprintf("Region where to create the floating IP. (mutually exclusive with %s)",
			doit.ArgDropletID))
	AddIntFlag(cmdFloatingIPCreate, doit.ArgDropletID, 0,
		fmt.Sprintf("ID of the droplet to assign the IP to. (mutually exclusive with %s)",
			doit.ArgRegionSlug))

	CmdBuilder(cmd, RunFloatingIPGet, "get <floating-ip>", "get the details of a floating IP", Writer,
		aliasOpt("g"), displayerType(&floatingIP{}), docCategories("floatingip"))

	CmdBuilder(cmd, RunFloatingIPDelete, "delete <floating-ip>", "delete a floating IP address", Writer, aliasOpt("d"))

	cmdFloatingIPList := CmdBuilder(cmd, RunFloatingIPList, "list", "list all floating IP addresses", Writer,
		aliasOpt("ls"), displayerType(&floatingIP{}), docCategories("floatingip"))
	AddStringFlag(cmdFloatingIPList, doit.ArgRegionSlug, "", "Floating IP region")

	return cmd
}

// RunFloatingIPCreate runs floating IP create.
func RunFloatingIPCreate(c *CmdConfig) error {
	fis := c.FloatingIPs()

	// ignore errors since we don't know which one is valid
	region, _ := c.Doit.GetString(c.NS, doit.ArgRegionSlug)
	dropletID, _ := c.Doit.GetInt(c.NS, doit.ArgDropletID)

	if region == "" && dropletID == 0 {
		return doit.NewMissingArgsErr("region and droplet id can't both be blank")
	}

	if region != "" && dropletID != 0 {
		return fmt.Errorf("specify region or droplet id when creating a floating ip")
	}

	req := &godo.FloatingIPCreateRequest{
		Region:    region,
		DropletID: dropletID,
	}

	ip, err := fis.Create(req)
	if err != nil {
		fmt.Println(err)
		return err
	}

	item := &floatingIP{floatingIPs: do.FloatingIPs{*ip}}
	return c.Display(item)
}

// RunFloatingIPGet retrieves a floating IP's details.
func RunFloatingIPGet(c *CmdConfig) error {
	fis := c.FloatingIPs()

	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	if len(ip) < 1 {
		return errors.New("invalid ip address")
	}

	fip, err := fis.Get(ip)
	if err != nil {
		return err
	}

	item := &floatingIP{floatingIPs: do.FloatingIPs{*fip}}
	return c.Display(item)
}

// RunFloatingIPDelete runs floating IP delete.
func RunFloatingIPDelete(c *CmdConfig) error {
	fis := c.FloatingIPs()

	if len(c.Args) != 1 {
		return doit.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	return fis.Delete(ip)
}

// RunFloatingIPList runs floating IP create.
func RunFloatingIPList(c *CmdConfig) error {
	fis := c.FloatingIPs()

	region, err := c.Doit.GetString(c.NS, doit.ArgRegionSlug)
	if err != nil {
		return err
	}

	list, err := fis.List()
	if err != nil {
		return err
	}

	fips := &floatingIP{floatingIPs: do.FloatingIPs{}}
	for _, fip := range list {
		var skip bool
		if region != "" && region != fip.Region.Slug {
			skip = true
		}

		if !skip {
			fips.floatingIPs = append(fips.floatingIPs, fip)
		}
	}

	item := fips
	return c.Display(item)
}
