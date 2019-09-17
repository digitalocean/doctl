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

// FloatingIPAction creates the floating IP action command.
func FloatingIPAction() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "floating-ip-action",
			Short:   "floating IP action commands",
			Long:    "floating IP action commands",
			Aliases: []string{"fipa"},
		},
	}

	CmdBuilder(cmd, RunFloatingIPActionsGet,
		"get <floating-ip> <action-id>", "get floating-ip action", Writer,
		displayerType(&displayers.Action{}))

	CmdBuilder(cmd, RunFloatingIPActionsAssign,
		"assign <floating-ip> <droplet-id>", "assign a floating IP to a droplet", Writer,
		displayerType(&displayers.Action{}))

	CmdBuilder(cmd, RunFloatingIPActionsUnassign,
		"unassign <floating-ip>", "unassign a floating IP to a droplet", Writer,
		displayerType(&displayers.Action{}))

	return cmd
}

// RunFloatingIPActionsGet retrieves an action for a floating IP.
func RunFloatingIPActionsGet(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	fia := c.FloatingIPActions()

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

// RunFloatingIPActionsAssign assigns a floating IP to a droplet.
func RunFloatingIPActionsAssign(c *CmdConfig) error {
	if len(c.Args) != 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	fia := c.FloatingIPActions()

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

// RunFloatingIPActionsUnassign unassigns a floating IP to a droplet.
func RunFloatingIPActionsUnassign(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	ip := c.Args[0]

	fia := c.FloatingIPActions()

	a, err := fia.Unassign(ip)
	if err != nil {
		checkErr(fmt.Errorf("could not unassign IP to droplet: %v", err))
	}

	item := &displayers.Action{Actions: do.Actions{*a}}
	return c.Display(item)
}
