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

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/gobwas/glob"
	"github.com/spf13/cobra"
)

// Snapshot creates the snapshot command
func Snapshot() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "snapshot",
			Aliases: []string{"s"},
			Short:   "snapshot commands",
			Long:    "snapshot is used to access snapshot commands",
		},
	}

	cmdRunSnapshotList := CmdBuilder(cmd, RunSnapshotList, "list [glob]", "list snapshots", Writer,
		aliasOpt("ls"), displayerType(&displayers.Snapshot{}))
	AddStringFlag(cmdRunSnapshotList, doctl.ArgResourceType, "", "", "Resource type")
	AddStringFlag(cmdRunSnapshotList, doctl.ArgRegionSlug, "", "", "Snapshot region")

	CmdBuilder(cmd, RunSnapshotGet, "get <snapshot-id>...", "get snapshots", Writer,
		aliasOpt("g"), displayerType(&displayers.Droplet{}))

	cmdRunSnapshotDelete := CmdBuilder(cmd, RunSnapshotDelete, "delete <snapshot-id>...", "delete snapshots", Writer,
		aliasOpt("d"), displayerType(&displayers.Droplet{}))
	AddBoolFlag(cmdRunSnapshotDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Force snapshot delete")

	return cmd
}

// RunSnapshotList returns a list of snapshots
func RunSnapshotList(c *CmdConfig) error {
	var err error
	ss := c.Snapshots()

	restype, err := c.Doit.GetString(c.NS, doctl.ArgResourceType)
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return err
	}

	matches := []glob.Glob{}
	for _, globStr := range c.Args {
		g, err := glob.Compile(globStr)
		if err != nil {
			return fmt.Errorf("unknown glob %q", globStr)
		}

		matches = append(matches, g)
	}

	var matchedList []do.Snapshot
	var list []do.Snapshot

	if restype == "droplet" {
		list, err = ss.ListDroplet()
		if err != nil {
			return err
		}
	} else if restype == "volume" {
		list, err = ss.ListVolume()
		if err != nil {
			return err
		}
	} else {
		list, err = ss.List()
		if err != nil {
			return err
		}
	}

	for _, snapshot := range list {
		var skip = true
		if len(matches) == 0 {
			skip = false
		} else {
			for _, m := range matches {
				if m.Match(snapshot.ID) {
					skip = false
				}
				if m.Match(snapshot.Name) {
					skip = false
				}
			}
		}

		if !skip && region != "" {
			for _, snapshotRegion := range snapshot.Regions {
				if region != snapshotRegion {
					skip = true
				} else {
					skip = false
					break
				}
			}

		}

		if !skip {
			matchedList = append(matchedList, snapshot)
		}
	}

	item := &displayers.Snapshot{Snapshots: matchedList}
	return c.Display(item)
}

// RunSnapshotGet returns a snapshot
func RunSnapshotGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	ss := c.Snapshots()
	ids := c.Args

	var matchedList []do.Snapshot

	for _, id := range ids {
		s, err := ss.Get(id)
		if err != nil {
			return err
		}
		matchedList = append(matchedList, *s)
	}
	item := &displayers.Snapshot{Snapshots: matchedList}
	return c.Display(item)
}

// RunSnapshotDelete destroys snapshot(s) by id
func RunSnapshotDelete(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	ss := c.Snapshots()
	ids := c.Args

	if force || AskForConfirm("delete snapshot(s)") == nil {
		for _, id := range ids {
			err := ss.Delete(id)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("operation aborted")
	}
	return nil
}
