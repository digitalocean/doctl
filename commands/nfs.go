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
	"strings"

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
	AddStringFlag(cmdNfsCreate, "performance-tier", "", "", "the performance tier of the NFS share", requiredOpt())
	cmdNfsCreate.Example =
		`doctl nfs create --name sammy-nfs-share --region 'atl1' --size 50 --vpc-ids 74922c16-5466-42a5-ac58-0e8069918b6b --performance-tier standard
doctl nfs create --name my-nfs-share --region 'nyc2' --size 100 --vpc-ids 74922c16-5466-42a5-ac58-0e8069918b6b --performance-tier standard`

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

	cmdNfsAttach := CmdBuilder(cmd, nfsAttach, "attach [flags]", "Attach an NFS share to a VPC", "Attaches an NFS share to a VPC with the given ID and region.", Writer)
	AddStringFlag(cmdNfsAttach, "id", "", "", "the ID of the NFS share", requiredOpt())
	AddStringFlag(cmdNfsAttach, "region", "r", "", "the region where the NFS share resides", requiredOpt())
	AddStringFlag(cmdNfsAttach, "vpc-id", "", "", "the id of the VPC we want to attach NFS share to", requiredOpt())
	AddBoolFlag(cmdNfsAttach, doctl.ArgCommandWait, "", false, "Wait for action to complete")
	cmdNfsAttach.Example =
		`doctl nfs attach --region 'atl1' --id b050990d-4337-4a9d-9c8d-9f759a83936a --vpc-id example-vpc-id`

	cmdNfsDetach := CmdBuilder(cmd, nfsDetach, "detach [flags]", "Detach an NFS share from a VPC", "Detaches an NFS share from a VPC with the given ID and region.", Writer)
	AddStringFlag(cmdNfsDetach, "id", "", "", "the ID of the NFS share", requiredOpt())
	AddStringFlag(cmdNfsDetach, "region", "r", "", "the region where the NFS share resides", requiredOpt())
	AddStringFlag(cmdNfsDetach, "vpc-id", "", "", "the id of the VPC we want to detach NFS share from", requiredOpt())
	AddBoolFlag(cmdNfsDetach, doctl.ArgCommandWait, "", false, "Wait for action to complete")
	cmdNfsDetach.Example =
		`doctl nfs detach --region 'atl1' --id b050990d-4337-4a9d-9c8d-9f759a83936a --vpc-id example-vpc-id`

	cmdNfsReassign := CmdBuilder(cmd, nfsReassign, "reassign [flags]", "Reassign an NFS share between VPCs", "Reassigns an NFS share from one VPC to another with the given ID.", Writer)
	AddStringFlag(cmdNfsReassign, "id", "", "", "the ID of the NFS share", requiredOpt())
	AddStringFlag(cmdNfsReassign, "old-vpc-id", "", "", "the id of the VPC we want to detach NFS share from", requiredOpt())
	AddStringFlag(cmdNfsReassign, "new-vpc-id", "", "", "the id of the VPC we want to attach NFS share to", requiredOpt())
	AddBoolFlag(cmdNfsReassign, doctl.ArgCommandWait, "", false, "Wait for action to complete")
	cmdNfsReassign.Example =
		`doctl nfs reassign --id b050990d-4337-4a9d-9c8d-9f759a83936a --old-vpc-id old-vpc-id --new-vpc-id new-vpc-id`

	cmdNfsSwitchPerformanceTier := CmdBuilder(cmd, nfsSwitchPerformanceTier, "switch-performance-tier [flags]", "Switch the performance tier of an NFS share", "Switch the performance tier of an NFS share with the given ID and tier.", Writer)
	AddStringFlag(cmdNfsSwitchPerformanceTier, "id", "", "", "the ID of the NFS share", requiredOpt())
	AddStringFlag(cmdNfsSwitchPerformanceTier, "performance-tier", "", "", "the performance tier of the NFS share", requiredOpt())
	AddBoolFlag(cmdNfsSwitchPerformanceTier, doctl.ArgCommandWait, "", false, "Wait for action to complete")
	cmdNfsSwitchPerformanceTier.Example =
		`doctl nfs switch-performance-tier --id b050990d-4337-4a9d-9c8d-9f759a83936a --performance-tier high`

	cmd.AddCommand(nfsSnapshots())
	cmd.AddCommand(nfsAccessPoints())

	return cmd
}

