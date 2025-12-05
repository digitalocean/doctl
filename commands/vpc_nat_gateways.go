package commands

import (
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func VPCNATGateway() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "vpc-nat-gateway",
			Aliases: []string{"vng"},
			Short:   "Display commands to manage VPC NAT Gateways",
			Long: `Use the subcommands of ` + "`" + `doctl compute vpc-nat-gateway` + "`" + ` to perform actions on VPC NAT Gateways.

You can use vpc-nat-gateway to perform CRUD operations on a VPC NAT Gateway.`,
		},
	}

	cmdVPCNATGatewayCreate := CmdBuilder(cmd, RunVPCNATGatewayCreate, "create", "Create a new VPC NAT Gateway", "", Writer, displayerType(&displayers.VPCNATGateways{}), aliasOpt("c"))

	cmdVPCNATGatewayUpdate := CmdBuilder(cmd, RunVPCNATGatewayUpdate, "update <gateway-id>", "Update an active VPC NAT Gateway", "", Writer, displayerType(&displayers.VPCNATGateways{}), aliasOpt("u"))

	for _, c := range []*Command{
		cmdVPCNATGatewayCreate,
		cmdVPCNATGatewayUpdate,
	} {
		AddStringFlag(c, doctl.ArgVPCNATGatewayName, "", "", "Name of the VPC NAT Gateway", requiredOpt())
		AddStringFlag(c, doctl.ArgVPCNATGatewayType, "", "PUBLIC", "Gateway type")
		AddStringFlag(c, doctl.ArgVPCNATGatewayRegion, "", "", "Gateway region", requiredOpt())
		AddIntFlag(c, doctl.ArgVPCNATGatewaySize, "", 1, "Gateway size")
		AddStringSliceFlag(c, doctl.ArgVPCNATGatewayVPCs, "", []string{}, "Ingress VPCs, takes a kv-pair of Ingress VPC ID and optional 'default' to indicate the gateway to be set as default for the VPC"+
			" (e.g. --vpcs 6df2c5f4-d2da-4bce-b8dc-e9d2b7bd5db6:default,abcd8994-7f1b-4512-bc2e-13d47ca68632)")
		AddIntFlag(c, doctl.ArgVPCNATGatewayUDPTimeout, "", 30, "UDP connection timeout (seconds)")
		AddIntFlag(c, doctl.ArgVPCNATGatewayICMPTimeout, "", 30, "ICMP connection timeout (seconds)")
		AddIntFlag(c, doctl.ArgVPCNATGatewayTCPTimeout, "", 300, "TCP connection timeout (seconds)")
	}

	AddStringFlag(cmdVPCNATGatewayCreate, doctl.ArgProjectID, "", "",
		"Indicates which project to associate the VPC NAT Gateway with. If not specified, the VPC NAT Gateway will be placed in your default project.")

	CmdBuilder(cmd, RunVPCNATGatewayGet, "get <gateway-id>", "Get a VPC NAT Gateway", "", Writer, displayerType(&displayers.VPCNATGateways{}))

	CmdBuilder(cmd, RunVPCNATGatewayList, "list", "List all active VPC NAT Gateways", "", Writer, displayerType(&displayers.VPCNATGateways{}), aliasOpt("ls"))

	cmdVPCNATGatewayDelete := CmdBuilder(cmd, RunVPCNATGatewayDelete, "delete <gateway-id>", "Delete a VPC NAT Gateway", "", Writer, aliasOpt("d", "rm"))
	AddBoolFlag(cmdVPCNATGatewayDelete, doctl.ArgForce, "", false, "Force delete without a confirmation prompt")

	return cmd
}

type requestType int

const (
	createRequestType requestType = iota + 1
	updateRequestType
)

