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

// ReservedIPAction creates the reserved IP action command.
func ReservedIPAction() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "reserved-ip-action",
			Short:   "Display commands to associate reserved IP addresses with Droplets",
			Long:    "Reserved IP actions are commands that are used to manage DigitalOcean reserved IP addresses.",
			Aliases: []string{"fipa", "floating-ip-action", "floating-ip-actions", "reserved-ip-actions"},
		},
	}
	flipActionDetail := `

	- The unique numeric ID used to identify and reference a reserved IP action.
	- The status of the reserved IP action. This will be either "in-progress", "completed", or "errored".
	- A time value given in ISO8601 combined date and time format that represents when the action was initiated.
	- A time value given in ISO8601 combined date and time format that represents when the action was completed.
	- The resource ID, which is a unique identifier for the resource that the action is associated with.
	- The type of resource that the action is associated with.
	- The region where the action occurred.
	- The slug for the region where the action occurred.
`
	CmdBuilder(cmd, RunReservedIPActionsGet,
		"get <reserved-ip> <action-id>", "Retrieve the status of a reserved IP action", `Use this command to retrieve the status of a reserved IP action. Outputs the following information:`+flipActionDetail, Writer,
		displayerType(&displayers.Action{}))

	CmdBuilder(cmd, RunReservedIPActionsAssign,
		"assign <reserved-ip> <droplet-id>", "Assign a reserved IP address to a Droplet", "Use this command to assign a reserved IP address to a Droplet by specifying the `droplet_id`.", Writer,
		displayerType(&displayers.Action{}))

	CmdBuilder(cmd, RunReservedIPActionsUnassign,
		"unassign <reserved-ip>", "Unassign a reserved IP address from a Droplet", `Use this command to unassign a reserved IP address from a Droplet. The reserved IP address will be reserved in the region but not assigned to a Droplet.`, Writer,
		displayerType(&displayers.Action{}))

	return cmd
}

// RunReservedIPActionsGet retrieves an action for a reserved IP.
func RunReservedIPActionsGet(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	fia := c.ReservedIPActions()

	actionID, err := strconv.Atoi(c.Args[1])
	if err != nil {
		return err
	}

	a, err := fia.Get(ip, actionID)
	if err != nil {
		return err
	}

	item := &displayers.Action{Actions: do.Actions{*a}}
	return c.Display(item)
}

// RunReservedIPActionsAssign assigns a reserved IP to a droplet.
func RunReservedIPActionsAssign(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	fia := c.ReservedIPActions()

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
func RunReservedIPActionsUnassign(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	ip := c.Args[0]

	fia := c.ReservedIPActions()

	a, err := fia.Unassign(ip)
	if err != nil {
		checkErr(fmt.Errorf("could not unassign IP to droplet: %v", err))
	}

	item := &displayers.Action{Actions: do.Actions{*a}}
	return c.Display(item)
}
