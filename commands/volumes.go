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
	"github.com/digitalocean/godo"
	"github.com/dustin/go-humanize"
	"github.com/gobwas/glob"
	"github.com/spf13/cobra"
)

// Volume creates the Volume command
func Volume() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "volume",
			Short: "Display commands to manage block storage volumes",
			Long: `The sub-commands of ` + "`" + `doctl compute volume` + "`" + ` manage your block storage volumes.

Block storage volumes provide expanded storage capacity for your Droplets, ranging in size from 1GiB to 16TiB.

Volumes function as raw block devices, meaning they appear to the operating system as locally attached storage which can be formatted using any filesystem supported by the OS. They can be moved between Droplets located in the same region as the volume.`,
		},
	}

	cmdRunVolumeList := CmdBuilder(cmd, RunVolumeList, "list", "List block storage volumes by ID", `Lists all of the block storage volumes on your account.`, Writer,
		aliasOpt("ls"), displayerType(&displayers.Volume{}))
	AddStringFlag(cmdRunVolumeList, doctl.ArgRegionSlug, "", "", "Filter's volumes by the specified region")
	cmdRunVolumeList.Example = `The following example retrieves a list of volumes on your account in the ` + "`" + `nyc1` + "`" + ` region. The command also uses the ` + "`" + `--format` + "`" + ` flag to return only the name and size of each volume: doctl compute volume list --region nyc1 --format Name,Size`

	cmdVolumeCreate := CmdBuilder(cmd, RunVolumeCreate, "create <volume-name>", "Create a block storage volume", `Creates a block storage volume on your account.

You can use flags to specify the volume size, region, description, filesystem type, tags, and to create a volume from an existing volume snapshot.

Use the `+"`"+`doctl compute volume-action attach <volume-id> <droplet-id>`+"`"+` command to attach a new volume to a Droplet.`, Writer,
		aliasOpt("c"), displayerType(&displayers.Volume{}))
	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeSize, "", "4TiB", "Volume size",
		requiredOpt())
	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeDesc, "", "", "A description of the volume")
	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeRegion, "", "", "The volume's region. Not compatible with the `--snapshot` flag")
	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeSnapshot, "", "", "Creates a volume from the specified snapshot ID. Not compatible with the `--region` flag")
	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeFilesystemType, "", "", "The volume's filesystem type: ext4 or xfs. If not specified, the volume is left unformatted")
	AddStringFlag(cmdVolumeCreate, doctl.ArgVolumeFilesystemLabel, "", "", "The volume's filesystem label")
	AddStringSliceFlag(cmdVolumeCreate, doctl.ArgTag, "", []string{}, "A comma-separated list of tags to apply to the volume. For example, `--tag frontend` or `--tag frontend,backend`")
	cmdVolumeCreate.Example = `The following example creates a 4TiB volume named ` + "`" + `example-volume` + "`" + ` in the ` + "`" + `nyc1` + "`" + ` region. The command also applies two tags to the volume: doctl compute volume create example-volume --region nyc1 --size 4TiB --tag frontend,backend`

	cmdRunVolumeDelete := CmdBuilder(cmd, RunVolumeDelete, "delete <volume-id>", "Delete a block storage volume", `Deletes a block storage volume by ID, destroying all of its data and removing it from your account. This is irreversible.`, Writer,
		aliasOpt("d", "rm"))
	AddBoolFlag(cmdRunVolumeDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Delete the volume without prompting for confirmation")
	cmdRunVolumeDelete.Example = `The following example deletes a volume with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute volume delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdVolumeGet := CmdBuilder(cmd, RunVolumeGet, "get <volume-id>", "Retrieve an existing block storage volume", `Retrieves information about a block storage volume.`, Writer, aliasOpt("g"),
		displayerType(&displayers.Volume{}))
	cmdVolumeGet.Example = `The following example retrieves information about a volume with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute volume get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdRunVolumeSnapshot := CmdBuilder(cmd, RunVolumeSnapshot, "snapshot <volume-id>", "Create a block storage volume snapshot", `Creates a snapshot of a block storage volume by ID.

You can use a block storage volume snapshot ID as a flag with `+"`"+`doctl volume create`+"`"+` to create a new block storage volume with the same data as the volume the snapshot was taken from.`, Writer,
		aliasOpt("s"), displayerType(&displayers.Volume{}))
	AddStringFlag(cmdRunVolumeSnapshot, doctl.ArgSnapshotName, "", "", "The snapshot name", requiredOpt())
	AddStringFlag(cmdRunVolumeSnapshot, doctl.ArgSnapshotDesc, "", "", "A description of the snapshot")
	AddStringSliceFlag(cmdRunVolumeSnapshot, doctl.ArgTag, "", []string{}, "A comma-separate list of tags to apply to the snapshot. For example, `--tag frontend` or `--tag frontend,backend`")
	cmdRunVolumeSnapshot.Example = `The following example creates a snapshot of a volume with the UUID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl compute volume snapshot f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --snapshot-name example-snapshot --tag frontend,backend`

	return cmd

}

