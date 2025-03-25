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

	cmd.AddCommand(PartnerNetworkConnects())

	return cmd
}

// PartnerNetworkConnects creates the partner network connects command.
func PartnerNetworkConnects() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "connect",
			Short: "Display commands that manage Partner Attachment",
			Long: `The commands under ` + "`" + `doctl network connect` + "`" + ` are for managing your Partner Attachment.

With the Partner Attachment commands, you can get, list, create, update, or delete Partner Attachment, and manage their configuration details.`,
		},
	}

	cmdPartnerNCCreate := CmdBuilder(cmd, RunPartnerNCCreate, "create",
		"Create a Partner Attachment", "Use this command to create a new Partner Attachment on your account.", Writer, aliasOpt("c"), displayerType(&displayers.PartnerNetworkConnect{}))
	AddStringFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCType, "", "partner", "Specify connect type (e.g., partner)")

	AddStringFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCName, "", "", "Name of the Partner Attachment", requiredOpt())
	AddIntFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCBandwidthInMbps, "", 0, "Connection Bandwidth in Mbps", requiredOpt())
	AddStringFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCRegion, "", "", "Region", requiredOpt())
	AddStringFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCNaaSProvider, "", "", "NaaS Provider", requiredOpt())
	AddStringSliceFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCVPCIDs, "", []string{}, "VPC network IDs", requiredOpt())
	AddIntFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCBGPLocalASN, "", 0, "BGP Local ASN")
	AddStringFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCBGPLocalRouterIP, "", "", "BGP Local Router IP")
	AddIntFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCBGPPeerASN, "", 0, "BGP Peer ASN")
	AddStringFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCBGPPeerRouterIP, "", "", "BGP Peer Router IP")
	AddStringFlag(cmdPartnerNCCreate, doctl.ArgPartnerNCBGPAuthKey, "", "", "BGP Auth Key")
	cmdPartnerNCCreate.Example = `The following example creates a Partner Attachment: doctl network connect create --name "example-pia" --connection-bandwidth-in-mbps 50 --naas-provider "MEGAPORT" --region "nyc" --vpc-ids "c5537207-ebf0-47cb-bc10-6fac717cd672"`

	partnerNetworkConnectDetails := `
- The Partner Attachment Connect ID
- The Partner Attachment Connect Name
- The Partner Attachment Connect State
- The Partner Attachment Connect Connection Bandwidth in Mbps
- The Partner Attachment Connect Region
- The Partner Attachment Connect NaaS Provider
- The Partner Attachment Connect VPC network IDs
- The Partner Attachment Connect creation date, in ISO8601 combined date and time format
- The Partner Attachment Connect BGP Local ASN
- The Partner Attachment Connect BGP Local Router IP
- The Partner Attachment Connect BGP Peer ASN
- The Partner Attachment Connect BGP Peer Router IP`

	cmdPartnerNCGet := CmdBuilder(cmd, RunPartnerNCGet, "get <partner-attachment-id>",
		"Retrieves a Partner Attachment",
		"Retrieves information about a Partner Attachment, including:"+partnerNetworkConnectDetails, Writer,
		aliasOpt("g"), displayerType(&displayers.PartnerNetworkConnect{}))
	AddStringFlag(cmdPartnerNCGet, doctl.ArgPartnerNCType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerNCGet.Example = `The following example retrieves information about a Partner Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" connect get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPartnerNCList := CmdBuilder(cmd, RunPartnerNCList, "list", "List Partner Attachment",
		"Retrieves a list of the Partner Attachment on your account, including the following information for each:"+partnerNetworkConnectDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.PartnerNetworkConnect{}))
	AddStringFlag(cmdPartnerNCList, doctl.ArgPartnerNCType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerNCList.Example = `The following example lists the Partner Attachment on your account :` +
		` doctl network --type "partner" connect list --format Name,VPCIDs `

	cmdPartnerNCDelete := CmdBuilder(cmd, RunPartnerNCDelete, "delete <partner-attachment-id>",
		"Deletes a Partner Attachment",
		"Deletes information about a Partner Attachment. This is irreversible ", Writer,
		aliasOpt("rm"), displayerType(&displayers.PartnerNetworkConnect{}))
	AddBoolFlag(cmdPartnerNCDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the Partner Attachment without any confirmation prompt")
	AddBoolFlag(cmdPartnerNCDelete, doctl.ArgCommandWait, "", false,
		"Boolean that specifies whether to wait for a Partner Attachment deletion to complete before returning control to the terminal")
	AddStringFlag(cmdPartnerNCDelete, doctl.ArgPartnerNCType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerNCDelete.Example = `The following example deletes a Partner Attachments with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" connect delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPartnerNCUpdate := CmdBuilder(cmd, RunPartnerNCUpdate, "update <partner-attachment-id>",
		"Update a Partner Attachments name and configuration",
		`Use this command to update the name and and configuration of a Partner Attachment`, Writer, aliasOpt("u"))
	AddStringFlag(cmdPartnerNCUpdate, doctl.ArgPartnerNCType, "", "partner", "Specify connect type (e.g., partner)")
	AddStringFlag(cmdPartnerNCUpdate, doctl.ArgPartnerNCName, "", "",
		"The Partner Attachment name", requiredOpt())
	AddStringFlag(cmdPartnerNCUpdate, doctl.ArgPartnerNCVPCIDs, "", "",
		"The Partner Attachment vpc ids", requiredOpt())
	cmdPartnerNCUpdate.Example = `The following example updates the name of a Partner Attachment with the ID ` +
		"`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to ` + "`" + `new-name` + "`" +
		`: doctl network --type "partner" connect update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name "new-name" --
vpc-ids "270a76ed-1bb7-4c5d-a6a5-e863de086940"`

	partnerNCRouteDetails := `
- The Partner Attachment ID
- The Partner Attachment Cidr`

	cmdPartnerNCRouteList := CmdBuilder(cmd, RunPartnerNCRouteList, "list-routes <partner-attachment-id>",
		"List Partner Attachment Routes",
		"Retrieves a list of the Partner Attachment Routes on your account, including the following information for each:"+partnerNCRouteDetails, Writer,
		aliasOpt("ls-routes"), displayerType(&displayers.PartnerNetworkConnect{}))
	AddStringFlag(cmdPartnerNCRouteList, doctl.ArgPartnerNCType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerNCRouteList.Example = `The following example lists the Partner Attachment Routes on your account :` +
		` doctl network --type "partner" connect list-routes f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --format ID,Cidr `

	cmdPartnerNCRegenerateServiceKey := CmdBuilder(cmd, RunPartnerNCRegenerateServiceKey, "regenerate-service-key <partner-attachment-id>",
		"Regenerates a Service key of Partner Attachment",
		"Regenerates information about a Service key of Partner Attachment", Writer,
		aliasOpt("regen-service-key"), displayerType(&displayers.PartnerNCRegenerateServiceKey{}))
	AddStringFlag(cmdPartnerNCRegenerateServiceKey, doctl.ArgPartnerNCType, "", "partner", "Specify connect type (e.g., partner)")
	cmdPartnerNCRegenerateServiceKey.Example = `The following example retrieves information about a Service key of Partner Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" connect regenerate-service-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdGetPartnerNCGetBGPAuthKey := CmdBuilder(cmd, RunGetPartnerNCBGPAuthKey, "get-bgp-auth-key <partner-attachment-id>",
		"Retrieves a BGP Auth key of Partner Attachment",
		"Retrieves information about a BGP Auth key of Partner Attachment", Writer,
		aliasOpt("g-bgp-auth-key"), displayerType(&displayers.PartnerNCBgpAuthKey{}))
	AddStringFlag(cmdGetPartnerNCGetBGPAuthKey, doctl.ArgPartnerNCType, "", "partner", "Specify connect type (e.g., partner)")
	cmdGetPartnerNCGetBGPAuthKey.Example = `The following example retrieves information about a Service key of Partner Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" connect get-bgp-auth-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	partnerNCServiceKeyDetails := `
- The Service key Value
- The Service key State
- The Service key CreatedAt`

	cmdGetPartnerIAServiceKey := CmdBuilder(cmd, RunGetPartnerNCServiceKey, "get-service-key <partner-attachment-id>",
		"Retrieves a Service key of Partner Attachment",
		"Retrieves information about a Service key of Partner Attachment, including:"+partnerNCServiceKeyDetails, Writer,
		aliasOpt("g-service-key"), displayerType(&displayers.PartnerNCServiceKey{}))
	AddStringFlag(cmdGetPartnerIAServiceKey, doctl.ArgPartnerNCType, "", "partner", "Specify connect type (e.g., partner)")
	cmdGetPartnerIAServiceKey.Example = `The following example retrieves information about a Service key of Partner Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" connect get-service-key f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	return cmd
}