func nfsAccessPoints() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "access-point",
			Short: "Display commands for NFS access points",
			Long:  "The commands under `doctl nfs access-point` are for managing NFS share access points.",
		},
	}

	create := CmdBuilder(cmd, nfsAccessPointCreate, "create [flags]", "Create an NFS access point", "Create an NFS access point for a share.", Writer, displayerType(&displayers.NfsAction{}), overrideCmdNS("nfs-access-point"))
	AddStringFlag(create, "share-id", "", "", "the ID of the NFS share", requiredOpt())
	AddStringFlag(create, "name", "n", "", "the access point name", requiredOpt())
	AddStringFlag(create, "path", "", "", "the export path for the access point, must start with /", requiredOpt())
	AddStringSliceFlag(create, "protocols", "", []string{"NFS4"}, "allowed protocols (NFS, NFS4)")
	AddStringFlag(create, "squash-config", "", "ROOT_SQUASH", "squash mode (NO_SQUASH, ROOT_SQUASH, ALL_SQUASH)")
	AddStringFlag(create, "anonuid", "", "65534", "anon uid for squashed users")
	AddStringFlag(create, "anongid", "", "65534", "anon gid for squashed users")
	AddBoolFlag(create, "identity-enforcement-enabled", "", false, "enable identity enforcement for the export")
	AddStringFlag(create, "vpc-id", "", "", "the VPC ID for this access point", requiredOpt())

	get := CmdBuilder(cmd, nfsAccessPointGet, "get [flags]", "Get an NFS access point", "Get an NFS access point by ID.", Writer, displayerType(&displayers.NfsAccessPoint{}), overrideCmdNS("nfs-access-point"))
	AddStringFlag(get, "id", "", "", "the ID of the NFS access point", requiredOpt())

	list := CmdBuilder(cmd, nfsAccessPointList, "list [flags]", "List NFS access points for a share", "List NFS access points for a share.", Writer, aliasOpt("ls"), displayerType(&displayers.NfsAccessPoint{}), overrideCmdNS("nfs-access-point"))
	AddStringFlag(list, "share-id", "", "", "the ID of the NFS share", requiredOpt())
	AddStringFlag(list, "status", "", "", "optional status filter (ACCESS_POINT_CREATING, ACCESS_POINT_ACTIVE, ACCESS_POINT_FAILED, ACCESS_POINT_DELETED)")

	deleteCmd := CmdBuilder(cmd, nfsAccessPointDelete, "delete [flags]", "Delete an NFS access point", "Delete an NFS access point by ID.", Writer, aliasOpt("rm"), displayerType(&displayers.NfsAction{}), overrideCmdNS("nfs-access-point"))
	AddStringFlag(deleteCmd, "id", "", "", "the ID of the NFS access point", requiredOpt())

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

	performanceTier, _ := c.Doit.GetString(c.NS, "performance-tier")
	r := &godo.NfsCreateRequest{
		Name:            name,
		Region:          region,
		SizeGib:         size,
		VpcIDs:          vpcIDs,
		PerformanceTier: performanceTier,
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
	region, _ := c.Doit.GetString(c.NS, "region")
	share, err := c.Nfs().Get(id, region)
	if err != nil {
		return err
	}

	return displayNfs(c, *share)
}

func nfsList(c *CmdConfig) error {
	region, _ := c.Doit.GetString(c.NS, "region")
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

	region, _ := c.Doit.GetString(c.NS, "region")

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

	region, _ := c.Doit.GetString(c.NS, "region")

	action, err := c.NfsActions().Snapshot(shareID, name, region)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	actionID, err := strconv.Atoi(action.ID)
	if err != nil {
		return err
	}

	if wait {
		_, err := actionWait(c, actionID, 5)
		if err != nil {
			return err
		}
	}

	item := &displayers.NfsAction{NfsActions: []do.NfsAction{*action}}
	return c.Display(item)
}