func buildVPCNATGatewayRequestFromArgs(c *CmdConfig, r *godo.VPCNATGatewayRequest, requestType requestType) error {
	var hydrators = []func() error{
		func() error {
			name, err := c.Doit.GetString(c.NS, doctl.ArgVPCNATGatewayName)
			if err != nil {
				return err
			}
			r.Name = name
			return nil
		},
		func() error {
			gatewayType, err := c.Doit.GetString(c.NS, doctl.ArgVPCNATGatewayType)
			if err != nil {
				return err
			}
			r.Type = gatewayType
			return nil
		},
		func() error {
			region, err := c.Doit.GetString(c.NS, doctl.ArgVPCNATGatewayRegion)
			if err != nil {
				return err
			}
			r.Region = region
			return nil
		},
		func() error {
			size, err := c.Doit.GetInt(c.NS, doctl.ArgVPCNATGatewaySize)
			if err != nil {
				return err
			}
			r.Size = uint32(size)
			return nil
		},
		func() error {
			vpcs, err := c.Doit.GetStringSlice(c.NS, doctl.ArgVPCNATGatewayVPCs)
			if err != nil {
				return err
			}
			for _, vpc := range vpcs {
				if pieces := strings.Split(vpc, ":"); len(pieces) > 0 {
					r.VPCs = append(r.VPCs, &godo.IngressVPC{
						VpcUUID: pieces[0],
						DefaultGateway: func() bool {
							if len(pieces) > 1 && strings.EqualFold(pieces[1], "default") {
								return true
							}
							return false
						}(),
					})
				}
			}
			return nil
		},
		func() error {
			timeout, err := c.Doit.GetInt(c.NS, doctl.ArgVPCNATGatewayUDPTimeout)
			if err != nil {
				return err
			}
			r.UDPTimeoutSeconds = uint32(timeout)
			return nil
		},
		func() error {
			timeout, err := c.Doit.GetInt(c.NS, doctl.ArgVPCNATGatewayICMPTimeout)
			if err != nil {
				return err
			}
			r.ICMPTimeoutSeconds = uint32(timeout)
			return nil
		},
		func() error {
			timeout, err := c.Doit.GetInt(c.NS, doctl.ArgVPCNATGatewayTCPTimeout)
			if err != nil {
				return err
			}
			r.TCPTimeoutSeconds = uint32(timeout)
			return nil
		},
	}
	if requestType == createRequestType {
		hydrators = append(hydrators,
			func() error {
				projectID, err := c.Doit.GetString(c.NS, doctl.ArgProjectID)
				if err != nil {
					return err
				}
				r.ProjectID = projectID
				return nil
			},
		)
	}
	for _, hydrate := range hydrators {
		if err := hydrate(); err != nil {
			return err
		}
	}
	return nil
}

// RunVPCNATGatewayCreate creates a VPC NAT Gateway
func RunVPCNATGatewayCreate(c *CmdConfig) error {
	createReq := new(godo.VPCNATGatewayRequest)
	if err := buildVPCNATGatewayRequestFromArgs(c, createReq, createRequestType); err != nil {
		return err
	}
	gateway, err := c.VPCNATGateways().Create(createReq)
	if err != nil {
		return err
	}
	item := &displayers.VPCNATGateways{VPCNATGateways: []*godo.VPCNATGateway{gateway}}
	return c.Display(item)
}

// RunVPCNATGatewayUpdate updates a VPC NAT Gateway
func RunVPCNATGatewayUpdate(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	updateReq := new(godo.VPCNATGatewayRequest)
	if err = buildVPCNATGatewayRequestFromArgs(c, updateReq, updateRequestType); err != nil {
		return err
	}
	gateway, err := c.VPCNATGateways().Update(id, updateReq)
	if err != nil {
		return err
	}
	item := &displayers.VPCNATGateways{VPCNATGateways: []*godo.VPCNATGateway{gateway}}
	return c.Display(item)
}

// RunVPCNATGatewayGet retrieves a VPC NAT Gateway
func RunVPCNATGatewayGet(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	gateway, err := c.VPCNATGateways().Get(id)
	if err != nil {
		return err
	}
	item := &displayers.VPCNATGateways{VPCNATGateways: []*godo.VPCNATGateway{gateway}}
	return c.Display(item)
}

// RunVPCNATGatewayList lists all VPC NAT Gateways
func RunVPCNATGatewayList(c *CmdConfig) error {
	gateways, err := c.VPCNATGateways().List()
	if err != nil {
		return err
	}
	item := &displayers.VPCNATGateways{VPCNATGateways: gateways}
	return c.Display(item)
}

// RunVPCNATGatewayDelete deletes a VPC NAT Gateway
func RunVPCNATGatewayDelete(c *CmdConfig) error {
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	id := c.Args[0]
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}
	if force || AskForConfirmDelete("vpc nat gateway", 1) == nil {
		if err = c.VPCNATGateways().Delete(id); err != nil {
			return err
		}
	} else {
		return errOperationAborted
	}
	return nil
}
