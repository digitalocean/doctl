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
		},
	}

	cmd.AddCommand(PartnerAttachments())

	return cmd
}

// PartnerAttachments creates the partner attachments command.
func PartnerAttachments() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "attachment",
			Short: "Display commands that manage Partner Attachment",
			Long: `The commands under ` + "`" + `doctl network attachment` + "`" + ` are for managing your Partner Attachment.

With the Partner Attachment commands, you can get, list, create, update, or delete Partner Attachment, and manage their configuration details.`,
		},
	}

	cmdPartnerAttachmentCreate := CmdBuilder(cmd, RunPartnerAttachmentCreate, "create",
		"Create a Partner Attachment", "Use this command to create a new Partner Attachment on your account.", Writer, aliasOpt("c"), displayerType(&displayers.PartnerAttachment{}))
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")

	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentName, "", "", "Name of the Partner Attachment", requiredOpt())
	AddIntFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBandwidthInMbps, "", 0, "Connection Bandwidth in Mbps", requiredOpt())
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentRegion, "", "", "Region", requiredOpt())
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentNaaSProvider, "", "", "NaaS Provider", requiredOpt())
	AddStringSliceFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentVPCIDs, "", []string{}, "VPC network IDs", requiredOpt())
	AddIntFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBGPLocalASN, "", 0, "BGP Local ASN")
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBGPLocalRouterIP, "", "", "BGP Local Router IP")
	AddIntFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBGPPeerASN, "", 0, "BGP Peer ASN")
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBGPPeerRouterIP, "", "", "BGP Peer Router IP")
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBGPAuthKey, "", "", "BGP Auth Key")
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentRedundancyZone, "", "", "Redundancy Zone (optional)")
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentParentUUID, "", "", "HA Parent UUID (optional)")
	cmdPartnerAttachmentCreate.Example = `The following example creates a Partner Attachment: doctl network connect create --name "example-pia" --connection-bandwidth-in-mbps 50 --naas-provider "MEGAPORT" --region "nyc" --vpc-ids "c5537207-ebf0-47cb-bc10-6fac717cd672"`

	partnerAttachmentDetails := `
- The Partner Attachment ID
- The Partner Attachment Name
- The Partner Attachment State
- The Partner Attachment Connection Bandwidth in Mbps
- The Partner Attachment Region
- The Partner Attachment NaaS Provider
- The Partner Attachment VPC network IDs
- The Partner Attachment creation date, in ISO8601 combined date and time format
- The Partner Attachment BGP Local ASN
- The Partner Attachment BGP Local Router IP
- The Partner Attachment BGP Peer ASN
- The Partner Attachment BGP Peer Router IP
- The Partner Attachment Redundancy Zone
- The Partner Attachment Parent
- The Partner Attachment Children`

	cmdPartnerAttachmentGet := CmdBuilder(cmd, RunPartnerAttachmentGet, "get <partner-attachment-id>",
		"Retrieves a Partner Attachment",
		"Retrieves information about a Partner Attachment, including:"+partnerAttachmentDetails, Writer,
		aliasOpt("g"), displayerType(&displayers.PartnerAttachment{}))
	AddStringFlag(cmdPartnerAttachmentGet, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerAttachmentGet.Example = `The following example retrieves information about a Partner Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" attachment get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPartnerAttachmentList := CmdBuilder(cmd, RunPartnerAttachmentList, "list", "List Partner Attachment",
		"Retrieves a list of the Partner Attachment on your account, including the following information for each:"+partnerAttachmentDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.PartnerAttachment{}))
	AddStringFlag(cmdPartnerAttachmentList, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerAttachmentList.Example = `The following example lists the Partner Attachment on your account :` +
		` doctl network --type "partner" attachment list --format Name,VPCIDs `

	cmdPartnerAttachmentDelete := CmdBuilder(cmd, RunPartnerAttachmentDelete, "delete <partner-attachment-id>",
		"Deletes a Partner Attachment",
		"Deletes information about a Partner Attachment. This is irreversible ", Writer,
		aliasOpt("rm"), displayerType(&displayers.PartnerAttachment{}))
	AddBoolFlag(cmdPartnerAttachmentDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the Partner Attachment without any confirmation prompt")
	AddBoolFlag(cmdPartnerAttachmentDelete, doctl.ArgCommandWait, "", false,
		"Boolean that specifies whether to wait for a Partner Attachment deletion to complete before returning control to the terminal")
	AddStringFlag(cmdPartnerAttachmentDelete, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerAttachmentDelete.Example = `The following example deletes a Partner Attachments with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" attachment delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPartnerAttachmentUpdate := CmdBuilder(cmd, RunPartnerAttachmentUpdate, "update <partner-attachment-id>",
		"Update a Partner Attachments name and configuration",
		`Use this command to update the name and and configuration of a Partner Attachment`, Writer, aliasOpt("u"))
	AddStringFlag(cmdPartnerAttachmentUpdate, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")
	AddStringFlag(cmdPartnerAttachmentUpdate, doctl.ArgPartnerAttachmentName, "", "",
		"The Partner Attachment name", requiredOpt())
	AddStringFlag(cmdPartnerAttachmentUpdate, doctl.ArgPartnerAttachmentVPCIDs, "", "",
		"The Partner Attachment vpc ids", requiredOpt())
	cmdPartnerAttachmentUpdate.Example = `The following example updates the name of a Partner Attachment with the ID ` +
		"`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to ` + "`" + `new-name` + "`" +
		`: doctl network --type "partner" attachment update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name "new-name" --
vpc-ids "270a76ed-1bb7-4c5d-a6a5-e863de086940"`

	partnerAttachmentRouteDetails := `
- The Partner Attachment ID
- The Partner Attachment Cidr`

	cmdPartnerAttachmentRouteList := CmdBuilder(cmd, RunPartnerAttachmentRouteList, "list-routes <partner-attachment-id>",
		"List Partner Attachment Routes",
		"Retrieves a list of the Partner Attachment Routes on your account, including the following information for each:"+partnerAttachmentRouteDetails, Writer,
		aliasOpt("ls-routes"), displayerType(&displayers.PartnerAttachment{}))
	AddStringFlag(cmdPartnerAttachmentRouteList, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerAttachmentRouteList.Example = `The following example lists the Partner Attachment Routes on your account :` +
		` doctl network --type "partner" attachment list-routes f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --format ID,Cidr `

	cmdPartnerAttachmentRegenerateServiceKey := CmdBuilder(cmd, RunPartnerAttachmentRegenerateServiceKey, "regenerate-service-key <partner-attachment-id>",
		"Regenerates a Service key of Partner Attachment",
		"Regenerates information about a Service key of Partner Attachment", Writer,
		aliasOpt("regen-service-key"), displayerType(&displayers.PartnerAttachmentRegenerateServiceKey{}))
	AddStringFlag(cmdPartnerAttachmentRegenerateServiceKey, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerAttachmentRegenerateServiceKey.Example = `The following example retrieves information about a Service key of Partner Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" attachment regenerate-service-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdGetPartnerAttachmentGetBGPAuthKey := CmdBuilder(cmd, RunGetPartnerAttachmentBGPAuthKey, "get-bgp-auth-key <partner-attachment-id>",
		"Retrieves a BGP Auth key of Partner Attachment",
		"Retrieves information about a BGP Auth key of Partner Attachment", Writer,
		aliasOpt("g-bgp-auth-key"), displayerType(&displayers.PartnerAttachmentBgpAuthKey{}))
	AddStringFlag(cmdGetPartnerAttachmentGetBGPAuthKey, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")
	cmdGetPartnerAttachmentGetBGPAuthKey.Example = `The following example retrieves information about a Service key of Partner Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" attachment get-bgp-auth-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	partnerAttachmentServiceKeyDetails := `
- The Service key Value
- The Service key State
- The Service key CreatedAt`

	cmdGetPartnerIAServiceKey := CmdBuilder(cmd, RunGetPartnerAttachmentServiceKey, "get-service-key <partner-attachment-id>",
		"Retrieves a Service key of Partner Attachment",
		"Retrieves information about a Service key of Partner Attachment, including:"+partnerAttachmentServiceKeyDetails, Writer,
		aliasOpt("g-service-key"), displayerType(&displayers.PartnerAttachmentServiceKey{}))
	AddStringFlag(cmdGetPartnerIAServiceKey, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")
	cmdGetPartnerIAServiceKey.Example = `The following example retrieves information about a Service key of Partner Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" attachment get-service-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	return cmd
}

func ensurePartnerAttachmentType(c *CmdConfig) error {
	attachmentType, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentType)
	if err != nil {
		return err
	}
	if attachmentType != "partner" {
		return fmt.Errorf("unsupported attachment type: %s", attachmentType)
	}
	return nil
}

// RunPartnerAttachmentCreate creates a new Partner Attachment with a given configuration.
func RunPartnerAttachmentCreate(c *CmdConfig) error {
	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	r := new(godo.PartnerAttachmentCreateRequest)
	name, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentName)
	if err != nil {
		return err
	}
	r.Name = name

	connBandwidth, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerAttachmentBandwidthInMbps)
	if err != nil {
		return err
	}
	r.ConnectionBandwidthInMbps = connBandwidth

	region, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentRegion)
	if err != nil {
		return err
	}
	r.Region = region

	naasProvider, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentNaaSProvider)
	if err != nil {
		return err
	}
	r.NaaSProvider = naasProvider

	vpcIDs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgPartnerAttachmentVPCIDs)
	if err != nil {
		return err
	}
	r.VPCIDs = vpcIDs

	bgpConfig := new(godo.BGP)

	bgpLocalASN, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerAttachmentBGPLocalASN)
	if err != nil {
		return err
	}
	bgpConfig.LocalASN = bgpLocalASN

	bgpLocalRouterIP, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentBGPLocalRouterIP)
	if err != nil {
		return err
	}
	bgpConfig.LocalRouterIP = bgpLocalRouterIP

	bgpPeerASN, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerAttachmentBGPPeerASN)
	if err != nil {
		return err
	}
	bgpConfig.PeerASN = bgpPeerASN

	bgpPeerRouterIP, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentBGPPeerRouterIP)
	if err != nil {
		return err
	}
	bgpConfig.PeerRouterIP = bgpPeerRouterIP

	bgpAuthKey, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentBGPAuthKey)
	if err != nil {
		return err
	}
	bgpConfig.AuthKey = bgpAuthKey

	redundancyZone, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentRedundancyZone)
	if err != nil {
		return err
	}
	r.RedundancyZone = redundancyZone

	parentUUID, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentParentUUID)
	if err != nil {
		return err
	}
	r.ParentUuid = parentUUID

	pas := c.PartnerAttachments()
	pa, err := pas.Create(r)
	if err != nil {
		return err
	}

	item := &displayers.PartnerAttachment{PartnerAttachments: do.PartnerAttachments{*pa}}
	return c.Display(item)
}