func nfsSnapshotList(c *CmdConfig) error {
	region, _ := c.Doit.GetString(c.NS, "region")

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

	region, _ := c.Doit.GetString(c.NS, "region")

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

	region, _ := c.Doit.GetString(c.NS, "region")

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

	region, _ := c.Doit.GetString(c.NS, "region")

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
		actionID, err := strconv.Atoi(action.ID)
		if err != nil {
			return err
		}

		_, err = actionWait(c, actionID, 5)
		if err != nil {
			return err
		}
	}

	item := &displayers.NfsAction{NfsActions: []do.NfsAction{*action}}
	return c.Display(item)
}

func nfsAttach(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}

	region, _ := c.Doit.GetString(c.NS, "region")

	vpcIdStr, err := c.Doit.GetString(c.NS, "vpc-id")
	if err != nil {
		return err
	}

	action, err := c.NfsActions().Attach(id, vpcIdStr, region)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		actionID, err := strconv.Atoi(action.ID)
		if err != nil {
			return err
		}

		_, err = actionWait(c, actionID, 5)
		if err != nil {
			return err
		}
	}

	item := &displayers.NfsAction{NfsActions: []do.NfsAction{*action}}
	return c.Display(item)
}

func nfsDetach(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}

	region, _ := c.Doit.GetString(c.NS, "region")
	vpcIdStr, err := c.Doit.GetString(c.NS, "vpc-id")
	if err != nil {
		return err
	}

	action, err := c.NfsActions().Detach(id, vpcIdStr, region)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		actionID, err := strconv.Atoi(action.ID)
		if err != nil {
			return err
		}

		_, err = actionWait(c, actionID, 5)
		if err != nil {
			return err
		}
	}

	item := &displayers.NfsAction{NfsActions: []do.NfsAction{*action}}
	return c.Display(item)
}

func nfsReassign(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}
	oldVpcID, err := c.Doit.GetString(c.NS, "old-vpc-id")
	if err != nil {
		return err
	}
	newVpcID, err := c.Doit.GetString(c.NS, "new-vpc-id")
	if err != nil {
		return err
	}

	action, err := c.NfsActions().Reassign(id, oldVpcID, newVpcID)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		actionID, err := strconv.Atoi(action.ID)
		if err != nil {
			return err
		}

		_, err = actionWait(c, actionID, 5)
		if err != nil {
			return err
		}
	}

	item := &displayers.NfsAction{NfsActions: []do.NfsAction{*action}}
	return c.Display(item)
}

func nfsSwitchPerformanceTier(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}

	performanceTier, err := c.Doit.GetString(c.NS, "performance-tier")
	if err != nil {
		return err
	}

	action, err := c.NfsActions().SwitchPerformanceTier(id, performanceTier)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		actionID, err := strconv.Atoi(action.ID)
		if err != nil {
			return err
		}

		_, err = actionWait(c, actionID, 5)
		if err != nil {
			return err
		}
	}

	item := &displayers.NfsAction{NfsActions: []do.NfsAction{*action}}
	return c.Display(item)
}

func nfsAccessPointCreate(c *CmdConfig) error {
	shareID, err := c.Doit.GetString(c.NS, "share-id")
	if err != nil {
		return err
	}

	name, err := c.Doit.GetString(c.NS, "name")
	if err != nil {
		return err
	}

	path, err := c.Doit.GetString(c.NS, "path")
	if err != nil {
		return err
	}

	protocols, err := c.Doit.GetStringSlice(c.NS, "protocols")
	if err != nil {
		return err
	}

	squashConfig, err := c.Doit.GetString(c.NS, "squash-config")
	if err != nil {
		return err
	}

	anonUIDStr, err := c.Doit.GetString(c.NS, "anonuid")
	if err != nil {
		return err
	}
	anonUID, err := strconv.ParseUint(anonUIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid anonuid value: %v", err)
	}

	anonGIDStr, err := c.Doit.GetString(c.NS, "anongid")
	if err != nil {
		return err
	}
	anonGID, err := strconv.ParseUint(anonGIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid anongid value: %v", err)
	}

	identityEnforcementEnabled, err := c.Doit.GetBool(c.NS, "identity-enforcement-enabled")
	if err != nil {
		return err
	}

	vpcID, err := c.Doit.GetString(c.NS, "vpc-id")
	if err != nil {
		return err
	}

	request := &do.NfsAccessPointCreateRequest{
		Name:  name,
		Path:  path,
		VpcID: vpcID,
		AccessPolicy: do.NfsAccessPointPolicy{
			Anonuid:                    anonUID,
			Anongid:                    anonGID,
			Protocols:                  normalizeProtocols(protocols),
			SquashConfig:               strings.ToUpper(squashConfig),
			IdentityEnforcementEnabled: identityEnforcementEnabled,
		},
	}

	resp, err := c.Nfs().CreateAccessPoint(shareID, request)
	if err != nil {
		return err
	}

	item := &displayers.NfsAction{NfsActions: []do.NfsAction{{NfsAction: resp.Action}}}
	return c.Display(item)
}

