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
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// ImageAction creates the image action command.
func ImageAction() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "image-action",
			Short: "Display commands to perform actions on images",
			Long:  "The sub-commands of `doctl compute image-actions` can be used to perform actions on images.",
		},
	}
	actionDetail := `

- The unique numeric ID used to identify and reference an image action.
- The status of the image action. This will be either ` + "`" + `in-progress` + "`" + `, ` + "`" + `completed` + "`" + `, or ` + "`" + `errored` + "`" + `.
- A time value given in ISO8601 combined date and time format that represents when the action was initiated.
- A time value given in ISO8601 combined date and time format that represents when the action was completed.
- The resource ID, which is a unique identifier for the resource that the action is associated with.
- The type of resource that the action is associated with.
- The region where the action occurred.
- The slug for the region where the action occurred.
`
	cmdImageActionsGet := CmdBuilder(cmd, RunImageActionsGet,
		"get <image-id>", "Retrieve the status of an image action", `Use this command to retrieve the status of an image action, including the following details:`+actionDetail, Writer,
		displayerType(&displayers.Action{}))
	AddIntFlag(cmdImageActionsGet, doctl.ArgActionID, "", 0, "action id", requiredOpt())

	cmdImageActionsTransfer := CmdBuilder(cmd, RunImageActionsTransfer,
		"transfer <image-id>", "Transfer an image to another datacenter region", `Use this command to transfer an image to a different datacenter region. Also outputs the following details:`+actionDetail, Writer,
		displayerType(&displayers.Action{}))
	AddStringFlag(cmdImageActionsTransfer, doctl.ArgRegionSlug, "", "", "region", requiredOpt())
	AddBoolFlag(cmdImageActionsTransfer, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	return cmd
}

// RunImageActionsGet retrieves an action for an image.
func RunImageActionsGet(c *CmdConfig) error {
	ias := c.ImageActions()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	imageID, err := strconv.Atoi(c.Args[0])
	if err != nil {
		return err
	}

	actionID, err := c.Doit.GetInt(c.NS, doctl.ArgActionID)
	if err != nil {
		return err
	}

	a, err := ias.Get(imageID, actionID)
	if err != nil {
		return err
	}

	item := &displayers.Action{Actions: do.Actions{*a}}
	return c.Display(item)
}

// RunImageActionsTransfer an image.
func RunImageActionsTransfer(c *CmdConfig) error {
	ias := c.ImageActions()

	err := ensureOneArg(c)
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(c.Args[0])
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}

	req := &godo.ActionRequest{
		"type":   "transfer",
		"region": region,
	}

	a, err := ias.Transfer(id, req)
	if err != nil {
		checkErr(fmt.Errorf("Could not transfer image: %v", err))
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