// RunVolumeList returns a list of volumes.
func RunVolumeList(c *CmdConfig) error {

	al := c.Volumes()

	region, err := c.Doit.GetString(c.NS, doctl.ArgRegionSlug)
	if err != nil {
		return nil
	}

	matches := make([]glob.Glob, 0, len(c.Args))
	for _, globStr := range c.Args {
		g, err := glob.Compile(globStr)
		if err != nil {
			return fmt.Errorf("Unknown glob %q", globStr)
		}

		matches = append(matches, g)
	}

	list, err := al.List()
	if err != nil {
		return err
	}
	var matchedList []do.Volume

	for _, volume := range list {
		var skip = true
		if len(matches) == 0 {
			skip = false
		} else {
			for _, m := range matches {
				if m.Match(volume.ID) {
					skip = false
				}
				if m.Match(volume.Name) {
					skip = false
				}
			}
		}

		if !skip && region != "" {
			if region != volume.Region.Slug {
				skip = true
			}
		}

		if !skip {
			matchedList = append(matchedList, volume)
		}
	}
	item := &displayers.Volume{Volumes: matchedList}
	return c.Display(item)
}

// RunVolumeCreate creates a volume.
func RunVolumeCreate(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	name := c.Args[0]

	sizeStr, err := c.Doit.GetString(c.NS, doctl.ArgVolumeSize)
	if err != nil {
		return err
	}
	size, err := humanize.ParseBytes(sizeStr)
	if err != nil {
		return err
	}

	desc, err := c.Doit.GetString(c.NS, doctl.ArgVolumeDesc)
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, doctl.ArgVolumeRegion)
	if err != nil {
		return err
	}

	snapshotID, err := c.Doit.GetString(c.NS, doctl.ArgVolumeSnapshot)
	if err != nil {
		return err
	}

	if region == "" && snapshotID == "" {
		errorMsg := fmt.Sprintf("%s.%s || %s.%s", c.NS, doctl.ArgVolumeRegion, c.NS, doctl.ArgVolumeSnapshot)
		return doctl.NewMissingArgsErr(errorMsg)
	}

	fsType, err := c.Doit.GetString(c.NS, doctl.ArgVolumeFilesystemType)
	if err != nil {
		return err
	}
	fsLabel, err := c.Doit.GetString(c.NS, doctl.ArgVolumeFilesystemLabel)
	if err != nil {
		return err
	}

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return err
	}

	var createVolume godo.VolumeCreateRequest

	createVolume.Name = name
	createVolume.SizeGigaBytes = int64(size / (1 << 30))
	createVolume.Description = desc
	createVolume.Region = region
	createVolume.SnapshotID = snapshotID
	createVolume.FilesystemType = fsType
	createVolume.FilesystemLabel = fsLabel
	createVolume.Tags = tags

	al := c.Volumes()

	d, err := al.CreateVolume(&createVolume)
	if err != nil {
		return err
	}
	item := &displayers.Volume{Volumes: []do.Volume{*d}}
	return c.Display(item)

}

// RunVolumeDelete deletes a volume.
func RunVolumeDelete(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)

	}

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("volume", 1) == nil {
		id := c.Args[0]
		return c.Volumes().DeleteVolume(id)
	}
	return errOperationAborted
}

// RunVolumeGet gets a volume.
func RunVolumeGet(c *CmdConfig) error {
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)

	}
	id := c.Args[0]
	al := c.Volumes()
	d, err := al.Get(id)
	if err != nil {
		return err
	}
	item := &displayers.Volume{Volumes: []do.Volume{*d}}
	return c.Display(item)
}

// RunVolumeSnapshot creates a snapshot of a volume
func RunVolumeSnapshot(c *CmdConfig) error {
	var err error
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	id := c.Args[0]

	name, err := c.Doit.GetString(c.NS, doctl.ArgSnapshotName)
	if err != nil {
		return err
	}

	desc, err := c.Doit.GetString(c.NS, doctl.ArgSnapshotDesc)
	if err != nil {
		return nil
	}

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return err
	}

	req := &godo.SnapshotCreateRequest{
		VolumeID:    id,
		Name:        name,
		Description: desc,
		Tags:        tags,
	}

	_, err = c.Volumes().CreateSnapshot(req)
	return err
}