func nfsAccessPointGet(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}

	ap, err := c.Nfs().GetAccessPoint(id)
	if err != nil {
		return err
	}

	accessPoints, err := enrichNfsAccessPointsForShare(c, ap.ShareID, []do.NfsAccessPoint{*ap})
	if err != nil {
		return err
	}

	return displayNfsAccessPoints(c, accessPoints...)
}

func nfsAccessPointList(c *CmdConfig) error {
	shareID, err := c.Doit.GetString(c.NS, "share-id")
	if err != nil {
		return err
	}

	status, err := c.Doit.GetString(c.NS, "status")
	if err != nil {
		return err
	}

	accessPoints, err := c.Nfs().ListAccessPoints(shareID, strings.ToUpper(status))
	if err != nil {
		return err
	}

	accessPoints, err = enrichNfsAccessPointsForShare(c, shareID, accessPoints)
	if err != nil {
		return err
	}

	return displayNfsAccessPoints(c, accessPoints...)
}

// enrichNfsAccessPointsForShare fills VPC IDs on the default access point from the
// parent share. The access point API does not return VPCs for the default "/"
// export, and the share can be looked up by ID without a region.
func enrichNfsAccessPointsForShare(c *CmdConfig, shareID string, accessPoints []do.NfsAccessPoint) ([]do.NfsAccessPoint, error) {
	needsShareVPCs := false
	for _, ap := range accessPoints {
		if ap.IsDefault {
			needsShareVPCs = true
			break
		}
	}
	if !needsShareVPCs {
		return accessPoints, nil
	}

	share, err := c.Nfs().Get(shareID, "")
	if err != nil {
		return nil, err
	}

	out := make([]do.NfsAccessPoint, len(accessPoints))
	for i, ap := range accessPoints {
		out[i] = ap
		if ap.IsDefault {
			out[i].VpcIDs = share.VpcIDs
		}
	}

	return out, nil
}

func nfsAccessPointDelete(c *CmdConfig) error {
	id, err := c.Doit.GetString(c.NS, "id")
	if err != nil {
		return err
	}

	resp, err := c.Nfs().DeleteAccessPoint(id)
	if err != nil {
		return err
	}

	item := &displayers.NfsAction{NfsActions: []do.NfsAction{{NfsAction: resp.Action}}}
	return c.Display(item)
}

func normalizeProtocols(protocols []string) []string {
	out := make([]string, 0, len(protocols))
	for _, p := range protocols {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, strings.ToUpper(trimmed))
		}
	}
	if len(out) == 0 {
		return []string{"NFS4"}
	}
	return out
}

func displayNfs(c *CmdConfig, shares ...do.Nfs) error {
	item := &displayers.Nfs{NfsShares: shares}
	return c.Display(item)
}

func displayNfsSnapshots(c *CmdConfig, snapshots ...do.NfsSnapshot) error {
	item := &displayers.NfsSnapshot{NfsSnapshots: snapshots}
	return c.Display(item)
}

func displayNfsAccessPoints(c *CmdConfig, accessPoints ...do.NfsAccessPoint) error {
	item := &displayers.NfsAccessPoint{NfsAccessPoints: accessPoints}
	return c.Display(item)
}
