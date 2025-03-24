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
			Use:   "connect",
			Short: "Display commands that manage Partner Network Connect",
			Long: `The commands under ` + "`" + `doctl network connect` + "`" + ` are for managing your Partner Network Connect.

With the Partner Network Connect commands, you can get, list, create, update, or delete Partner Network Connect, and manage their configuration details.`,
		},
	}

	cmdPartnerAttachmentCreate := CmdBuilder(cmd, RunPartnerInterconnectAttachmentCreate, "create",
		"Create a Partner Network Connect", "Use this command to create a new Partner Network Connect on your account.", Writer, aliasOpt("c"), displayerType(&displayers.PartnerNetworkConnect{}))
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")

	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentName, "", "", "Name of the Partner Network Connect", requiredOpt())
	AddIntFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentConnectionBandwidthInMbps, "", 0, "Connection Bandwidth in Mbps", requiredOpt())
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentRegion, "", "", "Region", requiredOpt())
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentNaaSProvider, "", "", "NaaS Provider", requiredOpt())
	AddStringSliceFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentVPCIDs, "", []string{}, "VPC network IDs", requiredOpt())
	AddIntFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBGPLocalASN, "", 0, "BGP Local ASN")
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBGPLocalRouterIP, "", "", "BGP Local Router IP")
	AddIntFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBGPPeerASN, "", 0, "BGP Peer ASN")
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBGPPeerRouterIP, "", "", "BGP Peer Router IP")
	AddStringFlag(cmdPartnerAttachmentCreate, doctl.ArgPartnerAttachmentBGPAuthKey, "", "", "BGP Auth Key")
	cmdPartnerAttachmentCreate.Example = `The following example creates a Partner Network Connect: doctl network connect create --name "example-pia" --connection-bandwidth-in-mbps 50 --naas-provider "MEGAPORT" --region "nyc" --vpc-ids "c5537207-ebf0-47cb-bc10-6fac717cd672"`

	partnerNetworkConnectDetails := `
- The Partner Network Connect ID
- The Partner Network Connect Name
- The Partner Network Connect State
- The Partner Network Connect Connection Bandwidth in Mbps
- The Partner Network Connect Region
- The Partner Network Connect NaaS Provider
- The Partner Network Connect VPC network IDs
- The Partner Network Connect creation date, in ISO8601 combined date and time format
- The Partner Network Connect BGP Local ASN
- The Partner Network Connect BGP Local Router IP
- The Partner Network Connect BGP Peer ASN
- The Partner Network Connect BGP Peer Router IP`

	cmdPartnerNCGet := CmdBuilder(cmd, RunPartnerNCGet, "get <partner-network-connect-id>",
		"Retrieves a Partner Network Connect", "Retrieves information about a Partner Network Connect, including:"+partnerNetworkConnectDetails, Writer,
		aliasOpt("g"), displayerType(&displayers.PartnerNetworkConnect{}))
	AddStringFlag(cmdPartnerNCGet, doctl.ArgPartnerAttachmentType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerNCGet.Example = `The following example retrieves information about a Partner Network Connect with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" connect get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPartnerIAList := CmdBuilder(cmd, RunPartnerAttachmentList, "list", "List Network Interconnect Attachments", "Retrieves a list of the Network Interconnect Attachments on your account, including the following information for each:"+partnerNetworkConnectDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.PartnerNetworkConnect{}))
	AddStringFlag(cmdPartnerIAList, doctl.ArgPartnerAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdPartnerIAList.Example = `The following example lists the Network Interconnect Attachments on your account :` +
		` doctl network --type "partner" partner-attachment list --format Name,VPCIDs `

	cmdPartnerIADelete := CmdBuilder(cmd, RunPartnerNetworkAttachmentDelete, "delete <partner-network-connect-id>",
		"Deletes a Partner Interconnect Attachment", "Deletes information about a Partner Interconnect Attachment. This is irreversible ", Writer,
		aliasOpt("rm"), displayerType(&displayers.PartnerNetworkConnect{}))
	AddBoolFlag(cmdPartnerIADelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the VPC Peering without any confirmation prompt")
	AddBoolFlag(cmdPartnerIADelete, doctl.ArgCommandWait, "", false,
		"Boolean that specifies whether to wait for a VPC Peering deletion to complete before returning control to the terminal")
	AddStringFlag(cmdPartnerIADelete, doctl.ArgPartnerAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdPartnerIADelete.Example = `The following example deletes a Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" partner-attachment delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPartnerIAUpdate := CmdBuilder(cmd, RunPartnerAttachmentUpdate, "update <partner-network-connect-id>",
		"Update a Partner Interconnect Attachment's name and configuration", `Use this command to update the name and and configuration of a Partner Interconnect Attachment`, Writer, aliasOpt("u"))
	AddStringFlag(cmdPartnerIAUpdate, doctl.ArgPartnerAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	AddStringFlag(cmdPartnerIAUpdate, doctl.ArgPartnerAttachmentName, "", "",
		"The Partner Interconnect Attachment's name", requiredOpt())
	AddStringFlag(cmdPartnerIAUpdate, doctl.ArgPartnerAttachmentVPCIDs, "", "",
		"The Partner Interconnect Attachment's vpc ids", requiredOpt())
	cmdPartnerIAUpdate.Example = `The following example updates the name of a Partner Interconnect Attachment with the ID ` +
		"`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to ` + "`" + `new-name` + "`" +
		`: doctl network --type "partner" partner-attachment update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name "new-name" --
vpc-ids "270a76ed-1bb7-4c5d-a6a5-e863de086940"`

	interconnectAttachmentRouteDetails := `
- The Partner Interconnect Attachment ID
- The Partner Interconnect Attachment Cidr`

	cmdPartnerIARouteList := CmdBuilder(cmd, RunPartnerAttachmentRouteList, "list-routes", "List Network Interconnect Attachment Routes", "Retrieves a list of the Network Interconnect Attachment Routes on your account, including the following information for each:"+interconnectAttachmentRouteDetails, Writer,
		aliasOpt("ls-routes"), displayerType(&displayers.PartnerNetworkConnect{}))
	AddStringFlag(cmdPartnerIARouteList, doctl.ArgPartnerAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdPartnerIARouteList.Example = `The following example lists the Network Interconnect Attachments on your account :` +
		` doctl network --type "partner" partner-attachment list-routes --format ID,Cidr `

	cmdGetPartnerIARegenerateServiceKey := CmdBuilder(cmd, RunPartnerAttachmentRegenerateServiceKey, "regenerate-service-key <partner-network-connect-id>",
		"Regenerates a Service key of Partner Interconnect Attachment", "Regenerates information about a Service key of Partner Interconnect Attachment", Writer,
		aliasOpt("regen-service-key"), displayerType(&displayers.PartnerAttachmentRegenerateServiceKey{}))
	AddStringFlag(cmdGetPartnerIARegenerateServiceKey, doctl.ArgPartnerAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdGetPartnerIARegenerateServiceKey.Example = `The following example retrieves information about a Service key of Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" partner-attachment regenerate-service-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdGetPartnerIAGetBGPAuthKey := CmdBuilder(cmd, RunGetPartnerAttachmentBGPAuthKey, "get-bgp-auth-key <partner-network-connect-id>",
		"Retrieves a BGP Auth key of Partner Interconnect Attachment", "Retrieves information about a BGP Auth key of Partner Interconnect Attachment", Writer,
		aliasOpt("g-bgp-auth-key"), displayerType(&displayers.PartnerAttachmentBgpAuthKey{}))
	AddStringFlag(cmdGetPartnerIAGetBGPAuthKey, doctl.ArgPartnerAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdGetPartnerIAGetBGPAuthKey.Example = `The following example retrieves information about a Service key of Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" partner-attachment get-bgp-auth-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	interconnectAttachmentServiceKeyDetails := `
- The Service key Value
- The Service key State
- The Service key CreatedAt`

	cmdGetPartnerIAServiceKey := CmdBuilder(cmd, RunGetPartnerAttachmentServiceKey, "get-service-key <partner-network-connect-id>",
		"Retrieves a Service key of Partner Interconnect Attachment", "Retrieves information about a Service key of Partner Interconnect Attachment, including:"+interconnectAttachmentServiceKeyDetails, Writer,
		aliasOpt("g-service-key"), displayerType(&displayers.PartnerAttachmentServiceKey{}))
	AddStringFlag(cmdGetPartnerIAServiceKey, doctl.ArgPartnerAttachmentType, "", "partner", "Specify interconnect attachment type (e.g., partner)")
	cmdGetPartnerIAServiceKey.Example = `The following example retrieves information about a Service key of Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" partner-attachment get-service-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

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

// RunPartnerInterconnectAttachmentCreate creates a new Partner Attachment with a given configuration.
func RunPartnerInterconnectAttachmentCreate(c *CmdConfig) error {
	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	r := new(godo.PartnerNetworkConnectCreateRequest)
	name, err := c.Doit.GetString(c.NS, doctl.ArgPartnerAttachmentName)
	if err != nil {
		return err
	}
	r.Name = name

	connBandwidth, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerAttachmentConnectionBandwidthInMbps)
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
		bgpConfig.AuthKey = bgpAuthKey
	}

	pias := c.PartnerInterconnectAttachments()
	pia, err := pias.Create(r)
	if err != nil {
		return err
	}

	item := &displayers.PartnerNetworkConnect{PartnerNetworkConnects: do.PartnerNetworkConnects{*pia}}
	return c.Display(item)
}

// RunPartnerNCGet retrieves an existing Partner Network Connect by its identifier.
func RunPartnerNCGet(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pias := c.PartnerInterconnectAttachments()
	networkConnects, err := pias.GetPartnerInterconnectAttachment(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerNetworkConnect{
		PartnerNetworkConnects: do.PartnerNetworkConnects{*networkConnects},
	}
	return c.Display(item)
}

// RunPartnerAttachmentList lists Partner Attachment
func RunPartnerAttachmentList(c *CmdConfig) error {

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	pias := c.PartnerInterconnectAttachments()
	list, err := pias.ListPartnerInterconnectAttachments()
	if err != nil {
		return err
	}

	item := &displayers.PartnerNetworkConnect{PartnerNetworkConnects: list}
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
	iaID := c.Args[0]

	r := new(godo.PartnerNetworkConnectUpdateRequest)
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

	interconnectAttachment, err := c.PartnerInterconnectAttachments().UpdatePartnerInterconnectAttachment(iaID, r)
	if err != nil {
		return err
	}

	item := &displayers.PartnerNetworkConnect{
		PartnerNetworkConnects: do.PartnerNetworkConnects{*interconnectAttachment},
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

	pias := c.PartnerInterconnectAttachments()
	regenerateServiceKey, err := pias.RegenerateServiceKey(iaID)
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
	iaID := c.Args[0]

	pias := c.PartnerInterconnectAttachments()
	bgpAuthKey, err := pias.GetBGPAuthKey(iaID)
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
	iaID := c.Args[0]

	pias := c.PartnerInterconnectAttachments()
	serviceKey, err := pias.GetServiceKey(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerAttachmentServiceKey{
		Key: *serviceKey,
	}
	return c.Display(item)
}

// RunPartnerNetworkAttachmentDelete deletes an existing Partner Network Attachment by its identifier.
func RunPartnerNetworkAttachmentDelete(c *CmdConfig) error {

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

	if force || AskForConfirmDelete("Partner Network Connect", 1) == nil {

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
			notice("Partner Network Connect is in progress, waiting for Partner Network Connect to be deleted")

			err := waitForPIA(pias, iaID, "DELETED", true)
			if err != nil {
				return fmt.Errorf("Partner Network Connect couldn't be deleted : %v", err)
			}
			notice("Partner Network Connect is successfully deleted")
		} else {
			notice("Partner Network Connect deletion request accepted")
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

	pias := c.PartnerInterconnectAttachments()
	routeList, err := pias.ListPartnerAttachmentRoutes(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerAttachmentRoute{PartnerAttachmentRoutes: routeList}
	return c.Display(item)
}

func waitForPIA(pias do.PartnerNetworkConnectsService, iaID string, wantStatus string, terminateOnNotFound bool) error {
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
			return fmt.Errorf("Partner Network Connect (%s) entered status `%s`", iaID, errStatus)
		}

		if interconnectAttachment.State == wantStatus {
			return nil
		}

		attempts++
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for Partner Network Connect (%s) to become %s", iaID, wantStatus)
}
