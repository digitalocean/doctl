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
	"github.com/spf13/cobra"
)

func Snapshot() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "snapshot",
			Aliases: []string{"s"},
			Short:   "snapshot commands",
			Long:    "snapshot is used to access snapshot commands",
		},
		IsIndex: true,
	}

	cmdRunSnapshotList := CmdBuilder(cmd, RunSnapshotList, "list [GLOB]", "list snapshots", Writer,
		aliasOpt("ls"), displayerType(&snapshot{}), docCategories("droplet"))
	AddStringFlag(cmdRunSnapshotList, doctl.ArgResourceType, "", "Resource type")

	CmdBuilder(cmd, RunSnapshotGet, "get", "get snapshot", Writer,
		aliasOpt("g"), displayerType(&droplet{}), docCategories("droplet"))

	cmdRunSnapshotDelete := CmdBuilder(cmd, RunSnapshotDelete, "delete", "delete snapshot", Writer,
		aliasOpt("d"), displayerType(&droplet{}), docCategories("droplet"))
	AddBoolFlag(cmdRunSnapshotDelete, doctl.ArgDeleteForce, false, "Force snapshot delete")

	return cmd
}

func RunSnapshotList(c *CmdConfig) error {
	ss := c.Snapshots()

	restype, err := c.Doit.GetString(c.NS, doctl.ArgResourceType)
	if err != nil {
		return err
	}

	if restype == "droplet" {
		list, err := ss.ListDroplet()
		if err != nil {
			return err
		}
		item := &snapshot{snapshots: list}
		return c.Display(item)
	} else if restype == "volume" {
		list, err := ss.ListVolume()
		if err != nil {
			return err
		}
		item := &snapshot{snapshots: list}
		return c.Display(item)
	} else {
		list, err := ss.List()
		if err != nil {
			return err
		}
		item := &snapshot{snapshots: list}
		return c.Display(item)
	}
	return nil
}

func RunSnapshotGet(c *CmdConfig) error {
	snapshotId, errId := getSnapshotIdArg(c.NS, c.Args)
	if errId != nil {
		return errId
	}

	ss := c.Snapshots()

	s, err := ss.Get(snapshotId)
	if err != nil {
		return err
	}

	item := &snapshot{snapshots: do.Snapshots{*s}}

	return c.Display(item)
}

func RunSnapshotDelete(c *CmdConfig) error {
	snapshotId, id_err := getSnapshotIdArg(c.NS, c.Args)
	if id_err != nil {
		return id_err
	}

	force, f_err := c.Doit.GetBool(c.NS, doctl.ArgDeleteForce)
	if f_err != nil {
		return f_err
	}

	ss := c.Snapshots()

	if force || AskForConfirm("delete snapshot(s)") == nil {

		err := ss.Delete(snapshotId)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Operation aborted.")
	}
	return nil
}

func getSnapshotIdArg(ns string, args []string) (string, error) {
	if len(args) != 1 {
		return "", doctl.NewMissingArgsErr(ns)
	}

	return args[0], nil
}
