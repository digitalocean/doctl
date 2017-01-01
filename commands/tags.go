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
	"fmt"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Tags creates the tag commands heirarchy.
func Tags() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "tag",
			Short: "tag commands",
			Long:  "tag is used to access tag commands",
		},
		DocCategories: []string{"tag"},
		IsIndex:       true,
	}

	CmdBuilder(cmd, RunCmdTagCreate, "create NAME", "create tag", Writer,
		docCategories("tag"))

	CmdBuilder(cmd, RunCmdTagGet, "get NAME", "get tag", Writer,
		docCategories("tag"))

	CmdBuilder(cmd, RunCmdTagList, "list", "list tags", Writer,
		aliasOpt("ls"), docCategories("tag"))

	cmdTagUpdate := CmdBuilder(cmd, RunCmdTagUpdate, "update NAME", "update tag", Writer,
		docCategories("tag"))
	AddStringFlag(cmdTagUpdate, doctl.ArgTagName, "", "", "Tag name",
		requiredOpt())

	cmdRunTagDelete := CmdBuilder(cmd, RunCmdTagDelete, "delete NAME", "delete tag", Writer,
		docCategories("tag"))
	AddBoolFlag(cmdRunTagDelete, doctl.ArgDeleteForce, doctl.ArgShortDeleteForce, false, "Force tag delete")

	return cmd
}

// RunCmdTagCreate runs tag create.
func RunCmdTagCreate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name := c.Args[0]
	ts := c.Tags()

	tcr := &godo.TagCreateRequest{Name: name}
	t, err := ts.Create(tcr)
	if err != nil {
		return err
	}

	return c.Display(&tag{tags: do.Tags{*t}})
}

// RunCmdTagGet runs tag get.
func RunCmdTagGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name := c.Args[0]
	ts := c.Tags()
	t, err := ts.Get(name)
	if err != nil {
		return err
	}

	return c.Display(&tag{tags: do.Tags{*t}})
}

// RunCmdTagList runs tag list.
func RunCmdTagList(c *CmdConfig) error {
	ts := c.Tags()
	tags, err := ts.List()
	if err != nil {
		return err
	}

	return c.Display(&tag{tags: tags})
}

// RunCmdTagUpdate runs tag update.
func RunCmdTagUpdate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name := c.Args[0]

	newName, err := c.Doit.GetString(c.NS, doctl.ArgTagName)
	if err != nil {
		return err
	}

	ts := c.Tags()
	tur := &godo.TagUpdateRequest{Name: newName}
	return ts.Update(name, tur)
}

// RunCmdTagDelete runs tag delete.
func RunCmdTagDelete(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgDeleteForce)
	if err != nil {
		return err
	}

	if force || AskForConfirm("delete tag(s)") == nil {
		for id := range c.Args {
			name := c.Args[id]
			ts := c.Tags()
			if err := ts.Delete(name); err != nil {
				return err
			}
		}
	} else {
		fmt.Errorf("operation aborted")
	}

	return nil
}
