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
	"github.com/spf13/cobra"
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

	//	cmd.PersistentFlags().String(doctl.ArgInterconnectAttachmentType, "partner", "Specify interconnect attachment type (e.g., partner)")
	//	viper.BindPFlag(strings.Join([]string{cmd.Use, doctl.ArgInterconnectAttachmentType}, "."), cmd.PersistentFlags().Lookup(doctl.ArgInterconnectAttachmentType))

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
	cmdPartnerIAGet.Example = `The following example retrieves information about a Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl network --type "partner" interconnect-attachment get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPartnerIAList := CmdBuilder(cmd, RunPartnerInterconnectAttachmentList, "list", "List Network Interconnect Attachments", "Retrieves a list of the Network Interconnect Attachments on your account, including the following information for each:"+interconnectAttachmentDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.PartnerInterconnectAttachment{}))
	cmdPartnerIAList.Example = `The following example lists the Network Interconnect Attachments on your account :" + 
		" doctl network --type "partner" interconnect-attachment list --format Name,VPCIDs`

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
