package commands

import (
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func EgressGateway() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "egress-gateway",
			Aliases: []string{"eg"},
			Short:   "Display commands to manage Egress Gateways",
			Long: `Use the subcommands of ` + "`" + `doctl compute egress-gateway` + "`" + ` to perform actions on Egress Gateways.

You can use egress-gateway to perform CRUD operations on an Egress Gateway.`,
		},
	}

	cmdEgressGatewayCreate := CmdBuilder(cmd, RunEgressGatewayCreate, "create", "Create a new Egress Gateway", "", Writer, displayerType(&displayers.EgressGateways{}))

	cmdEgressGatewayUpdate := CmdBuilder(cmd, RunEgressGatewayUpdate, "update <gateway-id>", "Update an active Egress Gateway", "", Writer, displayerType(&displayers.EgressGateways{}))

	for _, c := range []*Command{
		cmdEgressGatewayCreate,
		cmdEgressGatewayUpdate,
	} {
		AddStringFlag(c, doctl.ArgEgressGatewayName, "", "", "Name of the Egress Gateway", requiredOpt())
		AddStringFlag(c, doctl.ArgEgressGatewayType, "", "", "Gateway type", requiredOpt())
		AddStringFlag(c, doctl.ArgEgressGatewayRegion, "", "", "Gateway region", requiredOpt())
		AddStringSliceFlag(c, doctl.ArgEgressGatewayVPCs, "", []string{}, "Ingress VPCs", requiredOpt())
		AddIntFlag(c, doctl.ArgEgressGatewayUDPTimeout, "", 0, "UDP connection timeout (seconds)")
		AddIntFlag(c, doctl.ArgEgressGatewayICMPTimeout, "", 0, "ICMP connection timeout (seconds)")
		AddIntFlag(c, doctl.ArgEgressGatewayTCPTimeout, "", 0, "TCP connection timeout (seconds)")
	}

	CmdBuilder(cmd, RunEgressGatewayGet, "get <gateway-id>", "Get an Egress Gateway", "", Writer, displayerType(&displayers.EgressGateways{}))

	CmdBuilder(cmd, RunEgressGatewayList, "list", "List all active Egress Gateways", "", Writer, displayerType(&displayers.EgressGateways{}), aliasOpt("ls"))

	cmdEgressGatewayDeleteDangerous := CmdBuilder(cmd, RunEgressGatewayDelete, "delete <gateway-id>", "Delete an Egress Gateway", "", Writer)
	AddBoolFlag(cmdEgressGatewayDeleteDangerous, doctl.ArgForce, "", false, "Force delete without a confirmation prompt")

	return cmd
}

func buildEgressGatewayRequestFromArgs(c *CmdConfig, r *godo.EgressGatewayRequest) error {
	var hydrators = []func() error{
		func() error {
			name, err := c.Doit.GetString(c.NS, doctl.ArgEgressGatewayName)
			if err != nil {
				return err
			}
			r.Name = name
			return nil
		},
		func() error {
			gatewayType, err := c.Doit.GetString(c.NS, doctl.ArgEgressGatewayType)
			if err != nil {
				return err
			}
			r.Type = gatewayType
			return nil
		},
		func() error {
			region, err := c.Doit.GetString(c.NS, doctl.ArgEgressGatewayRegion)
			if err != nil {
				return err
			}
			r.Region = region
			return nil
		},
		func() error {
			vpcs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgEgressGatewayVPCs)
			if err != nil {
				return err
			}
			for _, vpc := range vpcs {
				r.VPCs = append(r.VPCs, &godo.IngressVPC{VpcUUID: vpc})
			}
			return nil
		},
		func() error {
			timeout, err := c.Doit.GetInt(c.NS, doctl.ArgEgressGatewayUDPTimeout)
			if err != nil {
				return err
			}
			r.UDPTimeoutSeconds = uint32(timeout)
			return nil
		},
		func() error {
			timeout, err := c.Doit.GetInt(c.NS, doctl.ArgEgressGatewayICMPTimeout)
			if err != nil {
				return err
			}
			r.ICMPTimeoutSeconds = uint32(timeout)
			return nil
		},
		func() error {
			timeout, err := c.Doit.GetInt(c.NS, doctl.ArgEgressGatewayTCPTimeout)
			if err != nil {
				return err
			}
			r.TCPTimeoutSeconds = uint32(timeout)
			return nil
		},
	}
	for _, hydrate := range hydrators {
		if err := hydrate(); err != nil {
			return err
		}
	}
	return nil
}

// RunEgressGatewayCreate creates an Egress Gateway
func RunEgressGatewayCreate(c *CmdConfig) error {
	createReq := new(godo.EgressGatewayRequest)
	if err := buildEgressGatewayRequestFromArgs(c, createReq); err != nil {
		return err
	}
	gateway, err := c.EgressGateways().Create(createReq)
	if err != nil {
		return err
	}
	item := &displayers.EgressGateways{EgressGateways: []*godo.EgressGateway{gateway}}
	return c.Display(item)
}

// RunEgressGatewayUpdate updates an Egress Gateway
func RunEgressGatewayUpdate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	updateReq := new(godo.EgressGatewayRequest)
	if err = buildEgressGatewayRequestFromArgs(c, updateReq); err != nil {
		return err
	}
	gateway, err := c.EgressGateways().Update(id, updateReq)
	if err != nil {
		return err
	}
	item := &displayers.EgressGateways{EgressGateways: []*godo.EgressGateway{gateway}}
	return c.Display(item)
}

// RunEgressGatewayGet retrieves an Egress Gateway
func RunEgressGatewayGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	gateway, err := c.EgressGateways().Get(id)
	if err != nil {
		return err
	}
	item := &displayers.EgressGateways{EgressGateways: []*godo.EgressGateway{gateway}}
	return c.Display(item)
}

// RunEgressGatewayList lists all Egress Gateways
func RunEgressGatewayList(c *CmdConfig) error {
	gateways, err := c.EgressGateways().List()
	if err != nil {
		return err
	}
	item := &displayers.EgressGateways{EgressGateways: gateways}
	return c.Display(item)
}

// RunEgressGatewayDelete deletes an Egress Gateway
func RunEgressGatewayDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if force || AskForConfirmDelete("egress gateway", 1) == nil {
		if err = c.EgressGateways().Delete(id); err != nil {
			return err
		}
	} else {
		return errOperationAborted
	}
	return nil
}
