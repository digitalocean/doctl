package commands

import (
	"errors"
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

// VPCPeerings creates the vpc peerings command.
func VPCPeerings() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "peerings",
			Short: "Display commands that manage VPC Peerings",
			Long: `The commands under ` + "`" + `doctl vpcs peerings` + "`" + ` are for managing your VPC Peerings.
With the VPC Peerings commands, you can get, list, create, update, or delete VPC Peerings, and manage their configuration details.`,
			Hidden: true,
		},
	}

	peeringDetails := `
- The VPC Peering ID
- The VPC Peering Name
- The Peered VPC network IDs
- The VPC Peering Status
- The VPC Peering creation date, in ISO8601 combined date and time format
`
	cmdPeeringGet := CmdBuilder(cmd, RunVPCPeeringGet, "get <id>",
		"Retrieves a VPC Peering", "Retrieves information about a VPC Peering, including:"+peeringDetails, Writer,
		aliasOpt("g"), displayerType(&displayers.VPCPeering{}))
	cmdPeeringGet.Example = `The following example retrieves information about a VPC Peering with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + `: doctl vpcs peerings get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdPeeringList := CmdBuilder(cmd, RunVPCPeeringList, "list", "List VPC Peerings", "Retrieves a list of the VPC Peerings on your account, including the following information for each:"+peeringDetails, Writer,
		aliasOpt("ls"), displayerType(&displayers.VPCPeering{}))
	AddStringFlag(cmdPeeringList, doctl.ArgVPCPeeringVPCID, "", "",
		"VPC ID")
	cmdPeeringList.Example = `The following example lists the VPC Peerings on your account : doctl vpcs peerings list --format Name,VPCIDs`

	cmdPeeringCreate := CmdBuilder(cmd, RunVPCPeeringCreate, "create",
		"Create a new VPC Peering", "Use this command to create a new VPC Peering on your account.", Writer, aliasOpt("c"))
	AddStringFlag(cmdPeeringCreate, doctl.ArgVPCPeeringVPCIDs, "", "",
		"Peering VPC IDs should be comma separated", requiredOpt())
	AddBoolFlag(cmdPeeringCreate, doctl.ArgCommandWait, "", false, "Boolean that specifies whether to wait for a VPC Peering creation to complete before returning control to the terminal")
	cmdPeeringCreate.Example = `The following example creates a VPC Peering named ` +
		"`" + `example-peering-name` + "`" +
		` : doctl vpcs peerings create example-peering-name --vpc-ids f81d4fae-7dec-11d0-a765-00a0c91e6bf6,3f900b61-30d7-40d8-9711-8c5d6264b268`

	cmdPeeringUpdate := CmdBuilder(cmd, RunVPCPeeringUpdate, "update <id>",
		"Update a VPC Peering's name", `Use this command to update the name of a VPC Peering`, Writer, aliasOpt("u"))
	AddStringFlag(cmdPeeringUpdate, doctl.ArgVPCPeeringName, "", "",
		"The VPC Peering's name", requiredOpt())
	cmdPeeringUpdate.Example = `The following example updates the name of a VPC Peering with the ID ` +
		"`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" + ` to ` + "`" + `new-name` + "`" +
		`: doctl vpcs peerings update f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --name new-name`

	cmdPeeringDelete := CmdBuilder(cmd, RunVPCPeeringDelete, "delete <id>",
		"Permanently delete a VPC Peering", `Permanently deletes the specified VPC Peering. This is irreversible.`, Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdPeeringDelete, doctl.ArgForce, doctl.ArgShortForce, false,
		"Delete the VPC Peering without any confirmation prompt")
	AddBoolFlag(cmdPeeringDelete, doctl.ArgCommandWait, "", false,
		"Boolean that specifies whether to wait for a VPC Peering deletion to complete before returning control to the terminal")
	cmdPeeringDelete.Example = `The following example deletes the VPC Peering with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		`: doctl vpcs peerings delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

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
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	peeringName := c.Args[0]

	r := new(godo.VPCPeeringCreateRequest)
	r.Name = peeringName

	vpcIDs, err := c.Doit.GetString(c.NS, doctl.ArgVPCPeeringVPCIDs)
	if err != nil {
		return err
	}

	for _, v := range strings.Split(vpcIDs, ",") {
		if v == "" {
			return errors.New("VPC ID is empty")
		}

		r.VPCIDs = append(r.VPCIDs, strings.TrimSpace(v))
	}

	if len(r.VPCIDs) != 2 {
		return errors.New("VPC IDs length should be 2")
	}

	vpcService := c.VPCs()
	peering, err := vpcService.CreateVPCPeering(r)
	if err != nil {
		return err
	}

	wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
	if err != nil {
		return err
	}

	if wait {
		notice("VPC Peering creation is in progress, waiting for VPC Peering to become active")

		err := waitForVPCPeering(vpcService, peering.ID, "ACTIVE", false)
		if err != nil {
			return fmt.Errorf("VPC Peering couldn't enter `active` state: %v", err)
		}

		peering, _ = vpcService.GetPeering(peering.ID)
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
	if len(c.Args) == 0 {
		return doctl.NewMissingArgsErr(c.NS)
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

		wait, err := c.Doit.GetBool(c.NS, doctl.ArgCommandWait)
		if err != nil {
			return err
		}

		if wait {
			notice("VPC Peering deletion is in progress, waiting for VPC Peering to be deleted")

			err := waitForVPCPeering(vpcs, peeringID, "DELETED", true)
			if err != nil {
				return fmt.Errorf("VPC Peering couldn't be deleted : %v", err)
			}
			notice("VPC Peering is successfully deleted")
		} else {
			notice("VPC Peering deletion request accepted")
		}

	} else {
		return fmt.Errorf("operation aborted")
	}

	return nil
}

func waitForVPCPeering(vpcService do.VPCsService, peeringID string, wantStatus string, terminateOnNotFound bool) error {
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

		peering, err := vpcService.GetPeering(peeringID)
		if err != nil {
			if terminateOnNotFound && strings.Contains(err.Error(), "not found") {
				return nil
			}
			return err
		}

		if peering.Status == errStatus {
			return fmt.Errorf("VPC Peering (%s) entered status `%s`", peeringID, errStatus)
		}

		if peering.Status == wantStatus {
			return nil
		}

		attempts++
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for VPC Peering (%s) to become %s", peeringID, wantStatus)
}
