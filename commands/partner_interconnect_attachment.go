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
	"os"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
)

// Network creates the network command.
func Network() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "network",
			Short:   "Display commands that manage network products",
			Long:    `The commands under ` + "`" + `doctl network` + "`" + ` are for managing network products`,
			GroupID: manageResourcesGroup,
			Hidden:  true,
		},
	}

	cmd.AddCommand(PartnerInterconnectAttachments())

	return cmd
}

// PartnerInterconnectAttachments creates the interconnect attachments command.
func PartnerInterconnectAttachments() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "interconnect-attachment",
			Short: "Display commands that manage Partner Interconnect Attachments",
			Long: `The commands under ` + "`" + `doctl network interconnect-attachment` + "`" + ` are for managing your Partner Interconnect Attachments.

With the Partner Interconnect Attachments commands, you can get, list, create, update, or delete Partner Interconnect Attachments, and manage their configuration details.`,
		},
	}

	cmdPartnerIACreate := CmdBuilder(cmd, RunPartnerInterconnectAttachmentCreate, "create",
		"Create a Partner Interconnect Attachment", "Use this command to create a new Partner Interconnect Attachment on your account.", Writer, aliasOpt("c"), displayerType(&displayers.PartnerInterconnectAttachment{}))
	AddStringFlag(cmdPartnerIACreate, doctl.ArgInterconnectAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")

	AddStringFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentName, "", "", "Name of the Partner Interconnect Attachment", requiredOpt())
	AddIntFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentConnectionBandwidthInMbps, "", 0, "Connection Bandwidth in Mbps", requiredOpt())
	AddStringFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentRegion, "", "", "Region", requiredOpt())
	AddStringFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentNaaSProvider, "", "", "NaaS Provider", requiredOpt())
	AddStringSliceFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentVPCIDs, "", []string{}, "VPC network IDs", requiredOpt())
	AddIntFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentBGPLocalASN, "", 0, "BGP Local ASN")
	AddStringFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentBGPLocalRouterIP, "", "", "BGP Local Router IP")
	AddIntFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentBGPPeerASN, "", 0, "BGP Peer ASN")
	AddStringFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentBGPPeerRouterIP, "", "", "BGP Peer Router IP")
	cmdPartnerIACreate.Example = `The following example creates a Partner Interconnect Attachment: doctl network interconnect-attachment create --name "example-pia" --connection-bandwidth-in-mbps 50 --naas-provider "MEGAPORT" --region "nyc" --vpc-ids "c5537207-ebf0-47cb-bc10-6fac717cd672"`

	interconnectAttachmentDetails := `
- The Partner Interconnect Attachment ID
- The Partner Interconnect Attachment Name
- The Partner Interconnect Attachment State
- The Partner Interconnect Attachment Connection Bandwidth in Mbps
- The Partner Interconnect Attachment Region
- The Partner Interconnect Attachment NaaS Provider
- The Partner Interconnect Attachment VPC network IDs
- The Partner Interconnect Attachment creation date, in ISO8601 combined date and time format
- The Partner Interconnect Attachment BGP Local ASN
- The Partner Interconnect Attachment BGP Local Router IP
- The Partner Interconnect Attachment BGP Peer ASN
- The Partner Interconnect Attachment BGP Peer Router IP`

	cmdPartnerIAGet := CmdBuilder(cmd, RunPartnerInterconnectAttachmentGet, "get <interconnect-attachment-id>",
		"Retrieves a Partner Interconnect Attachment", "Retrieves information about a Partner Interconnect Attachment, including:"+interconnectAttachmentDetails, Writer,
		aliasOpt("g"), displayerType(&displayers.PartnerInterconnectAttachment{}))
	AddStringFlag(cmdPartnerIAGet, doctl.ArgInterconnectAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdPartnerIAGet.Example = `The following example retrieves information about a Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" interconnect-attachment get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPartnerIAList := CmdBuilder(cmd, RunPartnerInterconnectAttachmentList, "list", "List Network Interconnect Attachments", "Retrieves a list of the Network Interconnect Attachments on your account, including the following information for each:"+interconnectAttachmentDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.PartnerInterconnectAttachment{}))
	AddStringFlag(cmdPartnerIAList, doctl.ArgInterconnectAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdPartnerIAList.Example = `The following example lists the Network Interconnect Attachments on your account :` +
		` doctl network --type "partner" interconnect-attachment list --format Name,VPCIDs `

	cmdPartnerIADelete := CmdBuilder(cmd, RunPartnerInterconnectAttachmentDelete, "delete <interconnect-attachment-id>",
		"Deletes a Partner Interconnect Attachment", "Deletes information about a Partner Interconnect Attachment. This is irreversible ", Writer,
		aliasOpt("rm"), displayerType(&displayers.PartnerInterconnectAttachment{}))
	AddBoolFlag(cmdPartnerIADelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the VPC Peering without any confirmation prompt")
	AddBoolFlag(cmdPartnerIADelete, doctl.ArgCommandWait, "", false,
		"Boolean that specifies whether to wait for a VPC Peering deletion to complete before returning control to the terminal")
	AddStringFlag(cmdPartnerIADelete, doctl.ArgInterconnectAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdPartnerIADelete.Example = `The following example deletes a Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" interconnect-attachment delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPartnerIAUpdate := CmdBuilder(cmd, RunPartnerInterconnectAttachmentUpdate, "update <interconnect-attachment-id>",
		"Update a Partner Interconnect Attachment's name and configuration", `Use this command to update the name and and configuration of a Partner Interconnect Attachment`, Writer, aliasOpt("u"))
	AddStringFlag(cmdPartnerIAUpdate, doctl.ArgInterconnectAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	AddStringFlag(cmdPartnerIAUpdate, doctl.ArgPartnerInterconnectAttachmentName, "", "",
		"The Partner Interconnect Attachment's name", requiredOpt())
	AddStringFlag(cmdPartnerIAUpdate, doctl.ArgPartnerInterconnectAttachmentVPCIDs, "", "",
		"The Partner Interconnect Attachment's vpc ids", requiredOpt())
	cmdPartnerIAUpdate.Example = `The following example updates the name of a Partner Interconnect Attachment with the ID ` +
		"`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to ` + "`" + `new-name` + "`" +
		`: doctl network --type "partner" interconnect-attachment update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name "new-name" --
vpc-ids "270a76ed-1bb7-4c5d-a6a5-e863de086940"`

	interconnectAttachmentRouteDetails := `
- The Partner Interconnect Attachment ID
- The Partner Interconnect Attachment Cidr`

	cmdPartnerIARouteList := CmdBuilder(cmd, RunPartnerInterconnectAttachmentRouteList, "list-routes", "List Network Interconnect Attachment Routes", "Retrieves a list of the Network Interconnect Attachment Routes on your account, including the following information for each:"+interconnectAttachmentRouteDetails, Writer,
		aliasOpt("ls-routes"), displayerType(&displayers.PartnerInterconnectAttachment{}))
	AddStringFlag(cmdPartnerIARouteList, doctl.ArgInterconnectAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdPartnerIARouteList.Example = `The following example lists the Network Interconnect Attachments on your account :` +
		` doctl network --type "partner" interconnect-attachment list-routes --format ID,Cidr `

	interconnectAttachmentServiceKeyDetails := `
- The Service key Value
- The Service key State`

	cmdGetPartnerIAServiceKey := CmdBuilder(cmd, RunGetPartnerInterconnectAttachmentServiceKey, "get-service-key <interconnect-attachment-id>",
		"Retrieves a Service key of Partner Interconnect Attachment", "Retrieves information about a Service key of Partner Interconnect Attachment, including:"+interconnectAttachmentServiceKeyDetails, Writer,
		aliasOpt("g-service-key"), displayerType(&displayers.PartnerInterconnectAttachmentServiceKey{}))
	AddStringFlag(cmdGetPartnerIAServiceKey, doctl.ArgInterconnectAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdGetPartnerIAServiceKey.Example = `The following example retrieves information about a Service key of Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" interconnect-attachment get-service-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdGetPartnerIARegenerateServiceKey := CmdBuilder(cmd, RunGetPartnerInterconnectAttachmentRegenerateServiceKey, "regenerate-service-key <interconnect-attachment-id>",
		"Regenerates a Service key of Partner Interconnect Attachment", "Regenerates information about a Service key of Partner Interconnect Attachment", Writer,
		aliasOpt("regen-service-key"), displayerType(&displayers.PartnerInterconnectAttachmentRegenerateServiceKey{}))
	AddStringFlag(cmdGetPartnerIARegenerateServiceKey, doctl.ArgInterconnectAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdGetPartnerIARegenerateServiceKey.Example = `The following example retrieves information about a Service key of Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" interconnect-attachment regen-service-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdGetPartnerIAGetBGPAuthKey := CmdBuilder(cmd, RunGetPartnerInterconnectAttachmentBGPAuthKey, "get-bgp-auth-key <interconnect-attachment-id>",
		"Retrieves a BGP Auth key of Partner Interconnect Attachment", "Retrieves information about a BGP Auth key of Partner Interconnect Attachment", Writer,
		aliasOpt("g-bgp-auth-key"), displayerType(&displayers.PartnerInterconnectAttachmentServiceKey{}))
	AddStringFlag(cmdGetPartnerIAGetBGPAuthKey, doctl.ArgInterconnectAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdGetPartnerIAGetBGPAuthKey.Example = `The following example retrieves information about a Service key of Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" interconnect-attachment get-bgp-auth-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	return cmd
}

func ensurePartnerAttachmentType(c *CmdConfig) error {
	attachmentType, err := c.Doit.GetString(c.NS, doctl.ArgInterconnectAttachmentType)
	if err != nil {
		return err
	}
	if attachmentType != "partner" {
		return fmt.Errorf("unsupported attachment type: %s", attachmentType)
	}
	return nil
}

// RunPartnerInterconnectAttachmentCreate creates a new Partner Interconnect Attachment with a given configuration.
func RunPartnerInterconnectAttachmentCreate(c *CmdConfig) error {
	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	r := new(godo.PartnerInterconnectAttachmentCreateRequest)
	name, err := c.Doit.GetString(c.NS, doctl.ArgPartnerInterconnectAttachmentName)
	if err != nil {
		return err
	}
	r.Name = name

	connBandwidth, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerInterconnectAttachmentConnectionBandwidthInMbps)
	if err != nil {
		return err
	}
	r.ConnectionBandwidthInMbps = connBandwidth

	region, err := c.Doit.GetString(c.NS, doctl.ArgPartnerInterconnectAttachmentRegion)
	if err != nil {
		return err
	}
	r.Region = region

	naasProvider, err := c.Doit.GetString(c.NS, doctl.ArgPartnerInterconnectAttachmentNaaSProvider)
	if err != nil {
		return err
	}
	r.NaaSProvider = naasProvider

	vpcIDs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgPartnerInterconnectAttachmentVPCIDs)
	if err != nil {
		return err
	}
	r.VPCIDs = vpcIDs

	bgpConfig := new(godo.BGP)

	bgpLocalASN, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerInterconnectAttachmentBGPLocalASN)
	if err != nil {
		return err
	}
	bgpConfig.LocalASN = bgpLocalASN

	bgpLocalRouterIP, err := c.Doit.GetString(c.NS, doctl.ArgPartnerInterconnectAttachmentBGPLocalRouterIP)
	if err != nil {
		return err
	}
	bgpConfig.LocalRouterIP = bgpLocalRouterIP

	bgpPeerASN, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerInterconnectAttachmentBGPPeerASN)
	if err != nil {
		return err
	}
	bgpConfig.PeerASN = bgpPeerASN

	bgpPeerRouterIP, err := c.Doit.GetString(c.NS, doctl.ArgPartnerInterconnectAttachmentBGPPeerRouterIP)
	if err != nil {
		return err
	}
	bgpConfig.PeerRouterIP = bgpPeerRouterIP

	pias := c.PartnerInterconnectAttachments()
	pia, err := pias.Create(r)
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachment{PartnerInterconnectAttachments: do.PartnerInterconnectAttachments{*pia}}
	return c.Display(item)
}

// RunPartnerInterconnectAttachmentGet retrieves an existing Partner Interconnect Attachment by its identifier.
func RunPartnerInterconnectAttachmentGet(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pias := c.PartnerInterconnectAttachments()
	interconnectAttachment, err := pias.GetPartnerInterconnectAttachment(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachment{
		PartnerInterconnectAttachments: do.PartnerInterconnectAttachments{*interconnectAttachment},
	}
	return c.Display(item)
}

// RunPartnerInterconnectAttachmentList lists Partner Interconnect Attachment
func RunPartnerInterconnectAttachmentList(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	pias := c.PartnerInterconnectAttachments()
	list, err := pias.ListPartnerInterconnectAttachments()
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachment{PartnerInterconnectAttachments: list}
	return c.Display(item)
}

// RunPartnerInterconnectAttachmentUpdate updates an existing Partner Interconnect Attachment with new configuration.
func RunPartnerInterconnectAttachmentUpdate(c *CmdConfig) error {
	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	r := new(godo.PartnerInterconnectAttachmentUpdateRequest)
	name, err := c.Doit.GetString(c.NS, doctl.ArgPartnerInterconnectAttachmentName)
	if err != nil {
		return err
	}
	r.Name = name

	vpcIDs, err := c.Doit.GetString(c.NS, doctl.ArgPartnerInterconnectAttachmentVPCIDs)
	if err != nil {
		return err
	}
	r.VPCIDs = strings.Split(vpcIDs, ",")

	interconnectAttachment, err := c.PartnerInterconnectAttachments().UpdatePartnerInterconnectAttachment(iaID, r)
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachment{
		PartnerInterconnectAttachments: do.PartnerInterconnectAttachments{*interconnectAttachment},
	}
	return c.Display(item)
}

// RunGetPartnerInterconnectAttachmentServiceKey retrieves service key of existing Partner Interconnect Attachment
func RunGetPartnerInterconnectAttachmentServiceKey(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pias := c.PartnerInterconnectAttachments()
	serviceKey, err := pias.GetServiceKey(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachmentServiceKey{
		Key: *serviceKey,
	}
	return c.Display(item)
}

// RunGetPartnerInterconnectAttachmentRegenerateServiceKey regenerates a service key of existing Partner Interconnect Attachment
func RunGetPartnerInterconnectAttachmentRegenerateServiceKey(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pias := c.PartnerInterconnectAttachments()
	regenerateServiceKey, err := pias.RegenerateServiceKey(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachmentRegenerateServiceKey{
		RegenerateKey: *regenerateServiceKey,
	}
	return c.Display(item)
}

// RunGetPartnerInterconnectAttachmentBGPAuthKey get a bgp auth key of existing Partner Interconnect Attachment
func RunGetPartnerInterconnectAttachmentBGPAuthKey(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pias := c.PartnerInterconnectAttachments()
	bgpAuthKey, err := pias.GetBGPAuthKey(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachmentBgpAuthKey{
		Key: *bgpAuthKey,
	}
	return c.Display(item)
}

// RunPartnerInterconnectAttachmentDelete deletes an existing Partner Interconnect Attachment by its identifier.
func RunPartnerInterconnectAttachmentDelete(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("Partner Interconnect Attachment", 1) == nil {

		pias := c.PartnerInterconnectAttachments()
		err := pias.DeletePartnerInterconnectAttachment(iaID)
		if err != nil {
			return err
		}

		wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
		if err != nil {
			return err
		}

		if wait {
			notice("Partner Interconnect Attachment is in progress, waiting for Partner Interconnect Attachment to be deleted")

			err := waitForPIA(pias, iaID, "DELETED", true)
			if err != nil {
				return fmt.Errorf("Partner Interconnect Attachment couldn't be deleted : %v", err)
			}
			notice("Partner Interconnect Attachment is successfully deleted")
		} else {
			notice("Partner Interconnect Attachment deletion request accepted")
		}

	} else {
		return fmt.Errorf("operation aborted")
	}

	return nil
}

// RunPartnerInterconnectAttachmentRouteList lists Partner Interconnect Attachment routes
func RunPartnerInterconnectAttachmentRouteList(c *CmdConfig) error {
	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pias := c.PartnerInterconnectAttachments()
	routeList, err := pias.ListPartnerInterconnectAttachmentRoutes(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachmentRoute{PartnerInterconnectAttachmentRoutes: routeList}
	return c.Display(item)
}

func waitForPIA(pias do.PartnerInterconnectAttachmentsService, iaID string, wantStatus string, terminateOnNotFound bool) error {
	const maxAttempts = 360
	const errStatus = "ERROR"
	attempts := 0
	printNewLineSet := false

	for i := 0; i < maxAttempts; i++ {
		if attempts != 0 {
			fmt.Fprint(os.Stderr, ".")
			if !printNewLineSet {
				printNewLineSet = true
				defer fmt.Fprintln(os.Stderr)
			}
		}

		interconnectAttachment, err := pias.GetPartnerInterconnectAttachment(iaID)
		if err != nil {
			if terminateOnNotFound && strings.Contains(err.Error(), "not found") {
				return nil
			}
			return err
		}

		if interconnectAttachment.State == errStatus {
			return fmt.Errorf("Partner Interconnect Attachment (%s) entered status `%s`", iaID, errStatus)
		}

		if interconnectAttachment.State == wantStatus {
			return nil
		}

		attempts++
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for Partner Interconnect Attachment (%s) to become %s", iaID, wantStatus)
}
