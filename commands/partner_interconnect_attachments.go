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
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Partner creates the partner interconnect attachments command.
func Partner() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "partner",
			Short:   "Display commands that manage Partner products",
			Long:    `The commands under ` + "`" + `doctl partner` + "`" + ` are for managing Partner products`,
			GroupID: manageResourcesGroup,
		},
	}

	cmd.AddCommand(InterconnectAttachments())

	return cmd
}

// InterconnectAttachments creates the interconnect attachments command.
func InterconnectAttachments() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "interconnect-attachment",
			Short: "Display commands that manage Partner Interconnect Attachments",
			Long: `The commands under ` + "`" + `doctl partner interconnect-attachment` + "`" + ` are for managing your Partner Interconnect Attachments.

With the Partner Interconnect Attachments commands, you can get, list, create, update, or delete Partner Interconnect Attachments, and manage their configuration details.`,
		},
	}

	cmdPartnerIACreate := CmdBuilder(cmd, RunPartnerInterconnectAttachmentCreate, "create",
		"Create a Partner Interconnect Attachment", "Use this command to create a new Partner Interconnect Attachment on your account.", Writer, aliasOpt("c"), displayerType(&displayers.PartnerInterconnectAttachment{}))
	AddStringFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentName, "", "", "Name of the Partner Interconnect Attachment", requiredOpt())
	AddIntFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentConnectionBandwidthInMbps, "", 0, "Connection Bandwidth in Mbps", requiredOpt())
	AddStringFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentRegion, "", "", "Region", requiredOpt())
	AddStringFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentNaaSProvider, "", "", "NaaS Provider", requiredOpt())
	AddStringSliceFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentVPCIDs, "", []string{}, "VPC network IDs", requiredOpt())
	AddIntFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentBGPLocalASN, "", 0, "BGP Local ASN")
	AddStringFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentBGPLocalRouterIP, "", "", "BGP Local Router IP")
	AddIntFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentBGPPeerASN, "", 0, "BGP Peer ASN")
	AddStringFlag(cmdPartnerIACreate, doctl.ArgPartnerInterconnectAttachmentBGPPeerRouterIP, "", "", "BGP Peer Router IP")
	cmdPartnerIACreate.Example = `The following example creates a Partner Interconnect Attachment: doctl partner interconnect-attachment create --name "example-pia" --connection-bandwidth-in-mbps 50 --naas-provider "MEGAPORT" --region "nyc" --vpc-ids "c5537207-ebf0-47cb-bc10-6fac717cd672"`

	return cmd
}

// RunPartnerInterconnectAttachmentCreate creates a new Partner Interconnect Attachment with a given configuration.
func RunPartnerInterconnectAttachmentCreate(c *CmdConfig) error {
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

	bgpLocalASN, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerInterconnectAttachmentBGPLocalASN)
	if err != nil {
		return err
	}
	r.BGP.LocalASN = bgpLocalASN

	bgpLocalRouterIP, err := c.Doit.GetString(c.NS, doctl.ArgPartnerInterconnectAttachmentBGPLocalRouterIP)
	if err != nil {
		return err
	}
	r.BGP.LocalRouterIP = bgpLocalRouterIP

	bgpPeerASN, err := c.Doit.GetInt(c.NS, doctl.ArgPartnerInterconnectAttachmentBGPPeerASN)
	if err != nil {
		return err
	}
	r.BGP.PeerASN = bgpPeerASN

	bgpPeerRouterIP, err := c.Doit.GetString(c.NS, doctl.ArgPartnerInterconnectAttachmentBGPPeerRouterIP)
	if err != nil {
		return err
	}
	r.BGP.PeerRouterIP = bgpPeerRouterIP

	pias := c.PartnerInterconnectAttachments()
	pia, err := pias.Create(r)
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachment{PartnerInterconnectAttachments: do.PartnerInterconnectAttachments{*pia}}
	return c.Display(item)
}
