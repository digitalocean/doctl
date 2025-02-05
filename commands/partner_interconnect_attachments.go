package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
)

// Network creates the partner commands
func Network() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "partner",
			Short:   "Display commands that manage Network products",
			Long:    `The commands under ` + "`" + `doctl network` + "`" + ` are for managing Network products`,
			GroupID: manageResourcesGroup,
		},
	}

	cmd.PersistentFlags().String(doctl.ArgInterconnectAttachmentType, "partner", "Specify interconnect attachment type (e.g., partner)")
	viper.BindPFlag(strings.Join([]string{cmd.Use, doctl.ArgInterconnectAttachmentType}, "."), cmd.PersistentFlags().Lookup(doctl.ArgInterconnectAttachmentType))

	cmd.AddCommand(PartnerInterconnectAttachments())

	return cmd
}

// PartnerInterconnectAttachments creates the partner interconnect attachment command
func PartnerInterconnectAttachments() *Command {
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

	cmdPartnerIAList := CmdBuilder(cmd, RunPartnerInterconnectAttachmentList, "list", "List Network Interconnect Attachments", "Retrieves a list of the Network Interconnect Attachments on your account, including the following information for each:"+interconnectAttachmentDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.PartnerInterconnectAttachment{}))
	cmdPartnerIAList.Example = `The following example lists the Network Interconnect Attachments on your account : doctl network interconnect-attachment list --format Name,VPCIDs`

	return cmd
}

func ensurePartnerAttachmentType(c *CmdConfig) error {
	attachmentType, err := c.Doit.GetString("network", doctl.ArgInterconnectAttachmentType)
	if err != nil {
		return err
	}
	if attachmentType != "partner" {
		return fmt.Errorf("unsupported attachment type: %s", attachmentType)
	}
	return nil
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

	if err := ensurePartnerAttachmentType(c); err != nil {
		return err
	}

	list, err := c.VPCs().ListPartnerInterconnectAttachments()
	if err != nil {
		return err
	}

	item := &displayers.PartnerInterconnectAttachment{PartnerInterconnectAttachments: list}
	return c.Display(item)
}