// RunPartnerAttachmentGet retrieves an existing Partner Attachment by its identifier.
func RunPartnerAttachmentGet(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pas := c.PartnerAttachments()
	networkConnects, err := pas.GetPartnerAttachment(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerAttachment{
		PartnerAttachments: do.PartnerAttachments{*networkConnects},
	}
	return c.Display(item)
}

// RunPartnerAttachmentList lists Partner Attachments
func RunPartnerAttachmentList(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	pias := c.PartnerAttachments()
	list, err := pias.ListPartnerAttachments()
	if err != nil {
		return err
	}

	item := &displayers.PartnerAttachment{PartnerAttachments: list}
	return c.Display(item)
}

// RunPartnerAttachmentUpdate updates an existing Partner Attachment with new configuration.
func RunPartnerAttachmentUpdate(c *CmdConfig) error {
	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	paID := c.Args[0]

	r := new(godo.PartnerAttachmentUpdateRequest)
	name, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentName)
	if err != nil {
		return err
	}
	r.Name = name

	vpcIDs, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentVPCIDs)
	if err != nil {
		return err
	}
	r.VPCIDs = strings.Split(vpcIDs, ",")

	pa, err := c.PartnerAttachments().UpdatePartnerAttachment(paID, r)
	if err != nil {
		return err
	}

	item := &displayers.PartnerAttachment{
		PartnerAttachments: do.PartnerAttachments{*pa},
	}
	return c.Display(item)
}

