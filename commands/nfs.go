/*
Copyright 2025 The Doctl Authors All rights reserved.
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

// Nfs creates a new command that groups the subcommands for managing DigitalOcean NFS.
func Nfs() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "nfs",
			Aliases: []string{},
			Short:   "Display commands to manage network file storage",
			Long:    "The subcommands of `doctl nfs` allow you to access and manage Network File Storage.",
			GroupID: manageResourcesGroup,
		},
	}

	cmdNfsCreate := CmdBuilder(cmd, nfsCreate, "create [flags]", "Create an NFS share", "Create an NFS share with the provided config.", Writer)
	AddStringFlag(cmdNfsCreate, "name", "n", "", "the name of the NFS share", requiredOpt())
	AddStringFlag(cmdNfsCreate, "region", "r", "", "the region where the NFS share resides", requiredOpt())
	AddStringFlag(cmdNfsCreate, "size", "s", "", "the size of the NFS share in GiB", requiredOpt())
	AddStringSliceFlag(cmdNfsCreate, "vpc-ids", "", nil, "the list of VPC IDs that should be able to access the share", requiredOpt())
	cmdNfsCreate.Example =
		`doctl nfs create --name sammy-nfs-share --region 'atl1' --size 50 --vpc-ids 74922c16-5466-42a5-ac58-0e8069918b6b
doctl nfs create --name my-nfs-share --region 'nyc2' --size 100 --vpc-ids 74922c16-5466-42a5-ac58-0e8069918b6b`

	cmdNfsGet := CmdBuilder(cmd, nfsGet, "get [flags]", "Get an NFS share by ID", "Get an NFS share with the given ID and region.", Writer, displayerType(&displayers.Nfs{}))
	AddStringFlag(cmdNfsGet, "id", "", "", "the ID of the NFS share", requiredOpt())
	AddStringFlag(cmdNfsGet, "region", "r", "", "the region where the NFS share resides", requiredOpt())
	cmdNfsGet.Example =
		`doctl nfs get --region 'atl1' --id b050990d-4337-4a9d-9c8d-9f759a83936a
doctl nfs get --region 'atl1' --id b050990d-4337-4a9d-9c8d-9f759a83936a --format ID,Name,Status`

	cmdNfsList := CmdBuilder(cmd, nfsList, "list [flags]", "List all NFS shares by region", "List all NFS shares in the given region.", Writer, aliasOpt("ls"), displayerType(&displayers.Nfs{}))
	AddStringFlag(cmdNfsList, "region", "r", "", "the region where the NFS shares reside", requiredOpt())
	cmdNfsList.Example =
		`doctl nfs list --region 'atl1'
doctl nfs list --region 'atl1' --format ID,Name,Size,Status`

	cmdNfsDelete := CmdBuilder(cmd, nfsDelete, "delete [flags]", "Delete an NFS share by ID", "Delete an NFS share with the given ID and region.", Writer, aliasOpt("rm"))
	AddStringFlag(cmdNfsDelete, "id", "", "", "the ID of the NFS share", requiredOpt())
	AddStringFlag(cmdNfsDelete, "region", "r", "", "the region where the NFS share resides", requiredOpt())
	cmdNfsDelete.Example =
		`doctl nfs delete --region 'atl1' --id b050990d-4337-4a9d-9c8d-9f759a83936a`

	cmdNfsResize := CmdBuilder(cmd, nfsResize, "resize [flags]", "Resize an NFS share", "Resize an NFS share with the given ID and region.", Writer)
	AddStringFlag(cmdNfsResize, "id", "", "", "the ID of the NFS share", requiredOpt())
	AddStringFlag(cmdNfsResize, "region", "r", "", "the region where the NFS share resides", requiredOpt())
	AddStringFlag(cmdNfsResize, "size", "s", "", "the size of the NFS share in GiB", requiredOpt())
	AddBoolFlag(cmdNfsResize, doctl.ArgCommandWait, "", false, "Wait for action to complete")
	cmdNfsResize.Example =
		`doctl nfs resize --region 'atl1' --id b050990d-4337-4a9d-9c8d-9f759a83936a --size 1024`

	cmd.AddCommand(nfsSnapshots())

	return cmd
}

func nfsSnapshots() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "snapshot",
			Short: "Display commands for NFS share's snapshots",
			Long:  "The commands under `doctl nfs snapshot` are for managing NFS share's snapshots.",
		},
	}

	cmdNfsSnapshotCreate := CmdBuilder(cmd, nfsSnapshotCreate, "create [flags]", "Creates a snapshot of the NFS share", "Creates a snapshot of the NFS share with the given share ID.", Writer, overrideCmdNS("nfs-snapshot"))
	cmdNfsSnapshotCreate.Example = `The following example creates a snapshot for a specified NFS share: doctl nfs snapshot create --name my-snapshot --region 'atl1' --share-id 0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d`
	AddStringFlag(cmdNfsSnapshotCreate, "name", "n", "", "the name of the NFS snapshot", requiredOpt())
	AddStringFlag(cmdNfsSnapshotCreate, "share-id", "", "", "the ID of the NFS share to snapshot", requiredOpt())
	AddStringFlag(cmdNfsSnapshotCreate, "region", "r", "", "the region where the NFS share resides", requiredOpt())
	AddBoolFlag(cmdNfsSnapshotCreate, doctl.ArgCommandWait, "", false, "Wait for action to complete")

	cmdNfsSnapshotGet := CmdBuilder(cmd, nfsSnapshotGet, "get [flags]", "Get an NFS snapshot by ID", "Get an NFS snapshot with the given ID and region.", Writer, displayerType(&displayers.NfsSnapshot{}), overrideCmdNS("nfs-snapshot"))
	AddStringFlag(cmdNfsSnapshotGet, "id", "", "", "the ID of the NFS snapshot", requiredOpt())
	AddStringFlag(cmdNfsSnapshotGet, "region", "r", "", "the region where the NFS snapshot resides", requiredOpt())
	cmdNfsSnapshotGet.Example =
		`doctl nfs snapshot get --region 'atl1' --id b050990d-4337-4a9d-9c8d-9f759a83936a
doctl nfs snapshot get --region 'atl1' --id b050990d-4337-4a9d-9c8d-9f759a83936a --format ID,Name,Status`

	cmdNfsSnapshotList := CmdBuilder(cmd, nfsSnapshotList, "list [flags]", "List all NFS snapshots by region", "List all NFS snapshots in the given region.", Writer, aliasOpt("ls"), displayerType(&displayers.NfsSnapshot{}), overrideCmdNS("nfs-snapshot"))
	AddStringFlag(cmdNfsSnapshotList, "share-id", "", "", "the NFS share ID to which snapshots belong")
	AddStringFlag(cmdNfsSnapshotList, "region", "r", "", "the region where the NFS shares reside", requiredOpt())
	cmdNfsSnapshotList.Example =
		`doctl nfs snapshot list --region 'atl1'
doctl nfs snapshot list --region 'atl1' --share-id b050990d-4337-4a9d-9c8d-9f759a83936
doctl nfs snapshot list --region 'atl1' --format ID,Name,Status,ShareID`

	cmdNfsSnapshotDelete := CmdBuilder(cmd, nfsSnapshotDelete, "delete [flags]", "Delete an NFS share by ID", "Delete an NFS share with the given ID and region.", Writer, aliasOpt("rm"), overrideCmdNS("nfs-snapshot"))
	AddStringFlag(cmdNfsSnapshotDelete, "id", "", "", "the ID of the NFS snapshot", requiredOpt())
	AddStringFlag(cmdNfsSnapshotDelete, "region", "r", "", "the region where the NFS share resides", requiredOpt())
	cmdNfsSnapshotDelete.Example =
		`doctl nfs snapshot delete  --region 'atl1' --id b050990d-4337-4a9d-9c8d-9f759a83936a`

	return cmd
}

func nfsCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, "name")
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, "region")
	if err != nil {
		return err
	}

	sizeStr, err := c.Doit.GetString(c.NS, "size")
	if err != nil {
		return err
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return fmt.Errorf("invalid size value: %v", err)
	}

	vpcIDs, err := c.Doit.GetStringSlice(c.NS, "vpc-ids")
	if err != nil {
		return err
	}

	r := &godo.NfsCreateRequest{
		Name:    name,
		Region:  region,
		SizeGib: size,
		VpcIDs:  vpcIDs,
	}

	share, err := c.Nfs().Create(r)
	if err != nil {
		return err
	}

	return displayNfs(c, *share)
}

func nfsGet(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, "region")
	if err != nil {
		return err
	}

	share, err := c.Nfs().Get(id, region)
	if err != nil {
		return err
	}

	return displayNfs(c, *share)
}

func nfsList(c *CmdConfig) error {
	region, err := c.Doit.GetString(c.NS, "region")
	if err != nil {
		return err
	}

	shares, err := c.Nfs().List(region)
	if err != nil {
		return err
	}

	return displayNfs(c, shares...)
}

func nfsDelete(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, "region")
	if err != nil {
		return err
	}

	err = c.Nfs().Delete(id, region)
	if err != nil {
		return err
	}

	return nil
}

func nfsSnapshotCreate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, "name")
	if err != nil {
		return err
	}

	shareID, err := c.Doit.GetString(c.NS, "share-id")
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, "region")
	if err != nil {
		return err
	}

	action, err := c.NfsActions().Snapshot(shareID, name, region)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		_, err := actionWait(c, action.ID, 5)
		if err != nil {
			return err
		}
	}

	item := &displayers.NfsAction{NfsActions: []do.NfsAction{*action}}
	return c.Display(item)
}

func nfsSnapshotList(c *CmdConfig) error {
	region, err := c.Doit.GetString(c.NS, "region")
	if err != nil {
		return err
	}

	shareId, _ := c.Doit.GetString(c.NS, "share-id")

	snapshots, err := c.Nfs().ListSnapshots(shareId, region)
	if err != nil {
		return err
	}

	return displayNfsSnapshots(c, snapshots...)
}

func nfsSnapshotGet(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, "region")
	if err != nil {
		return err
	}

	snapshot, err := c.Nfs().GetSnapshot(id, region)
	if err != nil {
		return err
	}

	return displayNfsSnapshots(c, *snapshot)
}

func nfsSnapshotDelete(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, "region")
	if err != nil {
		return err
	}

	err = c.Nfs().DeleteSnapshot(id, region)
	if err != nil {
		return err
	}

	return nil
}

func nfsResize(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, "region")
	if err != nil {
		return err
	}

	sizeStr, err := c.Doit.GetString(c.NS, "size")
	if err != nil {
		return err
	}

	size, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid size value: %v", err)
	}

	action, err := c.NfsActions().Resize(id, size, region)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		_, err := actionWait(c, action.ID, 5)
		if err != nil {
			return err
		}
	}

	item := &displayers.NfsAction{NfsActions: []do.NfsAction{*action}}
	return c.Display(item)
}

func displayNfs(c *CmdConfig, shares ...do.Nfs) error {
	item := &displayers.Nfs{NfsShares: shares}
	return c.Display(item)
}

func displayNfsSnapshots(c *CmdConfig, snapshots ...do.NfsSnapshot) error {
	item := &displayers.NfsSnapshot{NfsSnapshots: snapshots}
	return c.Display(item)
}