func ensurePartnerConnectType(c *CmdConfig) error {
	attachmentType, err := c.Doit.GetString(c.NS, doctl.ArgPartnerNCType)
	if err != nil {
		return err
	}
	if attachmentType != "partner" {
		return fmt.Errorf("unsupported attachment type: %s", attachmentType)
	}
	return nil
}

// RunPartnerNCCreate creates a new Partner Network Connect with a given configuration.
func RunPartnerNCCreate(c *CmdConfig) error {
	if err := ensurePartnerConnectType(c); err != nil {
		return err
	}

	r := new(godo.PartnerNetworkConnectCreateRequest)
	name, err := c.Doit.GetString(c.NS, doctl.ArgPartnerNCName)
	if err != nil {
		return err
	}
	r.Name = name

	connBandwidth, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerNCBandwidthInMbps)
	if err != nil {
		return err
	}
	r.ConnectionBandwidthInMbps = connBandwidth

	region, err := c.Doit.GetString(c.NS, doctl.ArgPartnerNCRegion)
	if err != nil {
		return err
	}
	r.Region = region

	naasProvider, err := c.Doit.GetString(c.NS, doctl.ArgPartnerNCNaaSProvider)
	if err != nil {
		return err
	}
	r.NaaSProvider = naasProvider

	vpcIDs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgPartnerNCVPCIDs)
	if err != nil {
		return err
	}
	r.VPCIDs = vpcIDs

	bgpConfig := new(godo.BGP)

	bgpLocalASN, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerNCBGPLocalASN)
	if err != nil {
		return err
	}
	bgpConfig.LocalASN = bgpLocalASN

	bgpLocalRouterIP, err := c.Doit.GetString(c.NS, doctl.ArgPartnerNCBGPLocalRouterIP)
	if err != nil {
		return err
	}
	bgpConfig.LocalRouterIP = bgpLocalRouterIP

	bgpPeerASN, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerNCBGPPeerASN)
	if err != nil {
		return err
	}
	bgpConfig.PeerASN = bgpPeerASN

	bgpPeerRouterIP, err := c.Doit.GetString(c.NS, doctl.ArgPartnerNCBGPPeerRouterIP)
	if err != nil {
		return err
	}
	bgpConfig.PeerRouterIP = bgpPeerRouterIP

	bgpAuthKey, err := c.Doit.GetString(c.NS, doctl.ArgPartnerNCBGPAuthKey)
	if err != nil {
		bgpConfig.AuthKey = bgpAuthKey
	}

	pncs := c.PartnerNetworkConnects()
	pnc, err := pncs.Create(r)
	if err != nil {
		return err
	}

	item := &displayers.PartnerNetworkConnect{PartnerNetworkConnects: do.PartnerNetworkConnects{*pnc}}
	return c.Display(item)
}

