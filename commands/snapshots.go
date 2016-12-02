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
	"github.com/digitalocean/doctl"
	"github.com/spf13/cobra"
)

func Snapshot() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "snapshot",
			Aliases: []string{"d"},
			Short:   "snapshot commands",
			Long:    "snapshot is used to access snapshot commands",
		},
		IsIndex: true,
	}

	cmdRunSnapshotList := CmdBuilder(cmd, RunSnapshotList, "list [GLOB]", "list snapshots", Writer,
		aliasOpt("ls"), displayerType(&snapshot{}), docCategories("droplet"))
	AddStringFlag(cmdRunSnapshotList, doctl.ArgResourceType, "", "Resource type")

	/*CmdBuilder(cmd, RunSnapshotListVolume, "lvolume", "list volume", Writer,
		aliasOpt("lsv"), displayerType(&snapshot{}), docCategories("droplet"))

	CmdBuilder(cmd, RunSnapshotListDroplet, "ldroplet", "list droplet", Writer,
		aliasOpt("lsd"), displayerType(&snapshot{}), docCategories("droplet"))

		cmdRunDropletGet := CmdBuilder(cmd, RunSnapshotGet, "get", "get snapshot", Writer,
		aliasOpt("g"), displayerType(&droplet{}), docCategories("droplet"))*/

	CmdBuilder(cmd, RunSnapshotDelete, "delete", "delete snapshot", Writer,
		aliasOpt("d"), displayerType(&droplet{}), docCategories("droplet"))

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

/*func RunSnapshotListVolume(c *CmdConfig) error {
	ss := c.Snapshots()

	list, err := ss.ListVolume()
	if err != nil {
		return err
	}
	item := &snapshot{snapshots: list}
	return c.Display(item)
}

func RunSnapshotListDroplet(c *CmdConfig) error {
	ss := c.Snapshots()

	list, err := ss.ListDroplet()
	if err != nil {
		return err
	}
	item := &snapshot{snapshots: list}
	return c.Display(item)
}*/

/*func RunSnapshotGet(c *CmdConfig) error {
	snapshotId, err := getSnapshotIDArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	ss := c.Snapshots()

	s, err := ss.Get(snapshotId)
	if err != nil {
		return err
	}

	//item := &snapshot{snapshots: do.Snapshot{*s}}

	return nil
}*/

func RunSnapshotDelete(c *CmdConfig) error {
	snapshotId, err := getSnapshotStringArg(c.NS, c.Args)
	if err != nil {
		return err
	}

	ss := c.Snapshots()

	derr := ss.Delete(snapshotId)
	if derr != nil {
		return derr
	}

	return nil
}

func getSnapshotStringArg(ns string, args []string) (string, error) {
	if len(args) != 1 {
		return "", doctl.NewMissingArgsErr(ns)
	}

	return args[0], nil
}
