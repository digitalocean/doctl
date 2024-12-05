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
	"fmt"
	"strconv"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/spf13/cobra"
)

// ReservedIPv6Action creates the reserved IPv6 action command.
func ReservedIPv6Action() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "reserved-ipv6-action",
			Short:   "Display commands to associate reserved IPv6 addresses with Droplets",
			Long:    "Reserved IP actions are commands that are used to manage DigitalOcean reserved IPv6 addresses.",
			Aliases: []string{"reserved-ipv6-actions"},
			Hidden:  true,
		},
	}

	cmdReservedIPv6ActionsAssign := CmdBuilder(cmd, RunReservedIPv6ActionsAssign,
		"assign <reserved-ipv6> <droplet-id>", "Assign a reserved IPv6 address to a Droplet", "Assigns a reserved IPv6 address to the specified Droplet.", Writer,
		displayerType(&displayers.Action{}))
	cmdReservedIPv6ActionsAssign.Example = `The following example assigns the reserved IPv6 address ` + "`" + `5a11:a:b0a7` + "`" + ` to a Droplet with the ID ` + "`" + `386734086` + "`" + `: doctl compute reserved-ipv6-action assign 5a11:a:b0a7 386734086`

	cmdReservedIPv6ActionsUnassign := CmdBuilder(cmd, RunReservedIPv6ActionsUnassign,
		"unassign <reserved-ipv6>", "Unassign a reserved IPv6 address from a Droplet", `Unassigns a reserved IPv6 address from a Droplet.`, Writer,
		displayerType(&displayers.Action{}))
	cmdReservedIPv6ActionsUnassign.Example = `The following example unassigns the reserved IPv6 address ` + "`" + `5a11:a:b0a7` + "`" + ` from a resource: doctl compute reserved-ipv6-action unassign 5a11:a:b0a7`

	return cmd
}

// RunReservedIPv6ActionsAssign assigns a reserved IP to a droplet.
func RunReservedIPv6ActionsAssign(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	fia := c.ReservedIPv6Actions()

	dropletID, err := strconv.Atoi(c.Args[1])
	if err != nil {
		return err
	}

	a, err := fia.Assign(ip, dropletID)
	if err != nil {
		checkErr(fmt.Errorf("could not assign IP to droplet: %v", err))
	}

	item := &displayers.Action{Actions: do.Actions{*a}}
	return c.Display(item)
}

// RunReservedIPActionsUnassign unassigns a reserved IP to a droplet.
func RunReservedIPv6ActionsUnassign(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	ip := c.Args[0]

	fia := c.ReservedIPv6Actions()

	a, err := fia.Unassign(ip)
	if err != nil {
		checkErr(fmt.Errorf("could not unassign IP to droplet: %v", err))
	}

	item := &displayers.Action{Actions: do.Actions{*a}}
	return c.Display(item)
}