// RunPartnerNCGet retrieves an existing Partner Network Connect by its identifier.
func RunPartnerNCGet(c *CmdConfig) error {

	if err := ensurePartnerConnectType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pncs := c.PartnerNetworkConnects()
	networkConnects, err := pncs.GetPartnerNetworkConnect(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerNetworkConnect{
		PartnerNetworkConnects: do.PartnerNetworkConnects{*networkConnects},
	}
	return c.Display(item)
}

// RunPartnerNCList lists Partner Network Connects
func RunPartnerNCList(c *CmdConfig) error {

	if err := ensurePartnerConnectType(c); err != nil {
		return err
	}

	pias := c.PartnerNetworkConnects()
	list, err := pias.ListPartnerNetworkConnects()
	if err != nil {
		return err
	}

	item := &displayers.PartnerNetworkConnect{PartnerNetworkConnects: list}
	return c.Display(item)
}

// RunPartnerNCUpdate updates an existing Partner Network Connect with new configuration.
func RunPartnerNCUpdate(c *CmdConfig) error {
	if err := ensurePartnerConnectType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	pncID := c.Args[0]

	r := new(godo.PartnerNetworkConnectUpdateRequest)
	name, err := c.Doit.GetString(c.NS, doctl.ArgPartnerNCName)
	if err != nil {
		return err
	}
	r.Name = name

	vpcIDs, err := c.Doit.GetString(c.NS, doctl.ArgPartnerNCVPCIDs)
	if err != nil {
		return err
	}
	r.VPCIDs = strings.Split(vpcIDs, ",")

	pnc, err := c.PartnerNetworkConnects().UpdatePartnerNetworkConnect(pncID, r)
	if err != nil {
		return err
	}

	item := &displayers.PartnerNetworkConnect{
		PartnerNetworkConnects: do.PartnerNetworkConnects{*pnc},
	}
	return c.Display(item)
}

// RunPartnerNCRegenerateServiceKey regenerates a service key of existing Partner Network Connect
func RunPartnerNCRegenerateServiceKey(c *CmdConfig) error {

	if err := ensurePartnerConnectType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pncs := c.PartnerNetworkConnects()
	regenerateServiceKey, err := pncs.RegenerateServiceKey(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerNCRegenerateServiceKey{
		RegenerateKey: *regenerateServiceKey,
	}
	return c.Display(item)
}

// RunGetPartnerNCBGPAuthKey get a bgp auth key of existing Partner Network Connect
func RunGetPartnerNCBGPAuthKey(c *CmdConfig) error {

	if err := ensurePartnerConnectType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	pncID := c.Args[0]

	pncs := c.PartnerNetworkConnects()
	bgpAuthKey, err := pncs.GetBGPAuthKey(pncID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerNCBgpAuthKey{
		Key: *bgpAuthKey,
	}
	return c.Display(item)
}

// RunGetPartnerNCServiceKey retrieves service key of existing Partner Network Connect
func RunGetPartnerNCServiceKey(c *CmdConfig) error {

	if err := ensurePartnerConnectType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	pncID := c.Args[0]

	pncs := c.PartnerNetworkConnects()
	serviceKey, err := pncs.GetServiceKey(pncID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerNCServiceKey{
		Key: *serviceKey,
	}
	return c.Display(item)
}

// RunPartnerNCDelete deletes an existing Partner Network Connect by its identifier.
func RunPartnerNCDelete(c *CmdConfig) error {

	if err := ensurePartnerConnectType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	pncID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("Partner Network Connect", 1) == nil {

		pncs := c.PartnerNetworkConnects()
		err := pncs.DeletePartnerNetworkConnect(pncID)
		if err != nil {
			return err
		}

		wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
		if err != nil {
			return err
		}

		if wait {
			notice("Partner Attachment is in progress, waiting for Partner Attachment to be deleted")

			err := waitForPNC(pncs, pncID, "DELETED", true)
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

// RunPartnerNCRouteList lists Partner Network Connect routes
func RunPartnerNCRouteList(c *CmdConfig) error {
	if err := ensurePartnerConnectType(c); err != nil {
		return err
	}

	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	pncs := c.PartnerNetworkConnects()
	routeList, err := pncs.ListPartnerNetworkConnectRoutes(iaID)
	if err != nil {
		return err
	}

	item := &displayers.PartnerNCRoute{PartnerNetworkConnectRoutes: routeList}
	return c.Display(item)
}

func waitForPNC(pncs do.PartnerNetworkConnectsService, iaID string, wantStatus string, terminateOnNotFound bool) error {
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

		pnc, err := pncs.GetPartnerNetworkConnect(iaID)
		if err != nil {
			if terminateOnNotFound && strings.Contains(err.Error(), "not found") {
				return nil
			}
			return err
		}

		if pnc.PartnerNetworkConnect.State == errStatus {
			return fmt.Errorf("Partner Network Connect (%s) entered status `%s`", iaID, errStatus)
		}

		if pnc.PartnerNetworkConnect.State == wantStatus {
			return nil
		}

		attempts++
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for Partner Network Connect (%s) to become %s", iaID, wantStatus)
}
