package commands

import (
	"github.com/spf13/cobra"

	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
)

// Partner creates the partner commands
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

// InterconnectAttachments creates the interconnect attachment command
func InterconnectAttachments() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "interconnect-attachment",
			Short: "Display commands that manage Partner Interconnect Attachments",
			Long: `The commands under ` + "`" + `doctl partner interconnect-attachment` + "`" + ` are for managing your Partner Interconnect Attachments.
With the Partner Interconnect Attachments commands, you can get or list, create, update, or delete Partner Interconnect Attachments, and manage their configuration details.`,
		},
	}

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
- The Partner Interconnect Attachment BGP Peer Router IP
`

	cmdPartnerIAGet := CmdBuilder(cmd, RunPartnerInterconnectAttachmentGet, "get <interconnect-attachment-id>",
		"Retrieves a Partner Interconnect Attachment", "Retrieves information about a Partner Interconnect Attachment, including:"+interconnectAttachmentDetails, Writer,
		aliasOpt("g"), displayerType(&displayers.PartnerInterconnectAttachment{}))
	cmdPartnerIAGet.Example = `The following example retrieves information about a Partner Interconnect Attachment with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl partner interconnect-attachment get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPartnerIAList := CmdBuilder(cmd, RunPartnerInterconnectAttachmentList, "list", "List Partner Interconnect Attachments", "Retrieves a list of the Partner Interconnect Attachments on your account, including the following information for each:"+interconnectAttachmentDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.PartnerInterconnectAttachment{}))
	cmdPartnerIAList.Example = `The following example lists the Partner Interconnect Attachments on your account : doctl partner interconnect-attachment list --format Name,VPCIDs`

	return cmd
}

// RunPartnerInterconnectAttachmentGet retrieves an existing Partner Interconnect Attachment by its identifier.
func RunPartnerInterconnectAttachmentGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	iaID := c.Args[0]

	interconnectAttachment, err := c.VPCs().GetPartnerInterconnectAttachment(iaID)
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

	list, err := c.VPCs().ListPartnerInterconnectAttachments()
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachment{PartnerInterconnectAttachments: list}
	return c.Display(item)
}