// RunPartnerAttachmentRegenerateServiceKey regenerates a service key of existing Partner Attachment
func RunPartnerAttachmentRegenerateServiceKey(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pas := c.PartnerAttachments()
	regenerateServiceKey, err := pas.RegenerateServiceKey(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerAttachmentRegenerateServiceKey{
		RegenerateKey: *regenerateServiceKey,
	}
	return c.Display(item)
}

// RunGetPartnerAttachmentBGPAuthKey get a bgp auth key of existing Partner Attachment
func RunGetPartnerAttachmentBGPAuthKey(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	paID := c.Args[0]

	pas := c.PartnerAttachments()
	bgpAuthKey, err := pas.GetBGPAuthKey(paID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerAttachmentBgpAuthKey{
		Key: *bgpAuthKey,
	}
	return c.Display(item)
}

// RunGetPartnerAttachmentServiceKey retrieves service key of existing Partner Attachment
func RunGetPartnerAttachmentServiceKey(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	paID := c.Args[0]

	pas := c.PartnerAttachments()
	serviceKey, err := pas.GetServiceKey(paID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerAttachmentServiceKey{
		Key: *serviceKey,
	}
	return c.Display(item)
}

// RunPartnerAttachmentDelete deletes an existing Partner Attachment by its identifier.
func RunPartnerAttachmentDelete(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	paID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("Partner Attachment", 1) == nil {

		pas := c.PartnerAttachments()
		err := pas.DeletePartnerAttachment(paID)
		if err != nil {
			return err
		}

		wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
		if err != nil {
			return err
		}

		if wait {
			notice("Partner Attachment is in progress, waiting for Partner Attachment to be deleted")

			err := waitForPNC(pas, paID, "DELETED", true)
			if err != nil {
				return fmt.Errorf("Partner Attachment couldn't be deleted : %v", err)
			}
			notice("Partner Attachment is successfully deleted")
		} else {
			notice("Partner Attachment deletion request accepted")
		}

	} else {
		return fmt.Errorf("operation aborted")
	}

	return nil
}

// RunPartnerAttachmentRouteList lists Partner Attachment routes
func RunPartnerAttachmentRouteList(c *CmdConfig) error {
	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pas := c.PartnerAttachments()
	routeList, err := pas.ListPartnerAttachmentRoutes(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerAttachmentRoute{PartnerAttachmentRoutes: routeList}
	return c.Display(item)
}

func waitForPNC(pas do.PartnerAttachmentsService, iaID string, wantStatus string, terminateOnNotFound bool) error {
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

		pa, err := pas.GetPartnerAttachment(iaID)
		if err != nil {
			if terminateOnNotFound && strings.Contains(err.Error(), "not found") {
				return nil
			}
			return err
		}

		if pa.PartnerAttachment.State == errStatus {
			return fmt.Errorf("Partner Attachment (%s) entered status `%s`", iaID, errStatus)
		}

		if pa.PartnerAttachment.State == wantStatus {
			return nil
		}

		attempts++
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for Partner Attachment (%s) to become %s", iaID, wantStatus)
}
