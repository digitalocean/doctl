package commands

import (
	"fmt"
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// VPCPeerings creates the vpc peerings command.
func VPCPeerings() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "vpc-peerings",
			Short: "Display commands that manage VPC Peerings",
			Long: `The commands under ` + "`" + `doctl vpc peerings` + "`" + ` are for managing your VPC Peerings.

With the VPC Peerings commands, you can get, list, create, update, or delete VPC Peerings, and manage their configuration details.`,
			GroupID: manageResourcesGroup,
		},
	}

	peeringDetails := `
- The VPC Peering ID
- The VPC Peering Name
- The First VPC network's ID
- The Second VPC network's ID
- The VPC Peering Status
- The VPC Peering creation date, in ISO8601 combined date and time format
`
	cmdPeeringGet := CmdBuilder(cmd, RunVPCPeeringGet, "get <id>",
		"Retrieve a VPC network", "Retrieve information about a VPC Peering, including:"+peeringDetails, Writer,
		aliasOpt("g"), displayerType(&displayers.VPCPeering{}))
	cmdPeeringGet.Example = `The following example retrieves information about a VPC Peering with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl vpc-peerings get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPeeringList := CmdBuilder(cmd, RunVPCPeeringList, "list", "List VPC Peerings", "Retrieves a list of the VPC Peerings on your account, including the following information for each:"+peeringDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.VPC{}))
	AddStringFlag(cmdPeeringList, doctl.ArgVPCPeeringVPCID, "", "",
		"VPC ID")
	cmdPeeringList.Example = `The following example lists the VPC Peerings on your account : doctl vpc-peerings list --format Name,FirstVpcID,SecondVpcID`

	cmdPeeringCreate := CmdBuilder(cmd, RunVPCPeeringCreate, "create",
		"Create a new VPC Peering", "Use this command to create a new VPC Peering on your account.", Writer, aliasOpt("c"))
	AddStringFlag(cmdPeeringCreate, doctl.ArgVPCPeeringName, "", "",
		"The VPC Peering's name", requiredOpt())
	AddStringFlag(cmdPeeringCreate, doctl.ArgVPCPeeringFirstVPCID, "", "",
		"First VPC ID")
	AddStringFlag(cmdPeeringCreate, doctl.ArgVPCPeeringSecondVPCID, "", "",
		"Second VPC ID")
	cmdPeeringCreate.Example = `The following example creates a VPC Peering named ` +
		"`" + `example-peering` + "`" +
		` : doctl vpc-peerings create --name example-peering --first-vpc-id f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --second-vpc-id 3f900b61-30d7-40d8-9711-8c5d6264b268`

	cmdPeeringUpdate := CmdBuilder(cmd, RunVPCPeeringUpdate, "update <id>",
		"Update a VPC Peering's configuration", `Updates a VPC network's configuration. You can update its name.`, Writer, aliasOpt("u"))
	AddStringFlag(cmdPeeringUpdate, doctl.ArgVPCPeeringName, "", "",
		"The VPC network's name")
	cmdPeeringUpdate.Example = `The following example updates the name of a VPC Peering with the ID ` +
		"`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to ` + "`" + `new-name` + "`" +
		`: doctl vpc-peerings update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name new-name`

	cmdPeeringDelete := CmdBuilder(cmd, RunVPCPeeringDelete, "delete <id>",
		"Permanently delete a VPC Peering", `Permanently deletes the specified VPC Peering. This is irreversible.`, Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdPeeringDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the VPC Peering without a confirmation prompt")
	cmdPeeringDelete.Example = `The following example deletes the VPC Peering with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl vpc-peerings delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	return cmd
}

// RunVPCPeeringGet retrieves an existing VPC Peering by its identifier.
func RunVPCPeeringGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	peeringID := c.Args[0]

	peering, err := c.VPCs().GetPeering(peeringID)
	if err != nil {
		return err
	}

	item := &displayers.VPCPeering{VPCPeerings: do.VPCPeerings{*peering}}
	return c.Display(item)
}

// RunVPCPeeringCreate creates a new VPC Peering with a given configuration.
func RunVPCPeeringCreate(c *CmdConfig) error {

	r := new(godo.VPCPeeringCreateRequest)
	name, err := c.Doit.GetString(c.NS, doctl.ArgVPCPeeringName)
	if err != nil {
		return err
	}
	r.Name = name

	firstVpcID, err := c.Doit.GetString(c.NS, doctl.ArgVPCPeeringFirstVPCID)
	if err != nil {
		return err
	}

	secondVPCID, err := c.Doit.GetString(c.NS, doctl.ArgVPCPeeringSecondVPCID)
	if err != nil {
		return err
	}

	r.VPCIDs = []string{firstVpcID, secondVPCID}

	peering, err := c.VPCs().CreateVPCPeering(r)
	if err != nil {
		return err
	}

	item := &displayers.VPCPeering{VPCPeerings: do.VPCPeerings{*peering}}
	return c.Display(item)
}

// RunVPCPeeringList lists VPC Peerings
func RunVPCPeeringList(c *CmdConfig) error {

	vpcID, err := c.Doit.GetString(c.NS, doctl.ArgVPCPeeringVPCID)
	if err != nil {
		return err
	}

	var list do.VPCPeerings
	if vpcID == "" {
		list, err = c.VPCs().ListVPCPeerings()
		if err != nil {
			return err
		}
	} else {
		list, err = c.VPCs().ListVPCPeeringsByVPCID(vpcID)
		if err != nil {
			return err
		}
	}

	item := &displayers.VPCPeering{VPCPeerings: list}
	return c.Display(item)
}

// RunVPCPeeringUpdate updates an existing VPC  Peering with new configuration.
func RunVPCPeeringUpdate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	peeringID := c.Args[0]

	r := new(godo.VPCPeeringUpdateRequest)
	name, err := c.Doit.GetString(c.NS, doctl.ArgVPCPeeringName)
	if err != nil {
		return err
	}
	r.Name = name

	peering, err := c.VPCs().UpdateVPCPeering(peeringID, r)
	if err != nil {
		return err
	}

	item := &displayers.VPCPeering{VPCPeerings: do.VPCPeerings{*peering}}
	return c.Display(item)
}

// RunVPCPeeringDelete deletes a VPC Peering by its identifier.
func RunVPCPeeringDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	peeringID := c.Args[0]

	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("VPC Peering", 1) == nil {
		vpcs := c.VPCs()
		if err := vpcs.DeleteVPCPeering(peeringID); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("operation aborted")
	}
	return nil
}
